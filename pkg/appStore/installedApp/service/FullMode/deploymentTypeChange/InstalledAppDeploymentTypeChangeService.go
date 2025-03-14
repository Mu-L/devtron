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

package deploymentTypeChange

import (
	"context"
	"errors"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apiclient/application"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/argoproj/gitops-engine/pkg/utils/kube"
	k8s2 "github.com/devtron-labs/common-lib/utils/k8s"
	client "github.com/devtron-labs/devtron/api/helm-app/service"
	helmBean "github.com/devtron-labs/devtron/api/helm-app/service/bean"
	"github.com/devtron-labs/devtron/client/argocdServer"
	"github.com/devtron-labs/devtron/internal/constants"
	appRepository "github.com/devtron-labs/devtron/internal/sql/repository/app"
	appStatus2 "github.com/devtron-labs/devtron/internal/sql/repository/appStatus"
	"github.com/devtron-labs/devtron/internal/util"
	appStoreBean "github.com/devtron-labs/devtron/pkg/appStore/bean"
	"github.com/devtron-labs/devtron/pkg/appStore/chartGroup"
	repository2 "github.com/devtron-labs/devtron/pkg/appStore/installedApp/repository"
	deployment2 "github.com/devtron-labs/devtron/pkg/appStore/installedApp/service/EAMode/deployment"
	"github.com/devtron-labs/devtron/pkg/appStore/installedApp/service/FullMode/deployment"
	util2 "github.com/devtron-labs/devtron/pkg/appStore/util"
	"github.com/devtron-labs/devtron/pkg/argoApplication"
	bean4 "github.com/devtron-labs/devtron/pkg/argoApplication/bean"
	"github.com/devtron-labs/devtron/pkg/bean"
	"github.com/devtron-labs/devtron/pkg/cluster"
	"github.com/devtron-labs/devtron/pkg/cluster/environment/repository"
	"github.com/devtron-labs/devtron/pkg/cluster/read"
	repository5 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	"github.com/devtron-labs/devtron/pkg/deployment/common"
	bean3 "github.com/devtron-labs/devtron/pkg/deployment/common/bean"
	"github.com/devtron-labs/devtron/pkg/deployment/gitOps/config"
	bean2 "github.com/devtron-labs/devtron/pkg/deployment/trigger/devtronApps/bean"
	"github.com/devtron-labs/devtron/pkg/k8s"
	util3 "github.com/devtron-labs/devtron/util"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"time"
)

type InstalledAppDeploymentTypeChangeService interface {
	// MigrateDeploymentType migrates the deployment type of installed app and then trigger in loop
	MigrateDeploymentType(ctx context.Context, request *bean.DeploymentAppTypeChangeRequest) (*bean.DeploymentAppTypeChangeResponse, error)
	// TriggerAfterMigration triggers all the installed apps for which the deployment types were migrated via MigrateDeploymentType
	TriggerAfterMigration(ctx context.Context, request *bean.DeploymentAppTypeChangeRequest) (*bean.DeploymentAppTypeChangeResponse, error)
}

type InstalledAppDeploymentTypeChangeServiceImpl struct {
	logger                        *zap.SugaredLogger
	installedAppRepository        repository2.InstalledAppRepository
	installedAppRepositoryHistory repository2.InstalledAppVersionHistoryRepository
	appStatusRepository           appStatus2.AppStatusRepository
	appRepository                 appRepository.AppRepository
	gitOpsConfigReadService       config.GitOpsConfigReadService
	environmentRepository         repository.EnvironmentRepository
	k8sCommonService              k8s.K8sCommonService
	k8sUtil                       k8s2.K8sService
	fullModeDeploymentService     deployment.FullModeDeploymentService
	eaModeDeploymentService       deployment2.EAModeDeploymentService
	argoClientWrapperService      argocdServer.ArgoClientWrapperService
	chartGroupService             chartGroup.ChartGroupService
	helmAppService                client.HelmAppService
	clusterService                cluster.ClusterService
	clusterReadService            read.ClusterReadService
	deploymentConfigService       common.DeploymentConfigService
	argoApplicationService        argoApplication.ArgoApplicationService
}

