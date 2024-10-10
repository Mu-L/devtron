/*
 * Copyright (c) 2024. Devtron Inc.
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

package devtronApps

import (
	"context"
	"encoding/json"
	"fmt"
	blob_storage "github.com/devtron-labs/common-lib/blob-storage"
	bean2 "github.com/devtron-labs/devtron/api/bean"
	gitSensorClient "github.com/devtron-labs/devtron/client/gitSensor"
	"github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig/bean/workflow/cdWorkflow"
	"github.com/devtron-labs/devtron/internal/util"
	bean6 "github.com/devtron-labs/devtron/pkg/attributes/bean"
	bean4 "github.com/devtron-labs/devtron/pkg/bean"
	repository2 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	bean5 "github.com/devtron-labs/devtron/pkg/deployment/common/bean"
	adapter2 "github.com/devtron-labs/devtron/pkg/deployment/trigger/devtronApps/adapter"
	"github.com/devtron-labs/devtron/pkg/deployment/trigger/devtronApps/bean"
	"github.com/devtron-labs/devtron/pkg/imageDigestPolicy"
	"github.com/devtron-labs/devtron/pkg/pipeline"
	"github.com/devtron-labs/devtron/pkg/pipeline/adapter"
	pipelineConfigBean "github.com/devtron-labs/devtron/pkg/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline/bean/CiPipeline"
	repository3 "github.com/devtron-labs/devtron/pkg/pipeline/history/repository"
	"github.com/devtron-labs/devtron/pkg/pipeline/types"
	"github.com/devtron-labs/devtron/pkg/plugin"
	bean3 "github.com/devtron-labs/devtron/pkg/plugin/bean"
	"github.com/devtron-labs/devtron/pkg/resourceQualifiers"
	"github.com/devtron-labs/devtron/pkg/sql"
	util3 "github.com/devtron-labs/devtron/pkg/util"
	repository5 "github.com/devtron-labs/devtron/pkg/variables/repository"
	util4 "github.com/devtron-labs/devtron/util"
	util2 "github.com/devtron-labs/devtron/util/event"
	"github.com/go-pg/pg"
	"go.opentelemetry.io/otel"
	"maps"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	APP_LABEL_KEY_PREFIX         = "APP_LABEL_KEY"
	APP_LABEL_VALUE_PREFIX       = "APP_LABEL_VALUE"
	APP_LABEL_COUNT              = "APP_LABEL_COUNT"
	CHILD_CD_ENV_NAME_PREFIX     = "CHILD_CD_ENV_NAME"
	CHILD_CD_CLUSTER_NAME_PREFIX = "CHILD_CD_CLUSTER_NAME"
	CHILD_CD_COUNT               = "CHILD_CD_COUNT"
)

func (impl *TriggerServiceImpl) TriggerPreStage(request bean.TriggerRequest) error {
	request.WorkflowType = bean2.CD_WORKFLOW_TYPE_PRE
	//setting triggeredAt variable to have consistent data for various audit log places in db for deployment time
	triggeredAt := time.Now()
	triggeredBy := request.TriggeredBy
	artifact := request.Artifact
	pipeline := request.Pipeline
	ctx := request.TriggerContext.Context
	env, namespace, err := impl.getEnvAndNsIfRunStageInEnv(ctx, request)
	if err != nil {
		impl.logger.Errorw("error, getEnvAndNsIfRunStageInEnv", "err", err, "pipeline", pipeline, "stage", request.WorkflowType)
		return nil
	}
	request.RunStageInEnvNamespace = namespace
	cdWf, runner, err := impl.createStartingWfAndRunner(request, triggeredAt)
	if err != nil {
		impl.logger.Errorw("error in creating wf starting and runner entry", "err", err, "request", request)
		return err
	}

	envDeploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(pipeline.AppId, pipeline.EnvironmentId)
	if err != nil {
		impl.logger.Errorw("error in fetching deployment config by appId and envId", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
		return err
	}

	// custom GitOps repo url validation --> Start
	err = impl.handleCustomGitOpsRepoValidation(runner, pipeline, envDeploymentConfig, triggeredBy)
	if err != nil {
		impl.logger.Errorw("custom GitOps repository validation error, TriggerPreStage", "err", err)
		return err
	}
	// custom GitOps repo url validation --> Ends

	//checking vulnerability for the selected image
	err = impl.checkVulnerabilityStatusAndFailWfIfNeeded(ctx, artifact, pipeline, runner, triggeredBy)
	if err != nil {
		impl.logger.Errorw("error, checkVulnerabilityStatusAndFailWfIfNeeded", "err", err, "runner", runner)
		return err
	}
	_, span := otel.Tracer("orchestrator").Start(ctx, "buildWFRequest")
	cdStageWorkflowRequest, err := impl.buildWFRequest(runner, cdWf, pipeline, envDeploymentConfig, triggeredBy)
	span.End()
	if err != nil {
		return err
	}
	cdStageWorkflowRequest.StageType = types.PRE
	// handling copyContainerImage plugin specific logic
	imagePathReservationIds, err := impl.setCopyContainerImagePluginDataAndReserveImages(cdStageWorkflowRequest, pipeline.Id, types.PRE, artifact)
	if err != nil {
		runner.Status = cdWorkflow.WorkflowFailed
		runner.Message = err.Error()
		_ = impl.cdWorkflowRepository.UpdateWorkFlowRunner(runner)
		return err
	} else {
		runner.ImagePathReservationIds = imagePathReservationIds
		_ = impl.cdWorkflowRepository.UpdateWorkFlowRunner(runner)
	}

	_, span = otel.Tracer("orchestrator").Start(ctx, "cdWorkflowService.SubmitWorkflow")
	cdStageWorkflowRequest.Pipeline = pipeline
	cdStageWorkflowRequest.Env = env
	cdStageWorkflowRequest.Type = pipelineConfigBean.CD_WORKFLOW_PIPELINE_TYPE
	_, err = impl.cdWorkflowService.SubmitWorkflow(cdStageWorkflowRequest)
	span.End()
	err = impl.sendPreStageNotification(ctx, cdWf, pipeline)
	if err != nil {
		return err
	}
	//creating cd config history entry
	_, span = otel.Tracer("orchestrator").Start(ctx, "prePostCdScriptHistoryService.CreatePrePostCdScriptHistory")
	err = impl.prePostCdScriptHistoryService.CreatePrePostCdScriptHistory(pipeline, nil, repository3.PRE_CD_TYPE, true, triggeredBy, triggeredAt)
	span.End()
	if err != nil {
		impl.logger.Errorw("error in creating pre cd script entry", "err", err, "pipeline", pipeline)
		return err
	}
	return nil
}

func (impl *TriggerServiceImpl) createStartingWfAndRunner(request bean.TriggerRequest, triggeredAt time.Time) (*pipelineConfig.CdWorkflow, *pipelineConfig.CdWorkflowRunner, error) {
	triggeredBy := request.TriggeredBy
	artifact := request.Artifact
	pipeline := request.Pipeline
	ctx := request.TriggerContext.Context
	//in case of pre stage manual trigger auth is already applied and for auto triggers there is no need for auth check here
	cdWf := request.CdWf
	var err error
	if cdWf == nil && request.WorkflowType == bean2.CD_WORKFLOW_TYPE_PRE {
		cdWf = &pipelineConfig.CdWorkflow{
			CiArtifactId: artifact.Id,
			PipelineId:   pipeline.Id,
			AuditLog:     sql.AuditLog{CreatedOn: triggeredAt, CreatedBy: 1, UpdatedOn: triggeredAt, UpdatedBy: 1},
		}
		err = impl.cdWorkflowRepository.SaveWorkFlow(ctx, cdWf)
		if err != nil {
			return nil, nil, err
		}
	}
	runner := &pipelineConfig.CdWorkflowRunner{
		Name:                  pipeline.Name,
		WorkflowType:          request.WorkflowType,
		ExecutorType:          impl.config.GetWorkflowExecutorType(),
		Status:                cdWorkflow.WorkflowStarting, // starting PreStage
		TriggeredBy:           triggeredBy,
		StartedOn:             triggeredAt,
		Namespace:             request.RunStageInEnvNamespace,
		BlobStorageEnabled:    impl.config.BlobStorageEnabled,
		CdWorkflowId:          cdWf.Id,
		LogLocation:           fmt.Sprintf("%s/%s%s-%s/main.log", impl.config.GetDefaultBuildLogsKeyPrefix(), strconv.Itoa(cdWf.Id), request.WorkflowType, pipeline.Name),
		AuditLog:              sql.AuditLog{CreatedOn: triggeredAt, CreatedBy: 1, UpdatedOn: triggeredAt, UpdatedBy: 1},
		RefCdWorkflowRunnerId: request.RefCdWorkflowRunnerId,
		ReferenceId:           request.TriggerContext.ReferenceId,
	}
	_, span := otel.Tracer("orchestrator").Start(ctx, "cdWorkflowRepository.SaveWorkFlowRunner")
	_, err = impl.cdWorkflowRepository.SaveWorkFlowRunner(runner)
	span.End()
	if err != nil {
		return nil, nil, err
	}
	return cdWf, runner, nil
}

func (impl *TriggerServiceImpl) getEnvAndNsIfRunStageInEnv(ctx context.Context, request bean.TriggerRequest) (*repository2.Environment, string, error) {
	workflowStage := request.WorkflowType
	pipeline := request.Pipeline
	var env *repository2.Environment
	var err error
	namespace := impl.config.GetDefaultNamespace()
	runStageInEnv := false
	if workflowStage == bean2.CD_WORKFLOW_TYPE_PRE {
		runStageInEnv = pipeline.RunPreStageInEnv
	} else if workflowStage == bean2.CD_WORKFLOW_TYPE_POST {
		runStageInEnv = pipeline.RunPostStageInEnv
	}
	_, span := otel.Tracer("orchestrator").Start(ctx, "envRepository.FindById")
	env, err = impl.envRepository.FindById(pipeline.EnvironmentId)
	span.End()
	if err != nil {
		impl.logger.Errorw(" unable to find env ", "err", err)
		return nil, namespace, err
	}
	if runStageInEnv {
		namespace = env.Namespace
	}
	return env, namespace, nil
}

func (impl *TriggerServiceImpl) checkVulnerabilityStatusAndFailWfIfNeeded(ctx context.Context, artifact *repository.CiArtifact,
	cdPipeline *pipelineConfig.Pipeline, runner *pipelineConfig.CdWorkflowRunner, triggeredBy int32) error {
	//checking vulnerability for the selected image
	vulnerabilityCheckRequest := adapter2.GetVulnerabilityCheckRequest(cdPipeline, artifact.ImageDigest)
	isVulnerable, err := impl.imageScanService.GetArtifactVulnerabilityStatus(ctx, vulnerabilityCheckRequest)
	if err != nil {
		impl.logger.Errorw("error in getting Artifact vulnerability status, TriggerPreStage", "err", err)
		return err
	}
	if isVulnerable {
		// if image vulnerable, update timeline status and return
		runner.Status = cdWorkflow.WorkflowFailed
		runner.Message = cdWorkflow.FOUND_VULNERABILITY
		runner.FinishedOn = time.Now()
		runner.UpdatedOn = time.Now()
		runner.UpdatedBy = triggeredBy
		err = impl.cdWorkflowRepository.UpdateWorkFlowRunner(runner)
		if err != nil {
			impl.logger.Errorw("error in updating wfr status due to vulnerable image", "err", err)
			return err
		}
		return fmt.Errorf("found vulnerability for image digest %s", artifact.ImageDigest)
	}
	return nil
}

// setCopyContainerImagePluginDataAndReserveImages sets required fields in cdStageWorkflowRequest and reserve images generated by plugin
func (impl *TriggerServiceImpl) setCopyContainerImagePluginDataAndReserveImages(cdStageWorkflowRequest *types.WorkflowRequest, pipelineId int, pipelineStage string, artifact *repository.CiArtifact) ([]int, error) {

	copyContainerImagePluginDetail, err := impl.globalPluginService.GetRefPluginIdByRefPluginName(pipeline.COPY_CONTAINER_IMAGE)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting copyContainerImage plugin id", "err", err)
		return nil, err
	}

	pluginIdToVersionMap := make(map[int]string)
	for _, p := range copyContainerImagePluginDetail {
		pluginIdToVersionMap[p.Id] = p.Version
	}

	dockerImageTag, customTagId, err := impl.getDockerTagAndCustomTagIdForPlugin(pipelineStage, pipelineId, artifact)
	if err != nil {
		impl.logger.Errorw("error in getting docker tag", "err", err)
		return nil, err
	}

	var sourceDockerRegistryId string
	if artifact.DataSource == repository.PRE_CD || artifact.DataSource == repository.POST_CD || artifact.DataSource == repository.POST_CI {
		if artifact.CredentialsSourceType == repository.GLOBAL_CONTAINER_REGISTRY {
			sourceDockerRegistryId = artifact.CredentialSourceValue
		}
	} else {
		sourceDockerRegistryId = cdStageWorkflowRequest.DockerRegistryId
	}

	registryCredentialMap := make(map[string]bean3.RegistryCredentials)
	var allDestinationImages []string //saving all images to be reserved in this array

	for _, step := range cdStageWorkflowRequest.PrePostDeploySteps {
		if version, ok := pluginIdToVersionMap[step.RefPluginId]; ok {
			registryDestinationImageMap, credentialMap, err := impl.pluginInputVariableParser.HandleCopyContainerImagePluginInputVariables(step.InputVars, dockerImageTag, cdStageWorkflowRequest.CiArtifactDTO.Image, sourceDockerRegistryId)
			if err != nil {
				impl.logger.Errorw("error in parsing copyContainerImage input variable", "err", err)
				return nil, err
			}
			if version == pipeline.COPY_CONTAINER_IMAGE_VERSION_V1 {
				// this is needed in ci runner only for v1
				cdStageWorkflowRequest.RegistryDestinationImageMap = registryDestinationImageMap
			}
			for _, images := range registryDestinationImageMap {
				allDestinationImages = append(allDestinationImages, images...)
			}
			for k, v := range credentialMap {
				registryCredentialMap[k] = v
			}
		}
	}

	// set data in cdStageWorkflowRequest needed for copy container image plugin

	cdStageWorkflowRequest.RegistryCredentialMap = registryCredentialMap
	cdStageWorkflowRequest.DockerImageTag = dockerImageTag
	if pipelineStage == types.PRE {
		cdStageWorkflowRequest.PluginArtifactStage = repository.PRE_CD
	} else {
		cdStageWorkflowRequest.PluginArtifactStage = repository.POST_CD
	}

	// fetch already saved artifacts to check if they are already present

	savedCIArtifacts, err := impl.ciArtifactRepository.FindCiArtifactByImagePaths(allDestinationImages)
	if err != nil {
		impl.logger.Errorw("error in fetching artifacts by image path", "err", err)
		return nil, err
	}
	if len(savedCIArtifacts) > 0 {
		// if already present in ci artifact, return "image path already in use error"
		return nil, pipelineConfigBean.ErrImagePathInUse
	}
	// reserve all images where data will be
	imagePathReservationIds, err := impl.ReserveImagesGeneratedAtPlugin(customTagId, allDestinationImages)
	if err != nil {
		impl.logger.Errorw("error in reserving image", "err", err)
		return imagePathReservationIds, err
	}
	return imagePathReservationIds, nil
}

func (impl *TriggerServiceImpl) getDockerTagAndCustomTagIdForPlugin(pipelineStage string, pipelineId int, artifact *repository.CiArtifact) (string, int, error) {
	var pipelineStageEntityType int
	if pipelineStage == types.PRE {
		pipelineStageEntityType = pipelineConfigBean.EntityTypePreCD
	} else {
		pipelineStageEntityType = pipelineConfigBean.EntityTypePostCD
	}
	customTag, err := impl.customTagService.GetActiveCustomTagByEntityKeyAndValue(pipelineStageEntityType, strconv.Itoa(pipelineId))
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching custom tag data", "err", err)
		return "", 0, err
	}
	var DockerImageTag string
	customTagId := -1 // if customTag is not configured id=-1 will be saved in image_path_reservation table for image reservation
	if !customTag.Enabled {
		// case when custom tag is not configured - source image tag will be taken as docker image tag
		_, DockerImageTag, err = artifact.ExtractImageRepoAndTag()
		if err != nil {
			impl.logger.Errorw("error in getting image tag and repo", "err", err)
		}
	} else {
		// for copyContainerImage plugin parse destination images and save its data in image path reservation table
		customTagDbObject, customDockerImageTag, err := impl.customTagService.GetCustomTag(pipelineStageEntityType, strconv.Itoa(pipelineId))
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in fetching custom tag by entity key and value for CD", "err", err)
			return "", 0, err
		}
		if customTagDbObject != nil && customTagDbObject.Id > 0 {
			customTagId = customTagDbObject.Id
		}
		DockerImageTag = customDockerImageTag
	}
	return DockerImageTag, customTagId, nil
}

func (impl *TriggerServiceImpl) buildWFRequest(runner *pipelineConfig.CdWorkflowRunner, cdWf *pipelineConfig.CdWorkflow, cdPipeline *pipelineConfig.Pipeline, envDeploymentConfig *bean5.DeploymentConfig, triggeredBy int32) (*types.WorkflowRequest, error) {
	if cdPipeline.App.Id == 0 {
		appModel, err := impl.appRepository.FindById(cdPipeline.AppId)
		if err != nil {
			impl.logger.Errorw("error in getting app", "appId", cdPipeline.AppId, "err", err)
			return nil, err
		}
		cdPipeline.App = *appModel
	}

	workflowExecutor := runner.ExecutorType

	artifact, err := impl.ciArtifactRepository.Get(cdWf.CiArtifactId)
	if err != nil {
		return nil, err
	}
	// Migration of deprecated DataSource Type
	if artifact.IsMigrationRequired() {
		migrationErr := impl.ciArtifactRepository.MigrateToWebHookDataSourceType(artifact.Id)
		if migrationErr != nil {
			impl.logger.Warnw("unable to migrate deprecated DataSource", "artifactId", artifact.Id)
		}
	}
	ciMaterialInfo, err := repository.GetCiMaterialInfo(artifact.MaterialInfo, artifact.DataSource)
	if err != nil {
		impl.logger.Errorw("parsing error", "err", err)
		return nil, err
	}

	var ciProjectDetails []pipelineConfigBean.CiProjectDetails
	var ciPipeline *pipelineConfig.CiPipeline
	if cdPipeline.CiPipelineId > 0 {
		ciPipeline, err = impl.ciPipelineRepository.FindById(cdPipeline.CiPipelineId)
		if err != nil && !util.IsErrNoRows(err) {
			impl.logger.Errorw("cannot find ciPipelineRequest", "err", err)
			return nil, err
		}
		if ciPipeline != nil && util.IsErrNoRows(err) {
			ciPipeline.Id = 0
		}
		for _, m := range ciPipeline.CiPipelineMaterials {
			// git material should be active in this case
			if m == nil || m.GitMaterial == nil || !m.GitMaterial.Active {
				continue
			}
			var ciMaterialCurrent repository.CiMaterialInfo
			for _, ciMaterial := range ciMaterialInfo {
				if ciMaterial.Material.GitConfiguration.URL == m.GitMaterial.Url {
					ciMaterialCurrent = ciMaterial
					break
				}
			}
			gitMaterial, err := impl.materialRepository.FindById(m.GitMaterialId)
			if err != nil && !util.IsErrNoRows(err) {
				impl.logger.Errorw("could not fetch git materials", "err", err)
				return nil, err
			}

			ciProjectDetail := pipelineConfigBean.CiProjectDetails{
				GitRepository:   ciMaterialCurrent.Material.GitConfiguration.URL,
				MaterialName:    gitMaterial.Name,
				CheckoutPath:    gitMaterial.CheckoutPath,
				FetchSubmodules: gitMaterial.FetchSubmodules,
				SourceType:      m.Type,
				SourceValue:     m.Value,
				Type:            string(m.Type),
				GitOptions: pipelineConfigBean.GitOptions{
					UserName:      gitMaterial.GitProvider.UserName,
					Password:      gitMaterial.GitProvider.Password,
					SshPrivateKey: gitMaterial.GitProvider.SshPrivateKey,
					AccessToken:   gitMaterial.GitProvider.AccessToken,
					AuthMode:      gitMaterial.GitProvider.AuthMode,
				},
			}

			if len(ciMaterialCurrent.Modifications) > 0 {
				ciProjectDetail.CommitHash = ciMaterialCurrent.Modifications[0].Revision
				ciProjectDetail.Author = ciMaterialCurrent.Modifications[0].Author
				ciProjectDetail.GitTag = ciMaterialCurrent.Modifications[0].Tag
				ciProjectDetail.Message = ciMaterialCurrent.Modifications[0].Message
				commitTime, err := convert(ciMaterialCurrent.Modifications[0].ModifiedTime)
				if err != nil {
					return nil, err
				}
				ciProjectDetail.CommitTime = commitTime.Format(bean4.LayoutRFC3339)
			} else if ciPipeline.PipelineType == string(CiPipeline.CI_JOB) {
				// This has been done to resolve unmarshalling issue in ci-runner, in case of no commit time(eg- polling container images)
				ciProjectDetail.CommitTime = time.Time{}.Format(bean4.LayoutRFC3339)
			} else {
				impl.logger.Debugw("devtronbug#1062", ciPipeline.Id, cdPipeline.Id)
				return nil, fmt.Errorf("modifications not found for %d", ciPipeline.Id)
			}

			// set webhook data
			if m.Type == pipelineConfig.SOURCE_TYPE_WEBHOOK && len(ciMaterialCurrent.Modifications) > 0 {
				webhookData := ciMaterialCurrent.Modifications[0].WebhookData
				ciProjectDetail.WebhookData = pipelineConfig.WebhookData{
					Id:              webhookData.Id,
					EventActionType: webhookData.EventActionType,
					Data:            webhookData.Data,
				}
			}

			ciProjectDetails = append(ciProjectDetails, ciProjectDetail)
		}
	}
	var stageYaml string
	var deployStageWfr pipelineConfig.CdWorkflowRunner
	var deployStageTriggeredByUserEmail string
	var pipelineReleaseCounter int
	var preDeploySteps []*pipelineConfigBean.StepObject
	var postDeploySteps []*pipelineConfigBean.StepObject
	var refPluginsData []*pipelineConfigBean.RefPluginObject
	//if pipeline_stage_steps present for pre-CD or post-CD then no need to add stageYaml to cdWorkflowRequest in that
	//case add PreDeploySteps and PostDeploySteps to cdWorkflowRequest, this is done for backward compatibility
	pipelineStage, err := impl.pipelineStageService.GetCdStageByCdPipelineIdAndStageType(cdPipeline.Id, runner.WorkflowType.WorkflowTypeToStageType())
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching CD pipeline stage", "cdPipelineId", cdPipeline.Id, "stage ", runner.WorkflowType.WorkflowTypeToStageType(), "err", err)
		return nil, err
	}
	env, err := impl.envRepository.FindById(cdPipeline.EnvironmentId)
	if err != nil {
		impl.logger.Errorw("error in getting environment by id", "err", err)
		return nil, err
	}

	//Scope will pick the environment of CD pipeline irrespective of in-cluster mode,
	//since user sees the environment of the CD pipeline
	scope := resourceQualifiers.Scope{
		AppId:     cdPipeline.App.Id,
		EnvId:     env.Id,
		ClusterId: env.ClusterId,
		SystemMetadata: &resourceQualifiers.SystemMetadata{
			EnvironmentName: env.Name,
			ClusterName:     env.Cluster.ClusterName,
			Namespace:       env.Namespace,
			Image:           artifact.Image,
			ImageTag:        util3.GetImageTagFromImage(artifact.Image),
		},
	}
	if pipelineStage != nil {
		var variableSnapshot map[string]string
		if runner.WorkflowType == bean2.CD_WORKFLOW_TYPE_PRE {
			//TODO: use const from pipeline.WorkflowService:95
			prePostAndRefPluginResponse, err := impl.pipelineStageService.BuildPrePostAndRefPluginStepsDataForWfRequest(cdPipeline.Id, "preCD", scope)
			if err != nil {
				impl.logger.Errorw("error in getting pre, post & refPlugin steps data for wf request", "err", err, "cdPipelineId", cdPipeline.Id)
				return nil, err
			}
			preDeploySteps = prePostAndRefPluginResponse.PreStageSteps
			refPluginsData = prePostAndRefPluginResponse.RefPluginData
			variableSnapshot = prePostAndRefPluginResponse.VariableSnapshot
		} else if runner.WorkflowType == bean2.CD_WORKFLOW_TYPE_POST {
			//TODO: use const from pipeline.WorkflowService:96
			prePostAndRefPluginResponse, err := impl.pipelineStageService.BuildPrePostAndRefPluginStepsDataForWfRequest(cdPipeline.Id, "postCD", scope)
			if err != nil {
				impl.logger.Errorw("error in getting pre, post & refPlugin steps data for wf request", "err", err, "cdPipelineId", cdPipeline.Id)
				return nil, err
			}
			postDeploySteps = prePostAndRefPluginResponse.PostStageSteps
			refPluginsData = prePostAndRefPluginResponse.RefPluginData
			variableSnapshot = prePostAndRefPluginResponse.VariableSnapshot
			deployStageWfr, deployStageTriggeredByUserEmail, pipelineReleaseCounter, err = impl.getDeployStageDetails(cdPipeline.Id)
			if err != nil {
				impl.logger.Errorw("error in getting deployStageWfr, deployStageTriggeredByUser and pipelineReleaseCounter wf request", "err", err, "cdPipelineId", cdPipeline.Id)
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unsupported workflow triggerd")
		}

		//Save Scoped VariableSnapshot
		var variableSnapshotHistories = util4.GetBeansPtr(
			repository5.GetSnapshotBean(runner.Id, repository5.HistoryReferenceTypeCDWORKFLOWRUNNER, variableSnapshot))
		if len(variableSnapshotHistories) > 0 {
			err = impl.scopedVariableManager.SaveVariableHistoriesForTrigger(variableSnapshotHistories, runner.TriggeredBy)
			if err != nil {
				impl.logger.Errorf("Not able to save variable snapshot for CD trigger %s %d %s", err, runner.Id, variableSnapshot)
			}
		}
	} else {
		//in this case no plugin script is not present for this cdPipeline hence going with attaching preStage or postStage config
		if runner.WorkflowType == bean2.CD_WORKFLOW_TYPE_PRE {
			stageYaml = cdPipeline.PreStageConfig
		} else if runner.WorkflowType == bean2.CD_WORKFLOW_TYPE_POST {
			stageYaml = cdPipeline.PostStageConfig
			deployStageWfr, deployStageTriggeredByUserEmail, pipelineReleaseCounter, err = impl.getDeployStageDetails(cdPipeline.Id)
			if err != nil {
				impl.logger.Errorw("error in getting deployStageWfr, deployStageTriggeredByUser and pipelineReleaseCounter wf request", "err", err, "cdPipelineId", cdPipeline.Id)
				return nil, err
			}

		} else {
			return nil, fmt.Errorf("unsupported workflow triggerd")
		}
	}

	digestConfigurationRequest := imageDigestPolicy.DigestPolicyConfigurationRequest{PipelineId: cdPipeline.Id}
	digestPolicyConfigurations, err := impl.imageDigestPolicyService.GetDigestPolicyConfigurations(digestConfigurationRequest)
	if err != nil {
		impl.logger.Errorw("error in checking if isImageDigestPolicyConfiguredForPipeline", "err", err, "pipelineId", cdPipeline.Id)
		return nil, err
	}
	image := artifact.Image
	if digestPolicyConfigurations.UseDigestForTrigger() {
		image = ReplaceImageTagWithDigest(image, artifact.ImageDigest)
	}

	host, err := impl.attributeService.GetByKey(bean6.HostUrlKey)
	if err != nil {
		impl.logger.Errorw("error in getting hostUrl", "err", err)
		return nil, err
	}
	cdStageWorkflowRequest := &types.WorkflowRequest{
		EnvironmentId:         cdPipeline.EnvironmentId,
		AppId:                 cdPipeline.AppId,
		WorkflowId:            cdWf.Id,
		WorkflowRunnerId:      runner.Id,
		WorkflowNamePrefix:    strconv.Itoa(runner.Id) + "-" + runner.Name,
		WorkflowPrefixForLog:  strconv.Itoa(cdWf.Id) + string(runner.WorkflowType) + "-" + runner.Name,
		CdImage:               impl.config.GetDefaultImage(),
		CdPipelineId:          cdWf.PipelineId,
		TriggeredBy:           triggeredBy,
		StageYaml:             stageYaml,
		CiProjectDetails:      ciProjectDetails,
		Namespace:             runner.Namespace,
		ActiveDeadlineSeconds: impl.config.GetDefaultTimeout(),
		CiArtifactDTO: types.CiArtifactDTO{
			Id:           artifact.Id,
			PipelineId:   artifact.PipelineId,
			Image:        artifact.Image,
			ImageDigest:  artifact.ImageDigest,
			MaterialInfo: artifact.MaterialInfo,
			DataSource:   artifact.DataSource,
			WorkflowId:   artifact.WorkflowId,
		},
		OrchestratorHost:  impl.config.OrchestratorHost,
		HostUrl:           host.Value,
		OrchestratorToken: impl.config.OrchestratorToken,
		CloudProvider:     impl.config.CloudProvider,
		WorkflowExecutor:  workflowExecutor,
		RefPlugins:        refPluginsData,
		Scope:             scope,
	}

	extraEnvVariables := make(map[string]string)
	if env != nil {
		extraEnvVariables[plugin.CD_PIPELINE_ENV_NAME_KEY] = env.Name
		if env.Cluster != nil {
			extraEnvVariables[plugin.CD_PIPELINE_CLUSTER_NAME_KEY] = env.Cluster.ClusterName
		}
	}
	ciWf, err := impl.ciWorkflowRepository.FindLastTriggeredWorkflowByArtifactId(artifact.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting ciWf by artifactId", "err", err, "artifactId", artifact.Id)
		return nil, err
	}

	var webhookAndCiData *gitSensorClient.WebhookAndCiData
	var gitTriggerEnvVariables map[string]string

	// get env variables of git trigger data and add it in the extraEnvVariables
	gitTriggerEnvVariables, webhookAndCiData, err = impl.ciCdPipelineOrchestrator.GetGitCommitEnvVarDataForCICDStage(ciWf.GitTriggers)
	if err != nil {
		impl.logger.Errorw("error in getting gitTrigger env data for stage", "gitTriggers", ciWf.GitTriggers, "err", err)
		return nil, err
	}
	maps.Copy(extraEnvVariables, gitTriggerEnvVariables)

	childCdIds, err := impl.appWorkflowRepository.FindChildCDIdsByParentCDPipelineId(cdPipeline.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting child cdPipelineIds by parent cdPipelineId", "err", err, "parent cdPipelineId", cdPipeline.Id)
		return nil, err
	}
	if len(childCdIds) > 0 {
		childPipelines, err := impl.pipelineRepository.FindByIdsIn(childCdIds)
		if err != nil {
			impl.logger.Errorw("error in getting pipelines by ids", "err", err, "ids", childCdIds)
			return nil, err
		}
		var childCdEnvVariables []types.ChildCdMetadata
		for i, childPipeline := range childPipelines {
			extraEnvVariables[fmt.Sprintf("%s_%d", CHILD_CD_ENV_NAME_PREFIX, i+1)] = childPipeline.Environment.Name
			extraEnvVariables[fmt.Sprintf("%s_%d", CHILD_CD_CLUSTER_NAME_PREFIX, i+1)] = childPipeline.Environment.Cluster.ClusterName

			childCdEnvVariables = append(childCdEnvVariables, types.ChildCdMetadata{
				ChildCdEnvName:     childPipeline.Environment.Name,
				ChildCdClusterName: childPipeline.Environment.Cluster.ClusterName,
			})
		}
		childCdEnvVariablesMetadata, err := json.Marshal(&childCdEnvVariables)
		if err != nil {
			impl.logger.Errorw("err while marshaling childCdEnvVariables", "err", err)
			return nil, err
		}
		extraEnvVariables[plugin.CHILD_CD_METADATA] = string(childCdEnvVariablesMetadata)

		extraEnvVariables[CHILD_CD_COUNT] = strconv.Itoa(len(childPipelines))
	}

	if ciPipeline != nil && ciPipeline.Id > 0 {
		sourceCiPipeline, err := impl.getSourceCiPipelineForArtifact(*ciPipeline)
		if err != nil {
			impl.logger.Errorw("error in getting source ciPipeline for artifact", "err", err)
			return nil, err
		}
		extraEnvVariables["APP_NAME"] = sourceCiPipeline.App.AppName
		cdStageWorkflowRequest.CiPipelineType = sourceCiPipeline.PipelineType
		buildRegistryConfig, dbErr := impl.getBuildRegistryConfigForArtifact(*sourceCiPipeline, *artifact)
		if dbErr != nil {
			impl.logger.Errorw("error in getting registry credentials for the artifact", "err", dbErr)
			return nil, dbErr
		}
		adapter.UpdateRegistryDetailsToWrfReq(cdStageWorkflowRequest, buildRegistryConfig)
	} else if cdPipeline.AppId > 0 {
		// the below flow is used for external ci base pipelines;
		extraEnvVariables["APP_NAME"] = cdPipeline.App.AppName
		buildRegistryConfig, err := impl.ciTemplateService.GetBaseDockerConfigForCiPipeline(cdPipeline.AppId)
		if err != nil {
			impl.logger.Errorw("error in getting build configurations", "err", err)
			return nil, fmt.Errorf("error in getting build configurations")
		}
		adapter.UpdateRegistryDetailsToWrfReq(cdStageWorkflowRequest, buildRegistryConfig)
		appLabels, err := impl.appLabelRepository.FindAllByAppId(cdPipeline.AppId)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in getting labels by appId", "err", err, "appId", cdPipeline.AppId)
			return nil, err
		}
		var appLabelEnvVariables []types.AppLabelMetadata
		for i, appLabel := range appLabels {
			extraEnvVariables[fmt.Sprintf("%s_%d", APP_LABEL_KEY_PREFIX, i+1)] = appLabel.Key
			extraEnvVariables[fmt.Sprintf("%s_%d", APP_LABEL_VALUE_PREFIX, i+1)] = appLabel.Value
			appLabelEnvVariables = append(appLabelEnvVariables, types.AppLabelMetadata{
				AppLabelKey:   appLabel.Key,
				AppLabelValue: appLabel.Value,
			})
		}
		if len(appLabels) > 0 {
			extraEnvVariables[APP_LABEL_COUNT] = strconv.Itoa(len(appLabels))
			appLabelEnvVariablesMetadata, err := json.Marshal(&appLabelEnvVariables)
			if err != nil {
				impl.logger.Errorw("err while marshaling appLabelEnvVariables", "err", err)
				return nil, err
			}
			extraEnvVariables[plugin.APP_LABEL_METADATA] = string(appLabelEnvVariablesMetadata)

		}
	}
	cdStageWorkflowRequest.ExtraEnvironmentVariables = extraEnvVariables
	cdStageWorkflowRequest.DeploymentTriggerTime = deployStageWfr.StartedOn
	cdStageWorkflowRequest.DeploymentTriggeredBy = deployStageTriggeredByUserEmail

	if pipelineReleaseCounter > 0 {
		cdStageWorkflowRequest.DeploymentReleaseCounter = pipelineReleaseCounter
	}
	cdWorkflowConfigCdCacheRegion := impl.config.GetDefaultCdLogsBucketRegion()
	// For Pre-CD / Post-CD workflow, cache is not uploaded; hence no need to set cache bucket
	cdWorkflowConfigCdCacheBucket := ""

	if runner.WorkflowType == bean2.CD_WORKFLOW_TYPE_PRE {
		//populate input variables of steps with extra env variables
		setExtraEnvVariableInDeployStep(preDeploySteps, extraEnvVariables, webhookAndCiData)
		cdStageWorkflowRequest.PrePostDeploySteps = preDeploySteps
	} else if runner.WorkflowType == bean2.CD_WORKFLOW_TYPE_POST {
		setExtraEnvVariableInDeployStep(postDeploySteps, extraEnvVariables, webhookAndCiData)
		cdStageWorkflowRequest.PrePostDeploySteps = postDeploySteps
	}
	cdStageWorkflowRequest.BlobStorageConfigured = runner.BlobStorageEnabled
	switch cdStageWorkflowRequest.CloudProvider {
	case types.BLOB_STORAGE_S3:
		//No AccessKey is used for uploading artifacts, instead IAM based auth is used
		cdStageWorkflowRequest.CdCacheRegion = cdWorkflowConfigCdCacheRegion
		cdStageWorkflowRequest.CdCacheLocation = cdWorkflowConfigCdCacheBucket
		cdStageWorkflowRequest.ArtifactLocation, cdStageWorkflowRequest.CiArtifactBucket, cdStageWorkflowRequest.CiArtifactFileName = impl.buildArtifactLocationForS3(cdWf, runner)
		cdStageWorkflowRequest.BlobStorageS3Config = &blob_storage.BlobStorageS3Config{
			AccessKey:                  impl.config.BlobStorageS3AccessKey,
			Passkey:                    impl.config.BlobStorageS3SecretKey,
			EndpointUrl:                impl.config.BlobStorageS3Endpoint,
			IsInSecure:                 impl.config.BlobStorageS3EndpointInsecure,
			CiCacheBucketName:          cdWorkflowConfigCdCacheBucket,
			CiCacheRegion:              cdWorkflowConfigCdCacheRegion,
			CiCacheBucketVersioning:    impl.config.BlobStorageS3BucketVersioned,
			CiArtifactBucketName:       cdStageWorkflowRequest.CiArtifactBucket,
			CiArtifactRegion:           cdWorkflowConfigCdCacheRegion,
			CiArtifactBucketVersioning: impl.config.BlobStorageS3BucketVersioned,
			CiLogBucketName:            impl.config.GetDefaultBuildLogsBucket(),
			CiLogRegion:                impl.config.GetDefaultCdLogsBucketRegion(),
			CiLogBucketVersioning:      impl.config.BlobStorageS3BucketVersioned,
		}
	case types.BLOB_STORAGE_GCP:
		cdStageWorkflowRequest.GcpBlobConfig = &blob_storage.GcpBlobConfig{
			CredentialFileJsonData: impl.config.BlobStorageGcpCredentialJson,
			ArtifactBucketName:     impl.config.GetDefaultBuildLogsBucket(),
			LogBucketName:          impl.config.GetDefaultBuildLogsBucket(),
		}
		cdStageWorkflowRequest.ArtifactLocation = impl.buildDefaultArtifactLocation(cdWf, runner)
		cdStageWorkflowRequest.CiArtifactFileName = cdStageWorkflowRequest.ArtifactLocation
	case types.BLOB_STORAGE_AZURE:
		cdStageWorkflowRequest.AzureBlobConfig = &blob_storage.AzureBlobConfig{
			Enabled:               true,
			AccountName:           impl.config.AzureAccountName,
			BlobContainerCiCache:  impl.config.AzureBlobContainerCiCache,
			AccountKey:            impl.config.AzureAccountKey,
			BlobContainerCiLog:    impl.config.AzureBlobContainerCiLog,
			BlobContainerArtifact: impl.config.AzureBlobContainerCiLog,
		}
		cdStageWorkflowRequest.BlobStorageS3Config = &blob_storage.BlobStorageS3Config{
			EndpointUrl:     impl.config.AzureGatewayUrl,
			IsInSecure:      impl.config.AzureGatewayConnectionInsecure,
			CiLogBucketName: impl.config.AzureBlobContainerCiLog,
			CiLogRegion:     "",
			AccessKey:       impl.config.AzureAccountName,
		}
		cdStageWorkflowRequest.ArtifactLocation = impl.buildDefaultArtifactLocation(cdWf, runner)
		cdStageWorkflowRequest.CiArtifactFileName = cdStageWorkflowRequest.ArtifactLocation
	default:
		if impl.config.BlobStorageEnabled {
			return nil, fmt.Errorf("blob storage %s not supported", cdStageWorkflowRequest.CloudProvider)
		}
	}
	cdStageWorkflowRequest.DefaultAddressPoolBaseCidr = impl.config.GetDefaultAddressPoolBaseCidr()
	cdStageWorkflowRequest.DefaultAddressPoolSize = impl.config.GetDefaultAddressPoolSize()
	return cdStageWorkflowRequest, nil
}

/*
getBuildRegistryConfigForArtifact performs the following logic to get Pre/Post CD Registry Credentials:

 1. CI Build:
    It will use the overridden credentials (if any) OR the base application level credentials.

 2. Link CI:
    It will fetch the parent CI pipeline first.
    Then will use the CI Build overridden credentials (if any) OR the Source application (App that contains CI Build) level credentials.

 3. Sync CD:
    It will fetch the parent CD pipeline first.

    - CASE CD Pipeline has CI Build as artifact provider:

    Then will use the CI Build overridden credentials (if any) OR the Source application (App that contains CI Build) level credentials.

    - CASE CD Pipeline has Link CI as artifact provider:

    It will fetch the parent CI pipeline of the Link CI  first.
    Then will use the CI Build overridden credentials (if any) OR the Source application (App that contains CI Build) level credentials.

 4. Skopeo Plugin:
    If any artifact has information about : credentials_source_type(global_container_registry) credentials_source_value(registry_id)
    Then we will use the credentials_source_value to derive the credentials.

 5. Polling plugin:
    If the ci_pipeline_type type is CI_JOB
    We will always fetch the registry credentials from the ci_template_override table
*/
func (impl *TriggerServiceImpl) getBuildRegistryConfigForArtifact(sourceCiPipeline pipelineConfig.CiPipeline, artifact repository.CiArtifact) (*types.DockerArtifactStoreBean, error) {
	// Handling for Skopeo Plugin
	if artifact.IsRegistryCredentialMapped() {
		dockerArtifactStore, err := impl.dockerArtifactStoreRepository.FindOne(artifact.CredentialSourceValue)
		if util.IsErrNoRows(err) {
			impl.logger.Errorw("source artifact registry not found", "registryId", artifact.CredentialSourceValue, "err", err)
			return nil, fmt.Errorf("source artifact registry '%s' not found", artifact.CredentialSourceValue)
		} else if err != nil {
			impl.logger.Errorw("error in fetching artifact info", "err", err)
			return nil, err
		}
		return adapter.GetDockerConfigBean(dockerArtifactStore), nil
	}

	// Handling for CI Job
	if adapter.IsCIJob(sourceCiPipeline) {
		// for bean.CI_JOB the source artifact is always driven from overridden ci template
		buildRegistryConfig, err := impl.ciTemplateService.GetAppliedDockerConfigForCiPipeline(sourceCiPipeline.Id, sourceCiPipeline.AppId, true)
		if err != nil {
			impl.logger.Errorw("error in getting build configurations", "err", err)
			return nil, fmt.Errorf("error in getting build configurations")
		}
		return buildRegistryConfig, nil
	}

	// Handling for Linked CI
	if adapter.IsLinkedCI(sourceCiPipeline) {
		parentCiPipeline, err := impl.ciPipelineRepository.FindById(sourceCiPipeline.ParentCiPipeline)
		if err != nil {
			impl.logger.Errorw("error in finding ciPipeline", "ciPipelineId", sourceCiPipeline.ParentCiPipeline, "err", err)
			return nil, err
		}
		buildRegistryConfig, err := impl.ciTemplateService.GetAppliedDockerConfigForCiPipeline(parentCiPipeline.Id, parentCiPipeline.AppId, parentCiPipeline.IsDockerConfigOverridden)
		if err != nil {
			impl.logger.Errorw("error in getting build configurations", "err", err)
			return nil, fmt.Errorf("error in getting build configurations")
		}
		return buildRegistryConfig, nil
	}

	// Handling for Build CI
	buildRegistryConfig, err := impl.ciTemplateService.GetAppliedDockerConfigForCiPipeline(sourceCiPipeline.Id, sourceCiPipeline.AppId, sourceCiPipeline.IsDockerConfigOverridden)
	if err != nil {
		impl.logger.Errorw("error in getting build configurations", "err", err)
		return nil, fmt.Errorf("error in getting build configurations")
	}
	return buildRegistryConfig, nil
}

