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

package pipeline

import (
	"context"
	"encoding/json"
	errors3 "errors"
	"fmt"
	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	bean2 "github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/api/bean/gitOps"
	models2 "github.com/devtron-labs/devtron/api/helm-app/models"
	client "github.com/devtron-labs/devtron/api/helm-app/service"
	helmBean "github.com/devtron-labs/devtron/api/helm-app/service/bean"
	"github.com/devtron-labs/devtron/client/argocdServer"
	"github.com/devtron-labs/devtron/internal/sql/models"
	"github.com/devtron-labs/devtron/internal/sql/repository"
	app2 "github.com/devtron-labs/devtron/internal/sql/repository/app"
	"github.com/devtron-labs/devtron/internal/sql/repository/appStatus"
	"github.com/devtron-labs/devtron/internal/sql/repository/appWorkflow"
	"github.com/devtron-labs/devtron/internal/sql/repository/chartConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig/bean/workflow/cdWorkflow"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/app"
	installedAppReader "github.com/devtron-labs/devtron/pkg/appStore/installedApp/read"
	"github.com/devtron-labs/devtron/pkg/bean"
	"github.com/devtron-labs/devtron/pkg/chart"
	chartRepoRepository "github.com/devtron-labs/devtron/pkg/chartRepo/repository"
	bean3 "github.com/devtron-labs/devtron/pkg/cluster/bean"
	clutserBean "github.com/devtron-labs/devtron/pkg/cluster/environment/bean"
	repository6 "github.com/devtron-labs/devtron/pkg/cluster/environment/repository"
	read2 "github.com/devtron-labs/devtron/pkg/cluster/read"
	repository2 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	"github.com/devtron-labs/devtron/pkg/deployment/common"
	bean4 "github.com/devtron-labs/devtron/pkg/deployment/common/bean"
	commonBean "github.com/devtron-labs/devtron/pkg/deployment/gitOps/common/bean"
	"github.com/devtron-labs/devtron/pkg/deployment/gitOps/config"
	"github.com/devtron-labs/devtron/pkg/deployment/gitOps/git"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deployedAppMetrics"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate"
	bean5 "github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/bean"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/read"
	config2 "github.com/devtron-labs/devtron/pkg/deployment/providerConfig"
	clientErrors "github.com/devtron-labs/devtron/pkg/errors"
	"github.com/devtron-labs/devtron/pkg/eventProcessor/out"
	"github.com/devtron-labs/devtron/pkg/imageDigestPolicy"
	pipelineConfigBean "github.com/devtron-labs/devtron/pkg/pipeline/bean"
	"github.com/devtron-labs/devtron/pkg/pipeline/history"
	repository4 "github.com/devtron-labs/devtron/pkg/pipeline/history/repository"
	repository5 "github.com/devtron-labs/devtron/pkg/pipeline/repository"
	resourceGroup2 "github.com/devtron-labs/devtron/pkg/resourceGroup"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/devtron-labs/devtron/pkg/variables"
	repository3 "github.com/devtron-labs/devtron/pkg/variables/repository"
	util2 "github.com/devtron-labs/devtron/util"
	"github.com/devtron-labs/devtron/util/rbac"
	"github.com/go-pg/pg"
	errors2 "github.com/juju/errors"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	chart2 "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/api/errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type CdPipelineConfigService interface {
	// GetCdPipelineById : Retrieve cdPipeline for given cdPipelineId.
	// getting cdPipeline,environment and strategies ,preDeployStage, postDeployStage,appWorkflowMapping from respective repository and service layer
	// converting above data in proper bean object and then assigning to CDPipelineConfigObject
	// if any error occur , will get empty object or nil
	GetCdPipelineById(pipelineId int) (cdPipeline *bean.CDPipelineConfigObject, err error)
	CreateCdPipelines(cdPipelines *bean.CdPipelines, ctx context.Context) (*bean.CdPipelines, error)
	// PatchCdPipelines : Handle CD pipeline patch requests, making necessary changes to the configuration and returning the updated version.
	// Performs Create ,Update and Delete operation.
	ValidateLinkExternalArgoCDRequest(request pipelineConfigBean.ArgoCDAppLinkValidationRequest) pipelineConfigBean.ArgoCdAppLinkValidationResponse
	PatchCdPipelines(cdPipelines *bean.CDPatchRequest, ctx context.Context) (*bean.CdPipelines, error)
	DeleteCdPipeline(pipeline *pipelineConfig.Pipeline, ctx context.Context, deleteAction int, acdDelete bool, userId int32) (*bean.AppDeleteResponseDTO, error)
	DeleteACDAppCdPipelineWithNonCascade(pipeline *pipelineConfig.Pipeline, ctx context.Context, forceDelete bool, userId int32) (err error)
	// GetTriggerViewCdPipelinesForApp :
	GetTriggerViewCdPipelinesForApp(appId int) (cdPipelines *bean.CdPipelines, err error)
	// GetCdPipelinesForApp : Retrieve cdPipeline for given appId
	GetCdPipelinesForApp(appId int) (cdPipelines *bean.CdPipelines, err error)
	// GetCdPipelinesForAppAndEnv : Retrieve cdPipeline for given appId and envId
	GetCdPipelinesForAppAndEnv(appId int, envId int) (cdPipelines *bean.CdPipelines, err error)
	/*	CreateCdPipelines(cdPipelines bean.CdPipelines) (*bean.CdPipelines, error)*/
	// GetCdPipelinesByEnvironment : lists cdPipeline for given environmentId and appIds
	GetCdPipelinesByEnvironment(request resourceGroup2.ResourceGroupingRequest, token string) (cdPipelines *bean.CdPipelines, err error)
	// GetCdPipelinesByEnvironmentMin : lists minimum detail of cdPipelines for given environmentId and appIds
	GetCdPipelinesByEnvironmentMin(request resourceGroup2.ResourceGroupingRequest, token string) (cdPipelines []*bean.CDPipelineConfigObject, err error)
	// PerformBulkActionOnCdPipelines :
	PerformBulkActionOnCdPipelines(dto *bean.CdBulkActionRequestDto, impactedPipelines []*pipelineConfig.Pipeline, ctx context.Context, dryRun bool, userId int32) ([]*bean.CdBulkActionResponseDto, error)
	// FindPipelineById : Retrieve Pipeline object from pipelineRepository for given cdPipelineId
	FindPipelineById(cdPipelineId int) (*pipelineConfig.Pipeline, error)
	// FindAppAndEnvDetailsByPipelineId : Retrieve app and env details for given cdPipelineId
	FindAppAndEnvDetailsByPipelineId(cdPipelineId int) (*pipelineConfig.Pipeline, error)
	// RetrieveParentDetails : Retrieve the parent id and type of the parent.
	// Here ParentId refers to Parent like parent of CD can be CI , PRE-CD .
	// It first fetches the workflow details from the appWorkflow repository.
	// If the workflow is a CD pipeline, it further checks for stage configurations.
	// If the workflow is a webhook, it returns the webhook workflow type.
	// In case of error , it returns 0 for parentId and empty string for parentType
	RetrieveParentDetails(pipelineId int) (parentId int, parentType bean2.WorkflowType, err error)
	// GetEnvironmentByCdPipelineId : Retrieve environmentId for given cdPipelineId
	GetEnvironmentByCdPipelineId(pipelineId int) (int, error)
	GetBulkActionImpactedPipelines(dto *bean.CdBulkActionRequestDto) ([]*pipelineConfig.Pipeline, error) //no usage
	// IsGitOpsRequiredForCD : Determine if GitOps is required for CD based on the provided pipeline creation request
	IsGitOpsRequiredForCD(pipelineCreateRequest *bean.CdPipelines) bool
	MarkGitOpsDevtronAppsDeletedWhereArgoAppIsDeleted(pipeline *pipelineConfig.Pipeline) (bool, error)
	// GetEnvironmentListForAutocompleteFilter : lists environment for given configuration
	GetEnvironmentListForAutocompleteFilter(envName string, clusterIds []int, offset int, size int, token string, checkAuthBatch func(token string, appObject []string, envObject []string) (map[string]bool, map[string]bool), ctx context.Context) (*clutserBean.ResourceGroupingResponse, error)
	RegisterInACD(ctx context.Context, chartGitAttr *commonBean.ChartGitAttribute, userId int32) error
	// DeleteHelmTypePipelineDeploymentApp : Deletes helm release for a pipeline with force flag
	DeleteHelmTypePipelineDeploymentApp(ctx context.Context, forceDelete bool, pipeline *pipelineConfig.Pipeline) error
}

type CdPipelineConfigServiceImpl struct {
	logger                            *zap.SugaredLogger
	pipelineRepository                pipelineConfig.PipelineRepository
	environmentRepository             repository6.EnvironmentRepository
	pipelineConfigRepository          chartConfig.PipelineConfigRepository
	appWorkflowRepository             appWorkflow.AppWorkflowRepository
	pipelineStageService              PipelineStageService
	appRepo                           app2.AppRepository
	appService                        app.AppService
	deploymentGroupRepository         repository.DeploymentGroupRepository
	ciCdPipelineOrchestrator          CiCdPipelineOrchestrator
	appStatusRepository               appStatus.AppStatusRepository
	ciPipelineRepository              pipelineConfig.CiPipelineRepository
	prePostCdScriptHistoryService     history.PrePostCdScriptHistoryService
	clusterRepository                 repository2.ClusterRepository
	helmAppService                    client.HelmAppService
	enforcerUtil                      rbac.EnforcerUtil
	pipelineStrategyHistoryService    history.PipelineStrategyHistoryService
	chartRepository                   chartRepoRepository.ChartRepository
	resourceGroupService              resourceGroup2.ResourceGroupService
	propertiesConfigService           PropertiesConfigService
	deploymentTemplateHistoryService  deploymentTemplate.DeploymentTemplateHistoryService
	scopedVariableManager             variables.ScopedVariableManager
	deploymentConfig                  *util2.DeploymentServiceTypeConfig
	customTagService                  CustomTagService
	ciPipelineConfigService           CiPipelineConfigService
	buildPipelineSwitchService        BuildPipelineSwitchService
	argoClientWrapperService          argocdServer.ArgoClientWrapperService
	deployedAppMetricsService         deployedAppMetrics.DeployedAppMetricsService
	gitOpsConfigReadService           config.GitOpsConfigReadService
	gitOperationService               git.GitOperationService
	chartService                      chart.ChartService
	imageDigestPolicyService          imageDigestPolicy.ImageDigestPolicyService
	pipelineConfigEventPublishService out.PipelineConfigEventPublishService
	deploymentTypeOverrideService     config2.DeploymentTypeOverrideService
	deploymentConfigService           common.DeploymentConfigService
	envConfigOverrideService          read.EnvConfigOverrideService
	chartRefRepository                chartRepoRepository.ChartRefRepository
	chartTemplateService              util.ChartTemplateService
	gitFactory                        *git.GitFactory
	clusterReadService                read2.ClusterReadService
	installedAppReadService           installedAppReader.InstalledAppReadService
}

func NewCdPipelineConfigServiceImpl(logger *zap.SugaredLogger, pipelineRepository pipelineConfig.PipelineRepository,
	environmentRepository repository6.EnvironmentRepository, pipelineConfigRepository chartConfig.PipelineConfigRepository,
	appWorkflowRepository appWorkflow.AppWorkflowRepository, pipelineStageService PipelineStageService,
	appRepo app2.AppRepository, appService app.AppService, deploymentGroupRepository repository.DeploymentGroupRepository,
	ciCdPipelineOrchestrator CiCdPipelineOrchestrator, appStatusRepository appStatus.AppStatusRepository,
	ciPipelineRepository pipelineConfig.CiPipelineRepository, prePostCdScriptHistoryService history.PrePostCdScriptHistoryService,
	clusterRepository repository2.ClusterRepository, helmAppService client.HelmAppService,
	enforcerUtil rbac.EnforcerUtil, pipelineStrategyHistoryService history.PipelineStrategyHistoryService,
	chartRepository chartRepoRepository.ChartRepository, resourceGroupService resourceGroup2.ResourceGroupService,
	propertiesConfigService PropertiesConfigService,
	deploymentTemplateHistoryService deploymentTemplate.DeploymentTemplateHistoryService,
	scopedVariableManager variables.ScopedVariableManager, envVariables *util2.EnvironmentVariables,
	customTagService CustomTagService,
	ciPipelineConfigService CiPipelineConfigService, buildPipelineSwitchService BuildPipelineSwitchService,
	argoClientWrapperService argocdServer.ArgoClientWrapperService,
	deployedAppMetricsService deployedAppMetrics.DeployedAppMetricsService,
	gitOpsConfigReadService config.GitOpsConfigReadService,
	gitOperationService git.GitOperationService,
	chartService chart.ChartService,
	imageDigestPolicyService imageDigestPolicy.ImageDigestPolicyService,
	pipelineConfigEventPublishService out.PipelineConfigEventPublishService,
	deploymentTypeOverrideService config2.DeploymentTypeOverrideService,
	deploymentConfigService common.DeploymentConfigService,
	envConfigOverrideService read.EnvConfigOverrideService,
	chartRefRepository chartRepoRepository.ChartRefRepository,
	chartTemplateService util.ChartTemplateService,
	gitFactory *git.GitFactory,
	clusterReadService read2.ClusterReadService,
	installedAppReadService installedAppReader.InstalledAppReadService) *CdPipelineConfigServiceImpl {
	return &CdPipelineConfigServiceImpl{
		logger:                            logger,
		pipelineRepository:                pipelineRepository,
		environmentRepository:             environmentRepository,
		pipelineConfigRepository:          pipelineConfigRepository,
		appWorkflowRepository:             appWorkflowRepository,
		pipelineStageService:              pipelineStageService,
		appRepo:                           appRepo,
		appService:                        appService,
		deploymentGroupRepository:         deploymentGroupRepository,
		ciCdPipelineOrchestrator:          ciCdPipelineOrchestrator,
		appStatusRepository:               appStatusRepository,
		ciPipelineRepository:              ciPipelineRepository,
		prePostCdScriptHistoryService:     prePostCdScriptHistoryService,
		clusterRepository:                 clusterRepository,
		helmAppService:                    helmAppService,
		enforcerUtil:                      enforcerUtil,
		pipelineStrategyHistoryService:    pipelineStrategyHistoryService,
		chartRepository:                   chartRepository,
		resourceGroupService:              resourceGroupService,
		propertiesConfigService:           propertiesConfigService,
		deploymentTemplateHistoryService:  deploymentTemplateHistoryService,
		scopedVariableManager:             scopedVariableManager,
		deploymentConfig:                  envVariables.DeploymentServiceTypeConfig,
		chartService:                      chartService,
		customTagService:                  customTagService,
		ciPipelineConfigService:           ciPipelineConfigService,
		buildPipelineSwitchService:        buildPipelineSwitchService,
		argoClientWrapperService:          argoClientWrapperService,
		deployedAppMetricsService:         deployedAppMetricsService,
		gitOpsConfigReadService:           gitOpsConfigReadService,
		gitOperationService:               gitOperationService,
		imageDigestPolicyService:          imageDigestPolicyService,
		pipelineConfigEventPublishService: pipelineConfigEventPublishService,
		deploymentTypeOverrideService:     deploymentTypeOverrideService,
		deploymentConfigService:           deploymentConfigService,
		envConfigOverrideService:          envConfigOverrideService,
		chartRefRepository:                chartRefRepository,
		chartTemplateService:              chartTemplateService,
		gitFactory:                        gitFactory,
		clusterReadService:                clusterReadService,
		installedAppReadService:           installedAppReadService,
	}
}

