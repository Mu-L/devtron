package delete

import (
	"fmt"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	dockerRegistryRepository "github.com/devtron-labs/devtron/internal/sql/repository/dockerRegistry"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/util"
	appStoreBean "github.com/devtron-labs/devtron/pkg/appStore/bean"
	repository2 "github.com/devtron-labs/devtron/pkg/appStore/installedApp/repository"
	"github.com/devtron-labs/devtron/pkg/chartRepo"
	"github.com/devtron-labs/devtron/pkg/cluster"
	clusterBean "github.com/devtron-labs/devtron/pkg/cluster/bean"

	"github.com/devtron-labs/devtron/pkg/cluster/repository"
	"github.com/devtron-labs/devtron/pkg/cluster/repository/bean"
	"github.com/devtron-labs/devtron/pkg/devtronResource"
	bean6 "github.com/devtron-labs/devtron/pkg/devtronResource/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline"
	"github.com/devtron-labs/devtron/pkg/team"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"net/http"
)

type DeleteServiceExtendedImpl struct {
	appRepository          app.AppRepository
	environmentRepository  repository.EnvironmentRepository
	pipelineRepository     pipelineConfig.PipelineRepository
	devtronResourceService devtronResource.DevtronResourceService
	*DeleteServiceImpl
}

func NewDeleteServiceExtendedImpl(logger *zap.SugaredLogger,
	teamService team.TeamService,
	clusterService cluster.ClusterService,
	environmentService cluster.EnvironmentService,
	appRepository app.AppRepository,
	environmentRepository repository.EnvironmentRepository,
	pipelineRepository pipelineConfig.PipelineRepository,
	chartRepositoryService chartRepo.ChartRepositoryService,
	installedAppRepository repository2.InstalledAppRepository,
	dockerRegistryConfig pipeline.DockerRegistryConfig,
	dockerRegistryRepository dockerRegistryRepository.DockerArtifactStoreRepository,
	devtronResourceService devtronResource.DevtronResourceService) *DeleteServiceExtendedImpl {
	return &DeleteServiceExtendedImpl{
		appRepository:          appRepository,
		environmentRepository:  environmentRepository,
		pipelineRepository:     pipelineRepository,
		devtronResourceService: devtronResourceService,
		DeleteServiceImpl: &DeleteServiceImpl{
			logger:                   logger,
			teamService:              teamService,
			clusterService:           clusterService,
			environmentService:       environmentService,
			chartRepositoryService:   chartRepositoryService,
			installedAppRepository:   installedAppRepository,
			dockerRegistryConfig:     dockerRegistryConfig,
			dockerRegistryRepository: dockerRegistryRepository,
		},
	}
}

func (impl DeleteServiceExtendedImpl) DeleteCluster(deleteRequest *clusterBean.ClusterBean, userId int32) error {
	//finding if there are env in this cluster or not, if yes then will not delete
	env, err := impl.environmentRepository.FindByClusterId(deleteRequest.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting cluster", "clusterName", deleteRequest.ClusterName, "err", err)
		return err
	}
	if len(env) > 0 {
		impl.logger.Errorw("err in deleting cluster, found env in this cluster", "clusterName", deleteRequest.ClusterName, "err", err)
		return &util.ApiError{HttpStatusCode: http.StatusBadRequest, UserMessage: " Please delete all related environments before deleting this cluster"}
	}
	err = impl.clusterService.DeleteFromDb(deleteRequest, userId)
	if err != nil {
		impl.logger.Errorw("error im deleting cluster", "err", err, "deleteRequest", deleteRequest)
		return err
	}
	go func() {
		errInResourceDelete := impl.devtronResourceService.DeleteObjectAndItsDependency(deleteRequest.Id, bean6.DevtronResourceCluster,
			"", bean6.DevtronResourceVersion1, userId)
		if errInResourceDelete != nil {
			impl.logger.Errorw("error in deleting cluster resource and dependency data", "err", err, "clusterId", deleteRequest.Id)
		}
	}()
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
func (impl DeleteServiceExtendedImpl) DeleteTeam(deleteRequest *team.TeamRequest) error {
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

func (impl DeleteServiceExtendedImpl) DeleteVirtualCluster(deleteRequest *clusterBean.VirtualClusterBean, userId int32) error {
	//finding if there are env in this cluster or not, if yes then will not delete
	env, err := impl.environmentRepository.FindByClusterId(deleteRequest.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting cluster", "clusterName", deleteRequest.ClusterName, "err", err)
		return err
	}
	if len(env) > 0 {
		impl.logger.Errorw("err in deleting cluster, found env in this cluster", "clusterName", deleteRequest.ClusterName, "err", err)
		return fmt.Errorf(" Please delete all related environments before deleting this cluster")
	}
	err = impl.clusterService.DeleteVirtualClusterFromDb(deleteRequest, userId)
	if err != nil {
		impl.logger.Errorw("error im deleting cluster", "err", err, "deleteRequest", deleteRequest)
		return err
	}
	return nil
}