func NewInstalledAppDeploymentTypeChangeServiceImpl(logger *zap.SugaredLogger,
	installedAppRepository repository2.InstalledAppRepository,
	installedAppRepositoryHistory repository2.InstalledAppVersionHistoryRepository,
	appStatusRepository appStatus2.AppStatusRepository,
	gitOpsConfigReadService config.GitOpsConfigReadService,
	environmentRepository repository.EnvironmentRepository,
	k8sCommonService k8s.K8sCommonService,
	k8sUtil k8s2.K8sService, fullModeDeploymentService deployment.FullModeDeploymentService,
	eaModeDeploymentService deployment2.EAModeDeploymentService,
	argoClientWrapperService argocdServer.ArgoClientWrapperService,
	chartGroupService chartGroup.ChartGroupService, helmAppService client.HelmAppService,
	clusterService cluster.ClusterService,
	clusterReadService read.ClusterReadService,
	appRepository appRepository.AppRepository,
	deploymentConfigService common.DeploymentConfigService,
	argoApplicationService argoApplication.ArgoApplicationService) *InstalledAppDeploymentTypeChangeServiceImpl {
	return &InstalledAppDeploymentTypeChangeServiceImpl{
		logger:                        logger,
		installedAppRepository:        installedAppRepository,
		installedAppRepositoryHistory: installedAppRepositoryHistory,
		appStatusRepository:           appStatusRepository,
		gitOpsConfigReadService:       gitOpsConfigReadService,
		environmentRepository:         environmentRepository,
		k8sCommonService:              k8sCommonService,
		k8sUtil:                       k8sUtil,
		fullModeDeploymentService:     fullModeDeploymentService,
		eaModeDeploymentService:       eaModeDeploymentService,
		argoClientWrapperService:      argoClientWrapperService,
		chartGroupService:             chartGroupService,
		helmAppService:                helmAppService,
		clusterService:                clusterService,
		clusterReadService:            clusterReadService,
		appRepository:                 appRepository,
		deploymentConfigService:       deploymentConfigService,
		argoApplicationService:        argoApplicationService,
	}
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) MigrateDeploymentType(ctx context.Context, request *bean.DeploymentAppTypeChangeRequest) (*bean.DeploymentAppTypeChangeResponse, error) {
	response := &bean.DeploymentAppTypeChangeResponse{
		EnvId:                 request.EnvId,
		DesiredDeploymentType: request.DesiredDeploymentType,
	}
	var err error

	var deleteDeploymentType bean2.DeploymentType
	var deployStatus appStoreBean.AppstoreDeploymentStatus
	if request.DesiredDeploymentType == bean2.ArgoCd {
		deleteDeploymentType = bean2.Helm
		deployStatus = appStoreBean.DEPLOY_INIT
	} else {
		deleteDeploymentType = bean2.ArgoCd
		deployStatus = appStoreBean.DEPLOY_SUCCESS
	}
	envBean, err := impl.environmentRepository.FindById(request.EnvId)
	if err != nil {
		impl.logger.Errorw("error in getting environment by envId", "envId", request.EnvId, "err", err)
		return response, err
	}
	//if cluster unreachable return with error, this is done to handle the case when cluster is unreachable and
	//delete req sent to argo cd the app deletion is stuck in deleting state
	isClusterReachable, err := impl.clusterReadService.IsClusterReachable(envBean.ClusterId)
	if err != nil {
		return response, err
	}
	if !isClusterReachable {
		return response, &util.ApiError{HttpStatusCode: http.StatusUnprocessableEntity, InternalMessage: err.Error(), UserMessage: "cluster unreachable"}
	}

	installedApps, err := impl.installedAppRepository.GetActiveInstalledAppByEnvIdAndDeploymentType(request.EnvId,
		deleteDeploymentType, util2.ConvertIntArrayToStringArray(request.ExcludeApps), util2.ConvertIntArrayToStringArray(request.IncludeApps))
	if err != nil {
		impl.logger.Errorw("error in fetching installed apps by env id and deployment type", "endId", request.EnvId, "deleteDeploymentType", deleteDeploymentType)
		return response, err
	}
	var installedAppIds []int
	for _, installedApp := range installedApps {
		if util2.IsExternalChartStoreApp(installedApp.App.DisplayName) {
			//for ext-apps, appName is a unique identifier pertaining to devtron environment hence changing appName to ReleaseName, as going
			//further interactions with helm/argo-cd will happen via release name only so refrain from doing any db updates using this installed apps
			installedApp.App.AppName = installedApp.App.DisplayName
		}
		installedAppIds = append(installedAppIds, installedApp.Id)
	}

	if len(installedAppIds) == 0 {
		return response, &util.ApiError{HttpStatusCode: http.StatusNotFound, UserMessage: fmt.Sprintf("no installed apps found for this desired deployment type %s", request.DesiredDeploymentType)}
	}
	if request.DesiredDeploymentType == bean2.Helm {
		//before deleting the installed app we'll first annotate CRD's manifest created by argo-cd with helm supported
		//annotations so that helm install doesn't throw crd already exist error while migrating from argo-cd to helm.
		for _, installedApp := range installedApps {
			err = impl.AnnotateCRDsIfExist(ctx, installedApp.App.AppName, installedApp.Environment.Name, installedApp.Environment.Namespace, installedApp.Environment.ClusterId)
			if err != nil {
				impl.logger.Errorw("error in annotating CRDs in manifest for argo-cd deployed installed apps", "installedAppId", installedApp.Id, "appId", installedApp.AppId)
				return response, err
			}
		}
	}

	deleteResponse, err := impl.deleteInstalledApps(ctx, installedApps, request.UserId, envBean.Cluster)
	if err != nil {
		return response, err
	}
	response.SuccessfulPipelines = deleteResponse.SuccessfulPipelines
	response.FailedPipelines = deleteResponse.FailedPipelines

	var successInstalledAppIds []int
	for _, item := range response.SuccessfulPipelines {
		successInstalledAppIds = append(successInstalledAppIds, item.InstalledAppId)
	}

	var successAppIds []*int
	for _, item := range response.SuccessfulPipelines {
		successAppIds = append(successAppIds, &item.AppId)
	}

	err = impl.performDbOperationsAfterMigrations(request.DesiredDeploymentType, successInstalledAppIds, successAppIds, request.UserId, int(deployStatus))
	if err != nil {
		impl.logger.Errorw("error in performing db operations for successful installed apps after migration",
			"envId", request.EnvId,
			"successfully deleted installedApp ids", successInstalledAppIds,
			"desired deployment type", request.DesiredDeploymentType,
			"err", err)

		return response, err
	}

	return response, nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) performDbOperationsAfterMigrations(desiredDeploymentType bean2.DeploymentType,
	successInstalledAppIds []int, successAppIds []*int, userId int32, deployStatus int) error {

	installedApps, err := impl.installedAppRepository.FindInstalledAppByIds(successInstalledAppIds)
	if err != nil {
		impl.logger.Errorw("error in getting installed apps by ids", "installedAppIds", successInstalledAppIds, "err", err)
		return err
	}

	for _, ia := range installedApps {
		deploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForHelmApp(ia.AppId, ia.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in getting deployment config by appId and envId", "appId", ia.AppId, "envId", ia.EnvironmentId, "err", err)
			return err
		}
		deploymentConfig.DeploymentAppType = desiredDeploymentType
		deploymentConfig, err = impl.deploymentConfigService.CreateOrUpdateConfig(nil, deploymentConfig, userId)
		if err != nil {
			impl.logger.Errorw("error in updating deployment config", "appId", ia.AppId, "envId", ia.EnvironmentId, "err", err)
			return err
		}
	}

	err = impl.installedAppRepository.UpdateDeploymentAppTypeInInstalledApp(desiredDeploymentType, successInstalledAppIds, userId, deployStatus)
	if err != nil {
		impl.logger.Errorw("failed to update deployment app type for successfully deleted installed apps in db",
			"successfully deleted installedApp ids", successInstalledAppIds,
			"desired deployment type", desiredDeploymentType,
			"err", err)

		return err
	}
	if desiredDeploymentType == bean2.ArgoCd {
		//this is to handle the case when an external helm app linked to devtron is being
		//migrated to argo_cd then it's app offering mode should be full mode
		err = impl.appRepository.UpdateAppOfferingModeForAppIds(successAppIds, util3.SERVER_MODE_FULL, userId)
		if err != nil {
			impl.logger.Errorw("error in updating app offering mode for successful migrated appIds",
				"successAppIds", successAppIds,
				"desired deployment type", desiredDeploymentType,
				"err", err)

			return err
		}
	}
	return nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) AnnotateCRDsIfExist(ctx context.Context, appName, envName, namespace string, clusterId int) error {
	deploymentAppName := util3.BuildDeployedAppName(appName, envName)
	query := &application.ResourcesQuery{
		ApplicationName: &deploymentAppName,
	}
	resp, err := impl.argoApplicationService.GetResourceTree(ctx, bean4.NewImperativeQueryRequest(query))
	if err != nil {
		impl.logger.Errorw("error in fetching resource tree", "err", err)
		err = &util.ApiError{
			HttpStatusCode:  http.StatusNotFound,
			Code:            constants.AppDetailResourceTreeNotFound,
			InternalMessage: err.Error(),
			UserMessage:     "failed to get resource tree from acd",
		}
		return err
	}
	crdsList := make([]v1alpha1.ResourceNode, 0)
	for _, node := range resp.ApplicationTree.Nodes {
		if node.ResourceRef.Kind == kube.CustomResourceDefinitionKind {
			crdsList = append(crdsList, node)
		}
	}
	restConfig, err, _ := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster Id", "clusterId", clusterId, "err", err)
		return err
	}
	for _, crd := range crdsList {
		gvk := schema.GroupVersionKind{
			Group:   crd.ResourceRef.Group,
			Version: crd.ResourceRef.Version,
			Kind:    crd.ResourceRef.Kind,
		}
		helmAnnotation := fmt.Sprintf(bean.HelmReleaseMetadataAnnotation, appName, namespace)
		_, err = impl.k8sUtil.PatchResourceRequest(ctx, restConfig, types.StrategicMergePatchType, helmAnnotation, crd.ResourceRef.Name, "", gvk)
		if err != nil {
			impl.logger.Errorw("error in patching release-name annotation in manifest", "appName", appName, "err", err)
			return err
		}
	}
	return nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) deleteInstalledApps(ctx context.Context, installedApps []*repository2.InstalledApps, userId int32, cluster *repository5.Cluster) (*bean.DeploymentAppTypeChangeResponse, error) {
	successfullyDeletedApps := make([]*bean.DeploymentChangeStatus, 0)
	failedToDeleteApps := make([]*bean.DeploymentChangeStatus, 0)

	gitOpsConfigStatus, gitOpsConfigErr := impl.gitOpsConfigReadService.IsGitOpsConfigured()

	for _, installedApp := range installedApps {
		installedApp.Environment.Cluster = cluster

		deploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForHelmApp(installedApp.AppId, installedApp.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in getiting deployment config db object by appId and envId", "appId", installedApp.AppId, "envId", installedApp.EnvironmentId, "err", err)
			return nil, err
		}

		var isValid bool
		// check if installed app info like app name and environment is empty or not
		if failedToDeleteApps, isValid = impl.isInstalledAppInfoValid(installedApp, failedToDeleteApps); !isValid {
			continue
		}

		var healthChkErr error
		// check health of the app if it is argo-cd deployment type
		if _, healthChkErr = impl.handleNotDeployedAppsIfArgoDeploymentType(installedApp, deploymentConfig, failedToDeleteApps); healthChkErr != nil {
			// cannot delete unhealthy app
			continue
		}

		deploymentAppName := util3.BuildDeployedAppName(installedApp.App.AppName, installedApp.Environment.Name)
		// delete request
		if deploymentConfig.DeploymentAppType == bean2.ArgoCd {
			err = impl.fullModeDeploymentService.DeleteACD(deploymentAppName, ctx, false)
		} else if deploymentConfig.DeploymentAppType == bean2.Helm {
			// For converting from Helm to ArgoCD, GitOps should be configured
			if gitOpsConfigErr != nil || !gitOpsConfigStatus.IsGitOpsConfiguredAndArgoCdInstalled() {
				err = &util.ApiError{HttpStatusCode: http.StatusBadRequest, Code: "200", UserMessage: errors.New("GitOps not configured or unable to fetch GitOps configuration")}
			}
			if err != nil {
				impl.logger.Errorw("error registering app on ACD with error: "+err.Error(),
					"deploymentAppName", deploymentAppName,
					"envId", installedApp.EnvironmentId,
					"appId", installedApp.AppId,
					"err", err)

				// deletion failed, append to the list of failed to delete installed apps
				failedToDeleteApps = impl.handleFailedInstalledAppChange(installedApp, failedToDeleteApps, appStoreBean.FAILED_TO_REGISTER_IN_ACD_ERROR+err.Error())
				continue
			}

			installAppVersionRequest := &appStoreBean.InstallAppVersionDTO{
				ClusterId: installedApp.Environment.ClusterId,
				AppName:   installedApp.App.AppName,
				Namespace: installedApp.Environment.Namespace,
			}
			err = impl.eaModeDeploymentService.DeleteInstalledApp(ctx, "", "", installAppVersionRequest, nil, nil)
		}

		if err != nil {
			impl.logger.Errorw("error deleting app on "+deploymentConfig.DeploymentAppType,
				"deployment app name", deploymentAppName,
				"err", err)

			// deletion failed, append to the list of failed pipelines
			failedToDeleteApps = impl.handleFailedInstalledAppChange(installedApp, failedToDeleteApps, appStoreBean.FAILED_TO_DELETE_APP_PREFIX_ERROR+err.Error())
			continue
		}
		// deletion successful, append to the list of successful pipelines
		successfullyDeletedApps = appendToDeploymentChangeStatusList(successfullyDeletedApps, installedApp, "", bean.INITIATED)

	}
	return &bean.DeploymentAppTypeChangeResponse{
		SuccessfulPipelines: successfullyDeletedApps,
		FailedPipelines:     failedToDeleteApps,
	}, nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) isInstalledAppInfoValid(installedApp *repository2.InstalledApps,
	failedToDeleteApps []*bean.DeploymentChangeStatus) ([]*bean.DeploymentChangeStatus, bool) {

	if len(installedApp.App.AppName) == 0 || len(installedApp.Environment.Name) == 0 {
		impl.logger.Errorw("app name or environment name is not present", "installed app id", installedApp.Id)

		failedToDeleteApps = impl.handleFailedInstalledAppChange(installedApp, failedToDeleteApps, appStoreBean.COULD_NOT_FETCH_APP_NAME_AND_ENV_NAME_ERR)

		return failedToDeleteApps, false
	}
	return failedToDeleteApps, true
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) handleNotDeployedAppsIfArgoDeploymentType(installedApp *repository2.InstalledApps,
	deploymentConfig *bean3.DeploymentConfig,
	failedToDeleteApps []*bean.DeploymentChangeStatus) ([]*bean.DeploymentChangeStatus, error) {

	if deploymentConfig.DeploymentAppType == bean2.ArgoCd {
		// check if app status is Healthy
		status, err := impl.appStatusRepository.Get(installedApp.AppId, installedApp.EnvironmentId)

		// case: missing status row in db
		if len(status.Status) == 0 {
			return failedToDeleteApps, nil
		}

		// cannot delete the app from argo-cd if app status is Progressing
		if err != nil {
			healthCheckErr := errors.New("unable to fetch app status")
			impl.logger.Errorw(healthCheckErr.Error(), "appId", installedApp.AppId, "environmentId", installedApp.EnvironmentId, "err", err)
			failedToDeleteApps = impl.handleFailedInstalledAppChange(installedApp, failedToDeleteApps, healthCheckErr.Error())
			return failedToDeleteApps, healthCheckErr
		}
		return failedToDeleteApps, nil
	}
	return failedToDeleteApps, nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) handleFailedInstalledAppChange(installedApp *repository2.InstalledApps,
	failedPipelines []*bean.DeploymentChangeStatus, err string) []*bean.DeploymentChangeStatus {

	return appendToDeploymentChangeStatusList(failedPipelines, installedApp, err, bean.Failed)
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) TriggerAfterMigration(ctx context.Context, request *bean.DeploymentAppTypeChangeRequest) (*bean.DeploymentAppTypeChangeResponse, error) {
	response := &bean.DeploymentAppTypeChangeResponse{
		EnvId:                 request.EnvId,
		DesiredDeploymentType: request.DesiredDeploymentType,
	}
	var err error

	installedApps, err := impl.installedAppRepository.GetActiveInstalledAppByEnvIdAndDeploymentType(request.EnvId, request.DesiredDeploymentType,
		util2.ConvertIntArrayToStringArray(request.ExcludeApps), util2.ConvertIntArrayToStringArray(request.IncludeApps))

	if err != nil {
		impl.logger.Errorw("Error fetching installed apps",
			"environmentId", request.EnvId,
			"desiredDeploymentAppType", request.DesiredDeploymentType,
			"err", err)
		return response, err
	}

	var installedAppIds []int
	for _, installedApp := range installedApps {
		if util2.IsExternalChartStoreApp(installedApp.App.DisplayName) {
			//for ext-apps, appName is a unique identifier pertaining to devtron environment hence changing appName to ReleaseName, as going
			//further interactions with helm/argo-cd will happen via release name only so refrain from doing any db updates using this installed apps
			installedApp.App.AppName = installedApp.App.DisplayName
		}
		installedAppIds = append(installedAppIds, installedApp.Id)
	}

	if len(installedAppIds) == 0 {
		return response, nil
	}

	deleteResponse := impl.fetchDeletedInstalledApp(ctx, installedApps)

	response.SuccessfulPipelines = deleteResponse.SuccessfulPipelines
	response.FailedPipelines = deleteResponse.FailedPipelines

	successfulInstalledAppIds := make([]int, 0, len(response.SuccessfulPipelines))
	for _, item := range response.SuccessfulPipelines {
		successfulInstalledAppIds = append(successfulInstalledAppIds, item.InstalledAppId)
	}

	successInstalledApps, err := impl.installedAppRepository.FindInstalledAppByIds(successfulInstalledAppIds)
	if err != nil {
		impl.logger.Errorw("failed to fetch installed app details",
			"ids", successfulInstalledAppIds,
			"err", err)

		return response, nil
	}
	installedAppVersionDTOList, err := impl.getDtoListForTriggerDeploymentEvent(request.DesiredDeploymentType, successInstalledApps)
	if err != nil {
		impl.logger.Errorw("error in getting Dto list for trigger deployment event",
			"environmentId", request.EnvId,
			"desiredDeploymentAppType", request.DesiredDeploymentType,
			"err", err)
		return response, err
	}

	impl.chartGroupService.TriggerDeploymentEventAndHandleStatusUpdate(installedAppVersionDTOList)

	err = impl.performDbOperationsAfterTrigger(request.DesiredDeploymentType, successInstalledApps)
	if err != nil {
		impl.logger.Errorw("error in performing db operations for successful installed apps after trigger",
			"envId", request.EnvId,
			"successfully deleted installedApp ids", successfulInstalledAppIds,
			"desired deployment type", request.DesiredDeploymentType,
			"err", err)

		return response, err
	}

	return response, nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) performDbOperationsAfterTrigger(desiredDeploymentType bean2.DeploymentType, successInstalledApps []*repository2.InstalledApps) error {
	if desiredDeploymentType == bean2.Helm {
		err := impl.deleteAppStatusEntryAfterTrigger(successInstalledApps)
		if err != nil && err == pg.ErrNoRows {
			impl.logger.Infow("app status already deleted or not found after trigger and migration from argo-cd to helm",
				"desiredDeploymentAppType", desiredDeploymentType)
		} else if err != nil {
			impl.logger.Errorw("error in getting deleting app status entry from db after trigger and migration from argo-cd to helm",
				"desiredDeploymentAppType", desiredDeploymentType,
				"err", err)
			return err
		}
	}
	return nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) getDtoListForTriggerDeploymentEvent(desiredDeploymentType bean2.DeploymentType, successInstalledApps []*repository2.InstalledApps) ([]*appStoreBean.InstallAppVersionDTO, error) {
	var installedAppVersionDTOList []*appStoreBean.InstallAppVersionDTO
	for _, installedApp := range successInstalledApps {
		installedAppVersion, err := impl.installedAppRepository.GetActiveInstalledAppVersionByInstalledAppId(installedApp.Id)
		if err != nil {
			impl.logger.Errorw("error in getting installedAppVersion from installedAppId", "installedAppId", installedApp.Id,
				"err", err)
			return nil, err
		}
		installedAppVersionHistory, err := impl.installedAppRepositoryHistory.GetLatestInstalledAppVersionHistory(installedAppVersion.Id)
		if err != nil {
			impl.logger.Errorw("error in getting installedAppVersionHistory from installedAppVersionId", "installedAppVersionId", installedAppVersion.Id,
				"err", err)
			return nil, err
		}
		err = impl.updateDeployedOnDataForTrigger(desiredDeploymentType, installedAppVersion, installedAppVersionHistory)
		if err != nil {
			impl.logger.Errorw("error in updating deployment on data for trigger", "err", err)
			return nil, err
		}
		installedAppVersionDTOList = append(installedAppVersionDTOList, &appStoreBean.InstallAppVersionDTO{
			InstalledAppVersionId:        installedAppVersion.Id,
			InstalledAppVersionHistoryId: installedAppVersionHistory.Id,
			Status:                       appStoreBean.DEPLOY_INIT,
		})
	}
	return installedAppVersionDTOList, nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) updateDeployedOnDataForTrigger(desiredDeploymentType bean2.DeploymentType, installedAppVersion *repository2.InstalledAppVersions, installedAppVersionHistory *repository2.InstalledAppVersionHistory) error {
	if desiredDeploymentType == bean2.Helm {
		//for helm, on ui we show  last deployed installed app versions table
		installedAppVersion.UpdatedOn = time.Now()
		_, err := impl.installedAppRepository.UpdateInstalledAppVersion(installedAppVersion, nil)
		if err != nil {
			impl.logger.Errorw("failed to update last deployed time in installed app version",
				"installedAppVersionId", installedAppVersion.Id,
				"err", err)
			return err
		}
	} else if desiredDeploymentType == bean2.ArgoCd {
		//for argo-cd deployments, on ui we show last deployed time from installed app version history table
		installedAppVersionHistory.StartedOn, installedAppVersionHistory.UpdatedOn = time.Now(), time.Now()

		_, err := impl.installedAppRepositoryHistory.UpdateInstalledAppVersionHistory(installedAppVersionHistory, nil)
		if err != nil {
			impl.logger.Errorw("failed to update deployed on time in installed app version history",
				"installedAppVersionHistoryId", installedAppVersionHistory.Id,
				"err", err)
			return err
		}
	}
	return nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) deleteAppStatusEntryAfterTrigger(successInstalledApps []*repository2.InstalledApps) error {
	dbConnection := impl.appStatusRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		impl.logger.Errorw("error in getting dbConnection", "err", err)
		return err
	}
	defer tx.Rollback()
	for _, installedApp := range successInstalledApps {
		_, err := impl.appStatusRepository.Get(installedApp.AppId, installedApp.EnvironmentId)
		if err != nil && err == pg.ErrNoRows {
			impl.logger.Errorw("app status for installed already deleted or not found",
				"appId", installedApp.AppId,
				"installedAppId", installedApp.Id,
				"err", err)
			continue
		}
		//delete entry from app_status table
		err = impl.appStatusRepository.Delete(tx, installedApp.AppId, installedApp.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in deleting appStatus for installed app",
				"appId", installedApp.AppId,
				"installedAppId", installedApp.Id,
				"err", err)
			return err
		}
	}
	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("error in committing db transaction", "err", err)
		return err
	}
	return nil
}