func (impl *CdPipelineConfigServiceImpl) GetCdPipelineById(pipelineId int) (cdPipeline *bean.CDPipelineConfigObject, err error) {
	dbPipeline, err := impl.pipelineRepository.FindById(pipelineId)
	if err != nil && errors.IsNotFound(err) {
		impl.logger.Errorw("error in fetching pipeline", "err", err)
		return cdPipeline, err
	}
	environment, err := impl.environmentRepository.FindById(dbPipeline.EnvironmentId)
	if err != nil && errors.IsNotFound(err) {
		impl.logger.Errorw("error in fetching pipeline", "err", err)
		return cdPipeline, err
	}
	if environment == nil || environment.Id == 0 {
		impl.logger.Errorw("environment doesn't exists", "environmentId", dbPipeline.EnvironmentId)
		return cdPipeline, err
	}
	strategies, err := impl.pipelineConfigRepository.GetAllStrategyByPipelineId(dbPipeline.Id)
	if err != nil && errors.IsNotFound(err) {
		impl.logger.Errorw("error in fetching strategies", "err", err)
		return cdPipeline, err
	}
	var strategiesBean []bean.Strategy
	var deploymentTemplate chartRepoRepository.DeploymentStrategy
	for _, item := range strategies {
		strategiesBean = append(strategiesBean, bean.Strategy{
			Config:             []byte(item.Config),
			DeploymentTemplate: item.Strategy,
			Default:            item.Default,
		})

		if item.Default {
			deploymentTemplate = item.Strategy
		}
	}

	preStage := bean.CdStage{}
	if len(dbPipeline.PreStageConfig) > 0 {
		preStage.Name = "Pre-Deployment"
		preStage.Config = dbPipeline.PreStageConfig
		preStage.TriggerType = dbPipeline.PreTriggerType
	}
	postStage := bean.CdStage{}
	if len(dbPipeline.PostStageConfig) > 0 {
		postStage.Name = "Post-Deployment"
		postStage.Config = dbPipeline.PostStageConfig
		postStage.TriggerType = dbPipeline.PostTriggerType
	}

	preStageConfigmapSecrets := bean.PreStageConfigMapSecretNames{}
	postStageConfigmapSecrets := bean.PostStageConfigMapSecretNames{}

	if dbPipeline.PreStageConfigMapSecretNames != "" {
		err = json.Unmarshal([]byte(dbPipeline.PreStageConfigMapSecretNames), &preStageConfigmapSecrets)
		if err != nil {
			impl.logger.Error(err)
			return nil, err
		}
	}

	if dbPipeline.PostStageConfigMapSecretNames != "" {
		err = json.Unmarshal([]byte(dbPipeline.PostStageConfigMapSecretNames), &postStageConfigmapSecrets)
		if err != nil {
			impl.logger.Error(err)
			return nil, err
		}
	}
	appWorkflowMapping, err := impl.appWorkflowRepository.FindWFCDMappingByCDPipelineId(pipelineId)
	if err != nil {
		return nil, err
	}

	var customTag *bean.CustomTagData
	var customTagStage repository5.PipelineStageType
	var customTagEnabled bool
	customTagPreCD, err := impl.customTagService.GetActiveCustomTagByEntityKeyAndValue(pipelineConfigBean.EntityTypePreCD, strconv.Itoa(pipelineId))
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching custom Tag precd")
		return nil, err
	}
	customTagPostCD, err := impl.customTagService.GetActiveCustomTagByEntityKeyAndValue(pipelineConfigBean.EntityTypePostCD, strconv.Itoa(pipelineId))
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching custom Tag precd")
		return nil, err
	}
	if customTagPreCD != nil && customTagPreCD.Id > 0 {
		customTag = &bean.CustomTagData{TagPattern: customTagPreCD.TagPattern,
			CounterX: customTagPreCD.AutoIncreasingNumber,
			Enabled:  customTagPreCD.Enabled,
		}
		customTagStage = repository5.PIPELINE_STAGE_TYPE_PRE_CD
		customTagEnabled = customTagPreCD.Enabled
	} else if customTagPostCD != nil && customTagPostCD.Id > 0 {
		customTag = &bean.CustomTagData{TagPattern: customTagPostCD.TagPattern,
			CounterX: customTagPostCD.AutoIncreasingNumber,
			Enabled:  customTagPostCD.Enabled,
		}
		customTagStage = repository5.PIPELINE_STAGE_TYPE_POST_CD
		customTagEnabled = customTagPostCD.Enabled
	}

	digestConfigurationRequest := imageDigestPolicy.DigestPolicyConfigurationRequest{PipelineId: pipelineId}
	digestPolicyConfigurations, err := impl.imageDigestPolicyService.GetDigestPolicyConfigurations(digestConfigurationRequest)
	if err != nil {
		impl.logger.Errorw("error in checking if isImageDigestPolicyConfiguredForPipeline", "err", err, "pipelineId", pipelineId)
		return nil, err
	}

	envDeploymentConfig, err := impl.deploymentConfigService.GetConfigForDevtronApps(dbPipeline.AppId, dbPipeline.EnvironmentId)
	if err != nil {
		impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", dbPipeline.AppId, "envId", dbPipeline.EnvironmentId, "err", err)
		return nil, err
	}

	cdPipeline = &bean.CDPipelineConfigObject{
		Id:                            dbPipeline.Id,
		Name:                          dbPipeline.Name,
		EnvironmentId:                 dbPipeline.EnvironmentId,
		EnvironmentName:               environment.Name,
		CiPipelineId:                  dbPipeline.CiPipelineId,
		DeploymentTemplate:            deploymentTemplate,
		TriggerType:                   dbPipeline.TriggerType,
		Strategies:                    strategiesBean,
		PreStage:                      preStage,
		PostStage:                     postStage,
		PreStageConfigMapSecretNames:  preStageConfigmapSecrets,
		PostStageConfigMapSecretNames: postStageConfigmapSecrets,
		RunPreStageInEnv:              dbPipeline.RunPreStageInEnv,
		RunPostStageInEnv:             dbPipeline.RunPostStageInEnv,
		CdArgoSetup:                   environment.Cluster.CdArgoSetup,
		ParentPipelineId:              appWorkflowMapping.ParentId,
		ParentPipelineType:            appWorkflowMapping.ParentType,
		DeploymentAppType:             envDeploymentConfig.DeploymentAppType,
		DeploymentAppCreated:          dbPipeline.DeploymentAppCreated,
		IsVirtualEnvironment:          dbPipeline.Environment.IsVirtualEnvironment,
		CustomTagObject:               customTag,
		CustomTagStage:                &customTagStage,
		EnableCustomTag:               customTagEnabled,
		AppId:                         dbPipeline.AppId,
		IsDigestEnforcedForPipeline:   digestPolicyConfigurations.DigestConfiguredForPipeline,
	}
	var preDeployStage *pipelineConfigBean.PipelineStageDto
	var postDeployStage *pipelineConfigBean.PipelineStageDto
	preDeployStage, postDeployStage, err = impl.pipelineStageService.GetCdPipelineStageDataDeepCopy(dbPipeline)
	if err != nil {
		impl.logger.Errorw("error in getting pre/post-CD stage data", "err", err, "cdPipelineId", dbPipeline.Id)
		return nil, err
	}
	cdPipeline.PreDeployStage = preDeployStage
	cdPipeline.PostDeployStage = postDeployStage

	return cdPipeline, err
}