func (impl *TriggerServiceImpl) getSourceCiPipelineForArtifact(ciPipeline pipelineConfig.CiPipeline) (*pipelineConfig.CiPipeline, error) {
	sourceCiPipeline := &ciPipeline
	if adapter.IsLinkedCD(ciPipeline) {
		sourceCdPipeline, err := impl.pipelineRepository.FindById(ciPipeline.ParentCiPipeline)
		if err != nil {
			impl.logger.Errorw("error in finding source cdPipeline for linked cd", "cdPipelineId", ciPipeline.ParentCiPipeline, "err", err)
			return nil, err
		}
		sourceCiPipeline, err = impl.ciPipelineRepository.FindOneWithAppData(sourceCdPipeline.CiPipelineId)
		if err != nil && !util.IsErrNoRows(err) {
			impl.logger.Errorw("error in finding ciPipeline for the cd pipeline", "CiPipelineId", sourceCdPipeline.Id, "CiPipelineId", sourceCdPipeline.CiPipelineId, "err", err)
			return nil, err
		}
	}
	return sourceCiPipeline, nil
}

func (impl *TriggerServiceImpl) ReserveImagesGeneratedAtPlugin(customTagId int, destinationImages []string) ([]int, error) {
	var imagePathReservationIds []int

	for _, image := range destinationImages {
		imagePathReservationData, err := impl.customTagService.ReserveImagePath(image, customTagId)
		if err != nil {
			impl.logger.Errorw("Error in marking custom tag reserved", "err", err)
			return imagePathReservationIds, err
		}
		if imagePathReservationData != nil {
			imagePathReservationIds = append(imagePathReservationIds, imagePathReservationData.Id)
		}
	}

	return imagePathReservationIds, nil
}