func (impl *InstalledAppDeploymentTypeChangeServiceImpl) fetchDeletedInstalledApp(ctx context.Context,
	installedApps []*repository2.InstalledApps) *bean.DeploymentAppTypeChangeResponse {

	successfulInstalledApps := make([]*bean.DeploymentChangeStatus, 0)
	failedInstalledApps := make([]*bean.DeploymentChangeStatus, 0)

	for _, installedApp := range installedApps {

		deploymentAppName := util3.BuildDeployedAppName(installedApp.App.AppName, installedApp.Environment.Name)
		var err error
		if installedApp.DeploymentAppType == bean2.ArgoCd {
			appIdentifier := &helmBean.AppIdentifier{
				ClusterId:   installedApp.Environment.ClusterId,
				ReleaseName: deploymentAppName,
				Namespace:   installedApp.Environment.Namespace,
			}
			_, err = impl.helmAppService.GetApplicationDetail(ctx, appIdentifier)
		} else {
			_, err = impl.argoClientWrapperService.GetArgoAppByName(ctx, deploymentAppName)
		}
		if err != nil {
			impl.logger.Errorw("error in getting application detail", "deploymentAppName", deploymentAppName, "err", err)
		}

		if err != nil && util2.CheckAppReleaseNotExist(err) {
			successfulInstalledApps = appendToDeploymentChangeStatusList(successfulInstalledApps, installedApp, "", bean.Success)
		} else {
			failError := appStoreBean.APP_NOT_DELETED_YET_ERROR
			failStatus := bean.NOT_YET_DELETED
			if util2.CheckPermissionErrorForArgoCd(err) {
				failError = string(bean.PermissionDenied)
				failStatus = bean.Failed
			}
			failedInstalledApps = appendToDeploymentChangeStatusList(failedInstalledApps, installedApp, failError, failStatus)
		}
	}

	return &bean.DeploymentAppTypeChangeResponse{
		SuccessfulPipelines: successfulInstalledApps,
		FailedPipelines:     failedInstalledApps,
	}
}

func appendToDeploymentChangeStatusList(installedApps []*bean.DeploymentChangeStatus,
	installedApp *repository2.InstalledApps, error string, status bean.Status) []*bean.DeploymentChangeStatus {

	return append(installedApps, &bean.DeploymentChangeStatus{
		InstalledAppId: installedApp.Id,
		AppId:          installedApp.AppId,
		AppName:        installedApp.App.AppName,
		EnvId:          installedApp.EnvironmentId,
		EnvName:        installedApp.Environment.Name,
		Error:          error,
		Status:         status,
	})
}