func (impl *CdPipelineConfigServiceImpl) CreateCdPipelines(pipelineCreateRequest *bean.CdPipelines, ctx context.Context) (*bean.CdPipelines, error) {

	//Validation for checking deployment App type
	gitOpsConfigurationStatus, err := impl.gitOpsConfigReadService.IsGitOpsConfigured()
	if err != nil {
		impl.logger.Errorw("error in checking if gitOps is configured or not", "err", err)
		return nil, err
	}
	envIds := make([]*int, 0)
	for _, pipeline := range pipelineCreateRequest.Pipelines {
		// skip creation of pipeline if envId is not set
		if pipeline.EnvironmentId <= 0 || pipeline.IsExternalArgoAppLinkRequest() {
			continue
		}
		//making environment array for fetching the clusterIds
		envIds = append(envIds, &pipeline.EnvironmentId)
		overrideDeploymentType, err := impl.deploymentTypeOverrideService.ValidateAndOverrideDeploymentAppType(pipeline.DeploymentAppType, gitOpsConfigurationStatus.IsGitOpsConfigured, pipeline.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("validation error in creating pipeline", "name", pipeline.Name, "err", err)
			return nil, err
		}
		pipeline.DeploymentAppType = overrideDeploymentType
	}

	if impl.deploymentConfig.ShouldCheckNamespaceOnClone {
		err = impl.checkIfNsExistsForEnvIds(envIds)
		if err != nil {
			impl.logger.Errorw("error in checking existence of namespace for env's", "envIds", envIds, "err", err)
			return nil, err
		}
	}

	isGitOpsRequiredForCD := impl.IsGitOpsRequiredForCD(pipelineCreateRequest)
	app, err := impl.appRepo.FindById(pipelineCreateRequest.AppId)

	if err != nil {
		impl.logger.Errorw("app not found", "err", err, "appId", pipelineCreateRequest.AppId)
		return nil, err
	}

	_, err = impl.validateCDPipelineRequest(pipelineCreateRequest)
	if err != nil {
		impl.logger.Errorw("error in validating cd pipeline create request", "pipelineCreateRequest", pipelineCreateRequest, "err", err)
		return nil, err
	}

	for _, pipeline := range pipelineCreateRequest.Pipelines {
		linkCDValidationResponse := impl.ValidateLinkExternalArgoCDRequest(pipelineConfigBean.ArgoCDAppLinkValidationRequest{
			AppId:         pipeline.AppId,
			ClusterId:     pipeline.ApplicationObjectClusterId,
			Namespace:     pipeline.ApplicationObjectNamespace,
			ArgoCDAppName: pipeline.DeploymentAppName,
		})

		if !linkCDValidationResponse.IsLinkable {
			return nil,
				util.NewApiError(http.StatusPreconditionFailed,
					linkCDValidationResponse.ErrorDetail.ValidationFailedMessage,
					string(linkCDValidationResponse.ErrorDetail.ValidationFailedReason))
		}
	}

	AppDeploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(app.Id, 0)
	if err != nil {
		impl.logger.Errorw("error in fetching deployment config by appId", "appId", app.Id, "err", err)
		return nil, err
	}

	// TODO: creating git repo for all apps irrespective of acd or helm
	if gitOpsConfigurationStatus.IsGitOpsConfigured &&
		isGitOpsRequiredForCD &&
		!pipelineCreateRequest.IsCloneAppReq &&
		!pipelineCreateRequest.Pipelines[0].IsExternalArgoAppLinkRequest() { //TODO: ayush revisit

		if gitOps.IsGitOpsRepoNotConfigured(AppDeploymentConfig.GetRepoURL()) {
			if gitOpsConfigurationStatus.AllowCustomRepository || AppDeploymentConfig.ConfigType == bean4.CUSTOM.String() {
				apiErr := &util.ApiError{
					HttpStatusCode:  http.StatusConflict,
					UserMessage:     cdWorkflow.GITOPS_REPO_NOT_CONFIGURED,
					InternalMessage: cdWorkflow.GITOPS_REPO_NOT_CONFIGURED,
				}
				return nil, apiErr
			}
			_, chartGitAttr, err := impl.appService.CreateGitOpsRepo(app, pipelineCreateRequest.UserId)
			if err != nil {
				impl.logger.Errorw("error in creating git repo", "err", err)
				return nil, fmt.Errorf("Create GitOps repository error: %s", err.Error())
			}
			err = impl.RegisterInACD(ctx, chartGitAttr, pipelineCreateRequest.UserId)
			if err != nil {
				impl.logger.Errorw("error in registering app in acd", "err", err)
				return nil, err
			}
			// below function will update gitRepoUrl for charts if user has not already provided gitOps repoURL
			AppDeploymentConfig, err = impl.chartService.ConfigureGitOpsRepoUrlForApp(pipelineCreateRequest.AppId, chartGitAttr.RepoUrl, chartGitAttr.ChartLocation, false, pipelineCreateRequest.UserId)
			if err != nil {
				impl.logger.Errorw("error in updating git repo url in charts", "err", err)
				return nil, err
			}
		}
	}

	for _, pipeline := range pipelineCreateRequest.Pipelines {
		// skip creation of DeploymentConfig if envId is not set
		var envDeploymentConfig *bean4.DeploymentConfig
		if pipeline.EnvironmentId > 0 {
			env, err := impl.environmentRepository.FindById(pipeline.EnvironmentId)
			if err != nil {
				impl.logger.Errorw("error in fetching environment", "environmentId", pipeline.EnvironmentId, "err", err)
				return nil, err
			}
			envDeploymentConfig = &bean4.DeploymentConfig{
				AppId:             app.Id,
				EnvironmentId:     pipeline.EnvironmentId,
				DeploymentAppType: pipeline.DeploymentAppType,
				RepoURL:           AppDeploymentConfig.RepoURL,
				ReleaseMode:       pipeline.ReleaseMode,
				Active:            true,
			}
			var releaseConfig *bean4.ReleaseConfiguration
			if pipeline.IsExternalArgoAppLinkRequest() {
				releaseConfig, err = impl.parseReleaseConfigForExternalAcdApp(pipeline.ApplicationObjectClusterId, pipeline.ApplicationObjectNamespace, pipeline.DeploymentAppName)
				if err != nil {
					impl.logger.Errorw("error in parsing deployment config for external acd app", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
					return nil, err
				}
				envDeploymentConfig.RepoURL = releaseConfig.ArgoCDSpec.Spec.Source.RepoURL //for backward compatibility
			} else if pipeline.DeploymentAppType == util.PIPELINE_DEPLOYMENT_TYPE_ACD {
				releaseConfig, err = impl.parseReleaseConfigForACDApp(app, AppDeploymentConfig, env)
				if err != nil {
					impl.logger.Errorw("error in parsing deployment config for acd app", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
					return nil, err
				}
			}
			envDeploymentConfig.ReleaseConfiguration = releaseConfig
			envDeploymentConfig, err = impl.deploymentConfigService.CreateOrUpdateConfig(nil, envDeploymentConfig, pipelineCreateRequest.UserId)
			if err != nil {
				impl.logger.Errorw("error in fetching creating env config", "appId", app.Id, "envId", pipeline.EnvironmentId, "err", err)
				return nil, err
			}
		}

		id, err := impl.createCdPipeline(ctx, app, pipeline, envDeploymentConfig, pipelineCreateRequest.UserId)
		if err != nil {
			impl.logger.Errorw("error in creating pipeline", "name", pipeline.Name, "err", err)
			return nil, err
		}
		pipeline.Id = id
		//go for stage creation if pipeline is created above
		if pipeline.Id > 0 {
			//creating pipeline_stage entry here after tx commit due to FK issue
			if pipeline.PreDeployStage != nil && len(pipeline.PreDeployStage.Steps) > 0 {
				err = impl.pipelineStageService.CreatePipelineStage(pipeline.PreDeployStage, repository5.PIPELINE_STAGE_TYPE_PRE_CD, id, pipelineCreateRequest.UserId)
				if err != nil {
					impl.logger.Errorw("error in creating pre-cd stage", "err", err, "preCdStage", pipeline.PreDeployStage, "pipelineId", id)
					return nil, err
				}
			}
			if pipeline.PostDeployStage != nil && len(pipeline.PostDeployStage.Steps) > 0 {
				err = impl.pipelineStageService.CreatePipelineStage(pipeline.PostDeployStage, repository5.PIPELINE_STAGE_TYPE_POST_CD, id, pipelineCreateRequest.UserId)
				if err != nil {
					impl.logger.Errorw("error in creating post-cd stage", "err", err, "postCdStage", pipeline.PostDeployStage, "pipelineId", id)
					return nil, err
				}
			}
		}
	}
	return pipelineCreateRequest, nil
}

func (impl *CdPipelineConfigServiceImpl) parseReleaseConfigForACDApp(app *app2.App, AppDeploymentConfig *bean4.DeploymentConfig, env *repository6.Environment) (*bean4.ReleaseConfiguration, error) {

	envOverride, err := impl.envConfigOverrideService.FindLatestChartForAppByAppIdAndEnvId(app.Id, env.Id)
	if err != nil && !errors2.IsNotFound(err) {
		impl.logger.Errorw("error in fetch")
		return nil, err
	}
	var latestChart *chartRepoRepository.Chart
	if envOverride == nil || envOverride.Id == 0 || (envOverride.Id > 0 && !envOverride.IsOverride) {
		latestChart, err = impl.chartRepository.FindLatestChartForAppByAppId(app.Id)
		if err != nil {
			return nil, err
		}
	} else {
		//if chart is overrides in env, it means it may have different version than app level.
		latestChart = envOverride.Chart
	}
	chartRefId := latestChart.ChartRefId

	chartRef, err := impl.chartRefRepository.FindById(chartRefId)
	if err != nil {
		impl.logger.Errorw("error in fetching chart", "chartRefId", chartRefId, "err", err)
		return nil, err
	}
	chartLocation := filepath.Join(chartRef.Location, latestChart.ChartVersion)

	return &bean4.ReleaseConfiguration{
		Version: bean4.Version,
		ArgoCDSpec: bean4.ArgoCDSpec{
			Metadata: bean4.ApplicationMetadata{
				ClusterId: bean3.DefaultClusterId,
				Namespace: argocdServer.DevtronInstalationNs,
			},
			Spec: bean4.ApplicationSpec{
				Destination: &bean4.Destination{
					Namespace: env.Namespace,
					Server:    env.Cluster.ServerUrl,
				},
				Source: &bean4.ApplicationSource{
					RepoURL:        AppDeploymentConfig.GetRepoURL(),
					Path:           chartLocation,
					TargetRevision: "master",
					Helm: &bean4.ApplicationSourceHelm{
						ValueFiles: []string{fmt.Sprintf("_%d-values.yaml", env.Id)},
					},
				},
			},
		},
	}, nil
}

func (impl *CdPipelineConfigServiceImpl) ValidateLinkExternalArgoCDRequest(request pipelineConfigBean.ArgoCDAppLinkValidationRequest) pipelineConfigBean.ArgoCdAppLinkValidationResponse {

	appId := request.AppId
	applicationObjectClusterId := request.ClusterId
	applicationObjectNamespace := request.Namespace
	acdAppName := request.ArgoCDAppName

	response := pipelineConfigBean.ArgoCdAppLinkValidationResponse{
		IsLinkable:          false,
		ApplicationMetadata: pipelineConfigBean.NewEmptyApplicationMetadata(),
	}

	application, err := impl.argoClientWrapperService.GetArgoAppByNameWithK8sClient(context.Background(), applicationObjectClusterId, applicationObjectNamespace, acdAppName)
	if err != nil {
		impl.logger.Errorw("error in fetching application", "deploymentAppName", acdAppName, "err", err)
		return response.SetUnknownErrorDetail(err)
	}

	applicationJSON, err := json.Marshal(application)
	if err != nil {
		impl.logger.Errorw("error in marshalling application", "applicationName", acdAppName, "err", err)
		return response.SetUnknownErrorDetail(err)
	}

	var argoApplicationSpec v1alpha1.Application
	if err := json.Unmarshal(applicationJSON, &argoApplicationSpec); err != nil {
		impl.logger.Errorw("error in unmarshalling application", "applicationName", acdAppName, "err", err)
		return response.SetUnknownErrorDetail(err)
	}

	if argoApplicationSpec.Spec.HasMultipleSources() {
		return response.SetErrorDetail(pipelineConfigBean.UnsupportedApplicationSpec, "application with multiple sources not supported")
	}
	if argoApplicationSpec.Spec.Source != nil && argoApplicationSpec.Spec.Source.Helm != nil && len(argoApplicationSpec.Spec.Source.Helm.ValueFiles) > 0 {
		return response.SetErrorDetail(pipelineConfigBean.UnsupportedApplicationSpec, "application with multiple helm value files are not supported")
	}

	response.ApplicationMetadata.Source.RepoURL = argoApplicationSpec.Spec.Source.RepoURL
	response.ApplicationMetadata.Source.ChartPath = argoApplicationSpec.Spec.Source.Chart
	response.ApplicationMetadata.Status = string(argoApplicationSpec.Status.Health.Status)

	pipelines, err := impl.pipelineRepository.GetArgoPipelineByArgoAppName(acdAppName)
	if err != nil && !errors3.Is(err, pg.ErrNoRows) {
		return response.SetUnknownErrorDetail(err)
	}

	for _, p := range pipelines {
		dc, err := impl.deploymentConfigService.GetConfigForDevtronApps(p.AppId, p.EnvironmentId)
		if err != nil {
			return response.SetUnknownErrorDetail(err)
		}
		if dc.ReleaseConfiguration.ArgoCDSpec.GetApplicationObjectClusterId() == applicationObjectClusterId &&
			dc.ReleaseConfiguration.ArgoCDSpec.GetApplicationObjectNamespace() == applicationObjectNamespace {
			return response.SetErrorDetail(pipelineConfigBean.ApplicationAlreadyPresent, pipelineConfigBean.PipelineAlreadyPresentMsg)
		}
	}

	installedApp, err := impl.installedAppReadService.GetInstalledAppByGitOpsAppName(acdAppName)
	if err != nil && !errors3.Is(err, pg.ErrNoRows) {
		return response.SetUnknownErrorDetail(err)
	}
	if installedApp != nil {
		// installed app found
		if bean3.DefaultClusterId == applicationObjectClusterId && argocdServer.DevtronInstalationNs == applicationObjectNamespace {
			return response.SetErrorDetail(pipelineConfigBean.ApplicationAlreadyPresent, pipelineConfigBean.HelmAppAlreadyPresentMsg)
		}
	}

	targetClusterURL := argoApplicationSpec.Spec.Destination.Server
	targetClusterNamespace := argoApplicationSpec.Spec.Destination.Namespace

	response.ApplicationMetadata.Destination.ClusterServerURL = targetClusterURL
	response.ApplicationMetadata.Destination.Namespace = targetClusterNamespace

	targetCluster, err := impl.clusterReadService.FindByClusterURL(targetClusterURL)
	if err != nil {
		impl.logger.Errorw("error in getting targetCluster by url", "clusterURL", targetClusterURL, "err", err)
		if errors3.Is(err, pg.ErrNoRows) {
			return response.SetErrorDetail(pipelineConfigBean.ClusterNotFound, "targetCluster not added in global configuration")
		}
		return response.SetUnknownErrorDetail(err)
	}
	response.ApplicationMetadata.Destination.ClusterName = targetCluster.ClusterName

	targetEnv, err := impl.environmentRepository.FindOneByNamespaceAndClusterId(targetClusterNamespace, targetCluster.Id)
	if err != nil {
		if errors3.Is(err, pg.ErrNoRows) {
			return response.SetErrorDetail(pipelineConfigBean.EnvironmentNotFound, "environment not added in global configuration")
		}
		return response.SetUnknownErrorDetail(err)
	}
	response.ApplicationMetadata.Destination.EnvironmentName = targetEnv.Name
	response.ApplicationMetadata.Destination.EnvironmentId = targetEnv.Id

	repoURL := argoApplicationSpec.Spec.Source.RepoURL
	_, err = impl.gitOpsConfigReadService.GetGitOpsProviderByRepoURL(repoURL)
	if err != nil {
		if strings.Contains(err.Error(), "no gitops config found in DB for given repoURL") {
			return response.SetErrorDetail(pipelineConfigBean.GitOpsNotFound, err.Error())
		}
		return response.SetUnknownErrorDetail(err)
	}

	repoName := impl.gitOpsConfigReadService.GetGitOpsRepoNameFromUrl(repoURL)
	repoURLFromConfiguredGitOpsClient, err := impl.gitOperationService.GetRepoUrlByRepoName(repoName)
	if err != nil {
		impl.logger.Errorw("error in fetching repo url by repoName", "repoName", repoName, "err", err)
		return response.SetErrorDetail(pipelineConfigBean.GitOpsNotFound, fmt.Sprintf("please configure valid GitOps credential for repo %s", repoURL))
	}

	if repoURLFromConfiguredGitOpsClient != repoURL {
		return response.SetErrorDetail(pipelineConfigBean.GitOpsNotFound, fmt.Sprintf("please configure valid GitOps credential for repo %s", repoURL))
	}

	chartPath := argoApplicationSpec.Spec.Source.Path
	helmChart, err := impl.extractHelmChartForExternalArgoApp(repoURL, chartPath)
	if err != nil {
		impl.logger.Errorw("error in extracting helm chart from application spec", "acdAppName", acdAppName, "err", err)
		return response.SetUnknownErrorDetail(err)
	}

	applicationChartName, applicationChartVersion := helmChart.Metadata.Name, helmChart.Metadata.Version
	latestChart, err := impl.chartService.FindLatestChartForAppByAppId(appId)
	if err != nil {
		impl.logger.Errorw("error in finding latest chart by appId", "appId", appId, "err", err)
		return response.SetUnknownErrorDetail(err)
	}

	chartRef, err := impl.chartRefRepository.FindById(latestChart.ChartRefId)
	if err != nil {
		impl.logger.Errorw("error in finding chart ref by chartRefId", "chartRefId", latestChart.ChartRefId, "err", err)
		return response.SetUnknownErrorDetail(err)
	}
	response.ApplicationMetadata.Source.ChartMetadata = pipelineConfigBean.ChartMetadata{
		ChartVersion:      applicationChartVersion,
		SavedChartName:    chartRef.Name,
		ValuesFilename:    argoApplicationSpec.Spec.Source.Helm.ValueFiles[0],
		RequiredChartName: applicationChartName,
	}

	if chartRef.Name != applicationChartName {
		return response.SetErrorDetail(pipelineConfigBean.ChartTypeMismatch, fmt.Sprintf(pipelineConfigBean.ChartTypeMismatchErrorMsg, applicationChartName, applicationChartVersion))
	}

	_, err = impl.chartRefRepository.FindByVersionAndName(applicationChartName, applicationChartVersion)
	if err != nil && !errors3.Is(err, pg.ErrNoRows) {
		impl.logger.Errorw("error in finding chart ref by chart name and version", "chartName", applicationChartName, "chartVersion", applicationChartVersion, "err", err)
		return response.SetUnknownErrorDetail(err)
	}
	if errors3.Is(err, pg.ErrNoRows) {
		return response.SetErrorDetail(pipelineConfigBean.ChartVersionNotFound, fmt.Sprintf(pipelineConfigBean.ChartVersionNotFoundErrorMsg, applicationChartVersion, applicationChartName))
	}

	return response
}

func (impl *CdPipelineConfigServiceImpl) parseReleaseConfigForExternalAcdApp(clusterId int, namespace, acdAppName string) (*bean4.ReleaseConfiguration, error) {
	application, err := impl.argoClientWrapperService.GetArgoAppByNameWithK8sClient(context.Background(), clusterId, namespace, acdAppName)
	if err != nil {
		impl.logger.Errorw("error in fetching application", "deploymentAppName", acdAppName, "err", err)
		return nil, err
	}
	applicationJSON, err := json.Marshal(application)
	if err != nil {
		impl.logger.Errorw("error in marshalling application", "applicationName", acdAppName, "err", err)
		return nil, err
	}
	var argoApplicationSpec bean4.ArgoCDSpec
	err = json.Unmarshal(applicationJSON, &argoApplicationSpec)
	if err != nil {
		impl.logger.Errorw("error in unmarshalling application", "applicationName", acdAppName, "err", err)
		return nil, err
	}
	argoApplicationSpec.SetApplicationObjectClusterId(clusterId)

	return &bean4.ReleaseConfiguration{
		Version:    bean4.Version,
		ArgoCDSpec: argoApplicationSpec,
	}, nil
}

func (impl *CdPipelineConfigServiceImpl) CDPipelineCustomTagDBOperations(pipeline *bean.CDPipelineConfigObject) error {

	if pipeline.EnableCustomTag && (pipeline.CustomTagObject != nil && len(pipeline.CustomTagObject.TagPattern) == 0) {
		return fmt.Errorf("please provide custom tag data if tag is enabled")
	}
	if pipeline.CustomTagObject != nil && pipeline.CustomTagObject.CounterX < 0 {
		return fmt.Errorf("value of {x} cannot be negative")
	}
	if !pipeline.EnableCustomTag {
		// disable custom tag if exist
		err := impl.DisableCustomTag(pipeline)
		if err != nil {
			return err
		}
		return nil
	} else {
		err := impl.SaveOrUpdateCustomTagForCDPipeline(pipeline)
		if err != nil {
			impl.logger.Errorw("error in creating custom tag for pipeline stage", "err", err)
			return err
		}
	}
	if *pipeline.CustomTagStage == repository5.PIPELINE_STAGE_TYPE_POST_CD {
		// delete entry for post stage if any
		preCDStageName := repository5.PIPELINE_STAGE_TYPE_PRE_CD
		err := impl.DeleteCustomTagByPipelineStageType(&preCDStageName, pipeline.Id)
		if err != nil {
			return err
		}
	} else if *pipeline.CustomTagStage == repository5.PIPELINE_STAGE_TYPE_PRE_CD {
		postCdStageName := repository5.PIPELINE_STAGE_TYPE_POST_CD
		err := impl.DeleteCustomTagByPipelineStageType(&postCdStageName, pipeline.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) DeleteCustomTag(pipeline *bean.CDPipelineConfigObject) error {
	preStage := repository5.PIPELINE_STAGE_TYPE_PRE_CD
	postStage := repository5.PIPELINE_STAGE_TYPE_POST_CD
	err := impl.DeleteCustomTagByPipelineStageType(&preStage, pipeline.Id)
	if err != nil {
		return err
	}
	err = impl.DeleteCustomTagByPipelineStageType(&postStage, pipeline.Id)
	if err != nil {
		return err
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) DisableCustomTag(pipeline *bean.CDPipelineConfigObject) error {
	preStage := repository5.PIPELINE_STAGE_TYPE_PRE_CD
	postStage := repository5.PIPELINE_STAGE_TYPE_POST_CD
	err := impl.DisableCustomTagByPipelineStageType(&preStage, pipeline.Id)
	if err != nil {
		return err
	}
	err = impl.DisableCustomTagByPipelineStageType(&postStage, pipeline.Id)
	if err != nil {
		return err
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) DeleteCustomTagByPipelineStageType(pipelineStageType *repository5.PipelineStageType, pipelineId int) error {
	err := impl.customTagService.DeleteCustomTagIfExists(
		bean2.CustomTag{EntityKey: getEntityTypeByPipelineStageType(*pipelineStageType),
			EntityValue: fmt.Sprintf("%d", pipelineId),
		})
	if err != nil {
		impl.logger.Errorw("error in deleting custom tag for pre stage", "err", err, "pipeline-id", pipelineId)
		return err
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) DisableCustomTagByPipelineStageType(pipelineStageType *repository5.PipelineStageType, pipelineId int) error {
	err := impl.customTagService.DisableCustomTagIfExist(
		bean2.CustomTag{EntityKey: getEntityTypeByPipelineStageType(*pipelineStageType),
			EntityValue: fmt.Sprintf("%d", pipelineId),
		})
	if err != nil {
		impl.logger.Errorw("error in deleting custom tag for pre stage", "err", err, "pipeline-id", pipelineId)
		return err
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) SaveOrUpdateCustomTagForCDPipeline(pipeline *bean.CDPipelineConfigObject) error {
	customTag, err := impl.ParseCustomTagPatchRequest(pipeline)
	if err != nil {
		impl.logger.Errorw("err", err)
		return err
	}
	err = impl.customTagService.CreateOrUpdateCustomTag(customTag)
	if err != nil {
		impl.logger.Errorw("error in creating custom tag", "err", err)
		return err
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) ParseCustomTagPatchRequest(pipelineRequest *bean.CDPipelineConfigObject) (*bean2.CustomTag, error) {
	entityType := getEntityTypeByPipelineStageType(*pipelineRequest.CustomTagStage)
	if entityType == 0 {
		return nil, fmt.Errorf("invalid stage for cd pipeline custom tag; pipelineStageType: %s ", string(*pipelineRequest.CustomTagStage))
	}
	customTag := &bean2.CustomTag{
		EntityKey:            entityType,
		EntityValue:          fmt.Sprintf("%d", pipelineRequest.Id),
		TagPattern:           pipelineRequest.CustomTagObject.TagPattern,
		AutoIncreasingNumber: pipelineRequest.CustomTagObject.CounterX,
		Metadata:             "",
		Enabled:              pipelineRequest.EnableCustomTag,
	}
	return customTag, nil
}

func getEntityTypeByPipelineStageType(pipelineStageType repository5.PipelineStageType) (customTagEntityType int) {
	switch pipelineStageType {
	case repository5.PIPELINE_STAGE_TYPE_PRE_CD:
		customTagEntityType = pipelineConfigBean.EntityTypePreCD
	case repository5.PIPELINE_STAGE_TYPE_POST_CD:
		customTagEntityType = pipelineConfigBean.EntityTypePostCD
	default:
		customTagEntityType = pipelineConfigBean.EntityNull
	}
	return customTagEntityType
}

func (impl *CdPipelineConfigServiceImpl) PatchCdPipelines(cdPipelines *bean.CDPatchRequest, ctx context.Context) (*bean.CdPipelines, error) {
	pipelineRequest := &bean.CdPipelines{
		UserId:    cdPipelines.UserId,
		AppId:     cdPipelines.AppId,
		Pipelines: []*bean.CDPipelineConfigObject{cdPipelines.Pipeline},
	}
	deleteAction := bean.CASCADE_DELETE
	if cdPipelines.ForceDelete {
		deleteAction = bean.FORCE_DELETE
	} else if cdPipelines.NonCascadeDelete {
		deleteAction = bean.NON_CASCADE_DELETE
	}
	switch cdPipelines.Action {
	case bean.CD_CREATE:
		return impl.CreateCdPipelines(pipelineRequest, ctx)
	case bean.CD_UPDATE:
		err := impl.updateCdPipeline(ctx, cdPipelines.Pipeline, cdPipelines.UserId)
		return pipelineRequest, err
	case bean.CD_DELETE:
		pipeline, err := impl.pipelineRepository.FindById(cdPipelines.Pipeline.Id)
		if err != nil {
			impl.logger.Errorw("error in getting cd pipeline by id", "err", err, "id", cdPipelines.Pipeline.Id)
			return pipelineRequest, err
		}
		deleteResponse, err := impl.DeleteCdPipeline(pipeline, ctx, deleteAction, false, cdPipelines.UserId)
		pipelineRequest.AppDeleteResponse = deleteResponse
		return pipelineRequest, err
	case bean.CD_DELETE_PARTIAL:
		pipeline, err := impl.pipelineRepository.FindById(cdPipelines.Pipeline.Id)
		if err != nil {
			impl.logger.Errorw("error in getting cd pipeline by id", "err", err, "id", cdPipelines.Pipeline.Id)
			return pipelineRequest, err
		}
		deleteResponse, err := impl.DeleteCdPipelinePartial(pipeline, ctx, deleteAction, cdPipelines.UserId)
		pipelineRequest.AppDeleteResponse = deleteResponse
		return pipelineRequest, err
	default:
		return nil, &util.ApiError{Code: "404", HttpStatusCode: 404, UserMessage: "operation not supported"}
	}
}

func (impl *CdPipelineConfigServiceImpl) DeleteCdPipeline(pipeline *pipelineConfig.Pipeline, ctx context.Context, deleteAction int, deleteFromAcd bool, userId int32) (*bean.AppDeleteResponseDTO, error) {
	cascadeDelete := true
	forceDelete := false
	deleteResponse := &bean.AppDeleteResponseDTO{
		DeleteInitiated:  false,
		ClusterReachable: true,
	}
	if deleteAction == bean.FORCE_DELETE {
		forceDelete = true
		cascadeDelete = false
	} else if deleteAction == bean.NON_CASCADE_DELETE {
		cascadeDelete = false
	}
	// updating cluster reachable flag
	clusterBean, err := impl.clusterRepository.FindById(pipeline.Environment.ClusterId)
	if err != nil {
		impl.logger.Errorw("error in getting cluster details", "err", err, "clusterId", pipeline.Environment.ClusterId)
	}
	deleteResponse.ClusterName = clusterBean.ClusterName
	if len(clusterBean.ErrorInConnecting) > 0 {
		deleteResponse.ClusterReachable = false
	}

	//getting children CD pipeline details
	childNodes, err := impl.appWorkflowRepository.FindWFCDMappingByParentCDPipelineId(pipeline.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting children cd details", "err", err)
		return deleteResponse, err
	}

	dbConnection := impl.pipelineRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return deleteResponse, err
	}
	// Rollback tx on error.
	defer tx.Rollback()
	if err = impl.ciCdPipelineOrchestrator.DeleteCdPipeline(pipeline.Id, userId, tx); err != nil {
		impl.logger.Errorw("err in deleting pipeline from db", "id", pipeline, "err", err)
		return deleteResponse, err
	}
	// delete entry in app_status table
	err = impl.appStatusRepository.Delete(tx, pipeline.AppId, pipeline.EnvironmentId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("err in deleting app_status from db", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
		return deleteResponse, err
	}
	//delete app workflow mapping
	appWorkflowMapping, err := impl.appWorkflowRepository.FindWFCDMappingByCDPipelineId(pipeline.Id)
	if err != nil {
		impl.logger.Errorw("error in deleting workflow mapping", "err", err)
		return deleteResponse, err
	}
	if appWorkflowMapping.ParentType == appWorkflow.WEBHOOK && len(childNodes) == 0 {
		childNodes, err := impl.appWorkflowRepository.FindWFCDMappingByExternalCiId(appWorkflowMapping.ParentId)
		if err != nil && !util.IsErrNoRows(err) {
			impl.logger.Errorw("error in fetching external ci", "err", err)
			return deleteResponse, err
		}
		noOtherChildNodes := true
		for _, childNode := range childNodes {
			if appWorkflowMapping.Id != childNode.Id {
				noOtherChildNodes = false
			}
		}
		if noOtherChildNodes {
			externalCiPipeline, err := impl.ciPipelineRepository.FindExternalCiById(appWorkflowMapping.ParentId)
			if err != nil {
				impl.logger.Errorw("error in deleting workflow mapping", "err", err)
				return deleteResponse, err
			}
			externalCiPipeline.Active = false
			externalCiPipeline.UpdatedOn = time.Now()
			externalCiPipeline.UpdatedBy = userId
			_, err = impl.ciPipelineRepository.UpdateExternalCi(externalCiPipeline, tx)
			if err != nil {
				impl.logger.Errorw("error in deleting workflow mapping", "err", err)
				return deleteResponse, err
			}

			appWorkflow, err := impl.appWorkflowRepository.FindById(appWorkflowMapping.AppWorkflowId)
			if err != nil {
				impl.logger.Errorw("error in deleting workflow mapping", "err", err)
				return deleteResponse, err
			}
			err = impl.appWorkflowRepository.DeleteAppWorkflow(appWorkflow, tx)
			if err != nil {
				impl.logger.Errorw("error in deleting workflow mapping", "err", err)
				return deleteResponse, err
			}
		}
	}
	appWorkflowMapping.UpdatedBy = userId
	appWorkflowMapping.UpdatedOn = time.Now()
	err = impl.appWorkflowRepository.DeleteAppWorkflowMapping(appWorkflowMapping, tx)
	if err != nil {
		impl.logger.Errorw("error in deleting workflow mapping", "err", err)
		return deleteResponse, err
	}

	if len(childNodes) > 0 {
		err = impl.appWorkflowRepository.UpdateParentComponentDetails(tx, appWorkflowMapping.ComponentId, appWorkflowMapping.Type, appWorkflowMapping.ParentId, appWorkflowMapping.ParentType, nil)
		if err != nil {
			impl.logger.Errorw("error updating wfm for children pipelines of pipeline", "err", err, "id", appWorkflowMapping.Id)
			return deleteResponse, err
		}
	}

	if pipeline.PreStageConfig != "" {
		err = impl.prePostCdScriptHistoryService.CreatePrePostCdScriptHistory(pipeline, tx, repository4.PRE_CD_TYPE, false, 0, time.Time{})
		if err != nil {
			impl.logger.Errorw("error in creating pre cd script entry", "err", err, "pipeline", pipeline)
			return deleteResponse, err
		}
	}
	if pipeline.PostStageConfig != "" {
		err = impl.prePostCdScriptHistoryService.CreatePrePostCdScriptHistory(pipeline, tx, repository4.POST_CD_TYPE, false, 0, time.Time{})
		if err != nil {
			impl.logger.Errorw("error in creating post cd script entry", "err", err, "pipeline", pipeline)
			return deleteResponse, err
		}
	}
	cdPipelinePluginDeleteReq, err := impl.GetCdPipelineById(pipeline.Id)
	if err != nil {
		impl.logger.Errorw("error in getting cdPipeline by id", "err", err, "id", pipeline.Id)
		return deleteResponse, err
	}
	if cdPipelinePluginDeleteReq.PreDeployStage != nil && cdPipelinePluginDeleteReq.PreDeployStage.Id > 0 {
		//deleting pre-stage
		err = impl.pipelineStageService.DeletePipelineStage(cdPipelinePluginDeleteReq.PreDeployStage, userId, tx)
		if err != nil {
			impl.logger.Errorw("error in deleting pre-CD stage", "err", err, "preDeployStage", cdPipelinePluginDeleteReq.PreDeployStage)
			return deleteResponse, err
		}
	}
	if cdPipelinePluginDeleteReq.PostDeployStage != nil && cdPipelinePluginDeleteReq.PostDeployStage.Id > 0 {
		//deleting post-stage
		err = impl.pipelineStageService.DeletePipelineStage(cdPipelinePluginDeleteReq.PostDeployStage, userId, tx)
		if err != nil {
			impl.logger.Errorw("error in deleting post-CD stage", "err", err, "postDeployStage", cdPipelinePluginDeleteReq.PostDeployStage)
			return deleteResponse, err
		}
	}
	if cdPipelinePluginDeleteReq.PreDeployStage != nil {
		tag := bean2.CustomTag{
			EntityKey:   pipelineConfigBean.EntityTypePreCD,
			EntityValue: strconv.Itoa(pipeline.Id),
		}
		err = impl.customTagService.DeleteCustomTagIfExists(tag)
		if err != nil {
			impl.logger.Errorw("error in deleting custom tag for pre-cd stage", "Err", err, "cd-pipeline-id", pipeline.Id)
		}
	}
	if cdPipelinePluginDeleteReq.PostDeployStage != nil {
		tag := bean2.CustomTag{
			EntityKey:   pipelineConfigBean.EntityTypePostCD,
			EntityValue: strconv.Itoa(pipeline.Id),
		}
		err = impl.customTagService.DeleteCustomTagIfExists(tag)
		if err != nil {
			impl.logger.Errorw("error in deleting custom tag for pre-cd stage", "Err", err, "cd-pipeline-id", pipeline.Id)
		}
	}
	_, err = impl.imageDigestPolicyService.DeletePolicyForPipeline(tx, pipeline.Id, userId)
	if err != nil {
		impl.logger.Errorw("error in deleting imageDigestPolicy for pipeline", "err", err, "pipelineId", pipeline.Id)
		return nil, err
	}
	envDeploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(pipeline.AppId, pipeline.EnvironmentId)
	if err != nil {
		impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
		return nil, err
	}
	envDeploymentConfig.Active = false
	envDeploymentConfig, err = impl.deploymentConfigService.CreateOrUpdateConfig(tx, envDeploymentConfig, userId)
	if err != nil {
		impl.logger.Errorw("error in deleting deployment config for pipeline", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
		return nil, err
	}
	//delete app from argo cd, if created
	if pipeline.DeploymentAppCreated == true {
		deploymentAppName := pipeline.DeploymentAppName
		if util.IsAcdApp(envDeploymentConfig.DeploymentAppType) {
			if !forceDelete && !deleteResponse.ClusterReachable {
				impl.logger.Errorw("cluster connection error", "err", clusterBean.ErrorInConnecting)
				if cascadeDelete {
					return deleteResponse, nil
				}
			}
			impl.logger.Debugw("acd app is already deleted for this pipeline", "pipeline", pipeline)
			if deleteFromAcd {
				//TODO: ayush test
				applicationObjectClusterId := envDeploymentConfig.GetApplicationObjectClusterId()
				applicationNamespace := envDeploymentConfig.GetDestinationNamespace()

				if err := impl.argoClientWrapperService.DeleteArgoAppWithK8sClient(ctx, applicationObjectClusterId, applicationNamespace, deploymentAppName, cascadeDelete); err != nil {
					impl.logger.Errorw("err in deleting pipeline on argocd", "id", pipeline, "err", err)

					if forceDelete {
						impl.logger.Warnw("error while deletion of app in acd, continue to delete in db as this operation is force delete", "error", err)
					} else {
						//statusError, _ := err.(*errors2.StatusError)
						if cascadeDelete && errors.IsNotFound(err) {
							err = &util.ApiError{
								UserMessage:     "Could not delete as application not found in argocd",
								InternalMessage: err.Error(),
							}
						} else {
							err = &util.ApiError{
								UserMessage:     "Could not delete application",
								InternalMessage: err.Error(),
							}
						}
						return deleteResponse, err
					}
				}
				impl.logger.Infow("app deleted from argocd", "id", pipeline.Id, "pipelineName", pipeline.Name, "app", deploymentAppName)
			}
		} else if util.IsHelmApp(envDeploymentConfig.DeploymentAppType) {
			err = impl.DeleteHelmTypePipelineDeploymentApp(ctx, forceDelete, pipeline)
			if err != nil {
				impl.logger.Errorw("error, DeleteHelmTypePipelineDeploymentApp", "err", err, "pipelineId", pipeline.Id)
				return deleteResponse, err

			}
		}
	}
	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("error in committing db transaction", "err", err)
		return deleteResponse, err
	}
	deleteResponse.DeleteInitiated = true
	impl.pipelineConfigEventPublishService.PublishCDPipelineDelete(pipeline.Id, userId)
	return deleteResponse, nil
}

func (impl *CdPipelineConfigServiceImpl) DeleteHelmTypePipelineDeploymentApp(ctx context.Context, forceDelete bool, pipeline *pipelineConfig.Pipeline) error {
	deploymentAppName := pipeline.DeploymentAppName
	appIdentifier := &helmBean.AppIdentifier{
		ClusterId:   pipeline.Environment.ClusterId,
		ReleaseName: deploymentAppName,
		Namespace:   pipeline.Environment.Namespace,
	}
	deleteResourceResponse, err := impl.helmAppService.DeleteApplication(ctx, appIdentifier)
	if forceDelete || errors3.As(err, &models2.NamespaceNotExistError{}) {
		impl.logger.Warnw("error while deletion of helm application, ignore error and delete from db since force delete req", "error", err, "pipelineId", pipeline.Id)
	} else {
		if err != nil {
			impl.logger.Errorw("error in deleting helm application", "error", err, "appIdentifier", appIdentifier)
			apiError := clientErrors.ConvertToApiError(err)
			if apiError != nil {
				err = apiError
			}
			return err
		}
		if deleteResourceResponse == nil || !deleteResourceResponse.GetSuccess() {
			return errors2.New("delete application response unsuccessful")
		}
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) DeleteACDAppCdPipelineWithNonCascade(pipeline *pipelineConfig.Pipeline, ctx context.Context, forceDelete bool, userId int32) error {
	if forceDelete {
		_, err := impl.DeleteCdPipeline(pipeline, ctx, bean.FORCE_DELETE, false, userId)
		return err
	}
	envDeploymentConfig, err := impl.deploymentConfigService.GetConfigForDevtronApps(pipeline.AppId, pipeline.EnvironmentId)
	if err != nil {
		impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
		return err
	}
	applicationObjectClusterId := envDeploymentConfig.GetApplicationObjectClusterId()
	applicationObjectNamespace := envDeploymentConfig.GetDestinationNamespace()
	//delete app from argo cd with non-cascade, if created
	if pipeline.DeploymentAppCreated && util.IsAcdApp(envDeploymentConfig.DeploymentAppType) {
		deploymentAppName := pipeline.DeploymentAppName
		impl.logger.Debugw("acd app is already deleted for this pipeline", "pipeline", pipeline)
		if err = impl.argoClientWrapperService.DeleteArgoAppWithK8sClient(ctx, applicationObjectClusterId, applicationObjectNamespace, deploymentAppName, false); err != nil {
			impl.logger.Errorw("err in deleting pipeline on argocd", "id", pipeline, "err", err)
			//statusError, _ := err.(*errors2.StatusError)
			if errors.IsNotFound(err) {
				err = &util.ApiError{
					UserMessage:     "Could not delete application",
					InternalMessage: err.Error(),
				}
				return err
			}
		}

	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) GetTriggerViewCdPipelinesForApp(appId int) (cdPipelines *bean.CdPipelines, err error) {
	triggerViewCdPipelinesResp, err := impl.ciCdPipelineOrchestrator.GetCdPipelinesForApp(appId)
	if err != nil {
		impl.logger.Errorw("error in fetching triggerViewCdPipelinesResp by appId", "err", err, "appId", appId)
		return triggerViewCdPipelinesResp, err
	}
	var dbPipelineIds []int
	for _, dbPipeline := range triggerViewCdPipelinesResp.Pipelines {
		dbPipelineIds = append(dbPipelineIds, dbPipeline.Id)
	}
	if len(dbPipelineIds) == 0 {
		return triggerViewCdPipelinesResp, nil
	}

	// construct strategiesMapping to get all strategies against pipelineId
	strategiesMapping, err := impl.getStrategiesMapping(dbPipelineIds)
	if err != nil {
		return triggerViewCdPipelinesResp, err
	}
	for _, dbPipeline := range triggerViewCdPipelinesResp.Pipelines {
		var strategies []*chartConfig.PipelineStrategy
		var deploymentTemplate chartRepoRepository.DeploymentStrategy
		if len(strategiesMapping[dbPipeline.Id]) != 0 {
			strategies = strategiesMapping[dbPipeline.Id]
		}
		for _, item := range strategies {
			if item.Default {
				deploymentTemplate = item.Strategy
			}
		}
		dbPipeline.DeploymentTemplate = deploymentTemplate
	}

	return triggerViewCdPipelinesResp, err
}

func (impl *CdPipelineConfigServiceImpl) GetCdPipelinesForApp(appId int) (cdPipelines *bean.CdPipelines, err error) {
	cdPipelines, err = impl.ciCdPipelineOrchestrator.GetCdPipelinesForApp(appId)
	if err != nil {
		impl.logger.Errorw("error in fetching cd Pipelines for appId", "err", err, "appId", appId)
		return nil, err
	}
	var envIds []*int
	var dbPipelineIds []int
	for _, dbPipeline := range cdPipelines.Pipelines {
		envIds = append(envIds, &dbPipeline.EnvironmentId)
		dbPipelineIds = append(dbPipelineIds, dbPipeline.Id)
	}
	if len(envIds) == 0 || len(dbPipelineIds) == 0 {
		return cdPipelines, nil
	}
	envMapping := make(map[int]*repository6.Environment)
	appWorkflowMapping := make(map[int]*appWorkflow.AppWorkflowMapping)

	envs, err := impl.environmentRepository.FindByIds(envIds)
	if err != nil && errors.IsNotFound(err) {
		impl.logger.Errorw("error in fetching environments", "err", err)
		return cdPipelines, err
	}
	//creating map for envId and respective env
	for _, env := range envs {
		envMapping[env.Id] = env
	}
	strategiesMapping, err := impl.getStrategiesMapping(dbPipelineIds)
	if err != nil {
		return cdPipelines, err
	}
	appWorkflowMappings, err := impl.appWorkflowRepository.FindByCDPipelineIds(dbPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in fetching app workflow mappings by pipelineIds", "err", err)
		return nil, err
	}
	for _, appWorkflow := range appWorkflowMappings {
		appWorkflowMapping[appWorkflow.ComponentId] = appWorkflow
	}

	var pipelines []*bean.CDPipelineConfigObject
	for _, dbPipeline := range cdPipelines.Pipelines {
		environment := &repository6.Environment{}
		var strategies []*chartConfig.PipelineStrategy
		appToWorkflowMapping := &appWorkflow.AppWorkflowMapping{}

		if envMapping[dbPipeline.EnvironmentId] != nil {
			environment = envMapping[dbPipeline.EnvironmentId]
		}
		if len(strategiesMapping[dbPipeline.Id]) != 0 {
			strategies = strategiesMapping[dbPipeline.Id]
		}
		if appWorkflowMapping[dbPipeline.Id] != nil {
			appToWorkflowMapping = appWorkflowMapping[dbPipeline.Id]
		}
		var strategiesBean []bean.Strategy
		var deploymentTemplate chartRepoRepository.DeploymentStrategy
		for _, item := range strategies {
			strategiesBean = append(strategiesBean, bean.Strategy{
				Config:             []byte(item.Config),
				DeploymentTemplate: item.Strategy,
				Default:            item.Default,
			})

			if item.Default {
				deploymentTemplate = item.Strategy
			}
		}
		var customTag *bean.CustomTagData
		var customTagStage repository5.PipelineStageType
		var customTagEnabled bool
		customTagPreCD, err := impl.customTagService.GetActiveCustomTagByEntityKeyAndValue(pipelineConfigBean.EntityTypePreCD, strconv.Itoa(dbPipeline.Id))
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in fetching custom Tag precd")
			return nil, err
		}
		customTagPostCD, err := impl.customTagService.GetActiveCustomTagByEntityKeyAndValue(pipelineConfigBean.EntityTypePostCD, strconv.Itoa(dbPipeline.Id))
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in fetching custom Tag precd")
			return nil, err
		}
		if customTagPreCD != nil && customTagPreCD.Id > 0 {
			customTag = &bean.CustomTagData{TagPattern: customTagPreCD.TagPattern,
				CounterX: customTagPreCD.AutoIncreasingNumber,
				Enabled:  customTagPreCD.Enabled,
			}
			customTagStage = repository5.PIPELINE_STAGE_TYPE_PRE_CD
			customTagEnabled = customTagPreCD.Enabled
		} else if customTagPostCD != nil && customTagPostCD.Id > 0 {
			customTag = &bean.CustomTagData{TagPattern: customTagPostCD.TagPattern,
				CounterX: customTagPostCD.AutoIncreasingNumber,
				Enabled:  customTagPostCD.Enabled,
			}
			customTagStage = repository5.PIPELINE_STAGE_TYPE_POST_CD
			customTagEnabled = customTagPostCD.Enabled
		}

		digestConfigurationRequest := imageDigestPolicy.DigestPolicyConfigurationRequest{PipelineId: dbPipeline.Id}
		digestPolicyConfigurations, err := impl.imageDigestPolicyService.GetDigestPolicyConfigurations(digestConfigurationRequest)
		if err != nil {
			impl.logger.Errorw("error in checking if isImageDigestPolicyConfiguredForPipeline", "err", err, "pipelineId", dbPipeline.Id)
			return nil, err
		}

		pipeline := &bean.CDPipelineConfigObject{
			Id:                            dbPipeline.Id,
			Name:                          dbPipeline.Name,
			EnvironmentId:                 dbPipeline.EnvironmentId,
			EnvironmentName:               environment.Name,
			Description:                   environment.Description,
			CiPipelineId:                  dbPipeline.CiPipelineId,
			DeploymentTemplate:            deploymentTemplate,
			TriggerType:                   dbPipeline.TriggerType,
			Strategies:                    strategiesBean,
			PreStage:                      dbPipeline.PreStage,
			PostStage:                     dbPipeline.PostStage,
			PreStageConfigMapSecretNames:  dbPipeline.PreStageConfigMapSecretNames,
			PostStageConfigMapSecretNames: dbPipeline.PostStageConfigMapSecretNames,
			RunPreStageInEnv:              dbPipeline.RunPreStageInEnv,
			RunPostStageInEnv:             dbPipeline.RunPostStageInEnv,
			DeploymentAppType:             dbPipeline.DeploymentAppType,
			DeploymentAppCreated:          dbPipeline.DeploymentAppCreated,
			ParentPipelineType:            appToWorkflowMapping.ParentType,
			ParentPipelineId:              appToWorkflowMapping.ParentId,
			DeploymentAppDeleteRequest:    dbPipeline.DeploymentAppDeleteRequest,
			IsVirtualEnvironment:          dbPipeline.IsVirtualEnvironment,
			PreDeployStage:                dbPipeline.PreDeployStage,
			PostDeployStage:               dbPipeline.PostDeployStage,
			CustomTagObject:               customTag,
			CustomTagStage:                &customTagStage,
			EnableCustomTag:               customTagEnabled,
			IsDigestEnforcedForPipeline:   digestPolicyConfigurations.DigestConfiguredForPipeline,
			IsDigestEnforcedForEnv:        digestPolicyConfigurations.DigestConfiguredForEnvOrCluster, // will always be false in oss
		}
		pipelines = append(pipelines, pipeline)
	}
	cdPipelines.Pipelines = pipelines
	return cdPipelines, err
}

func (impl *CdPipelineConfigServiceImpl) GetCdPipelinesForAppAndEnv(appId int, envId int) (cdPipelines *bean.CdPipelines, err error) {
	return impl.ciCdPipelineOrchestrator.GetCdPipelinesForAppAndEnv(appId, envId)
}

func (impl *CdPipelineConfigServiceImpl) GetCdPipelinesByEnvironment(request resourceGroup2.ResourceGroupingRequest, token string) (cdPipelines *bean.CdPipelines, err error) {
	_, span := otel.Tracer("orchestrator").Start(request.Ctx, "cdHandler.authorizationCdPipelinesForResourceGrouping")
	if request.ResourceGroupId > 0 {
		appIds, err := impl.resourceGroupService.GetResourceIdsByResourceGroupId(request.ResourceGroupId)
		if err != nil {
			return nil, err
		}
		//override appIds if already provided app group id in request.
		request.ResourceIds = appIds
	}
	cdPipelines, err = impl.ciCdPipelineOrchestrator.GetCdPipelinesForEnv(request.ParentResourceId, request.ResourceIds)
	if err != nil {
		impl.logger.Errorw("error in fetching pipeline", "err", err)
		return cdPipelines, err
	}
	pipelineIds := make([]int, 0)
	for _, pipeline := range cdPipelines.Pipelines {
		pipelineIds = append(pipelineIds, pipeline.Id)
	}
	if len(pipelineIds) == 0 {
		err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no matching pipeline found"}
		return cdPipelines, err
	}
	//authorization block starts here
	var appObjectArr []string
	var envObjectArr []string
	objects := impl.enforcerUtil.GetAppAndEnvObjectByPipeline(cdPipelines.Pipelines)
	pipelineIds = []int{}
	for _, object := range objects {
		appObjectArr = append(appObjectArr, object[0])
		envObjectArr = append(envObjectArr, object[1])
	}
	appResults, envResults := request.CheckAuthBatch(token, appObjectArr, envObjectArr)
	//authorization block ends here
	span.End()
	var pipelines []*bean.CDPipelineConfigObject
	authorizedPipelines := make(map[int]*bean.CDPipelineConfigObject)
	for _, dbPipeline := range cdPipelines.Pipelines {
		appObject := objects[dbPipeline.Id][0]
		envObject := objects[dbPipeline.Id][1]
		if !(appResults[appObject] && envResults[envObject]) {
			//if user unauthorized, skip items
			continue
		}
		pipelineIds = append(pipelineIds, dbPipeline.Id)
		authorizedPipelines[dbPipeline.Id] = dbPipeline
	}

	pipelineDeploymentTemplate := make(map[int]chartRepoRepository.DeploymentStrategy)
	pipelineWorkflowMapping := make(map[int]*appWorkflow.AppWorkflowMapping)
	if len(pipelineIds) == 0 {
		err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no authorized pipeline found"}
		return cdPipelines, err
	}
	_, span = otel.Tracer("orchestrator").Start(request.Ctx, "cdHandler.GetAllStrategyByPipelineIds")
	strategies, err := impl.pipelineConfigRepository.GetAllStrategyByPipelineIds(pipelineIds)
	span.End()
	if err != nil {
		impl.logger.Errorw("error in fetching strategies", "err", err)
		return cdPipelines, err
	}
	for _, item := range strategies {
		if item.Default {
			pipelineDeploymentTemplate[item.PipelineId] = item.Strategy
		}
	}
	_, span = otel.Tracer("orchestrator").Start(request.Ctx, "cdHandler.FindByCDPipelineIds")
	appWorkflowMappings, err := impl.appWorkflowRepository.FindByCDPipelineIds(pipelineIds)
	span.End()
	if err != nil {
		impl.logger.Errorw("error in fetching workflows", "err", err)
		return nil, err
	}
	for _, item := range appWorkflowMappings {
		pipelineWorkflowMapping[item.ComponentId] = item
	}
	isAppLevelGitOpsConfigured := false
	gitOpsConfigStatus, err := impl.gitOpsConfigReadService.IsGitOpsConfigured()
	if err != nil {
		impl.logger.Errorw("error in fetching global GitOps configuration")
		return nil, err
	}
	if gitOpsConfigStatus.IsGitOpsConfigured && !gitOpsConfigStatus.AllowCustomRepository {
		isAppLevelGitOpsConfigured = true
	}
	var strPipelineIds []string
	for _, pipelineId := range pipelineIds {
		strPipelineIds = append(strPipelineIds, strconv.Itoa(pipelineId))
	}
	customTagMapResponse, err := impl.customTagService.GetActiveCustomTagByValues(strPipelineIds)
	if err != nil {
		return nil, err
	}

	for _, dbPipeline := range authorizedPipelines {
		var customTag *bean.CustomTagData
		var customTagStage repository5.PipelineStageType
		customTagPreCD := customTagMapResponse.GetCustomTagForEntityKey(pipelineConfigBean.EntityTypePreCD, strconv.Itoa(dbPipeline.Id))
		customTagPostCD := customTagMapResponse.GetCustomTagForEntityKey(pipelineConfigBean.EntityTypePostCD, strconv.Itoa(dbPipeline.Id))
		if customTagPreCD != nil && customTagPreCD.Id > 0 {
			customTag = &bean.CustomTagData{TagPattern: customTagPreCD.TagPattern,
				CounterX: customTagPreCD.AutoIncreasingNumber,
			}
			customTagStage = repository5.PIPELINE_STAGE_TYPE_PRE_CD
		} else if customTagPostCD != nil && customTagPostCD.Id > 0 {
			customTag = &bean.CustomTagData{TagPattern: customTagPostCD.TagPattern,
				CounterX: customTagPostCD.AutoIncreasingNumber,
			}
			customTagStage = repository5.PIPELINE_STAGE_TYPE_POST_CD
		}
		if !isAppLevelGitOpsConfigured {
			isAppLevelGitOpsConfigured, err = impl.chartService.IsGitOpsRepoConfiguredForDevtronApp(dbPipeline.AppId)
			if err != nil {
				impl.logger.Errorw("error in fetching latest chart details for app by appId")
				return nil, err
			}
		}

		pipeline := &bean.CDPipelineConfigObject{
			Id:                            dbPipeline.Id,
			Name:                          dbPipeline.Name,
			EnvironmentId:                 dbPipeline.EnvironmentId,
			EnvironmentName:               dbPipeline.EnvironmentName,
			CiPipelineId:                  dbPipeline.CiPipelineId,
			DeploymentTemplate:            pipelineDeploymentTemplate[dbPipeline.Id],
			TriggerType:                   dbPipeline.TriggerType,
			PreStage:                      dbPipeline.PreStage,
			PostStage:                     dbPipeline.PostStage,
			PreStageConfigMapSecretNames:  dbPipeline.PreStageConfigMapSecretNames,
			PostStageConfigMapSecretNames: dbPipeline.PostStageConfigMapSecretNames,
			RunPreStageInEnv:              dbPipeline.RunPreStageInEnv,
			RunPostStageInEnv:             dbPipeline.RunPostStageInEnv,
			DeploymentAppType:             dbPipeline.DeploymentAppType,
			ParentPipelineType:            pipelineWorkflowMapping[dbPipeline.Id].ParentType,
			ParentPipelineId:              pipelineWorkflowMapping[dbPipeline.Id].ParentId,
			AppName:                       dbPipeline.AppName,
			AppId:                         dbPipeline.AppId,
			IsVirtualEnvironment:          dbPipeline.IsVirtualEnvironment,
			PreDeployStage:                dbPipeline.PreDeployStage,
			PostDeployStage:               dbPipeline.PostDeployStage,
			CustomTagObject:               customTag,
			CustomTagStage:                &customTagStage,
			IsGitOpsRepoNotConfigured:     !isAppLevelGitOpsConfigured,
		}
		pipelines = append(pipelines, pipeline)
	}
	cdPipelines.Pipelines = pipelines
	return cdPipelines, err
}

func (impl *CdPipelineConfigServiceImpl) GetCdPipelinesByEnvironmentMin(request resourceGroup2.ResourceGroupingRequest, token string) (cdPipelines []*bean.CDPipelineConfigObject, err error) {
	_, span := otel.Tracer("orchestrator").Start(request.Ctx, "cdHandler.authorizationCdPipelinesForResourceGrouping")
	if request.ResourceGroupId > 0 {
		appIds, err := impl.resourceGroupService.GetResourceIdsByResourceGroupId(request.ResourceGroupId)
		if err != nil {
			return cdPipelines, err
		}
		//override appIds if already provided app group id in request.
		request.ResourceIds = appIds
	}
	var pipelines []*pipelineConfig.Pipeline
	if len(request.ResourceIds) > 0 {
		pipelines, err = impl.pipelineRepository.FindActiveByInFilter(request.ParentResourceId, request.ResourceIds)
	} else {
		pipelines, err = impl.pipelineRepository.FindActiveByEnvId(request.ParentResourceId)
	}
	if err != nil {
		impl.logger.Errorw("error in fetching pipelines", "request", request, "err", err)
		return cdPipelines, err
	}
	//authorization block starts here
	var appObjectArr []string
	var envObjectArr []string
	objects := impl.enforcerUtil.GetAppAndEnvObjectByDbPipeline(pipelines)
	for _, object := range objects {
		appObjectArr = append(appObjectArr, object[0])
		envObjectArr = append(envObjectArr, object[1])
	}
	appResults, envResults := request.CheckAuthBatch(token, appObjectArr, envObjectArr)
	//authorization block ends here
	span.End()
	for _, dbPipeline := range pipelines {
		appObject := objects[dbPipeline.Id][0]
		envObject := objects[dbPipeline.Id][1]
		if !(appResults[appObject] && envResults[envObject]) {
			//if user unauthorized, skip items
			continue
		}

		envDeploymentConfig, err := impl.deploymentConfigService.GetConfigForDevtronApps(dbPipeline.AppId, dbPipeline.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", dbPipeline.AppId, "envId", dbPipeline.EnvironmentId, "err", err)
			return nil, err
		}

		pcObject := &bean.CDPipelineConfigObject{
			AppId:                dbPipeline.AppId,
			AppName:              dbPipeline.App.AppName,
			EnvironmentId:        dbPipeline.EnvironmentId,
			Id:                   dbPipeline.Id,
			DeploymentAppType:    envDeploymentConfig.DeploymentAppType,
			IsVirtualEnvironment: dbPipeline.Environment.IsVirtualEnvironment,
		}
		cdPipelines = append(cdPipelines, pcObject)
	}
	return cdPipelines, err
}

func (impl *CdPipelineConfigServiceImpl) PerformBulkActionOnCdPipelines(dto *bean.CdBulkActionRequestDto, impactedPipelines []*pipelineConfig.Pipeline, ctx context.Context, dryRun bool, userId int32) ([]*bean.CdBulkActionResponseDto, error) {
	switch dto.Action {
	case bean.CD_BULK_DELETE:
		deleteAction := bean.CASCADE_DELETE
		if dto.ForceDelete {
			deleteAction = bean.FORCE_DELETE
		} else if !dto.CascadeDelete {
			deleteAction = bean.NON_CASCADE_DELETE
		}
		bulkDeleteResp := impl.BulkDeleteCdPipelines(impactedPipelines, ctx, dryRun, deleteAction, userId)
		return bulkDeleteResp, nil
	default:
		return nil, &util.ApiError{Code: "400", HttpStatusCode: 400, UserMessage: "this action is not supported"}
	}
}

func (impl *CdPipelineConfigServiceImpl) FindPipelineById(cdPipelineId int) (*pipelineConfig.Pipeline, error) {
	return impl.pipelineRepository.FindById(cdPipelineId)
}

func (impl *CdPipelineConfigServiceImpl) FindAppAndEnvDetailsByPipelineId(cdPipelineId int) (*pipelineConfig.Pipeline, error) {
	return impl.pipelineRepository.FindAppAndEnvDetailsByPipelineId(cdPipelineId)
}

func (impl *CdPipelineConfigServiceImpl) RetrieveParentDetails(pipelineId int) (parentId int, parentType bean2.WorkflowType, err error) {

	workflow, err := impl.appWorkflowRepository.GetParentDetailsByPipelineId(pipelineId)
	if err != nil {
		impl.logger.Errorw("failed to get parent component details",
			"componentId", pipelineId,
			"err", err)
		return 0, "", err
	}

	if workflow.ParentType == appWorkflow.CDPIPELINE {
		// workflow is of type CD, check for stage
		// for older apps post cd script was stored in post_stage_config_yaml, for newer apps new stage is created in pipeline_stage
		parentPostStage, err := impl.pipelineStageService.GetCdStageByCdPipelineIdAndStageType(workflow.ParentId, repository5.PIPELINE_STAGE_TYPE_POST_CD, false)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in fetching post stage by pipeline id", "err", err, "cd-pipeline-id", parentId)
			return workflow.ParentId, bean2.CD_WORKFLOW_TYPE_DEPLOY, err
		}
		parentPipeline, err := impl.pipelineRepository.GetPostStageConfigById(workflow.ParentId)
		if err != nil {
			impl.logger.Errorw("failed to get the post_stage_config_yaml",
				"cdPipelineId", workflow.ParentId,
				"err", err)
			return 0, "", err
		}

		if len(parentPipeline.PostStageConfig) > 0 || parentPostStage.IsPipelineStageExists() {
			return workflow.ParentId, bean2.CD_WORKFLOW_TYPE_POST, nil
		}
		return workflow.ParentId, bean2.CD_WORKFLOW_TYPE_DEPLOY, nil

	} else if workflow.ParentType == appWorkflow.WEBHOOK {
		// For webhook type
		return workflow.ParentId, bean2.WEBHOOK_WORKFLOW_TYPE, nil
	}

	return workflow.ParentId, bean2.CI_WORKFLOW_TYPE, nil
}

func (impl *CdPipelineConfigServiceImpl) GetEnvironmentByCdPipelineId(pipelineId int) (int, error) {
	dbPipeline, err := impl.pipelineRepository.FindById(pipelineId)
	if err != nil || dbPipeline == nil {
		impl.logger.Errorw("error in fetching pipeline", "err", err)
		return 0, err
	}
	return dbPipeline.EnvironmentId, err
}

func (impl *CdPipelineConfigServiceImpl) GetBulkActionImpactedPipelines(dto *bean.CdBulkActionRequestDto) ([]*pipelineConfig.Pipeline, error) {
	if len(dto.EnvIds) == 0 || (len(dto.AppIds) == 0 && len(dto.ProjectIds) == 0) {
		//invalid payload, envIds are must and either of appIds or projectIds are must
		return nil, &util.ApiError{Code: "400", HttpStatusCode: 400, UserMessage: "invalid payload, can not get pipelines for this filter"}
	}
	var pipelineIdsByAppLevel []int
	var pipelineIdsByProjectLevel []int
	var err error
	if len(dto.AppIds) > 0 && len(dto.EnvIds) > 0 {
		//getting pipeline IDs for app level deletion request
		pipelineIdsByAppLevel, err = impl.pipelineRepository.FindIdsByAppIdsAndEnvironmentIds(dto.AppIds, dto.EnvIds)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in getting cd pipelines by appIds and envIds", "err", err)
			return nil, err
		}
	}
	if len(dto.ProjectIds) > 0 && len(dto.EnvIds) > 0 {
		//getting pipeline IDs for project level deletion request
		pipelineIdsByProjectLevel, err = impl.pipelineRepository.FindIdsByProjectIdsAndEnvironmentIds(dto.ProjectIds, dto.EnvIds)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in getting cd pipelines by projectIds and envIds", "err", err)
			return nil, err
		}
	}
	var pipelineIdsMerged []int
	//it might be possible that pipelineIdsByAppLevel & pipelineIdsByProjectLevel have some same values
	//we are still appending them to save operation cost of checking same ids as we will get pipelines from
	//in clause which gives correct results even if some values are repeating
	pipelineIdsMerged = append(pipelineIdsMerged, pipelineIdsByAppLevel...)
	pipelineIdsMerged = append(pipelineIdsMerged, pipelineIdsByProjectLevel...)
	var pipelines []*pipelineConfig.Pipeline
	if len(pipelineIdsMerged) > 0 {
		pipelines, err = impl.pipelineRepository.FindByIdsIn(pipelineIdsMerged)
		if err != nil {
			impl.logger.Errorw("error in getting cd pipelines by ids", "err", err, "ids", pipelineIdsMerged)
			return nil, err
		}
	}
	return pipelines, nil
}

func (impl *CdPipelineConfigServiceImpl) IsGitOpsRequiredForCD(pipelineCreateRequest *bean.CdPipelines) bool {

	// if deploymentAppType is not coming in request than hasAtLeastOneGitOps will be false

	haveAtLeastOneGitOps := false
	for _, pipeline := range pipelineCreateRequest.Pipelines {
		if pipeline.EnvironmentId > 0 && pipeline.DeploymentAppType == util.PIPELINE_DEPLOYMENT_TYPE_ACD {
			haveAtLeastOneGitOps = true
		}
	}
	return haveAtLeastOneGitOps
}

func (impl *CdPipelineConfigServiceImpl) MarkGitOpsDevtronAppsDeletedWhereArgoAppIsDeleted(pipeline *pipelineConfig.Pipeline) (bool, error) {

	acdAppFound := false

	acdAppName := pipeline.DeploymentAppName
	_, err := impl.argoClientWrapperService.GetArgoAppByName(context.Background(), acdAppName)
	if err == nil {
		// acd app is not yet deleted so return
		acdAppFound = true
		return acdAppFound, err
	}
	impl.logger.Warnw("app not found in argo, deleting from db ", "err", err)
	//make call to delete it from pipeline DB because it's ACD counterpart is deleted
	_, err = impl.DeleteCdPipeline(pipeline, context.Background(), bean.FORCE_DELETE, false, 1)
	if err != nil {
		impl.logger.Errorw("error in deleting cd pipeline", "err", err)
		return acdAppFound, err
	}
	return acdAppFound, nil
}

func (impl *CdPipelineConfigServiceImpl) GetEnvironmentListForAutocompleteFilter(envName string, clusterIds []int, offset int, size int, token string, checkAuthBatch func(token string, appObject []string, envObject []string) (map[string]bool, map[string]bool), ctx context.Context) (*clutserBean.ResourceGroupingResponse, error) {
	result := &clutserBean.ResourceGroupingResponse{}
	var models []*repository6.Environment
	var beans []clutserBean.EnvironmentBean
	var err error
	if len(envName) > 0 && len(clusterIds) > 0 {
		models, err = impl.environmentRepository.FindByEnvNameAndClusterIds(envName, clusterIds)
	} else if len(clusterIds) > 0 {
		models, err = impl.environmentRepository.FindByClusterIdsWithFilter(clusterIds)
	} else if len(envName) > 0 {
		models, err = impl.environmentRepository.FindByEnvName(envName)
	} else {
		models, err = impl.environmentRepository.FindAllActiveWithFilter()
	}
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching environment", "err", err)
		return result, err
	}
	var envIds []int
	for _, model := range models {
		envIds = append(envIds, model.Id)
	}
	if len(envIds) == 0 {
		err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no matching environment found"}
		return nil, err
	}
	_, span := otel.Tracer("orchestrator").Start(ctx, "pipelineBuilder.FindActiveByEnvIds")
	cdPipelines, err := impl.pipelineRepository.FindActiveByEnvIds(envIds)
	span.End()
	if err != nil && err != pg.ErrNoRows {
		return result, err
	}
	pipelineIds := make([]int, 0)
	for _, pipeline := range cdPipelines {
		pipelineIds = append(pipelineIds, pipeline.Id)
	}
	if len(pipelineIds) == 0 {
		err = &util.ApiError{Code: "404", HttpStatusCode: 200, UserMessage: "no matching pipeline found"}
		return nil, err
	}
	//authorization block starts here
	var appObjectArr []string
	var envObjectArr []string
	_, span = otel.Tracer("orchestrator").Start(ctx, "pipelineBuilder.GetAppAndEnvObjectByPipelineIds")
	objects := impl.enforcerUtil.GetAppAndEnvObjectByPipelineIds(pipelineIds)
	span.End()
	pipelineIds = []int{}
	for _, object := range objects {
		appObjectArr = append(appObjectArr, object[0])
		envObjectArr = append(envObjectArr, object[1])
	}
	_, span = otel.Tracer("orchestrator").Start(ctx, "pipelineBuilder.checkAuthBatch")
	appResults, envResults := checkAuthBatch(token, appObjectArr, envObjectArr)
	span.End()
	//authorization block ends here

	pipelinesMap := make(map[int][]*pipelineConfig.Pipeline)
	for _, pipeline := range cdPipelines {
		appObject := objects[pipeline.Id][0]
		envObject := objects[pipeline.Id][1]
		if !(appResults[appObject] && envResults[envObject]) {
			//if user unauthorized, skip items
			continue
		}
		pipelinesMap[pipeline.EnvironmentId] = append(pipelinesMap[pipeline.EnvironmentId], pipeline)
	}
	for _, model := range models {
		environment := clutserBean.EnvironmentBean{
			Id:                    model.Id,
			Environment:           model.Name,
			Namespace:             model.Namespace,
			CdArgoSetup:           model.Cluster.CdArgoSetup,
			EnvironmentIdentifier: model.EnvironmentIdentifier,
			ClusterName:           model.Cluster.ClusterName,
			IsVirtualEnvironment:  model.IsVirtualEnvironment,
		}

		//authorization block starts here
		appCount := 0
		envPipelines := pipelinesMap[model.Id]
		if _, ok := pipelinesMap[model.Id]; ok {
			appCount = len(envPipelines)
		}
		environment.AppCount = appCount
		beans = append(beans, environment)
	}

	envCount := len(beans)
	// Apply pagination
	if size > 0 {
		if offset+size <= len(beans) {
			beans = beans[offset : offset+size]
		} else {
			beans = beans[offset:]
		}
	}
	result.EnvList = beans
	result.EnvCount = envCount
	return result, nil
}

func (impl *CdPipelineConfigServiceImpl) validateCDPipelineRequest(pipelineCreateRequest *bean.CdPipelines) (bool, error) {
	envPipelineMap := make(map[int]string)
	for _, pipeline := range pipelineCreateRequest.Pipelines {
		if envPipelineMap[pipeline.EnvironmentId] != "" {
			err := &util.ApiError{
				HttpStatusCode:  http.StatusBadRequest,
				InternalMessage: "cd-pipelines already exist for this app and env, cannot create multiple cd-pipelines",
				UserMessage:     "cd-pipelines already exist for this app and env, cannot create multiple cd-pipelines",
			}
			return false, err
		}
		envPipelineMap[pipeline.EnvironmentId] = pipeline.Name

		existingCdPipelinesForEnv, pErr := impl.pipelineRepository.FindActiveByAppIdAndEnvironmentId(pipelineCreateRequest.AppId, pipeline.EnvironmentId)
		if pErr != nil && !util.IsErrNoRows(pErr) {
			impl.logger.Errorw("error in fetching cd pipelines ", "err", pErr, "appId", pipelineCreateRequest.AppId)
			return false, pErr
		}
		if len(existingCdPipelinesForEnv) > 0 {
			err := &util.ApiError{
				HttpStatusCode:  http.StatusBadRequest,
				InternalMessage: "cd-pipelines already exist for this app and env, cannot create multiple cd-pipelines",
				UserMessage:     "cd-pipelines already exist for this app and env, cannot create multiple cd-pipelines",
			}
			return false, err
		}
		if len(pipeline.PreStage.Config) > 0 && !strings.Contains(pipeline.PreStage.Config, "beforeStages") {
			err := &util.ApiError{
				HttpStatusCode:  http.StatusBadRequest,
				InternalMessage: "invalid yaml config, must include - beforeStages",
				UserMessage:     "invalid yaml config, must include - beforeStages",
			}
			return false, err
		}
		if len(pipeline.PostStage.Config) > 0 && !strings.Contains(pipeline.PostStage.Config, "afterStages") {
			err := &util.ApiError{
				HttpStatusCode:  http.StatusBadRequest,
				InternalMessage: "invalid yaml config, must include - afterStages",
				UserMessage:     "invalid yaml config, must include - afterStages",
			}
			return false, err
		}

	}

	return true, nil

}

func (impl *CdPipelineConfigServiceImpl) RegisterInACD(ctx context.Context, chartGitAttr *commonBean.ChartGitAttribute, userId int32) error {
	err := impl.argoClientWrapperService.RegisterGitOpsRepoInArgoWithRetry(ctx, chartGitAttr.RepoUrl, userId)
	if err != nil {
		impl.logger.Errorw("error while register git repo in argo", "err", err)
		return err
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) createCdPipeline(ctx context.Context, app *app2.App, pipeline *bean.CDPipelineConfigObject, deploymentConfig *bean4.DeploymentConfig, userId int32) (pipelineRes int, err error) {
	dbConnection := impl.pipelineRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return 0, err
	}
	// Rollback tx on error.
	defer tx.Rollback()
	if (pipeline.AppWorkflowId == 0 || pipeline.IsSwitchCiPipelineRequest()) && pipeline.ParentPipelineType == "WEBHOOK" {
		if pipeline.AppWorkflowId == 0 {
			wf := &appWorkflow.AppWorkflow{
				Name:     fmt.Sprintf("wf-%d-%s", app.Id, util2.Generate(4)),
				AppId:    app.Id,
				Active:   true,
				AuditLog: sql.AuditLog{CreatedBy: userId, CreatedOn: time.Now(), UpdatedOn: time.Now(), UpdatedBy: userId},
			}
			savedAppWf, err := impl.appWorkflowRepository.SaveAppWorkflowWithTx(wf, tx)
			if err != nil {
				impl.logger.Errorw("error in saving app workflow", "appId", app.Id, "err", err)
				return 0, err
			}
			pipeline.AppWorkflowId = savedAppWf.Id
		}
		externalCiPipelineId, appWorkflowMapping, err := impl.ciPipelineConfigService.CreateExternalCiAndAppWorkflowMapping(app.Id, pipeline.AppWorkflowId, userId, tx)
		if err != nil {
			impl.logger.Errorw("error in creating new external ci pipeline and new app workflow mapping", "appId", app.Id, "err", err)
			return 0, err
		}
		if pipeline.IsSwitchCiPipelineRequest() {
			err = impl.buildPipelineSwitchService.SwitchToExternalCi(tx, appWorkflowMapping, pipeline.SwitchFromCiPipelineId, userId)
			if err != nil {
				impl.logger.Errorw("error in switching external ci", "appId", app.Id, "switchFromExternalCiPipelineId", pipeline.SwitchFromCiPipelineId, "userId", userId, "err", err)
				return 0, err
			}
		}
		pipeline.ParentPipelineId = externalCiPipelineId
	}

	// do not create the pipeline if environment is not set
	pipelineId := 0
	if pipeline.EnvironmentId > 0 {
		latestChart, err := impl.chartRepository.FindLatestChartForAppByAppId(app.Id)
		if err != nil {
			return 0, err
		}
		//getting global app metrics for cd pipeline create because env level metrics is not created yet
		appLevelAppMetricsEnabled := false
		isAppLevelMetricsEnabled, err := impl.deployedAppMetricsService.GetMetricsFlagByAppId(app.Id)
		if err != nil {
			impl.logger.Errorw("error, GetMetricsFlagByAppId", "err", err, "appId", app.Id)
			return 0, err
		}
		appLevelAppMetricsEnabled = isAppLevelMetricsEnabled

		var (
			envOverride       *bean5.EnvConfigOverride
			updatedAppMetrics bool
		)
		if pipeline.IsExternalArgoAppLinkRequest() {
			overrideCreateRequest, err := impl.parseEnvOverrideCreateRequestForExternalAcdApp(deploymentConfig, latestChart, app, userId, pipeline, appLevelAppMetricsEnabled)
			envOverride, updatedAppMetrics, err = impl.propertiesConfigService.CreateIfRequired(overrideCreateRequest, tx)
			if err != nil {
				impl.logger.Errorw("error in creating env override", "appId", app.Id, "envId", envOverride.TargetEnvironment, "err", err)
				return 0, err
			}
		} else {
			overrideCreateRequest := &pipelineConfigBean.EnvironmentOverrideCreateInternalDTO{
				Chart:               latestChart,
				EnvironmentId:       pipeline.EnvironmentId,
				UserId:              userId,
				ManualReviewed:      false,
				ChartStatus:         models.CHARTSTATUS_NEW,
				IsOverride:          false,
				IsAppMetricsEnabled: appLevelAppMetricsEnabled,
				IsBasicViewLocked:   false,
				Namespace:           pipeline.Namespace,
				CurrentViewEditor:   latestChart.CurrentViewEditor,
				MergeStrategy:       "",
			}
			envOverride, updatedAppMetrics, err = impl.propertiesConfigService.CreateIfRequired(overrideCreateRequest, tx)
			if err != nil {
				return 0, err
			}
			appLevelAppMetricsEnabled = updatedAppMetrics
		}

		appLevelAppMetricsEnabled = updatedAppMetrics
		// Get pipeline override based on Deployment strategy
		//TODO: mark as created in our db
		pipelineId, err = impl.ciCdPipelineOrchestrator.CreateCDPipelines(pipeline, app.Id, userId, tx, app.AppName)
		if err != nil {
			impl.logger.Errorw("error in creating cd pipeline", "appId", app.Id, "pipeline", pipeline)
			return 0, err
		}
		if pipeline.RefPipelineId > 0 {
			pipeline.SourceToNewPipelineId[pipeline.RefPipelineId] = pipelineId
		}

		//adding pipeline to workflow
		_, err = impl.appWorkflowRepository.FindByIdAndAppId(pipeline.AppWorkflowId, app.Id)
		if err != nil && err != pg.ErrNoRows {
			return 0, err
		}
		if pipeline.AppWorkflowId > 0 {
			var parentPipelineId int
			var parentPipelineType string

			if pipeline.ParentPipelineId == 0 {
				parentPipelineId = pipeline.CiPipelineId
				parentPipelineType = "CI_PIPELINE"
			} else {
				parentPipelineId = pipeline.ParentPipelineId
				parentPipelineType = pipeline.ParentPipelineType
				if pipeline.ParentPipelineType != appWorkflow.WEBHOOK && pipeline.RefPipelineId > 0 && len(pipeline.SourceToNewPipelineId) > 0 {
					parentPipelineId = pipeline.SourceToNewPipelineId[pipeline.ParentPipelineId]
				}
			}

			if pipeline.CDPipelineAddType == bean.SEQUENTIAL {
				childPipelineIds := make([]int, 0)
				if pipeline.ChildPipelineId > 0 {
					childPipelineIds = append(childPipelineIds, pipeline.ChildPipelineId)
				}
				err = impl.appWorkflowRepository.UpdateParentComponentDetails(tx, parentPipelineId, parentPipelineType, pipelineId, "CD_PIPELINE", childPipelineIds)
				if err != nil {
					return 0, err
				}
			}

			appWorkflowMap := &appWorkflow.AppWorkflowMapping{
				AppWorkflowId: pipeline.AppWorkflowId,
				ParentId:      parentPipelineId,
				ParentType:    parentPipelineType,
				ComponentId:   pipelineId,
				Type:          "CD_PIPELINE",
				Active:        true,
				AuditLog:      sql.AuditLog{CreatedBy: userId, CreatedOn: time.Now(), UpdatedOn: time.Now(), UpdatedBy: userId},
			}
			_, err = impl.appWorkflowRepository.SaveAppWorkflowMapping(appWorkflowMap, tx)
			if err != nil {
				return 0, err
			}
		}

		err = impl.deploymentTemplateHistoryService.CreateDeploymentTemplateHistoryFromEnvOverrideTemplate(envOverride, tx, appLevelAppMetricsEnabled, pipelineId)
		if err != nil {
			impl.logger.Errorw("error in creating entry for env deployment template history", "err", err, "envOverride", envOverride)
			return 0, err
		}
		//VARIABLE_MAPPING_UPDATE
		err = impl.scopedVariableManager.ExtractAndMapVariables(envOverride.EnvOverrideValues, envOverride.Id, repository3.EntityTypeDeploymentTemplateEnvLevel, envOverride.UpdatedBy, tx)
		if err != nil {
			return 0, err
		}
		// strategies for pipeline ids, there is only one is default
		defaultCount := 0
		for _, item := range pipeline.Strategies {
			if item.Default {
				defaultCount = defaultCount + 1
				if defaultCount > 1 {
					impl.logger.Warnw("already have one strategy is default in this pipeline", "strategy", item.DeploymentTemplate)
					item.Default = false
				}
			}
			strategy := &chartConfig.PipelineStrategy{
				PipelineId: pipelineId,
				Strategy:   item.DeploymentTemplate,
				Config:     string(item.Config),
				Default:    item.Default,
				Deleted:    false,
				AuditLog:   sql.AuditLog{UpdatedBy: userId, CreatedBy: userId, UpdatedOn: time.Now(), CreatedOn: time.Now()},
			}
			err = impl.pipelineConfigRepository.Save(strategy, tx)
			if err != nil {
				impl.logger.Errorw("error in saving strategy", "strategy", item.DeploymentTemplate)
				return pipelineId, fmt.Errorf("pipeline created but failed to add strategy")
			}
			//creating history entry for strategy
			_, err = impl.pipelineStrategyHistoryService.CreatePipelineStrategyHistory(strategy, pipeline.TriggerType, tx)
			if err != nil {
				impl.logger.Errorw("error in creating strategy history entry", "err", err)
				return 0, err
			}

		}

		// save custom tag data
		err = impl.CDPipelineCustomTagDBOperations(pipeline)
		if err != nil {
			return pipelineId, err
		}

		if pipeline.IsDigestEnforcedForPipeline {
			_, err = impl.imageDigestPolicyService.CreatePolicyForPipeline(tx, pipelineId, pipeline.Name, userId)
			if err != nil {
				return pipelineId, err
			}
		}

	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	impl.logger.Debugw("pipeline created with GitMaterialId ", "id", pipelineId, "pipeline", pipeline)
	return pipelineId, nil
}

func (impl *CdPipelineConfigServiceImpl) parseEnvOverrideCreateRequestForExternalAcdApp(deploymentConfig *bean4.DeploymentConfig, latestChart *chartRepoRepository.Chart, app *app2.App, userId int32, pipeline *bean.CDPipelineConfigObject, appLevelAppMetricsEnabled bool) (*pipelineConfigBean.EnvironmentOverrideCreateInternalDTO, error) {
	values, chartMetadata, err := impl.GetValuesAndChartMetadataForExternalArgoCDApp(deploymentConfig.ReleaseConfiguration.ArgoCDSpec)
	if err != nil {
		impl.logger.Errorw("error in reading values for external argocd app", "acdAppName", deploymentConfig.ReleaseConfiguration.ArgoCDSpec.Metadata.Name, "err", err)
		return nil, err
	}

	chartName, chartVersion := chartMetadata.Name, chartMetadata.Version

	chartRef, err := impl.chartRefRepository.FindByVersionAndName(chartVersion, chartName)
	if err != nil {
		impl.logger.Errorw("error in getting chart ref by name and version", "chartName", chartName, "chartVersion", chartVersion, "err", err)
		return nil, err
	}

	chartForOverride, err := impl.chartRepository.FindChartByAppIdAndRefId(app.Id, chartRef.Id)
	if err != nil && errors3.Is(err, pg.ErrNoRows) {
		impl.logger.Errorw("error in finding chart for this app and chart_ref_id", "appId", app.Id, "chartRefId", chartRef.Id, "err", err)
		return nil, err
	}
	if errors3.Is(err, pg.ErrNoRows) {
		chartCreateRequest := chart.TemplateRequest{
			AppId:               app.Id,
			ChartRefId:          chartRef.Id,
			ValuesOverride:      []byte("{}"),
			UserId:              userId,
			IsAppMetricsEnabled: false,
		}
		_, err = impl.chartService.CreateChartFromEnvOverride(chartCreateRequest, context.Background())
		if err != nil {
			impl.logger.Errorw("error in creating chart from env override", "appId", app.Id, "chartRefId", chartRef.Id, "err", err)
			return nil, err
		}
		chartForOverride, err = impl.chartRepository.FindChartByAppIdAndRefId(app.Id, chartRef.Id)
		if err != nil && !errors3.Is(err, pg.ErrNoRows) {
			impl.logger.Errorw("error in finding chart for this app and chart_ref_id", "appId", app.Id, "chartRefId", chartRef.Id, "err", err)
			return nil, err
		}
	}
	chartForOverride.GlobalOverride = string(values)
	overrideCreateRequest := &pipelineConfigBean.EnvironmentOverrideCreateInternalDTO{
		Chart:               chartForOverride,
		EnvironmentId:       pipeline.EnvironmentId,
		UserId:              userId,
		ManualReviewed:      false,
		ChartStatus:         models.CHARTSTATUS_NEW,
		IsOverride:          true,
		IsAppMetricsEnabled: appLevelAppMetricsEnabled,
		IsBasicViewLocked:   false,
		Namespace:           pipeline.Namespace,
		CurrentViewEditor:   latestChart.CurrentViewEditor,
		MergeStrategy:       models.MERGE_STRATEGY_REPLACE,
	}
	return overrideCreateRequest, err
}

func (impl *CdPipelineConfigServiceImpl) GetValuesAndChartMetadataForExternalArgoCDApp(spec bean4.ArgoCDSpec) (json.RawMessage, *chart2.Metadata, error) {
	repoURL := spec.Spec.Source.RepoURL
	chartPath := spec.Spec.Source.Path
	//validation is performed before this step, so assuming ValueFiles array has one and only one entry
	valuesFileName := spec.Spec.Source.Helm.ValueFiles[0]
	helmChart, err := impl.extractHelmChartForExternalArgoApp(repoURL, chartPath)
	if err != nil {
		impl.logger.Errorw("error in extracting helm ")
		return nil, nil, err
	}
	for _, file := range helmChart.Files {
		if file.Name == valuesFileName {
			return file.Data, helmChart.Metadata, nil
		}
	}
	return nil, nil, errors2.New(fmt.Sprintf("values file with name %s not found in chart", valuesFileName))
}

func (impl *CdPipelineConfigServiceImpl) extractHelmChartForExternalArgoApp(repoURL string, chartPath string) (*chart2.Chart, error) {
	repoName := impl.gitOpsConfigReadService.GetGitOpsRepoNameFromUrl(repoURL)
	chartDir := fmt.Sprintf("%s-%s", repoName, impl.chartTemplateService.GetDir())
	clonedDir := impl.gitFactory.GitOpsHelper.GetCloneDirectory(chartDir)
	defer impl.chartTemplateService.CleanDir(clonedDir)
	_, err := impl.gitOperationService.CloneInDir(repoURL, clonedDir)
	if err != nil {
		impl.logger.Errorw("error in cloning in dir for external argo app", "repoURL", repoURL, "err", err)
		return nil, err
	}
	chartFullPath := filepath.Join(clonedDir, chartPath)
	helmChart, err := loader.Load(chartFullPath)
	if err != nil {
		impl.logger.Errorw("error in loading helm chart", "repoURL", repoURL, "chartPath", chartFullPath, "err", err)
		return nil, err
	}
	return helmChart, nil
}

func (impl *CdPipelineConfigServiceImpl) updateCdPipeline(ctx context.Context, pipeline *bean.CDPipelineConfigObject, userID int32) (err error) {

	if len(pipeline.PreStage.Config) > 0 && !strings.Contains(pipeline.PreStage.Config, "beforeStages") {
		err = &util.ApiError{
			HttpStatusCode:  http.StatusBadRequest,
			InternalMessage: "invalid yaml config, must include - beforeStages",
			UserMessage:     "invalid yaml config, must include - beforeStages",
		}
		return err
	}
	if len(pipeline.PostStage.Config) > 0 && !strings.Contains(pipeline.PostStage.Config, "afterStages") {
		err = &util.ApiError{
			HttpStatusCode:  http.StatusBadRequest,
			InternalMessage: "invalid yaml config, must include - afterStages",
			UserMessage:     "invalid yaml config, must include - afterStages",
		}
		return err
	}
	dbConnection := impl.pipelineRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return err
	}
	// Rollback tx on error.
	defer tx.Rollback()
	dbPipelineObj, err := impl.ciCdPipelineOrchestrator.UpdateCDPipeline(pipeline, userID, tx)
	if err != nil {
		impl.logger.Errorw("error in updating pipeline")
		return err
	}

	// strategies for pipeline ids, there is only one is default
	existingStrategies, err := impl.pipelineConfigRepository.GetAllStrategyByPipelineId(pipeline.Id)
	if err != nil && !errors2.IsNotFound(err) {
		impl.logger.Errorw("error in getting pipeline strategies", "err", err)
		return err
	}
	for _, oldItem := range existingStrategies {
		notFound := true
		for _, newItem := range pipeline.Strategies {
			if newItem.DeploymentTemplate == oldItem.Strategy {
				notFound = false
			}
		}

		if notFound {
			//delete from db
			err := impl.pipelineConfigRepository.Delete(oldItem, tx)
			if err != nil {
				impl.logger.Errorw("error in delete pipeline strategies", "err", err)
				return fmt.Errorf("error in delete pipeline strategies")
			}
		}
	}

	defaultCount := 0
	for _, item := range pipeline.Strategies {
		if item.Default {
			defaultCount = defaultCount + 1
			if defaultCount > 1 {
				impl.logger.Warnw("already have one strategy is default in this pipeline, skip this", "strategy", item.DeploymentTemplate)
				continue
			}
		}
		strategy, err := impl.pipelineConfigRepository.FindByStrategyAndPipelineId(item.DeploymentTemplate, pipeline.Id)
		if err != nil && pg.ErrNoRows != err {
			impl.logger.Errorw("error in getting strategy", "err", err)
			return err
		}
		if strategy.Id > 0 {
			strategy.Config = string(item.Config)
			strategy.Default = item.Default
			strategy.UpdatedBy = userID
			strategy.UpdatedOn = time.Now()
			err = impl.pipelineConfigRepository.Update(strategy, tx)
			if err != nil {
				impl.logger.Errorw("error in updating strategy", "strategy", item.DeploymentTemplate)
				return fmt.Errorf("pipeline updated but failed to update one strategy")
			}
			//creating history entry for strategy
			_, err = impl.pipelineStrategyHistoryService.CreatePipelineStrategyHistory(strategy, pipeline.TriggerType, tx)
			if err != nil {
				impl.logger.Errorw("error in creating strategy history entry", "err", err)
				return err
			}
		} else {
			strategy := &chartConfig.PipelineStrategy{
				PipelineId: pipeline.Id,
				Strategy:   item.DeploymentTemplate,
				Config:     string(item.Config),
				Default:    item.Default,
				Deleted:    false,
				AuditLog:   sql.AuditLog{UpdatedBy: userID, CreatedBy: userID, UpdatedOn: time.Now(), CreatedOn: time.Now()},
			}
			err = impl.pipelineConfigRepository.Save(strategy, tx)
			if err != nil {
				impl.logger.Errorw("error in saving strategy", "strategy", item.DeploymentTemplate)
				return fmt.Errorf("pipeline created but failed to add strategy")
			}
			//creating history entry for strategy
			_, err = impl.pipelineStrategyHistoryService.CreatePipelineStrategyHistory(strategy, pipeline.TriggerType, tx)
			if err != nil {
				impl.logger.Errorw("error in creating strategy history entry", "err", err)
				return err
			}
		}
	}
	// update custom tag data
	pipeline.Id = dbPipelineObj.Id // pipeline object is request received from FE
	err = impl.CDPipelineCustomTagDBOperations(pipeline)
	if err != nil {
		impl.logger.Errorw("error in updating custom tag data for pipeline", "err", err)
		return err
	}

	_, err = impl.handleDigestPolicyOperations(tx, pipeline.Id, pipeline.Name, pipeline.IsDigestEnforcedForPipeline, userID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (impl *CdPipelineConfigServiceImpl) handleDigestPolicyOperations(tx *pg.Tx, pipelineId int, pipelineName string, isDigestEnforcedForPipeline bool, userId int32) (resourceQualifierId int, err error) {
	if isDigestEnforcedForPipeline {
		resourceQualifierId, err = impl.imageDigestPolicyService.CreatePolicyForPipelineIfNotExist(tx, pipelineId, pipelineName, userId)
		if err != nil {
			impl.logger.Errorw("error in imageDigestPolicy operations for CD pipeline", "err", err, "pipelineId", pipelineId)
			return 0, err
		}
	} else {
		resourceQualifierId, err = impl.imageDigestPolicyService.DeletePolicyForPipeline(tx, pipelineId, userId)
		if err != nil {
			impl.logger.Errorw("error in deleting imageDigestPolicy for pipeline", "err", err, "pipelineId", pipelineId)
			return 0, err
		}
	}
	return resourceQualifierId, nil
}

func (impl *CdPipelineConfigServiceImpl) DeleteCdPipelinePartial(pipeline *pipelineConfig.Pipeline, ctx context.Context, deleteAction int, userId int32) (*bean.AppDeleteResponseDTO, error) {
	cascadeDelete := true
	forceDelete := false
	deleteResponse := &bean.AppDeleteResponseDTO{
		DeleteInitiated:  false,
		ClusterReachable: true,
	}
	if deleteAction == bean.FORCE_DELETE {
		forceDelete = true
		cascadeDelete = false
	} else if deleteAction == bean.NON_CASCADE_DELETE {
		cascadeDelete = false
	}
	//Updating clusterReachable flag
	clusterBean, err := impl.clusterRepository.FindById(pipeline.Environment.ClusterId)
	if err != nil {
		impl.logger.Errorw("error in getting cluster details", "err", err, "clusterId", pipeline.Environment.ClusterId)
	}
	deleteResponse.ClusterName = clusterBean.ClusterName
	if len(clusterBean.ErrorInConnecting) > 0 {
		deleteResponse.ClusterReachable = false
	}
	//getting children CD pipeline details
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting children cd details", "err", err)
		return deleteResponse, err
	}

	dbConnection := impl.pipelineRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return deleteResponse, err
	}
	// Rollback tx on error.
	defer tx.Rollback()

	//delete app from argo cd, if created
	if pipeline.DeploymentAppCreated && !pipeline.DeploymentAppDeleteRequest {
		envDeploymentConfig, err := impl.deploymentConfigService.GetAndMigrateConfigIfAbsentForDevtronApps(pipeline.AppId, pipeline.EnvironmentId)
		if err != nil {
			impl.logger.Errorw("error in fetching environment deployment config by appId and envId", "appId", pipeline.AppId, "envId", pipeline.EnvironmentId, "err", err)
			return deleteResponse, err
		}
		deploymentAppName := pipeline.DeploymentAppName
		if util.IsAcdApp(envDeploymentConfig.DeploymentAppType) {
			if !forceDelete && !deleteResponse.ClusterReachable {
				impl.logger.Errorw("cluster connection error", "err", clusterBean.ErrorInConnecting)
				if cascadeDelete {
					return deleteResponse, nil
				}
			}
			impl.logger.Debugw("acd app is already deleted for this pipeline", "pipeline", pipeline)
			if _, err := impl.argoClientWrapperService.DeleteArgoApp(ctx, deploymentAppName, cascadeDelete); err != nil {
				impl.logger.Errorw("err in deleting pipeline on argocd", "id", pipeline, "err", err)

				if forceDelete {
					impl.logger.Warnw("error while deletion of app in acd, continue to delete in db as this operation is force delete", "error", err)
				} else {
					//statusError, _ := err.(*errors2.StatusError)
					if cascadeDelete && strings.Contains(err.Error(), "code = NotFound") {
						err = &util.ApiError{
							UserMessage:     "Could not delete as application not found in argocd",
							InternalMessage: err.Error(),
						}
					} else {
						err = &util.ApiError{
							UserMessage:     "Could not delete application",
							InternalMessage: err.Error(),
						}
					}
					return deleteResponse, err
				}
			}
			impl.logger.Infow("app deleted from argocd", "id", pipeline.Id, "pipelineName", pipeline.Name, "app", deploymentAppName)
			err = impl.pipelineRepository.MarkPartiallyDeleted(pipeline.Id, userId, tx)
			if err != nil {
				impl.logger.Errorw("error in partially delete cd pipeline", "err", err)
				return deleteResponse, err
			}
		}
		deleteResponse.DeleteInitiated = true
	}
	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("error in committing db transaction", "err", err)
		return deleteResponse, err
	}
	return deleteResponse, nil
}

func (impl *CdPipelineConfigServiceImpl) getStrategiesMapping(dbPipelineIds []int) (map[int][]*chartConfig.PipelineStrategy, error) {
	strategiesMapping := make(map[int][]*chartConfig.PipelineStrategy)
	strategiesByPipelineIds, err := impl.pipelineConfigRepository.GetAllStrategyByPipelineIds(dbPipelineIds)
	if err != nil && !errors2.IsNotFound(err) {
		impl.logger.Errorw("error in fetching strategies by pipelineIds", "PipelineIds", dbPipelineIds, "err", err)
		return strategiesMapping, err
	}
	for _, strategy := range strategiesByPipelineIds {
		strategiesMapping[strategy.PipelineId] = append(strategiesMapping[strategy.PipelineId], strategy)
	}
	return strategiesMapping, nil
}

func (impl *CdPipelineConfigServiceImpl) BulkDeleteCdPipelines(impactedPipelines []*pipelineConfig.Pipeline, ctx context.Context, dryRun bool, deleteAction int, userId int32) []*bean.CdBulkActionResponseDto {
	var respDtos []*bean.CdBulkActionResponseDto
	for _, pipeline := range impactedPipelines {
		respDto := &bean.CdBulkActionResponseDto{
			PipelineName:    pipeline.Name,
			AppName:         pipeline.App.AppName,
			EnvironmentName: pipeline.Environment.Name,
		}
		if !dryRun {
			deleteResponse, err := impl.DeleteCdPipeline(pipeline, ctx, deleteAction, true, userId)
			if err != nil {
				impl.logger.Errorw("error in deleting cd pipeline", "err", err, "pipelineId", pipeline.Id)
				respDto.DeletionResult = fmt.Sprintf("Not able to delete pipeline, %v", err)
			} else if !(deleteResponse.DeleteInitiated || deleteResponse.ClusterReachable) {
				respDto.DeletionResult = fmt.Sprintf("Not able to delete pipeline, cluster connection error")
			} else {
				respDto.DeletionResult = "Pipeline deleted successfully."
			}
		}
		respDtos = append(respDtos, respDto)
	}
	return respDtos

}
func (impl *CdPipelineConfigServiceImpl) checkIfNsExistsForEnvIds(envIds []*int) error {

	if len(envIds) == 0 {
		return nil
	}
	//fetching environments for the given environment Ids
	environmentList, err := impl.environmentRepository.FindByIds(envIds)
	if err != nil {
		impl.logger.Errorw("error in fetching environment", "err", err)
		return fmt.Errorf("error in fetching environment err:", err)
	}
	clusterIdToNsMap := make(map[int]string, 0)
	for _, environment := range environmentList {

		clusterIdToNsMap[environment.ClusterId] = environment.Namespace
	}
	err = impl.helmAppService.CheckIfNsExistsForClusterIds(clusterIdToNsMap)
	if err != nil {
		return err
	}
	return nil
}