func setExtraEnvVariableInDeployStep(deploySteps []*pipelineConfigBean.StepObject, extraEnvVariables map[string]string, webhookAndCiData *gitSensorClient.WebhookAndCiData) {
	for _, deployStep := range deploySteps {
		for variableKey, variableValue := range extraEnvVariables {
			if isExtraVariableDynamic(variableKey, webhookAndCiData) && deployStep.StepType == "INLINE" {
				extraInputVar := &pipelineConfigBean.VariableObject{
					Name:                  variableKey,
					Format:                "STRING",
					Value:                 variableValue,
					VariableType:          pipelineConfigBean.VARIABLE_TYPE_REF_GLOBAL,
					ReferenceVariableName: variableKey,
				}
				deployStep.InputVars = append(deployStep.InputVars, extraInputVar)
			}
		}
	}
}

func (impl *TriggerServiceImpl) getDeployStageDetails(pipelineId int) (pipelineConfig.CdWorkflowRunner, string, int, error) {
	deployStageWfr := pipelineConfig.CdWorkflowRunner{}
	//getting deployment pipeline latest wfr by pipelineId
	deployStageWfr, err := impl.cdWorkflowRepository.FindLatestByPipelineIdAndRunnerType(pipelineId, bean2.CD_WORKFLOW_TYPE_DEPLOY)
	if err != nil {
		impl.logger.Errorw("error in getting latest status of deploy type wfr by pipelineId", "err", err, "pipelineId", pipelineId)
		return deployStageWfr, "", 0, err
	}
	deployStageTriggeredByUserEmail, err := impl.userService.GetActiveEmailById(deployStageWfr.TriggeredBy)
	if err != nil {
		impl.logger.Errorw("error in getting user email by id", "err", err, "userId", deployStageWfr.TriggeredBy)
		return deployStageWfr, "", 0, err
	}
	pipelineReleaseCounter, err := impl.pipelineOverrideRepository.GetCurrentPipelineReleaseCounter(pipelineId)
	if err != nil {
		impl.logger.Errorw("error occurred while fetching latest release counter for pipeline", "pipelineId", pipelineId, "err", err)
		return deployStageWfr, "", 0, err
	}
	return deployStageWfr, deployStageTriggeredByUserEmail, pipelineReleaseCounter, nil
}

