/*
 * Copyright (c) 2020-2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package client

import (
	"context"
	"fmt"
	buildBean "github.com/devtron-labs/devtron/pkg/build/pipeline/bean"
	repository4 "github.com/devtron-labs/devtron/pkg/cluster/environment/repository"
	"strings"
	"time"

	bean2 "github.com/devtron-labs/devtron/api/bean"
	repository2 "github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/internal/sql/repository/chartConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/pkg/auth/user/repository"
	"github.com/devtron-labs/devtron/pkg/bean"
	"github.com/devtron-labs/devtron/util/event"
	"github.com/go-pg/pg"
	"github.com/satori/go.uuid"
	"go.uber.org/zap"
)

type EventFactory interface {
	Build(eventType util.EventType, sourceId *int, appId int, envId *int, pipelineType util.PipelineType) (Event, error)
	BuildExtraCDData(event Event, wfr *pipelineConfig.CdWorkflowRunner, pipelineOverrideId int, stage bean2.WorkflowType) Event
	BuildExtraCIData(event Event, material *buildBean.MaterialTriggerInfo) Event
	//BuildFinalData(event Event) *Payload
}

type EventSimpleFactoryImpl struct {
	logger                       *zap.SugaredLogger
	cdWorkflowRepository         pipelineConfig.CdWorkflowRepository
	pipelineOverrideRepository   chartConfig.PipelineOverrideRepository
	ciWorkflowRepository         pipelineConfig.CiWorkflowRepository
	ciPipelineMaterialRepository pipelineConfig.CiPipelineMaterialRepository
	ciPipelineRepository         pipelineConfig.CiPipelineRepository
	pipelineRepository           pipelineConfig.PipelineRepository
	userRepository               repository.UserRepository
	envRepository                repository4.EnvironmentRepository
	ciArtifactRepository         repository2.CiArtifactRepository
}

func NewEventSimpleFactoryImpl(logger *zap.SugaredLogger, cdWorkflowRepository pipelineConfig.CdWorkflowRepository,
	pipelineOverrideRepository chartConfig.PipelineOverrideRepository, ciWorkflowRepository pipelineConfig.CiWorkflowRepository,
	ciPipelineMaterialRepository pipelineConfig.CiPipelineMaterialRepository,
	ciPipelineRepository pipelineConfig.CiPipelineRepository, pipelineRepository pipelineConfig.PipelineRepository,
	userRepository repository.UserRepository, envRepository repository4.EnvironmentRepository, ciArtifactRepository repository2.CiArtifactRepository) *EventSimpleFactoryImpl {
	return &EventSimpleFactoryImpl{
		logger:                       logger,
		cdWorkflowRepository:         cdWorkflowRepository,
		pipelineOverrideRepository:   pipelineOverrideRepository,
		ciWorkflowRepository:         ciWorkflowRepository,
		ciPipelineMaterialRepository: ciPipelineMaterialRepository,
		ciPipelineRepository:         ciPipelineRepository,
		pipelineRepository:           pipelineRepository,
		userRepository:               userRepository,
		ciArtifactRepository:         ciArtifactRepository,
		envRepository:                envRepository,
	}
}

func (impl *EventSimpleFactoryImpl) Build(eventType util.EventType, sourceId *int, appId int, envId *int, pipelineType util.PipelineType) (Event, error) {
	correlationId := uuid.NewV4()
	event := Event{}
	event.EventTypeId = int(eventType)
	if sourceId != nil {
		event.PipelineId = *sourceId
	}
	event.AppId = appId
	if envId != nil && *envId > 0 {
		env, err := impl.envRepository.FindById(*envId)
		if err != nil {
			impl.logger.Errorw("error in getting env", "envId", *envId, "err", err)
			return event, err
		}
		event.EnvId = *envId
		event.ClusterId = env.ClusterId
		event.IsProdEnv = env.Default
	}
	event.PipelineType = string(pipelineType)
	event.CorrelationId = fmt.Sprintf("%s", correlationId)
	event.EventTime = time.Now().Format(bean.LayoutRFC3339)
	return event, nil
}

func (impl *EventSimpleFactoryImpl) BuildExtraCDData(event Event, wfr *pipelineConfig.CdWorkflowRunner, pipelineOverrideId int, stage bean2.WorkflowType) Event {
	//event.CdWorkflowRunnerId =
	event.CdWorkflowType = stage
	payload := event.Payload
	if payload == nil {
		payload = &Payload{}
		payload.Stage = string(stage)
		event.Payload = payload
	}
	if wfr != nil {
		material, err := impl.getCiMaterialInfo(wfr.CdWorkflow.Pipeline.CiPipelineId, wfr.CdWorkflow.CiArtifactId)
		if err != nil {
			impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "event", event, "stage", stage, "workflow runner", wfr, "pipelineOverrideId", pipelineOverrideId)
		}
		payload.MaterialTriggerInfo = material
		payload.DockerImageUrl = wfr.CdWorkflow.CiArtifact.Image
		event.UserId = int(wfr.TriggeredBy)
		event.Payload = payload
		event.CdWorkflowRunnerId = wfr.Id
		event.CiArtifactId = wfr.CdWorkflow.CiArtifactId
	} else if pipelineOverrideId > 0 {
		pipelineOverride, err := impl.pipelineOverrideRepository.FindById(pipelineOverrideId)
		if err != nil {
			impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "event", event, "stage", stage, "workflow runner", wfr, "pipelineOverrideId", pipelineOverrideId)
		}
		if pipelineOverride != nil && pipelineOverride.Id > 0 {
			cdWorkflow, err := impl.cdWorkflowRepository.FindById(pipelineOverride.CdWorkflowId)
			if err != nil {
				impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "cdWorkflow", cdWorkflow, "event", event, "stage", stage, "workflow runner", wfr, "pipelineOverrideId", pipelineOverrideId)
			}
			wfr, err := impl.cdWorkflowRepository.FindByWorkflowIdAndRunnerType(context.Background(), cdWorkflow.Id, stage)
			if err != nil {
				impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "wfr", wfr, "event", event, "stage", stage, "workflow runner", wfr, "pipelineOverrideId", pipelineOverrideId)
			}
			if wfr.Id > 0 {
				event.CdWorkflowRunnerId = wfr.Id
				event.CiArtifactId = pipelineOverride.CiArtifactId

				material, err := impl.getCiMaterialInfo(pipelineOverride.CiArtifact.PipelineId, pipelineOverride.CiArtifactId)
				if err != nil {
					impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "material", material)
				}
				payload.MaterialTriggerInfo = material
				payload.DockerImageUrl = wfr.CdWorkflow.CiArtifact.Image
				event.UserId = int(wfr.TriggeredBy)
			}
		}
		event.Payload = payload
	} else if event.PipelineId > 0 {
		pipeline, err := impl.pipelineRepository.FindById(event.PipelineId)
		if err != nil {
			impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "pipeline", pipeline)
		}
		if pipeline != nil {
			material, err := impl.getCiMaterialInfo(pipeline.CiPipelineId, 0)
			if err != nil {
				impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "material", material)
			}
			payload.MaterialTriggerInfo = material
		}
		event.Payload = payload
	}

	if event.UserId > 0 {
		user, err := impl.userRepository.GetById(int32(event.UserId))
		if err != nil {
			impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "user", user)
		}
		payload = event.Payload
		payload.TriggeredBy = user.EmailId
		event.Payload = payload
	}
	event = impl.addExtraCdDataForEnterprise(event, wfr)
	return event
}

func (impl *EventSimpleFactoryImpl) BuildExtraCIData(event Event, material *buildBean.MaterialTriggerInfo) Event {
	if material == nil {
		materialInfo, err := impl.getCiMaterialInfo(event.PipelineId, event.CiArtifactId)
		if err != nil {
			impl.logger.Errorw("found error on payload build for ci, skipping this error ", "materialInfo", materialInfo)
		}
		material = materialInfo
	} else if material.CiMaterials == nil {
		materialInfo, err := impl.getCiMaterialInfo(event.PipelineId, 0)
		if err != nil {
			impl.logger.Errorw("found error on payload build for ci, skipping this error ", "materialInfo", materialInfo)
		}
		materialInfo.GitTriggers = material.GitTriggers
		material = materialInfo
	}
	payload := event.Payload
	if payload == nil {
		payload = &Payload{}
		event.Payload = payload
	}
	event.Payload.MaterialTriggerInfo = material

	if event.UserId > 0 {
		user, err := impl.userRepository.GetById(int32(event.UserId))
		if err != nil {
			impl.logger.Errorw("found error on payload build for cd stages, skipping this error ", "user", user)
		}
		payload = event.Payload
		payload.TriggeredBy = user.EmailId
		event.Payload = payload
	}

	// fetching all the envs which are directly or indirectly linked with the ci pipeline
	if event.PipelineId > 0 {
		// Get the pipeline to check if it's external
		ciPipeline, err := impl.ciPipelineRepository.FindById(event.PipelineId)
		if err != nil {
			impl.logger.Errorw("error in getting ci pipeline", "pipelineId", event.PipelineId, "err", err)
		} else {
			envs, err := impl.envRepository.FindEnvLinkedWithCiPipelines(ciPipeline.IsExternal, []int{event.PipelineId})
			if err != nil {
				impl.logger.Errorw("error in finding environments linked with ci pipeline", "pipelineId", event.PipelineId, "err", err)
			} else {
				event.EnvIdsForCiPipeline = make([]int, 0, len(envs))
				for _, env := range envs {
					event.EnvIdsForCiPipeline = append(event.EnvIdsForCiPipeline, env.Id)
				}
			}
		}
	}

	return event
}

func (impl *EventSimpleFactoryImpl) getCiMaterialInfo(ciPipelineId int, ciArtifactId int) (*buildBean.MaterialTriggerInfo, error) {
	materialTriggerInfo := &buildBean.MaterialTriggerInfo{}
	if ciPipelineId > 0 {
		ciMaterials, err := impl.ciPipelineMaterialRepository.GetByPipelineId(ciPipelineId)
		if err != nil {
			impl.logger.Errorw("error on fetching materials for", "ciPipelineId", ciPipelineId, "err", err)
			return nil, err
		}

		var ciMaterialsArr []buildBean.CiPipelineMaterialResponse
		for _, m := range ciMaterials {
			if m.GitMaterial == nil {
				impl.logger.Warnw("git material are empty", "material", m)
				continue
			}
			res := buildBean.CiPipelineMaterialResponse{
				Id:              m.Id,
				GitMaterialId:   m.GitMaterialId,
				GitMaterialName: m.GitMaterial.Name[strings.Index(m.GitMaterial.Name, "-")+1:],
				Type:            string(m.Type),
				Value:           m.Value,
				Active:          m.Active,
				Url:             m.GitMaterial.Url,
			}
			ciMaterialsArr = append(ciMaterialsArr, res)
		}
		materialTriggerInfo.CiMaterials = ciMaterialsArr
	}
	if ciArtifactId > 0 {
		ciArtifact, err := impl.ciArtifactRepository.Get(ciArtifactId)
		if err != nil {
			impl.logger.Errorw("error fetching artifact data", "err", err)
			return nil, err
		}

		// handling linked ci pipeline
		if ciArtifact.ParentCiArtifact > 0 && ciArtifact.WorkflowId == nil {
			ciArtifactId = ciArtifact.ParentCiArtifact
		}
		ciWf, err := impl.ciWorkflowRepository.FindLastTriggeredWorkflowByArtifactId(ciArtifactId)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error fetching ci workflow data by artifact", "err", err)
			return nil, err
		}
		if ciWf != nil {
			materialTriggerInfo.GitTriggers = ciWf.GitTriggers
		}
	}
	return materialTriggerInfo, nil
}
