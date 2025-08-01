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

package delete

import (
	"fmt"
	k8sUtil "github.com/devtron-labs/common-lib/utils/k8s"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	dockerRegistryRepository "github.com/devtron-labs/devtron/internal/sql/repository/dockerRegistry"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/util"
	appStoreBean "github.com/devtron-labs/devtron/pkg/appStore/bean"
	repository2 "github.com/devtron-labs/devtron/pkg/appStore/installedApp/repository"
	"github.com/devtron-labs/devtron/pkg/chartRepo"
	"github.com/devtron-labs/devtron/pkg/cluster"
	bean2 "github.com/devtron-labs/devtron/pkg/cluster/bean"
	"github.com/devtron-labs/devtron/pkg/cluster/environment"
	"github.com/devtron-labs/devtron/pkg/cluster/environment/bean"
	"github.com/devtron-labs/devtron/pkg/cluster/environment/repository"
	"github.com/devtron-labs/devtron/pkg/k8s/informer"
	"github.com/devtron-labs/devtron/pkg/pipeline"
	"github.com/devtron-labs/devtron/pkg/team"
	bean3 "github.com/devtron-labs/devtron/pkg/team/bean"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"net/http"
)

type DeleteServiceExtendedImpl struct {
	appRepository         app.AppRepository
	environmentRepository repository.EnvironmentRepository
	pipelineRepository    pipelineConfig.PipelineRepository
	*DeleteServiceImpl
}

func NewDeleteServiceExtendedImpl(logger *zap.SugaredLogger,
	teamService team.TeamService,
	clusterService cluster.ClusterService,
	environmentService environment.EnvironmentService,
	appRepository app.AppRepository,
	environmentRepository repository.EnvironmentRepository,
	pipelineRepository pipelineConfig.PipelineRepository,
	chartRepositoryService chartRepo.ChartRepositoryService,
	installedAppRepository repository2.InstalledAppRepository,
	dockerRegistryConfig pipeline.DockerRegistryConfig,
	dockerRegistryRepository dockerRegistryRepository.DockerArtifactStoreRepository,
	K8sService k8sUtil.K8sService,
	factory informer.K8sInformerFactory,
) *DeleteServiceExtendedImpl {
	return &DeleteServiceExtendedImpl{
		appRepository:         appRepository,
		environmentRepository: environmentRepository,
		pipelineRepository:    pipelineRepository,
		DeleteServiceImpl: &DeleteServiceImpl{
			logger:                   logger,
			teamService:              teamService,
			clusterService:           clusterService,
			environmentService:       environmentService,
			chartRepositoryService:   chartRepositoryService,
			installedAppRepository:   installedAppRepository,
			dockerRegistryConfig:     dockerRegistryConfig,
			dockerRegistryRepository: dockerRegistryRepository,
			K8sUtil:                  K8sService,
			k8sInformerFactory:       factory,
		},
	}
}

func (impl DeleteServiceExtendedImpl) DeleteCluster(deleteRequest *bean2.DeleteClusterBean, userId int32) error {
	//finding if there are env in this cluster or not, if yes then will not delete
	env, err := impl.environmentRepository.FindByClusterId(deleteRequest.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting cluster", "clusterId", deleteRequest.Id, "err", err)
		return err
	}
	if len(env) > 0 {
		impl.logger.Errorw("err in deleting cluster, found env in this cluster", "clusterId", deleteRequest.Id, "err", err)
		return &util.ApiError{HttpStatusCode: http.StatusBadRequest, UserMessage: " Please delete all related environments before deleting this cluster"}
	}
	clusterName, err := impl.clusterService.DeleteFromDb(deleteRequest, userId)
	if err != nil {
		impl.logger.Errorw("error im deleting cluster", "err", err, "deleteRequest", deleteRequest)
		return err
	}
	err = impl.DeleteClusterConfigMap(deleteRequest)
	if err != nil {
		impl.logger.Errorw("error in deleting cluster secret", "clusterId", deleteRequest.Id, "error", err)
		// We are not returning error as it is not a blocking call as cluster can be unreachable at that time, and we have already deleted cluster from db.
		//return err
	}
	impl.k8sInformerFactory.DeleteClusterFromCache(clusterName)
	return nil
}