func (impl *TriggerServiceImpl) buildArtifactLocationForS3(cdWf *pipelineConfig.CdWorkflow, runner *pipelineConfig.CdWorkflowRunner) (string, string, string) {
	cdArtifactLocationFormat := impl.config.GetArtifactLocationFormat()
	cdWorkflowConfigLogsBucket := impl.config.GetDefaultBuildLogsBucket()
	ArtifactLocation := fmt.Sprintf("s3://"+path.Join(cdWorkflowConfigLogsBucket, cdArtifactLocationFormat), cdWf.Id, runner.Id)
	artifactFileName := fmt.Sprintf(cdArtifactLocationFormat, cdWf.Id, runner.Id)
	return ArtifactLocation, cdWorkflowConfigLogsBucket, artifactFileName
}

func (impl *TriggerServiceImpl) buildDefaultArtifactLocation(savedWf *pipelineConfig.CdWorkflow, runner *pipelineConfig.CdWorkflowRunner) string {
	cdArtifactLocationFormat := impl.config.GetArtifactLocationFormat()
	ArtifactLocation := fmt.Sprintf(cdArtifactLocationFormat, savedWf.Id, runner.Id)
	return ArtifactLocation
}

func ReplaceImageTagWithDigest(image, digest string) string {
	imageWithoutTag := strings.Split(image, ":")[0]
	imageWithDigest := fmt.Sprintf("%s@%s", imageWithoutTag, digest)
	return imageWithDigest
}

func (impl *TriggerServiceImpl) sendPreStageNotification(ctx context.Context, cdWf *pipelineConfig.CdWorkflow, pipeline *pipelineConfig.Pipeline) error {
	wfr, err := impl.cdWorkflowRepository.FindByWorkflowIdAndRunnerType(ctx, cdWf.Id, bean2.CD_WORKFLOW_TYPE_PRE)
	if err != nil {
		return err
	}

	event, _ := impl.eventFactory.Build(util2.Trigger, &pipeline.Id, pipeline.AppId, &pipeline.EnvironmentId, util2.CD)
	impl.logger.Debugw("event PreStageTrigger", "event", event)
	event = impl.eventFactory.BuildExtraCDData(event, &wfr, 0, bean2.CD_WORKFLOW_TYPE_PRE)
	_, span := otel.Tracer("orchestrator").Start(ctx, "eventClient.WriteNotificationEvent")
	_, evtErr := impl.eventClient.WriteNotificationEvent(event)
	span.End()
	if evtErr != nil {
		impl.logger.Errorw("CD trigger event not sent", "error", evtErr)
	}
	return nil
}

func isExtraVariableDynamic(variableName string, webhookAndCiData *gitSensorClient.WebhookAndCiData) bool {
	if strings.Contains(variableName, types.GIT_COMMIT_HASH_PREFIX) || strings.Contains(variableName, types.GIT_SOURCE_TYPE_PREFIX) || strings.Contains(variableName, types.GIT_SOURCE_VALUE_PREFIX) ||
		strings.Contains(variableName, APP_LABEL_VALUE_PREFIX) || strings.Contains(variableName, APP_LABEL_KEY_PREFIX) ||
		strings.Contains(variableName, CHILD_CD_ENV_NAME_PREFIX) || strings.Contains(variableName, CHILD_CD_CLUSTER_NAME_PREFIX) ||
		strings.Contains(variableName, CHILD_CD_COUNT) || strings.Contains(variableName, APP_LABEL_COUNT) || strings.Contains(variableName, types.GIT_SOURCE_COUNT) ||
		webhookAndCiData != nil {

		return true
	}
	return false
}

func convert(ts string) (*time.Time, error) {
	t, err := time.Parse(bean4.LayoutRFC3339, ts)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