func (impl DeleteServiceExtendedImpl) DeleteEnvironment(deleteRequest *bean.EnvironmentBean, userId int32) error {
	//finding if this env is used in any cd pipelines, if yes then will not delete
	pipelines, err := impl.pipelineRepository.FindActiveByEnvId(deleteRequest.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting env", "envName", deleteRequest.Environment, "err", err)
		return err
	}
	//finding if this env is used in any helm apps, if yes then will not delete
	filter := &appStoreBean.AppStoreFilter{
		EnvIds: []int{deleteRequest.Id},
	}
	installedApps, err := impl.installedAppRepository.GetAllInstalledApps(filter)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting env", "envName", deleteRequest.Environment, "err", err)
		return err
	}
	if len(installedApps) > 0 && len(pipelines) > 0 {
		impl.logger.Errorw("err in deleting env, found cd pipelines and helm apps in this env", "envName", deleteRequest.Environment, "err", err)
		return &util.ApiError{HttpStatusCode: http.StatusBadRequest, UserMessage: " Please delete all related cd pipelines and helm apps before deleting this environment"}
	} else if len(installedApps) > 0 {
		impl.logger.Errorw("err in deleting env, found helm apps in this env", "envName", deleteRequest.Environment, "err", err)
		return &util.ApiError{HttpStatusCode: http.StatusBadRequest, UserMessage: " Please delete all related helm apps before deleting this environment"}
	} else if len(pipelines) > 0 {
		impl.logger.Errorw("err in deleting env, found cd pipelines in this env", "envName", deleteRequest.Environment, "err", err)
		return &util.ApiError{HttpStatusCode: http.StatusBadRequest, UserMessage: " Please delete all related cd pipelines before deleting this environment"}
	}

	err = impl.environmentService.Delete(deleteRequest, userId)
	if err != nil {
		impl.logger.Errorw("error in deleting environment", "err", err, "deleteRequest", deleteRequest)
		return err
	}
	return nil
}
func (impl DeleteServiceExtendedImpl) DeleteTeam(deleteRequest *bean3.TeamRequest) error {
	//finding if this project is used in some app; if yes, will not perform delete operation
	apps, err := impl.appRepository.FindAppsByTeamId(deleteRequest.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting team", "teamId", deleteRequest.Id, "err", err)
		return err
	}
	if len(apps) > 0 {
		impl.logger.Errorw("err in deleting team, found apps in team", "teamName", deleteRequest.Name, "err", err)
		return fmt.Errorf(" Please delete all apps in this project before deleting this project")
	}
	err = impl.teamService.Delete(deleteRequest)
	if err != nil {
		impl.logger.Errorw("error in deleting team", "err", err, "deleteRequest", deleteRequest)
		return err
	}
	return nil
}

func (impl DeleteServiceExtendedImpl) DeleteChartRepo(deleteRequest *chartRepo.ChartRepoDto) error {
	//finding if any charts is deployed using this repo, if yes then will not delete
	deployedCharts, err := impl.installedAppRepository.GetAllInstalledAppsByChartRepoId(deleteRequest.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting repo", "deleteRequest", deployedCharts)
		return err
	}
	if len(deployedCharts) > 0 {
		impl.logger.Errorw("err in deleting repo, found charts deployed using this repo", "deleteRequest", deployedCharts)
		return fmt.Errorf("cannot delete repo, found charts deployed in this repo")
	}
	err = impl.chartRepositoryService.DeleteChartRepo(deleteRequest)
	if err != nil {
		impl.logger.Errorw("error in deleting chart repo", "err", err, "deleteRequest", deleteRequest)
		return err
	}
	return nil
}
