/*
 * Copyright (c) 2020-2024. Devtron Inc.
 */

package pipeline

import (
	argoApplication "github.com/devtron-labs/devtron/client/argocdServer/bean"
	"github.com/devtron-labs/devtron/enterprise/pkg/deploymentWindow"
	"github.com/devtron-labs/devtron/internal/sql/repository/appWorkflow"
	"github.com/devtron-labs/devtron/internal/util"
	read3 "github.com/devtron-labs/devtron/pkg/appWorkflow/read"
	bean3 "github.com/devtron-labs/devtron/pkg/auth/user/bean"
	repository4 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	pipelineBean "github.com/devtron-labs/devtron/pkg/pipeline/bean"
	constants2 "github.com/devtron-labs/devtron/pkg/pipeline/constants"
	"github.com/devtron-labs/devtron/pkg/policyGovernance/artifactApproval/read"
	"github.com/devtron-labs/devtron/pkg/policyGovernance/artifactPromotion/constants"
	read2 "github.com/devtron-labs/devtron/pkg/policyGovernance/artifactPromotion/read"
	"github.com/devtron-labs/devtron/pkg/team"
	util2 "github.com/devtron-labs/devtron/util"
	"github.com/devtron-labs/devtron/util/rbac"
	"golang.org/x/exp/slices"
	"net/http"
	"sort"
	"strings"

	"github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/enterprise/pkg/resourceFilter"
	"github.com/devtron-labs/devtron/internal/sql/repository"
	dockerArtifactStoreRegistry "github.com/devtron-labs/devtron/internal/sql/repository/dockerRegistry"
	repository3 "github.com/devtron-labs/devtron/internal/sql/repository/imageTagging"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	bean2 "github.com/devtron-labs/devtron/pkg/bean"
	repository2 "github.com/devtron-labs/devtron/pkg/pipeline/repository"
	"github.com/devtron-labs/devtron/pkg/pipeline/types"
	"github.com/devtron-labs/devtron/pkg/resourceQualifiers"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type AppArtifactManager interface {
	// RetrieveArtifactsByCDPipeline : RetrieveArtifactsByCDPipeline returns all the artifacts for the cd pipeline (pre / deploy / post)
	RetrieveArtifactsByCDPipeline(pipeline *pipelineConfig.Pipeline, stage bean.WorkflowType, searchString string, count int, isApprovalNode bool) (*bean2.CiArtifactResponse, error)
	// RetrieveArtifactsForAppWorkflows - List all the artifacts available for the given combination of bean.WorkflowComponentsBean
	// and filtered by workflow component filter map (map of "componentType-componentId" and appWorkflowId, ex - {"CIPIPELINE-8": 8})
	// returns - bean2.CiArtifactResponse and error
	RetrieveArtifactsForAppWorkflows(workflowComponents *bean.WorkflowComponentsBean, workflowComponentFilterMap map[string]int) (bean2.CiArtifactResponse, error)

	RetrieveArtifactsByCDPipelineV2(ctx *util2.RequestCtx, pipeline *pipelineConfig.Pipeline, stage bean.WorkflowType, artifactListingFilterOpts *bean.ArtifactsListFilterOptions, isApprovalNode bool) (*bean2.CiArtifactResponse, error)

	FetchApprovalPendingArtifacts(pipeline *pipelineConfig.Pipeline, artifactListingFilterOpts *bean.ArtifactsListFilterOptions) (*bean2.CiArtifactResponse, error)

	// FetchArtifactForRollback :
	FetchArtifactForRollback(cdPipelineId, appId, offset, limit int, searchString string, app *bean2.CreateAppDTO, pipeline *pipelineConfig.Pipeline) (bean2.CiArtifactResponse, error)

	FetchArtifactForRollbackV2(ctx *util2.RequestCtx, cdPipelineId, appId, offset, limit int, searchString string, app *bean2.CreateAppDTO, deploymentPipeline *pipelineConfig.Pipeline) (bean2.CiArtifactResponse, error)

	BuildArtifactsForCdStage(pipelineId int, stageType bean.WorkflowType, ciArtifacts []bean2.CiArtifactBean, artifactMap map[int]int, parent bool, searchString string, limit int, parentCdId int) ([]bean2.CiArtifactBean, map[int]int, int, string, error)

	BuildArtifactsForParentStage(cdPipelineId int, parentId int, parentType bean.WorkflowType, ciArtifacts []bean2.CiArtifactBean, artifactMap map[int]int, searchString string, limit int, parentCdId int) ([]bean2.CiArtifactBean, error)

	FetchMaterialForArtifactPromotion(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest, imagePromoterAuth func(*util2.RequestCtx, []string) map[string]bool) (bean2.CiArtifactResponse, error)
}

type AppArtifactManagerImpl struct {
	logger                           *zap.SugaredLogger
	cdWorkflowRepository             pipelineConfig.CdWorkflowRepository
	userService                      user.UserService
	imageTaggingService              ImageTaggingService
	ciArtifactRepository             repository.CiArtifactRepository
	ciWorkflowRepository             pipelineConfig.CiWorkflowRepository
	pipelineStageService             PipelineStageService
	celService                       resourceFilter.CELEvaluatorService
	resourceFilterService            resourceFilter.ResourceFilterService
	resourceFilterAuditService       resourceFilter.FilterEvaluationAuditService
	config                           *types.CdConfig
	cdPipelineConfigService          CdPipelineConfigService
	dockerArtifactRegistry           dockerArtifactStoreRegistry.DockerArtifactStoreRepository
	CiPipelineRepository             pipelineConfig.CiPipelineRepository
	ciTemplateService                CiTemplateService
	imageTaggingRepository           repository3.ImageTaggingRepository
	artifactApprovalDataReadService  read.ArtifactApprovalDataReadService
	environmentRepository            repository4.EnvironmentRepository
	appWorkflowRepository            appWorkflow.AppWorkflowRepository
	artifactPromotionDataReadService read2.ArtifactPromotionDataReadService
	teamService                      team.TeamService
	appWorkflowDataReadService       read3.AppWorkflowDataReadService
	ciPipelineConfigService          CiPipelineConfigService
	enforcerUtil                     rbac.EnforcerUtil
	ciCdPipelineOrchestrator         CiCdPipelineOrchestrator
}

func NewAppArtifactManagerImpl(
	logger *zap.SugaredLogger,
	cdWorkflowRepository pipelineConfig.CdWorkflowRepository,
	userService user.UserService,
	imageTaggingService ImageTaggingService,
	ciArtifactRepository repository.CiArtifactRepository,
	ciWorkflowRepository pipelineConfig.CiWorkflowRepository,
	celService resourceFilter.CELEvaluatorService,
	resourceFilterService resourceFilter.ResourceFilterService,
	resourceFilterAuditService resourceFilter.FilterEvaluationAuditService,
	pipelineStageService PipelineStageService,
	cdPipelineConfigService CdPipelineConfigService,
	dockerArtifactRegistry dockerArtifactStoreRegistry.DockerArtifactStoreRepository,
	CiPipelineRepository pipelineConfig.CiPipelineRepository,
	ciTemplateService CiTemplateService,
	imageTaggingRepository repository3.ImageTaggingRepository,
	artifactApprovalDataReadService read.ArtifactApprovalDataReadService,
	environmentRepository repository4.EnvironmentRepository,
	appWorkflowRepository appWorkflow.AppWorkflowRepository,
	artifactPromotionDataReadService read2.ArtifactPromotionDataReadService,
	teamService team.TeamService,
	appWorkflowDataReadService read3.AppWorkflowDataReadService,
	ciPipelineConfigService CiPipelineConfigService,
	enforcerUtil rbac.EnforcerUtil,
	ciCdPipelineOrchestrator CiCdPipelineOrchestrator,
) *AppArtifactManagerImpl {
	cdConfig, err := types.GetCdConfig()
	if err != nil {
		return nil
	}
	return &AppArtifactManagerImpl{
		logger:                           logger,
		cdWorkflowRepository:             cdWorkflowRepository,
		userService:                      userService,
		imageTaggingService:              imageTaggingService,
		ciArtifactRepository:             ciArtifactRepository,
		ciWorkflowRepository:             ciWorkflowRepository,
		celService:                       celService,
		resourceFilterService:            resourceFilterService,
		resourceFilterAuditService:       resourceFilterAuditService,
		cdPipelineConfigService:          cdPipelineConfigService,
		pipelineStageService:             pipelineStageService,
		config:                           cdConfig,
		dockerArtifactRegistry:           dockerArtifactRegistry,
		CiPipelineRepository:             CiPipelineRepository,
		ciTemplateService:                ciTemplateService,
		imageTaggingRepository:           imageTaggingRepository,
		artifactApprovalDataReadService:  artifactApprovalDataReadService,
		environmentRepository:            environmentRepository,
		appWorkflowRepository:            appWorkflowRepository,
		artifactPromotionDataReadService: artifactPromotionDataReadService,
		teamService:                      teamService,
		appWorkflowDataReadService:       appWorkflowDataReadService,
		ciPipelineConfigService:          ciPipelineConfigService,
		enforcerUtil:                     enforcerUtil,
		ciCdPipelineOrchestrator:         ciCdPipelineOrchestrator,
	}
}

func (impl *AppArtifactManagerImpl) BuildArtifactsForParentStage(cdPipelineId int, parentId int, parentType bean.WorkflowType, ciArtifacts []bean2.CiArtifactBean, artifactMap map[int]int, searchString string, limit int, parentCdId int) ([]bean2.CiArtifactBean, error) {
	var ciArtifactsFinal []bean2.CiArtifactBean
	var err error
	if parentType == bean.CI_WORKFLOW_TYPE {
		ciArtifactsFinal, err = impl.BuildArtifactsForCIParent(cdPipelineId, parentId, parentType, ciArtifacts, artifactMap, searchString, limit)
	} else if parentType == bean.WEBHOOK_WORKFLOW_TYPE {
		ciArtifactsFinal, err = impl.BuildArtifactsForCIParent(cdPipelineId, parentId, parentType, ciArtifacts, artifactMap, searchString, limit)
	} else {
		// parent type is PRE, POST or DEPLOY type
		ciArtifactsFinal, _, _, _, err = impl.BuildArtifactsForCdStage(parentId, parentType, ciArtifacts, artifactMap, true, searchString, limit, parentCdId)
	}
	return ciArtifactsFinal, err
}

func (impl *AppArtifactManagerImpl) BuildArtifactsForCdStage(pipelineId int, stageType bean.WorkflowType, ciArtifacts []bean2.CiArtifactBean, artifactMap map[int]int, parent bool, searchString string, limit int, parentCdId int) ([]bean2.CiArtifactBean, map[int]int, int, string, error) {
	// getting running artifact id for parent cd
	parentCdRunningArtifactId := 0
	if parentCdId > 0 && parent {
		parentCdWfrList, err := impl.cdWorkflowRepository.FindArtifactByPipelineIdAndRunnerType(parentCdId, bean.CD_WORKFLOW_TYPE_DEPLOY, searchString, 1, nil)
		if err != nil || len(parentCdWfrList) == 0 {
			impl.logger.Errorw("error in getting artifact for parent cd", "parentCdPipelineId", parentCdId)
			return ciArtifacts, artifactMap, 0, "", err
		}
		parentCdRunningArtifactId = parentCdWfrList[0].CdWorkflow.CiArtifact.Id
	}
	// getting wfr for parent and updating artifacts
	parentWfrList, err := impl.cdWorkflowRepository.FindArtifactByPipelineIdAndRunnerType(pipelineId, stageType, searchString, limit, nil)
	if err != nil {
		impl.logger.Errorw("error in getting artifact for deployed items", "cdPipelineId", pipelineId)
		return ciArtifacts, artifactMap, 0, "", err
	}
	deploymentArtifactId := 0
	deploymentArtifactStatus := ""
	for index, wfr := range parentWfrList {
		if !parent && index == 0 {
			deploymentArtifactId = wfr.CdWorkflow.CiArtifact.Id
			deploymentArtifactStatus = wfr.Status
		}
		if wfr.Status == argoApplication.Healthy || wfr.Status == argoApplication.SUCCEEDED {
			lastSuccessfulTriggerOnParent := parent && index == 0
			latest := !parent && index == 0
			runningOnParentCd := parentCdRunningArtifactId == wfr.CdWorkflow.CiArtifact.Id
			if ciArtifactIndex, ok := artifactMap[wfr.CdWorkflow.CiArtifact.Id]; !ok {
				// entry not present, creating new entry
				mInfo, err := bean2.ParseMaterialInfo([]byte(wfr.CdWorkflow.CiArtifact.MaterialInfo), wfr.CdWorkflow.CiArtifact.DataSource)
				if err != nil {
					mInfo = []byte("[]")
					impl.logger.Errorw("Error in parsing artifact material info", "err", err)
				}
				ciArtifact := bean2.CiArtifactBean{
					Id:                            wfr.CdWorkflow.CiArtifact.Id,
					Image:                         wfr.CdWorkflow.CiArtifact.Image,
					ImageDigest:                   wfr.CdWorkflow.CiArtifact.ImageDigest,
					MaterialInfo:                  mInfo,
					LastSuccessfulTriggerOnParent: lastSuccessfulTriggerOnParent,
					Latest:                        latest,
					Scanned:                       wfr.CdWorkflow.CiArtifact.Scanned,
					ScanEnabled:                   wfr.CdWorkflow.CiArtifact.ScanEnabled,
					CiPipelineId:                  wfr.CdWorkflow.CiArtifact.PipelineId,
					CredentialsSourceType:         wfr.CdWorkflow.CiArtifact.CredentialsSourceType,
					CredentialsSourceValue:        wfr.CdWorkflow.CiArtifact.CredentialSourceValue,
				}
				if wfr.DeploymentApprovalRequest != nil {
					ciArtifact.UserApprovalMetadata = wfr.DeploymentApprovalRequest.ConvertToApprovalMetadata()
				}
				if !parent {
					ciArtifact.Deployed = true
					ciArtifact.DeployedTime = formatDate(wfr.StartedOn, bean2.LayoutRFC3339)
				}
				if runningOnParentCd {
					ciArtifact.RunningOnParentCd = runningOnParentCd
				}
				ciArtifacts = append(ciArtifacts, ciArtifact)
				// storing index of ci artifact for using when updating old entry
				artifactMap[wfr.CdWorkflow.CiArtifact.Id] = len(ciArtifacts) - 1
			} else {
				// entry already present, updating running on parent
				if parent {
					ciArtifacts[ciArtifactIndex].LastSuccessfulTriggerOnParent = lastSuccessfulTriggerOnParent
				}
				if runningOnParentCd {
					ciArtifacts[ciArtifactIndex].RunningOnParentCd = runningOnParentCd
				}
			}
		}
	}
	return ciArtifacts, artifactMap, deploymentArtifactId, deploymentArtifactStatus, nil
}

func (impl *AppArtifactManagerImpl) BuildArtifactsForCIParent(cdPipelineId int, parentId int, parentType bean.WorkflowType, ciArtifacts []bean2.CiArtifactBean, artifactMap map[int]int, searchString string, limit int) ([]bean2.CiArtifactBean, error) {
	artifacts, err := impl.ciArtifactRepository.GetArtifactsByCDPipeline(cdPipelineId, limit, parentId, searchString, parentType)
	if err != nil {
		impl.logger.Errorw("error in getting artifacts for ci", "err", err)
		return ciArtifacts, err
	}
	for _, artifact := range artifacts {
		if _, ok := artifactMap[artifact.Id]; !ok {
			mInfo, err := bean2.ParseMaterialInfo([]byte(artifact.MaterialInfo), artifact.DataSource)
			if err != nil {
				mInfo = []byte("[]")
				impl.logger.Errorw("Error in parsing artifact material info", "err", err, "artifact", artifact)
			}
			ciArtifacts = append(ciArtifacts, bean2.CiArtifactBean{
				Id:                     artifact.Id,
				Image:                  artifact.Image,
				ImageDigest:            artifact.ImageDigest,
				MaterialInfo:           mInfo,
				ScanEnabled:            artifact.ScanEnabled,
				Scanned:                artifact.Scanned,
				CiPipelineId:           artifact.PipelineId,
				CredentialsSourceType:  artifact.CredentialsSourceType,
				CredentialsSourceValue: artifact.CredentialSourceValue,
			})
		}
	}
	return ciArtifacts, nil
}

func (impl *AppArtifactManagerImpl) FetchArtifactForRollback(cdPipelineId, appId, offset, limit int, searchString string, app *bean2.CreateAppDTO, deploymentPipeline *pipelineConfig.Pipeline) (bean2.CiArtifactResponse, error) {
	var deployedCiArtifacts []bean2.CiArtifactBean
	var deployedCiArtifactsResponse bean2.CiArtifactResponse
	var pipeline *pipelineConfig.Pipeline

	cdWfrs, err := impl.cdWorkflowRepository.FetchArtifactsByCdPipelineId(cdPipelineId, bean.CD_WORKFLOW_TYPE_DEPLOY, offset, limit, searchString)
	if err != nil {
		impl.logger.Errorw("error in getting artifacts for rollback by cdPipelineId", "err", err, "cdPipelineId", cdPipelineId)
		return deployedCiArtifactsResponse, err
	}
	var ids []int32
	for _, item := range cdWfrs {
		ids = append(ids, item.TriggeredBy)
		if pipeline == nil && item.CdWorkflow != nil {
			pipeline = item.CdWorkflow.Pipeline
		}
	}
	userEmails := make(map[int32]string)
	users, err := impl.userService.GetByIds(ids)
	if err != nil {
		impl.logger.Errorw("unable to fetch users by ids", "err", err, "ids", ids)
	}
	for _, item := range users {
		userEmails[item.Id] = item.EmailId
	}

	imageTagsDataMap, err := impl.imageTaggingService.GetTagsDataMapByAppId(appId)
	if err != nil {
		impl.logger.Errorw("error in getting image tagging data with appId", "err", err, "appId", appId)
		return deployedCiArtifactsResponse, err
	}
	artifactIds := make([]int, 0)

	for _, cdWfr := range cdWfrs {
		ciArtifact := &repository.CiArtifact{}
		if cdWfr.CdWorkflow != nil && cdWfr.CdWorkflow.CiArtifact != nil {
			ciArtifact = cdWfr.CdWorkflow.CiArtifact
		}
		if ciArtifact == nil {
			continue
		}
		mInfo, err := bean2.ParseMaterialInfo([]byte(ciArtifact.MaterialInfo), ciArtifact.DataSource)
		if err != nil {
			mInfo = []byte("[]")
			impl.logger.Errorw("error in parsing ciArtifact material info", "err", err, "ciArtifact", ciArtifact)
		}
		userEmail := userEmails[cdWfr.TriggeredBy]
		deployedCiArtifacts = append(deployedCiArtifacts, bean2.CiArtifactBean{
			Id:           ciArtifact.Id,
			Image:        ciArtifact.Image,
			MaterialInfo: mInfo,
			DeployedTime: formatDate(cdWfr.StartedOn, bean2.LayoutRFC3339),
			WfrId:        cdWfr.Id,
			DeployedBy:   userEmail,
		})
		artifactIds = append(artifactIds, ciArtifact.Id)
	}
	imageCommentsDataMap, err := impl.imageTaggingService.GetImageCommentsDataMapByArtifactIds(artifactIds)
	if err != nil {
		impl.logger.Errorw("error in getting GetImageCommentsDataMapByArtifactIds", "err", err, "appId", appId, "artifactIds", artifactIds)
		return deployedCiArtifactsResponse, err
	}

	scope := resourceQualifiers.Scope{AppId: app.Id, EnvId: deploymentPipeline.EnvironmentId, ClusterId: deploymentPipeline.Environment.ClusterId, ProjectId: app.TeamId, IsProdEnv: deploymentPipeline.Environment.Default}
	impl.logger.Infow("scope for rollback deployment ", "scope", scope)
	filters, err := impl.resourceFilterService.GetFiltersByScope(scope)
	if err != nil {
		impl.logger.Errorw("error in getting resource filters for the pipeline", "pipelineId", pipeline.Id, "err", err)
		return deployedCiArtifactsResponse, err
	}

	for i, _ := range deployedCiArtifacts {
		imageTaggingResp := imageTagsDataMap[deployedCiArtifacts[i].Id]
		if imageTaggingResp != nil {
			deployedCiArtifacts[i].ImageReleaseTags = imageTaggingResp
		}
		if imageCommentResp := imageCommentsDataMap[deployedCiArtifacts[i].Id]; imageCommentResp != nil {
			deployedCiArtifacts[i].ImageComment = imageCommentResp
			releaseTags := make([]string, 0, len(imageTaggingResp))
			for _, imageTag := range imageTaggingResp {
				releaseTags = append(releaseTags, imageTag.TagName)
			}
			materialInfos, err := deployedCiArtifacts[i].GetMaterialInfo()
			if err != nil {
				impl.logger.Errorw("error in getting material info for the given artifact", "artifactId", deployedCiArtifacts[i].Id, "materialInfo", deployedCiArtifacts[i].MaterialInfo, "err", err)
				return deployedCiArtifactsResponse, err
			}
			filterState, _, err := impl.resourceFilterService.CheckForResource(filters, deployedCiArtifacts[i].Image, releaseTags, materialInfos)
			if err != nil {
				return deployedCiArtifactsResponse, err
			}
			deployedCiArtifacts[i].FilterState = filterState
		}
		deployedCiArtifactsResponse.ResourceFilters = filters
	}

	deployedCiArtifactsResponse.CdPipelineId = cdPipelineId
	if deployedCiArtifacts == nil {
		deployedCiArtifacts = []bean2.CiArtifactBean{}
	}
	if pipeline != nil && pipeline.ApprovalNodeConfigured() {
		deployedCiArtifacts, _, err = impl.overrideArtifactsWithUserApprovalData(pipeline, deployedCiArtifacts, false, 0)
		if err != nil {
			return deployedCiArtifactsResponse, err
		}
	}
	deployedCiArtifactsResponse.CiArtifacts = deployedCiArtifacts

	return deployedCiArtifactsResponse, nil
}

func (impl *AppArtifactManagerImpl) FetchArtifactForRollbackV2(ctx *util2.RequestCtx, cdPipelineId, appId, offset, limit int, searchString string, app *bean2.CreateAppDTO, deploymentPipeline *pipelineConfig.Pipeline) (bean2.CiArtifactResponse, error) {
	var deployedCiArtifactsResponse bean2.CiArtifactResponse
	imageTagsDataMap, err := impl.imageTaggingService.GetTagsDataMapByAppId(appId)
	if err != nil {
		impl.logger.Errorw("error in getting image tagging data with appId", "err", err, "appId", appId)
		return deployedCiArtifactsResponse, err
	}

	artifactListingFilterOpts := bean.ArtifactsListFilterOptions{}
	artifactListingFilterOpts.PipelineId = cdPipelineId
	artifactListingFilterOpts.StageType = bean.CD_WORKFLOW_TYPE_DEPLOY
	artifactListingFilterOpts.ApprovalNodeConfigured = deploymentPipeline.ApprovalNodeConfigured()
	artifactListingFilterOpts.SearchString = "%" + searchString + "%"
	artifactListingFilterOpts.Limit = limit
	artifactListingFilterOpts.Offset = offset
	if artifactListingFilterOpts.ApprovalNodeConfigured {
		approvalConfig, err := deploymentPipeline.GetApprovalConfig()
		if err != nil {
			impl.logger.Errorw("failed to unmarshal userApprovalConfig", "err", err, "cdPipelineId", deploymentPipeline.Id, "approvalConfig", approvalConfig)
			return deployedCiArtifactsResponse, err
		}
		artifactListingFilterOpts.ApproversCount = approvalConfig.RequiredCount
		deployedCiArtifactsResponse.UserApprovalConfig = &approvalConfig
	}

	deployedCiArtifacts, artifactIds, totalCount, err := impl.BuildRollbackArtifactsList(artifactListingFilterOpts)
	if err != nil {
		impl.logger.Errorw("error in building ci artifacts for rollback", "err", err, "cdPipelineId", cdPipelineId)
		return deployedCiArtifactsResponse, err
	}

	deployedCiArtifacts, err = impl.setPromotionArtifactMetadata(ctx, deployedCiArtifacts, artifactListingFilterOpts.PipelineId, constants.PROMOTED)
	if err != nil {
		impl.logger.Errorw("error in setting promotion artifact metadata for artifacts", "cdPipelineId", artifactListingFilterOpts.PipelineId, "err", err)
		return deployedCiArtifactsResponse, err
	}

	imageCommentsDataMap, err := impl.imageTaggingService.GetImageCommentsDataMapByArtifactIds(artifactIds)
	if err != nil {
		impl.logger.Errorw("error in getting GetImageCommentsDataMapByArtifactIds", "err", err, "appId", appId, "artifactIds", artifactIds)
		return deployedCiArtifactsResponse, err
	}

	scope := resourceQualifiers.Scope{AppId: app.Id, EnvId: deploymentPipeline.EnvironmentId, ClusterId: deploymentPipeline.Environment.ClusterId, ProjectId: app.TeamId, IsProdEnv: deploymentPipeline.Environment.Default}
	impl.logger.Infow("scope for rollback deployment ", "scope", scope)
	filters, err := impl.resourceFilterService.GetFiltersByScope(scope)
	if err != nil {
		impl.logger.Errorw("error in getting resource filters for the pipeline", "pipelineId", cdPipelineId, "err", err)
		return deployedCiArtifactsResponse, err
	}

	for i, _ := range deployedCiArtifacts {
		imageTaggingResp := imageTagsDataMap[deployedCiArtifacts[i].Id]
		if imageTaggingResp != nil {
			deployedCiArtifacts[i].ImageReleaseTags = imageTaggingResp
		}
		if imageCommentResp := imageCommentsDataMap[deployedCiArtifacts[i].Id]; imageCommentResp != nil {
			deployedCiArtifacts[i].ImageComment = imageCommentResp
		}
		releaseTags := make([]string, 0, len(imageTaggingResp))
		for _, imageTag := range imageTaggingResp {
			releaseTags = append(releaseTags, imageTag.TagName)
		}
		materialInfos, err := deployedCiArtifacts[i].GetMaterialInfo()
		if err != nil {
			impl.logger.Errorw("error in getting material info for the given artifact", "artifactId", deployedCiArtifacts[i].Id, "materialInfo", deployedCiArtifacts[i].MaterialInfo, "err", err)
			return deployedCiArtifactsResponse, err
		}
		filterState, _, err := impl.resourceFilterService.CheckForResource(filters, deployedCiArtifacts[i].Image, releaseTags, materialInfos)
		if err != nil {
			return deployedCiArtifactsResponse, err
		}
		deployedCiArtifacts[i].FilterState = filterState
		var dockerRegistryId string
		if deployedCiArtifacts[i].DataSource == repository.POST_CI || deployedCiArtifacts[i].DataSource == repository.PRE_CD || deployedCiArtifacts[i].DataSource == repository.POST_CD {
			if deployedCiArtifacts[i].CredentialsSourceType == repository.GLOBAL_CONTAINER_REGISTRY {
				dockerRegistryId = deployedCiArtifacts[i].CredentialsSourceValue
			}
		} else if deployedCiArtifacts[i].DataSource == repository.CI_RUNNER {
			ciPipeline, err := impl.CiPipelineRepository.FindByIdIncludingInActive(deployedCiArtifacts[i].CiPipelineId)
			if err != nil {
				impl.logger.Errorw("error in fetching ciPipeline", "ciPipelineId", ciPipeline.Id, "error", err)
				return deployedCiArtifactsResponse, err
			}
			dockerRegistryId = *ciPipeline.CiTemplate.DockerRegistryId
		}
		if len(dockerRegistryId) > 0 {
			dockerArtifact, err := impl.dockerArtifactRegistry.FindOne(dockerRegistryId)
			if err != nil {
				impl.logger.Errorw("error in getting docker registry details", "err", err, "dockerArtifactStoreId", dockerRegistryId)
			}
			deployedCiArtifacts[i].RegistryType = string(dockerArtifact.RegistryType)
			deployedCiArtifacts[i].RegistryName = dockerRegistryId
		}
		deployedCiArtifactsResponse.ResourceFilters = filters
	}

	deployedCiArtifactsResponse.CdPipelineId = cdPipelineId
	if deployedCiArtifacts == nil {
		deployedCiArtifacts = []bean2.CiArtifactBean{}
	}
	deployedCiArtifactsResponse.CiArtifacts = deployedCiArtifacts
	deployedCiArtifactsResponse.TotalCount = totalCount
	deployedCiArtifactsResponse.CanApproverDeploy = impl.config.CanApproverDeploy
	return deployedCiArtifactsResponse, nil
}

func (impl *AppArtifactManagerImpl) BuildRollbackArtifactsList(artifactListingFilterOpts bean.ArtifactsListFilterOptions) ([]bean2.CiArtifactBean, []int, int, error) {
	var deployedCiArtifacts []bean2.CiArtifactBean
	totalCount := 0

	// 1)get current deployed artifact on this pipeline
	latestWf, err := impl.cdWorkflowRepository.FindArtifactByPipelineIdAndRunnerType(artifactListingFilterOpts.PipelineId, artifactListingFilterOpts.StageType, "", 1, []string{argoApplication.Healthy, argoApplication.SUCCEEDED, argoApplication.Progressing})
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting latest workflow by pipelineId", "pipelineId", artifactListingFilterOpts.PipelineId, "currentStageType", artifactListingFilterOpts.StageType)
		return deployedCiArtifacts, nil, totalCount, err
	}
	if len(latestWf) > 0 {
		// we should never show current deployed artifact in rollback API
		artifactListingFilterOpts.ExcludeWfrIds = []int{latestWf[0].Id}
	}

	var ciArtifacts []repository.CiArtifactWithExtraData
	if artifactListingFilterOpts.ApprovalNodeConfigured {
		ciArtifacts, totalCount, err = impl.ciArtifactRepository.FetchApprovedArtifactsForRollback(artifactListingFilterOpts)
	} else {
		ciArtifacts, totalCount, err = impl.ciArtifactRepository.FetchArtifactsByCdPipelineIdV2(artifactListingFilterOpts)
	}

	if err != nil {
		impl.logger.Errorw("error in getting artifacts for rollback by cdPipelineId", "err", err, "cdPipelineId", artifactListingFilterOpts.PipelineId)
		return deployedCiArtifacts, nil, totalCount, err
	}

	var ids []int32
	for _, item := range ciArtifacts {
		ids = append(ids, item.TriggeredBy)
	}

	userEmails := make(map[int32]string)
	users, err := impl.userService.GetByIds(ids)
	if err != nil {
		impl.logger.Errorw("unable to fetch users by ids", "err", err, "ids", ids)
	}
	for _, item := range users {
		userEmails[item.Id] = item.EmailId
	}

	artifactIds := make([]int, 0)

	for _, ciArtifact := range ciArtifacts {
		mInfo, err := bean2.ParseMaterialInfo([]byte(ciArtifact.MaterialInfo), ciArtifact.DataSource)
		if err != nil {
			mInfo = []byte("[]")
			impl.logger.Errorw("error in parsing ciArtifact material info", "err", err, "ciArtifact", ciArtifact)
		}
		userEmail := userEmails[ciArtifact.TriggeredBy]
		deployedCiArtifacts = append(deployedCiArtifacts, bean2.CiArtifactBean{
			Id:                     ciArtifact.Id,
			Image:                  ciArtifact.Image,
			MaterialInfo:           mInfo,
			DeployedTime:           formatDate(ciArtifact.StartedOn, bean2.LayoutRFC3339),
			WfrId:                  ciArtifact.CdWorkflowRunnerId,
			DeployedBy:             userEmail,
			Scanned:                ciArtifact.Scanned,
			ScanEnabled:            ciArtifact.ScanEnabled,
			CiPipelineId:           ciArtifact.PipelineId,
			CredentialsSourceType:  ciArtifact.CredentialsSourceType,
			CredentialsSourceValue: ciArtifact.CredentialSourceValue,
			DataSource:             ciArtifact.DataSource,
		})
		artifactIds = append(artifactIds, ciArtifact.Id)
	}
	return deployedCiArtifacts, artifactIds, totalCount, nil

}

func (impl *AppArtifactManagerImpl) RetrieveArtifactsByCDPipeline(pipeline *pipelineConfig.Pipeline, stage bean.WorkflowType, searchString string, count int, isApprovalNode bool) (*bean2.CiArtifactResponse, error) {

	// retrieve parent details
	parentId, parentType, err := impl.cdPipelineConfigService.RetrieveParentDetails(pipeline.Id)
	if err != nil {
		impl.logger.Errorw("failed to retrieve parent details",
			"cdPipelineId", pipeline.Id,
			"err", err)
		return nil, err
	}

	parentCdId := 0
	if parentType == bean.CD_WORKFLOW_TYPE_POST || (parentType == bean.CD_WORKFLOW_TYPE_DEPLOY && stage != bean.CD_WORKFLOW_TYPE_POST) {
		// parentCdId is being set to store the artifact currently deployed on parent cd (if applicable).
		// Parent component is CD only if parent type is POST/DEPLOY
		parentCdId = parentId
	}

	if stage == bean.CD_WORKFLOW_TYPE_DEPLOY {
		pipelinePreStage, err := impl.pipelineStageService.GetCdStageByCdPipelineIdAndStageType(pipeline.Id, repository2.PIPELINE_STAGE_TYPE_PRE_CD)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in fetching PRE-CD stage by cd pipeline id", "pipelineId", pipeline.Id, "err", err)
			return nil, err
		}
		if (pipelinePreStage != nil && pipelinePreStage.Id != 0) || len(pipeline.PreStageConfig) > 0 {
			// Parent type will be PRE for DEPLOY stage
			parentId = pipeline.Id
			parentType = bean.CD_WORKFLOW_TYPE_PRE
		}
	}
	if stage == bean.CD_WORKFLOW_TYPE_POST {
		// Parent type will be DEPLOY for POST stage
		parentId = pipeline.Id
		parentType = bean.CD_WORKFLOW_TYPE_DEPLOY
	}

	// Build artifacts for cd stages
	var ciArtifacts []bean2.CiArtifactBean
	ciArtifactsResponse := &bean2.CiArtifactResponse{}

	artifactMap := make(map[int]int)
	limit := count

	ciArtifacts, artifactMap, latestWfArtifactId, latestWfArtifactStatus, err := impl.
		BuildArtifactsForCdStage(pipeline.Id, stage, ciArtifacts, artifactMap, false, searchString, limit, parentCdId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting artifacts for child cd stage", "err", err, "stage", stage)
		return nil, err
	}

	ciArtifacts, err = impl.BuildArtifactsForParentStage(pipeline.Id, parentId, parentType, ciArtifacts, artifactMap, searchString, limit, parentCdId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting artifacts for cd", "err", err, "parentStage", parentType, "stage", stage)
		return nil, err
	}

	// sorting ci artifacts on the basis of creation time
	if ciArtifacts != nil {
		sort.SliceStable(ciArtifacts, func(i, j int) bool {
			return ciArtifacts[i].Id > ciArtifacts[j].Id
		})
	}

	artifactIds := make([]int, 0, len(ciArtifacts))
	for _, artifact := range ciArtifacts {
		artifactIds = append(artifactIds, artifact.Id)
	}

	artifacts, err := impl.ciArtifactRepository.GetArtifactParentCiAndWorkflowDetailsByIdsInDesc(artifactIds)
	if err != nil {
		return ciArtifactsResponse, err
	}
	imageTagsDataMap, err := impl.imageTaggingService.GetTagsDataMapByAppId(pipeline.AppId)
	if err != nil {
		impl.logger.Errorw("error in getting image tagging data with appId", "err", err, "appId", pipeline.AppId)
		return ciArtifactsResponse, err
	}

	imageCommentsDataMap, err := impl.imageTaggingService.GetImageCommentsDataMapByArtifactIds(artifactIds)
	if err != nil {
		impl.logger.Errorw("error in getting GetImageCommentsDataMapByArtifactIds", "err", err, "appId", pipeline.AppId, "artifactIds", artifactIds)
		return ciArtifactsResponse, err
	}

	environment := pipeline.Environment
	scope := resourceQualifiers.Scope{AppId: pipeline.AppId, ProjectId: pipeline.App.TeamId, EnvId: pipeline.EnvironmentId, ClusterId: environment.ClusterId, IsProdEnv: environment.Default}
	filters, err := impl.resourceFilterService.GetFiltersByScope(scope)
	if err != nil {
		impl.logger.Errorw("error in getting resource filters for the pipeline", "pipelineId", pipeline.Id, "err", err)
		return ciArtifactsResponse, err
	}

	for i, artifact := range artifacts {
		imageTaggingResp := imageTagsDataMap[ciArtifacts[i].Id]
		if imageTaggingResp != nil {
			ciArtifacts[i].ImageReleaseTags = imageTaggingResp
		}
		if imageCommentResp := imageCommentsDataMap[ciArtifacts[i].Id]; imageCommentResp != nil {
			ciArtifacts[i].ImageComment = imageCommentResp
		}

		releaseTags := make([]string, 0, len(imageTaggingResp))
		for _, imageTag := range imageTaggingResp {
			releaseTags = append(releaseTags, imageTag.TagName)
		}
		materialInfos, err := ciArtifacts[i].GetMaterialInfo()
		if err != nil {
			impl.logger.Errorw("error in getting material info for the given artifact", "artifactId", ciArtifacts[i].Id, "materialInfo", ciArtifacts[i].MaterialInfo, "err", err)
			return ciArtifactsResponse, err
		}
		filterState, _, err := impl.resourceFilterService.CheckForResource(filters, ciArtifacts[i].Image, releaseTags, materialInfos)
		if err != nil {
			return ciArtifactsResponse, err
		}
		ciArtifacts[i].FilterState = filterState

		if artifact.ExternalCiPipelineId != 0 {
			// if external webhook continue
			continue
		}
		var dockerRegistryId string
		if artifact.PipelineId != 0 {
			ciPipeline, err := impl.CiPipelineRepository.FindByIdIncludingInActive(artifact.PipelineId)
			if err != nil {
				impl.logger.Errorw("error in fetching ciPipeline", "ciPipelineId", ciPipeline.Id, "error", err)
				return nil, err
			}
			dockerRegistryId = *ciPipeline.CiTemplate.DockerRegistryId
		} else {
			if artifact.CredentialsSourceType == repository.GLOBAL_CONTAINER_REGISTRY {
				dockerRegistryId = artifact.CredentialSourceValue
			}
		}
		if len(dockerRegistryId) > 0 {
			dockerArtifact, err := impl.dockerArtifactRegistry.FindOne(dockerRegistryId)
			if err != nil {
				impl.logger.Errorw("error in getting docker registry details", "err", err, "dockerArtifactStoreId", dockerRegistryId)
			}
			ciArtifacts[i].RegistryType = string(dockerArtifact.RegistryType)
			ciArtifacts[i].RegistryName = dockerRegistryId
		}
		var ciWorkflow *pipelineConfig.CiWorkflow
		if artifact.ParentCiArtifact != 0 {
			ciWorkflow, err = impl.ciWorkflowRepository.FindLastTriggeredWorkflowGitTriggersByArtifactId(artifact.ParentCiArtifact)
			if err != nil {
				impl.logger.Errorw("error in getting ci_workflow for artifacts", "err", err, "artifact", artifact, "parentStage", parentType, "stage", stage)
				return ciArtifactsResponse, err
			}

		} else {
			ciWorkflow, err = impl.ciWorkflowRepository.FindCiWorkflowGitTriggersById(*artifact.WorkflowId)
			if err != nil {
				impl.logger.Errorw("error in getting ci_workflow for artifacts", "err", err, "artifact", artifact, "parentStage", parentType, "stage", stage)
				return ciArtifactsResponse, err
			}
		}
		ciArtifacts[i].TriggeredBy = ciWorkflow.TriggeredBy
		ciArtifacts[i].CiConfigureSourceType = ciWorkflow.GitTriggers[ciWorkflow.CiPipelineId].CiConfigureSourceType
		ciArtifacts[i].CiConfigureSourceValue = ciWorkflow.GitTriggers[ciWorkflow.CiPipelineId].CiConfigureSourceValue
	}
	ciArtifactsResponse.ResourceFilters = filters
	ciArtifactsResponse.CdPipelineId = pipeline.Id
	ciArtifactsResponse.LatestWfArtifactId = latestWfArtifactId
	ciArtifactsResponse.LatestWfArtifactStatus = latestWfArtifactStatus
	if ciArtifacts == nil {
		ciArtifacts = []bean2.CiArtifactBean{}
	}
	ciArtifactsResponse.CiArtifacts = ciArtifacts

	if pipeline.ApprovalNodeConfigured() && stage == bean.CD_WORKFLOW_TYPE_DEPLOY { // for now, we are checking artifacts for deploy stage only
		ciArtifactsFinal, approvalConfig, err := impl.overrideArtifactsWithUserApprovalData(pipeline, ciArtifactsResponse.CiArtifacts, isApprovalNode, latestWfArtifactId)
		if err != nil {
			return ciArtifactsResponse, err
		}
		ciArtifactsResponse.UserApprovalConfig = &approvalConfig
		ciArtifactsResponse.CiArtifacts = ciArtifactsFinal
	}
	return ciArtifactsResponse, nil
}

func (impl *AppArtifactManagerImpl) RetrieveArtifactsForAppWorkflows(workflowComponents *bean.WorkflowComponentsBean,
	workflowComponentFilterMap map[string]int) (bean2.CiArtifactResponse, error) {
	var artifactEntities []repository.CiArtifact
	var totalCount int
	var err error
	if workflowComponents.Limit == 0 {
		workflowComponents.Limit = 20
	}
	if len(workflowComponents.SearchImageTag) != 0 {
		artifactEntity, err := impl.ciArtifactRepository.GetArtifactByImageTagAndAppId(workflowComponents.SearchImageTag, workflowComponents.AppId)
		if err != nil {
			impl.logger.Errorw("error in fetching artifacts for app workflow", "workflowComponents", workflowComponents, "err", err)
			return bean2.CiArtifactResponse{}, err
		}
		if artifactEntity.Id != 0 {
			artifactEntities = []repository.CiArtifact{artifactEntity}
			totalCount = 1
		}
	} else { //get by wf components
		artifactEntities, totalCount, err = impl.ciArtifactRepository.GetAllArtifactsForWfComponents(
			workflowComponents.ArtifactIds, workflowComponents.CiPipelineIds,
			workflowComponents.ExternalCiPipelineIds, workflowComponents.CdPipelineIds,
			workflowComponents.SearchArtifactTag, workflowComponents.Offset, workflowComponents.Limit)
		if err != nil {
			impl.logger.Errorw("error in fetching artifacts for app workflow", "workflowComponents", workflowComponents, "err", err)
			return bean2.CiArtifactResponse{}, err
		}
	}
	//not setting appWorkflowId because the artifact source may be a ci which is not present in the workflow yet
	//such cases will require us to check for every artifact whether it is fetched because of ci or cd
	//if fetched due to cd then we will take workflowId of cd pipeline else ci
	artifactResponse := bean2.CiArtifactResponse{
		CiArtifacts: bean2.ConvertArtifactEntityToModel(artifactEntities),
		TotalCount:  totalCount,
	}
	pipelineIdToEnvName, hasProdEnv, err := impl.getWfPipelinesMetadata(workflowComponents.CdPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in getting pipelineId to envName mapping", "cdPipelineIds", workflowComponents.CdPipelineIds, "err", err)
		return artifactResponse, err
	}
	if len(artifactResponse.CiArtifacts) > 0 {
		artifactResponse.CiArtifacts, err = impl.setAdditionalDataInArtifacts(artifactResponse.CiArtifacts, workflowComponents.AppId)
		if err != nil {
			impl.logger.Errorw("error in setting additional data in artifacts", "appId", workflowComponents.AppId, "err", err)
			return artifactResponse, err
		}
		deployedEnvironmentsForArtifacts, err := impl.getDeployedEnvironmentsForArtifacts(artifactResponse.CiArtifacts, workflowComponents.CdPipelineIds, pipelineIdToEnvName)
		if err != nil {
			impl.logger.Errorw("error in fetching environments on which artifact is currently deployed", "cdPipelineIds", workflowComponents.CdPipelineIds, "err", err)
			return artifactResponse, err
		}
		artifactResponse.CiArtifacts = bean2.GetArtifactResponseWithDeployedOnEnvironments(artifactResponse.CiArtifacts, deployedEnvironmentsForArtifacts)
		artifactResponse.CiArtifacts, err = impl.setGitTriggerData(artifactResponse.CiArtifacts)
		if err != nil {
			impl.logger.Errorw("error in setting gitTrigger data in fetched artifacts", "appId", workflowComponents.AppId, "err", err)
			return artifactResponse, err
		}
	}
	appTags, err := impl.imageTaggingService.GetUniqueTagsByAppId(workflowComponents.AppId)
	if err != nil {
		impl.logger.Errorw("service err, GetTagsByAppId", "appId", workflowComponents.AppId, "err", err)
		return artifactResponse, err
	}
	artifactResponse.TagsEditable = hasProdEnv
	artifactResponse.AppReleaseTagNames = appTags
	artifactResponse.RequestedUserId = workflowComponents.UserId
	return artifactResponse, err
}

func (impl *AppArtifactManagerImpl) FetchApprovalPendingArtifacts(pipeline *pipelineConfig.Pipeline, artifactListingFilterOpts *bean.ArtifactsListFilterOptions) (*bean2.CiArtifactResponse, error) {
	ciArtifactsResponse := &bean2.CiArtifactResponse{}

	if pipeline.ApprovalNodeConfigured() { // for now, we are checking artifacts for deploy stage only
		approvalConfig, err := pipeline.GetApprovalConfig()
		if err != nil {
			impl.logger.Errorw("failed to unmarshal userApprovalConfig", "err", err, "cdPipelineId", pipeline.Id, "approvalConfig", approvalConfig)
			return ciArtifactsResponse, err
		}
		requiredApprovals := approvalConfig.RequiredCount

		ciArtifacts, totalCount, err := impl.artifactApprovalDataReadService.FetchApprovalPendingArtifacts(pipeline.Id, artifactListingFilterOpts.Limit, artifactListingFilterOpts.Offset, requiredApprovals, artifactListingFilterOpts.SearchString)
		if err != nil {
			impl.logger.Errorw("failed to fetch approval request artifacts", "err", err, "cdPipelineId", pipeline.Id)
			return ciArtifactsResponse, err
		}

		environment := pipeline.Environment
		scope := resourceQualifiers.Scope{AppId: pipeline.AppId, ProjectId: pipeline.App.TeamId, EnvId: pipeline.EnvironmentId, ClusterId: environment.ClusterId, IsProdEnv: environment.Default}
		filters, err := impl.resourceFilterService.GetFiltersByScope(scope)
		if err != nil {
			impl.logger.Errorw("error in getting resource filters for the pipeline", "pipelineId", pipeline.Id, "err", err)
			return ciArtifactsResponse, err
		}

		if ciArtifacts != nil {
			// set userApprovalMetaData starts
			var artifactIds []int
			for _, item := range ciArtifacts {
				artifactIds = append(artifactIds, item.Id)
			}
			userApprovalMetadata, err := impl.artifactApprovalDataReadService.FetchApprovalDataForArtifacts(artifactIds, pipeline.Id, requiredApprovals) // it will fetch all the request data with nil cd_wfr_rnr_id
			if err != nil {
				impl.logger.Errorw("error occurred while fetching approval data for artifacts", "cdPipelineId", pipeline.Id, "artifactIds", artifactIds, "err", err)
				return ciArtifactsResponse, err
			}

			for i, artifact := range ciArtifacts {
				if approvalMetadataForArtifact, ok := userApprovalMetadata[artifact.Id]; ok {
					ciArtifacts[i].UserApprovalMetadata = approvalMetadataForArtifact
				}
			}
			// set userApprovalMetaData ends
			ciArtifacts, err = impl.setAdditionalDataInArtifacts(ciArtifacts, pipeline.AppId, filters...)
			if err != nil {
				impl.logger.Errorw("error in setting additional data in fetched artifacts", "pipelineId", pipeline.Id, "err", err)
				return ciArtifactsResponse, err
			}
		}

		ciArtifactsResponse.CdPipelineId = pipeline.Id
		if ciArtifacts == nil {
			ciArtifacts = []bean2.CiArtifactBean{}
		}
		ciArtifactsResponse.CiArtifacts = ciArtifacts
		ciArtifactsResponse.UserApprovalConfig = &approvalConfig
		ciArtifactsResponse.ResourceFilters = filters
		ciArtifactsResponse.TotalCount = totalCount

	}
	return ciArtifactsResponse, nil
}
func (impl *AppArtifactManagerImpl) overrideArtifactsWithUserApprovalData(pipeline *pipelineConfig.Pipeline, inputArtifacts []bean2.CiArtifactBean, isApprovalNode bool, latestArtifactId int) ([]bean2.CiArtifactBean, pipelineConfig.UserApprovalConfig, error) {
	impl.logger.Infow("approval node configured", "pipelineId", pipeline.Id, "isApproval", isApprovalNode)
	ciArtifactsFinal := make([]bean2.CiArtifactBean, 0, len(inputArtifacts))
	artifactIds := make([]int, 0, len(inputArtifacts))
	cdPipelineId := pipeline.Id
	approvalConfig, err := pipeline.GetApprovalConfig()
	if err != nil {
		impl.logger.Errorw("failed to unmarshal userApprovalConfig", "err", err, "cdPipelineId", cdPipelineId, "approvalConfig", approvalConfig)
		return ciArtifactsFinal, approvalConfig, err
	}

	for _, item := range inputArtifacts {
		artifactIds = append(artifactIds, item.Id)
	}

	var userApprovalMetadata map[int]*pipelineConfig.UserApprovalMetadata
	requiredApprovals := approvalConfig.RequiredCount
	userApprovalMetadata, err = impl.artifactApprovalDataReadService.FetchApprovalDataForArtifacts(artifactIds, cdPipelineId, requiredApprovals) // it will fetch all the request data with nil cd_wfr_rnr_id
	if err != nil {
		impl.logger.Errorw("error occurred while fetching approval data for artifacts", "cdPipelineId", cdPipelineId, "artifactIds", artifactIds, "err", err)
		return ciArtifactsFinal, approvalConfig, err
	}
	for _, artifact := range inputArtifacts {
		approvalRuntimeState := pipelineConfig.InitApprovalState
		approvalMetadataForArtifact, ok := userApprovalMetadata[artifact.Id]
		if ok { // either approved or requested
			approvalRuntimeState = approvalMetadataForArtifact.ApprovalRuntimeState
			artifact.UserApprovalMetadata = approvalMetadataForArtifact
		} else if artifact.Deployed {
			approvalRuntimeState = pipelineConfig.ConsumedApprovalState
		}

		allowed := false
		if isApprovalNode { // return all the artifacts with state in init, requested or consumed
			allowed = approvalRuntimeState == pipelineConfig.InitApprovalState || approvalRuntimeState == pipelineConfig.RequestedApprovalState || approvalRuntimeState == pipelineConfig.ConsumedApprovalState
		} else { // return only approved state artifacts
			allowed = approvalRuntimeState == pipelineConfig.ApprovedApprovalState || artifact.Latest || artifact.Id == latestArtifactId
		}
		if allowed {
			ciArtifactsFinal = append(ciArtifactsFinal, artifact)
		}
	}
	return ciArtifactsFinal, approvalConfig, nil
}

func (impl *AppArtifactManagerImpl) fillAppliedFiltersData(ciArtifactBeans []bean2.CiArtifactBean, pipelineId int, stage bean.WorkflowType) []bean2.CiArtifactBean {
	referenceType := resourceFilter.Pipeline
	referenceId := pipelineId
	if stage != bean.CD_WORKFLOW_TYPE_DEPLOY {
		referenceType = resourceFilter.PipelineStage
		stageType := repository2.PIPELINE_STAGE_TYPE_PRE_CD
		if stage == bean.CD_WORKFLOW_TYPE_POST {
			stageType = repository2.PIPELINE_STAGE_TYPE_POST_CD
		}
		pipelineStage, err := impl.pipelineStageService.GetCdStageByCdPipelineIdAndStageType(pipelineId, stageType)
		if err != nil {
			// not returning error by choice
			impl.logger.Errorw("error in fetching pipeline Stage", "stageType", stageType, "pipelineId", pipelineId, "err", err)
			return ciArtifactBeans
		}
		if pipelineStage != nil {
			referenceId = pipelineStage.Id
		} else { // this may happen if PRE-CD/POST-CD not yet migrated to pipeline_stage table
			if stageType == repository2.PIPELINE_STAGE_TYPE_PRE_CD {
				referenceType = resourceFilter.PrePipelineStageYaml
			} else if stageType == repository2.PIPELINE_STAGE_TYPE_POST_CD {
				referenceType = resourceFilter.PostPipelineStageYaml
			}
		}
	}
	artifactIds := make([]int, 0, len(ciArtifactBeans))
	for _, ciArtifactBean := range ciArtifactBeans {
		// we only want to get evaluated filters for un deployed artifacts
		if !ciArtifactBean.Deployed {
			artifactIds = append(artifactIds, ciArtifactBean.Id)
		}
	}
	if len(artifactIds) > 0 {
		subjectIdVsState, appliedFiltersMap, appliedFiltersTimeStampMap, err := impl.resourceFilterService.GetEvaluatedFiltersForSubjects(resourceFilter.Artifact, artifactIds, referenceId, referenceType)
		if err != nil {
			// not returning error by choice
			impl.logger.Errorw("error in fetching applied filters when this image was born", "stageType", stage, "pipelineId", pipelineId, "err", err)
			return ciArtifactBeans
		}
		for i, ciArtifactBean := range ciArtifactBeans {
			ciArtifactBeans[i].AppliedFilters = appliedFiltersMap[ciArtifactBean.Id]
			ciArtifactBeans[i].AppliedFiltersTimestamp = appliedFiltersTimeStampMap[ciArtifactBean.Id]
			ciArtifactBeans[i].AppliedFiltersState = subjectIdVsState[ciArtifactBean.Id]
		}
	}
	return ciArtifactBeans
}

func (impl *AppArtifactManagerImpl) getDeploymentWindowAuditData(artifactIds []int, pipelineId int, stage bean.WorkflowType) (map[int]deploymentWindow.DeploymentWindowAuditData, error) {

	referenceType := getResourceTypeForWorkflowType(stage)
	filters, err := impl.resourceFilterAuditService.GetLatestByRefAndMultiSubjectAndFilterType(referenceType, pipelineId, resourceFilter.Artifact, artifactIds, resourceFilter.DEPLOYMENT_WINDOW)
	if err != nil {
		return nil, err
	}
	dataMap := make(map[int]deploymentWindow.DeploymentWindowAuditData)
	for _, filter := range filters {
		if !(filter == nil || len(filter.FilterHistoryObjects) == 0) {
			data := deploymentWindow.GetAuditDataFromSerializedValue(filter.FilterHistoryObjects)
			if data.TriggerType != "AUTO" {
				continue
			}
			dataMap[filter.SubjectId] = data
		}
	}
	return dataMap, nil
}

func getResourceTypeForWorkflowType(stage bean.WorkflowType) resourceFilter.ReferenceType {
	var referenceType resourceFilter.ReferenceType
	switch stage {
	case bean.CD_WORKFLOW_TYPE_PRE:
		referenceType = resourceFilter.PreDeploy
	case bean.CD_WORKFLOW_TYPE_POST:
		referenceType = resourceFilter.PostDeploy
	case bean.CD_WORKFLOW_TYPE_DEPLOY:
		referenceType = resourceFilter.Deploy
	}
	return referenceType
}

func (impl *AppArtifactManagerImpl) RetrieveArtifactsByCDPipelineV2(ctx *util2.RequestCtx, pipeline *pipelineConfig.Pipeline, stage bean.WorkflowType, artifactListingFilterOpts *bean.ArtifactsListFilterOptions, isApprovalNode bool) (*bean2.CiArtifactResponse, error) {

	// retrieve parent details
	parentId, parentType, parentCdId, err := impl.cdPipelineConfigService.ExtractParentMetaDataByPipeline(pipeline, stage)
	if err != nil {
		impl.logger.Errorw("error in finding parent meta data for pipeline", "pipelineId", pipeline.Id, "pipelineStage", stage, "err", err)
		return nil, err
	}
	// Build artifacts for cd stages
	var ciArtifacts []bean2.CiArtifactBean
	ciArtifactsResponse := &bean2.CiArtifactResponse{}

	artifactListingFilterOpts.PipelineId = pipeline.Id
	// this will be 0 for external-ci cases, note: do not refer this for external-ci cases
	artifactListingFilterOpts.CiPipelineId = pipeline.CiPipelineId
	artifactListingFilterOpts.ParentId = parentId
	artifactListingFilterOpts.ParentCdId = parentCdId
	artifactListingFilterOpts.ParentStageType = parentType
	artifactListingFilterOpts.StageType = stage
	artifactListingFilterOpts.ApprovalNodeConfigured = pipeline.ApprovalNodeConfigured()
	artifactListingFilterOpts.SearchString = "%" + artifactListingFilterOpts.SearchString + "%"
	artifactListingFilterOpts.UseCdStageQueryV2 = impl.config.UseArtifactListingQueryV2
	if artifactListingFilterOpts.ApprovalNodeConfigured {
		approvalConfig, err := pipeline.GetApprovalConfig()
		if err != nil {
			impl.logger.Errorw("failed to unmarshal userApprovalConfig", "err", err, "cdPipelineId", pipeline.Id, "approvalConfig", approvalConfig)
			return ciArtifactsResponse, err
		}
		artifactListingFilterOpts.ApproversCount = approvalConfig.RequiredCount
		ciArtifactsResponse.UserApprovalConfig = &approvalConfig
	}

	ciArtifactsRefs, latestWfArtifactId, latestWfArtifactStatus, totalCount, err := impl.BuildArtifactsList(artifactListingFilterOpts, isApprovalNode)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting artifacts for child cd stage", "err", err, "stage", stage)
		return nil, err
	}

	for _, ciArtifactsRef := range ciArtifactsRefs {
		ciArtifacts = append(ciArtifacts, *ciArtifactsRef)
	}

	environment := pipeline.Environment
	scope := resourceQualifiers.Scope{AppId: pipeline.AppId, ProjectId: pipeline.App.TeamId, EnvId: pipeline.EnvironmentId, ClusterId: environment.ClusterId, IsProdEnv: environment.Default}
	filters, err := impl.resourceFilterService.GetFiltersByScope(scope)
	if err != nil {
		impl.logger.Errorw("error in getting resource filters for the pipeline", "pipelineId", pipeline.Id, "err", err)
		return ciArtifactsResponse, err
	}
	ciArtifactsResponse.ResourceFilters = filters

	// sorting ci artifacts on the basis of creation time
	if ciArtifacts != nil {
		sort.SliceStable(ciArtifacts, func(i, j int) bool {
			return ciArtifacts[i].Id > ciArtifacts[j].Id
		})
		ciArtifacts, err = impl.setAdditionalDataInArtifacts(ciArtifacts, pipeline.AppId, filters...)
		if err != nil {
			impl.logger.Errorw("error in setting additional data in fetched artifacts", "pipelineId", pipeline.Id, "err", err)
			return ciArtifactsResponse, err
		}

		ciArtifacts, err = impl.setDeploymentWindowMetadata(ciArtifacts, stage, pipeline)
		if err != nil {
			impl.logger.Errorw("error in setting additional data in fetched artifacts", "pipelineId", pipeline.Id, "err", err)
			return ciArtifactsResponse, err
		}

		ciArtifacts, err = impl.setGitTriggerData(ciArtifacts)
		if err != nil {
			impl.logger.Errorw("error in setting gitTrigger data in fetched artifacts", "pipelineId", pipeline.Id, "err", err)
			return ciArtifactsResponse, err
		}
		if !isApprovalNode {
			ciArtifacts = impl.fillAppliedFiltersData(ciArtifacts, pipeline.Id, stage)
		}
	}

	ciArtifacts, err = impl.setPromotionArtifactMetadata(ctx, ciArtifacts, pipeline.Id, constants.PROMOTED)
	if err != nil {
		impl.logger.Errorw("error in setting promotion artifact metadata for given pipeline", "pipelineId", pipeline.Id, "err", err)
		return ciArtifactsResponse, err
	}

	ciArtifactsResponse.CdPipelineId = pipeline.Id
	ciArtifactsResponse.LatestWfArtifactId = latestWfArtifactId
	ciArtifactsResponse.LatestWfArtifactStatus = latestWfArtifactStatus
	if ciArtifacts == nil {
		ciArtifacts = []bean2.CiArtifactBean{}
	}
	ciArtifactsResponse.CiArtifacts = ciArtifacts
	ciArtifactsResponse.TotalCount = totalCount
	ciArtifactsResponse.CanApproverDeploy = impl.config.CanApproverDeploy
	return ciArtifactsResponse, nil
}

func (impl *AppArtifactManagerImpl) setDeploymentWindowMetadata(ciArtifacts []bean2.CiArtifactBean, stage bean.WorkflowType, pipeline *pipelineConfig.Pipeline) ([]bean2.CiArtifactBean, error) {

	artifactIds := make([]int, 0, len(ciArtifacts))
	for _, artifact := range ciArtifacts {
		artifactIds = append(artifactIds, artifact.Id)
	}

	deploymentWindowDataMap, err := impl.getDeploymentWindowAuditData(artifactIds, pipeline.Id, stage)
	if err != nil {
		impl.logger.Errorw("error in getting deployment window audit data for artifacts", "err", err, "artifacts", artifactIds, "pipelineId", pipeline.Id)
		return ciArtifacts, err
	}

	for i, _ := range ciArtifacts {
		if data, ok := deploymentWindowDataMap[ciArtifacts[i].Id]; ok {
			ciArtifacts[i].DeploymentWindowArtifactMetadata = data
			ciArtifacts[i].AppliedFiltersTimestamp = data.TriggeredAt
		}
	}
	return ciArtifacts, nil
}

func (impl *AppArtifactManagerImpl) setDeployedOnEnvironmentsForArtifact(ciArtifacts []bean2.CiArtifactBean, wfMetadata *pipelineBean.PromotionRequestWfMetadata) ([]bean2.CiArtifactBean, error) {
	deployedEnvironmentsForArtifacts, err := impl.getDeployedEnvironmentsForArtifacts(ciArtifacts, wfMetadata.GetCdPipelineIds(), wfMetadata.GetPipelineIdToEnvNameMap())
	if err != nil {
		impl.logger.Errorw("error in fetching environments on which artifact is currently deployed", "cdPipelineIds", wfMetadata.GetCdPipelineIds(), "err", err)
		return ciArtifacts, err
	}
	return bean2.GetArtifactResponseWithDeployedOnEnvironments(ciArtifacts, deployedEnvironmentsForArtifacts), nil
}

func (impl *AppArtifactManagerImpl) getDeployedEnvironmentsForArtifacts(ciArtifacts []bean2.CiArtifactBean, wfCdPipelineIds []int, pipelineIdToEnvName map[int]string) (artifactIdToDeployedEnvMap map[int][]string, err error) {
	deployedArtifactToPipelineIDMapping, err := impl.getLatestArtifactDeployedOnPipelines(wfCdPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in getting latest artifact deployed on pipeline", "wfCdPipelineIds", wfCdPipelineIds, "err", err)
		return artifactIdToDeployedEnvMap, err
	}

	artifactToDeployedOnEnvsMapping := make(map[int][]string)
	for _, artifact := range ciArtifacts {
		if pipelineIds, ok := deployedArtifactToPipelineIDMapping[artifact.Id]; ok {
			artifactToDeployedOnEnvsMapping[artifact.Id] = append(artifactToDeployedOnEnvsMapping[artifact.Id],
				getEnvNamesForPipelineIds(pipelineIdToEnvName, pipelineIds)...)
		}
	}

	return artifactToDeployedOnEnvsMapping, nil
}

func getEnvNamesForPipelineIds(pipelineIdToEnvName map[int]string, pipelineIds []int) (envNames []string) {
	envNames = make([]string, 0, len(pipelineIds))
	for _, pipelineId := range pipelineIds {
		envNames = append(envNames, pipelineIdToEnvName[pipelineId])
	}
	return envNames
}

func (impl *AppArtifactManagerImpl) getWfPipelinesMetadata(wfCdPipelineIds []int) (map[int]string, bool, error) {
	pipelineIdToEnvNameMapping := make(map[int]string)
	pipelines, err := impl.cdPipelineConfigService.FindCdPipelinesByIds(wfCdPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in fetching deployed pipelines by pipelineIds", "pipelineIds", wfCdPipelineIds, "err", err)
		return pipelineIdToEnvNameMapping, false, err
	}
	pipelineIdToEnvName := make(map[int]string, 0)
	hasProdEnv := false
	for _, p := range pipelines {
		pipelineIdToEnvName[p.Id] = p.EnvironmentName
		hasProdEnv = p.IsProdEnv || hasProdEnv
	}
	return pipelineIdToEnvName, hasProdEnv, nil
}

func (impl *AppArtifactManagerImpl) getLatestArtifactDeployedOnPipelines(wfCdPipelineIds []int) (map[int][]int, error) {
	artifactDeployedCDWorkflowInfo, err := impl.cdWorkflowRepository.FindLatestSucceededWfsByCDPipelineIds(wfCdPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in fetching cd workflow info for pipelines on which artifacts is deployed", "cdPipelineIds", wfCdPipelineIds, "err", err)
		return nil, err
	}

	artifactToDeployedPipelineIDMapping := make(map[int][]int)
	for _, workflowMetadata := range artifactDeployedCDWorkflowInfo {
		if slices.Contains(artifactToDeployedPipelineIDMapping[workflowMetadata.CiArtifactId], workflowMetadata.PipelineId) {
			continue
		}
		artifactToDeployedPipelineIDMapping[workflowMetadata.CiArtifactId] =
			append(artifactToDeployedPipelineIDMapping[workflowMetadata.CiArtifactId], workflowMetadata.PipelineId)
	}
	return artifactToDeployedPipelineIDMapping, nil
}

func (impl *AppArtifactManagerImpl) getImageTagDataForArtifacts(ciArtifacts []bean2.CiArtifactBean, appId int) (map[int][]*repository3.ImageTag, map[int]*repository3.ImageComment, error) {
	artifactIds := make([]int, 0, len(ciArtifacts))
	for _, artifact := range ciArtifacts {
		artifactIds = append(artifactIds, artifact.Id)
	}

	imageTagsDataMap, err := impl.imageTaggingService.GetTagsDataMapByAppId(appId)
	if err != nil {
		impl.logger.Errorw("error in getting image tagging data with appId", "err", err, "appId", appId)
		return nil, nil, err
	}

	imageCommentsDataMap, err := impl.imageTaggingService.GetImageCommentsDataMapByArtifactIds(artifactIds)
	if err != nil {
		impl.logger.Errorw("error in getting GetImageCommentsDataMapByArtifactIds", "err", err, "appId", appId, "artifactIds", artifactIds)
		return nil, nil, err
	}
	return imageTagsDataMap, imageCommentsDataMap, nil
}

func (impl *AppArtifactManagerImpl) setAdditionalDataInArtifacts(ciArtifacts []bean2.CiArtifactBean, appId int, filters ...*resourceFilter.FilterMetaDataBean) ([]bean2.CiArtifactBean, error) {
	// TODO Extract out this logic to adapter
	imageTagsDataMap, imageCommentsDataMap, err := impl.getImageTagDataForArtifacts(ciArtifacts, appId)
	if err != nil {
		return nil, err
	}
	for i, _ := range ciArtifacts {
		imageTaggingResp := imageTagsDataMap[ciArtifacts[i].Id]
		if imageTaggingResp != nil {
			ciArtifacts[i].ImageReleaseTags = imageTaggingResp
		}
		if imageCommentResp := imageCommentsDataMap[ciArtifacts[i].Id]; imageCommentResp != nil {
			ciArtifacts[i].ImageComment = imageCommentResp
		}

		if len(filters) > 0 {
			materialInfos, err := ciArtifacts[i].GetMaterialInfo()
			if err != nil {
				impl.logger.Errorw("error in getting material info for the given artifact", "artifactId", ciArtifacts[i].Id, "materialInfo", ciArtifacts[i].MaterialInfo, "err", err)
				return nil, err
			}
			ciArtifacts[i].FilterState = impl.getFilterState(imageTaggingResp, filters, ciArtifacts[i].Image, materialInfos)
		}
		registryType, registryName, err := impl.getDockerRegistryDataForArtifact(ciArtifacts[i])
		if err != nil {
			return nil, err
		}
		ciArtifacts[i].RegistryType = registryType
		ciArtifacts[i].RegistryName = registryName
	}
	return ciArtifacts, nil

}

func (impl *AppArtifactManagerImpl) getDockerRegistryDataForArtifact(ciArtifact bean2.CiArtifactBean) (registryType, registryName string, err error) {
	dockerRegistryId, err := impl.getDockerRegistryId(ciArtifact)
	if err != nil {
		impl.logger.Errorw("error in getting dockerRegistryId", "ciPipelineId", ciArtifact.CiPipelineId, "error", err)
		return registryType, registryName, err
	}
	if len(dockerRegistryId) > 0 {
		dockerArtifact, err := impl.dockerArtifactRegistry.FindOne(dockerRegistryId)
		if err != nil {
			impl.logger.Errorw("error in getting docker registry details", "err", err, "dockerArtifactStoreId", dockerRegistryId)
		}
		if dockerArtifact != nil {
			registryType = string(dockerArtifact.RegistryType)
		}
		registryName = dockerRegistryId
	}
	return registryType, registryName, nil
}

func (impl *AppArtifactManagerImpl) getDockerRegistryId(ciArtifact bean2.CiArtifactBean) (dockerRegistryId string, err error) {
	if ciArtifact.DataSource == repository.POST_CI || ciArtifact.DataSource == repository.PRE_CD || ciArtifact.DataSource == repository.POST_CD {
		if ciArtifact.CredentialsSourceType == repository.GLOBAL_CONTAINER_REGISTRY {
			dockerRegistryId = ciArtifact.CredentialsSourceValue
		}
	} else if ciArtifact.DataSource == repository.CI_RUNNER {
		// need this if the artifact's ciPipeline gets switched, then the previous ci-pipeline will be in deleted state
		ciPipeline, err := impl.CiPipelineRepository.FindByIdIncludingInActive(ciArtifact.CiPipelineId)
		if err != nil {
			impl.logger.Errorw("error in fetching ciPipeline", "ciPipelineId", ciArtifact.CiPipelineId, "error", err)
			return dockerRegistryId, err
		}
		// need this if the artifact's ciPipeline gets switched, then the previous ci-pipeline will be in deleted state
		sourceCiPipeline, err := impl.ciCdPipelineOrchestrator.GetSourceCiPipelineForArtifact(ciPipeline)
		if err != nil {
			impl.logger.Errorw("error in fetching sourceCiPipeline", "ciPipelineId", ciArtifact.CiPipelineId, "error", err)
			return dockerRegistryId, err
		}
		// source ci could be empty where linked cd is derived from a webhook (external ci) linked pipeline
		if sourceCiPipeline != nil && sourceCiPipeline.Id != 0 {
			if !sourceCiPipeline.IsExternal && sourceCiPipeline.IsDockerConfigOverridden {
				ciTemplateBean, err := impl.ciTemplateService.FindTemplateOverrideByCiPipelineId(sourceCiPipeline.Id)
				if err != nil {
					impl.logger.Errorw("error in fetching template override", "sourceCiPipelineId", sourceCiPipeline.Id, "err", err)
					return dockerRegistryId, err
				}
				dockerRegistryId = ciTemplateBean.CiTemplateOverride.DockerRegistryId
			} else {
				dockerRegistryId = *sourceCiPipeline.CiTemplate.DockerRegistryId
			}
		}
	}
	return dockerRegistryId, nil
}

func (impl *AppArtifactManagerImpl) setGitTriggerData(ciArtifacts []bean2.CiArtifactBean) ([]bean2.CiArtifactBean, error) {
	directArtifactIndexes, directWorkflowIds, artifactsWithParentIndexes, parentArtifactIds := make([]int, 0), make([]int, 0), make([]int, 0), make([]int, 0)
	for i, artifact := range ciArtifacts {
		if artifact.ExternalCiPipelineId != 0 {
			// if external webhook continue
			continue
		}
		// linked ci case
		if artifact.ParentCiArtifact != 0 {
			artifactsWithParentIndexes = append(artifactsWithParentIndexes, i)
			parentArtifactIds = append(parentArtifactIds, artifact.ParentCiArtifact)
		} else {
			directArtifactIndexes = append(directArtifactIndexes, i)
			directWorkflowIds = append(directWorkflowIds, artifact.CiWorkflowId)
		}
	}
	ciWorkflowWithArtifacts, err := impl.ciWorkflowRepository.FindLastTriggeredWorkflowGitTriggersByArtifactIds(parentArtifactIds)
	if err != nil {
		impl.logger.Errorw("error in getting ci_workflow for artifacts", "err", err, "parentArtifactIds", parentArtifactIds)
		return ciArtifacts, err
	}

	parentArtifactIdVsCiWorkflowMap := make(map[int]*pipelineConfig.WorkflowWithArtifact)
	for _, ciWorkflow := range ciWorkflowWithArtifacts {
		parentArtifactIdVsCiWorkflowMap[ciWorkflow.CiArtifactId] = ciWorkflow
	}

	for _, index := range directArtifactIndexes {
		ciWorkflow := parentArtifactIdVsCiWorkflowMap[ciArtifacts[index].CiWorkflowId]
		if ciWorkflow != nil {
			ciArtifacts[index].TriggeredBy = ciWorkflow.TriggeredBy
			ciArtifacts[index].CiConfigureSourceType = ciWorkflow.GitTriggers[ciWorkflow.CiPipelineId].CiConfigureSourceType
			ciArtifacts[index].CiConfigureSourceValue = ciWorkflow.GitTriggers[ciWorkflow.CiPipelineId].CiConfigureSourceValue
		}
	}

	ciWorkflows, err := impl.ciWorkflowRepository.FindCiWorkflowGitTriggersByIds(directWorkflowIds)
	if err != nil {
		impl.logger.Errorw("error in getting ci_workflow for artifacts", "err", err, "ciWorkflowIds", directWorkflowIds)
		return ciArtifacts, err
	}
	ciWorkflowMap := make(map[int]*pipelineConfig.CiWorkflow)
	for _, ciWorkflow := range ciWorkflows {
		ciWorkflowMap[ciWorkflow.Id] = ciWorkflow
	}
	for _, index := range directArtifactIndexes {
		ciWorkflow := ciWorkflowMap[ciArtifacts[index].CiWorkflowId]
		if ciWorkflow != nil {
			ciArtifacts[index].TriggeredBy = ciWorkflow.TriggeredBy
			ciArtifacts[index].CiConfigureSourceType = ciWorkflow.GitTriggers[ciWorkflow.CiPipelineId].CiConfigureSourceType
			ciArtifacts[index].CiConfigureSourceValue = ciWorkflow.GitTriggers[ciWorkflow.CiPipelineId].CiConfigureSourceValue
		}
	}
	return ciArtifacts, nil
}

func (impl *AppArtifactManagerImpl) BuildArtifactsList(listingFilterOpts *bean.ArtifactsListFilterOptions, isApprovalNode bool) ([]*bean2.CiArtifactBean, int, string, int, error) {

	var ciArtifacts []*bean2.CiArtifactBean
	totalCount := 0
	// 1)get current deployed artifact on this pipeline
	latestWf, err := impl.cdWorkflowRepository.FindArtifactByPipelineIdAndRunnerType(listingFilterOpts.PipelineId, listingFilterOpts.StageType, "", 1, []string{argoApplication.Healthy, argoApplication.SUCCEEDED, argoApplication.Progressing})
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting latest workflow by pipelineId", "pipelineId", listingFilterOpts.PipelineId, "currentStageType", listingFilterOpts.StageType, "err", err)
		return ciArtifacts, 0, "", totalCount, err
	}

	var currentRunningArtifactBean *bean2.CiArtifactBean
	currentRunningArtifactId := 0
	currentRunningWorkflowStatus := ""

	// no artifacts deployed on this pipeline yet
	if len(latestWf) > 0 {

		currentRunningArtifact := latestWf[0].CdWorkflow.CiArtifact
		if !isApprovalNode {
			listingFilterOpts.ExcludeArtifactIds = []int{currentRunningArtifact.Id}
		}
		currentRunningArtifactId = currentRunningArtifact.Id
		currentRunningWorkflowStatus = latestWf[0].Status
		// current deployed artifact should always be computed, as we have to show it every time
		mInfo, err := bean2.ParseMaterialInfo([]byte(currentRunningArtifact.MaterialInfo), currentRunningArtifact.DataSource)
		if err != nil {
			mInfo = []byte("[]")
			impl.logger.Errorw("Error in parsing artifact material info", "err", err, "artifact", currentRunningArtifact)
		}
		currentRunningArtifactBean = &bean2.CiArtifactBean{
			Id:                     currentRunningArtifact.Id,
			Image:                  currentRunningArtifact.Image,
			ImageDigest:            currentRunningArtifact.ImageDigest,
			MaterialInfo:           mInfo,
			ScanEnabled:            currentRunningArtifact.ScanEnabled,
			Scanned:                currentRunningArtifact.Scanned,
			Deployed:               true,
			DeployedTime:           formatDate(latestWf[0].CdWorkflow.CreatedOn, bean2.LayoutRFC3339),
			Latest:                 true,
			CreatedTime:            formatDate(currentRunningArtifact.CreatedOn, bean2.LayoutRFC3339),
			DataSource:             currentRunningArtifact.DataSource,
			CiPipelineId:           currentRunningArtifact.PipelineId,
			CredentialsSourceType:  currentRunningArtifact.CredentialsSourceType,
			CredentialsSourceValue: currentRunningArtifact.CredentialSourceValue,
		}
		if currentRunningArtifact.WorkflowId != nil {
			currentRunningArtifactBean.CiWorkflowId = *currentRunningArtifact.WorkflowId
		}
	}
	// 2) get artifact list limited by filterOptions

	// if approval configured and request is for deploy stage, fetch approved images only
	if listingFilterOpts.ApprovalNodeConfigured && listingFilterOpts.StageType == bean.CD_WORKFLOW_TYPE_DEPLOY && !isApprovalNode { // currently approval node is configured for this deploy stage
		ciArtifacts, totalCount, err = impl.fetchApprovedArtifacts(listingFilterOpts, currentRunningArtifactBean)
		if err != nil {
			impl.logger.Errorw("error in fetching approved artifacts for cd pipeline", "pipelineId", listingFilterOpts.PipelineId, "err", err)
			return ciArtifacts, currentRunningArtifactId, currentRunningWorkflowStatus, totalCount, err
		}
		return ciArtifacts, currentRunningArtifactId, currentRunningWorkflowStatus, totalCount, nil

	} else {

		// if parent pipeline is CI/WEBHOOK, get all the ciArtifacts limited by listingFilterOpts
		if listingFilterOpts.ParentStageType == bean.CI_WORKFLOW_TYPE || listingFilterOpts.ParentStageType == bean.WEBHOOK_WORKFLOW_TYPE {
			ciArtifacts, totalCount, err = impl.buildArtifactsForCIParentV2(listingFilterOpts, isApprovalNode)
			if err != nil {
				impl.logger.Errorw("error in getting ci artifacts for ci/webhook type parent", "pipelineId", listingFilterOpts.PipelineId, "parentPipelineId", listingFilterOpts.ParentId, "parentStageType", listingFilterOpts.ParentStageType, "currentStageType", listingFilterOpts.StageType, "err", err)
				return ciArtifacts, 0, "", totalCount, err
			}
		} else {
			if listingFilterOpts.ParentStageType == pipelineBean.WorkflowTypePre {
				listingFilterOpts.PluginStage = repository.PRE_CD
			} else if listingFilterOpts.ParentStageType == pipelineBean.WorkflowTypePost {
				listingFilterOpts.PluginStage = repository.POST_CD
			}
			// if parent pipeline is PRE_CD/POST_CD/CD, then compute ciArtifacts using listingFilterOpts
			ciArtifacts, totalCount, err = impl.buildArtifactsForCdStageV2(listingFilterOpts, isApprovalNode)
			if err != nil {
				impl.logger.Errorw("error in getting ci artifacts for ci/webhook type parent", "pipelineId", listingFilterOpts.PipelineId, "parentPipelineId", listingFilterOpts.ParentId, "parentStageType", listingFilterOpts.ParentStageType, "currentStageType", listingFilterOpts.StageType, "err", err)
				return ciArtifacts, 0, "", totalCount, err
			}
		}
		artifactIds := make([]int, len(ciArtifacts))
		for i, artifact := range ciArtifacts {
			artifactIds[i] = artifact.Id
		}

		var userApprovalMetadata map[int]*pipelineConfig.UserApprovalMetadata
		if isApprovalNode {
			userApprovalMetadata, err = impl.artifactApprovalDataReadService.FetchApprovalDataForArtifacts(artifactIds, listingFilterOpts.PipelineId, listingFilterOpts.ApproversCount) // it will fetch all the request data with nil cd_wfr_rnr_id
			if err != nil {
				impl.logger.Errorw("error occurred while fetching approval data for artifacts", "cdPipelineId", listingFilterOpts.PipelineId, "artifactIds", artifactIds, "err", err)
				return ciArtifacts, 0, "", totalCount, err
			}
			for i, artifact := range ciArtifacts {
				if currentRunningArtifactBean != nil && artifact.Id == currentRunningArtifactBean.Id {
					ciArtifacts[i].Latest = true
					ciArtifacts[i].Deployed = true
					ciArtifacts[i].DeployedTime = currentRunningArtifactBean.DeployedTime

				}
				if approvalMetadataForArtifact, ok := userApprovalMetadata[artifact.Id]; ok {
					ciArtifacts[i].UserApprovalMetadata = approvalMetadataForArtifact
				}
			}
		}
	}
	// we don't need currently deployed artifact for approvalNode explicitly
	// if no artifact deployed skip adding currentRunningArtifactBean in ciArtifacts arr
	if !isApprovalNode && currentRunningArtifactBean != nil {
		// listingFilterOpts.SearchString is always like %?%
		searchString := listingFilterOpts.SearchString[1 : len(listingFilterOpts.SearchString)-1]
		// just send current deployed in approval configured pipeline or this is eligible in search
		if listingFilterOpts.ApprovalNodeConfigured || strings.Contains(currentRunningArtifactBean.Image, searchString) {
			ciArtifacts = append(ciArtifacts, currentRunningArtifactBean)
			totalCount += 1
		}
	}

	return ciArtifacts, currentRunningArtifactId, currentRunningWorkflowStatus, totalCount, nil
}

func (impl *AppArtifactManagerImpl) setPromotionArtifactMetadata(ctx *util2.RequestCtx, ciArtifacts []bean2.CiArtifactBean, cdPipelineId int, status constants.ArtifactPromotionRequestStatus) ([]bean2.CiArtifactBean, error) {

	artifactIds := util2.GetArrayObject(ciArtifacts, func(artifact bean2.CiArtifactBean) int {
		return artifact.Id
	})

	promotionApprovalArtifactIdToMetadataMap, err := impl.artifactPromotionDataReadService.FetchPromotionApprovalDataForArtifacts(artifactIds, cdPipelineId, status)
	if err != nil {
		impl.logger.Errorw("error in fetching promotion approval metadata for given artifactIds", "err", err)
		return ciArtifacts, err
	}
	for i, artifact := range ciArtifacts {
		if promotionApprovalMetadata, ok := promotionApprovalArtifactIdToMetadataMap[artifact.Id]; ok {
			ciArtifacts[i].PromotionApprovalMetadata = promotionApprovalMetadata
		}
	}
	return ciArtifacts, nil
}

func (impl *AppArtifactManagerImpl) buildArtifactsForCdStageV2(listingFilterOpts *bean.ArtifactsListFilterOptions, isApprovalNode bool) ([]*bean2.CiArtifactBean, int, error) {
	cdArtifacts, totalCount, err := impl.ciArtifactRepository.FindArtifactByListFilter(listingFilterOpts, isApprovalNode)
	if err != nil {
		impl.logger.Errorw("error in fetching cd workflow runners using filter", "filterOptions", listingFilterOpts, "err", err)
		return nil, totalCount, err
	}
	ciArtifacts := make([]*bean2.CiArtifactBean, 0, len(cdArtifacts))

	// get artifact running on parent cd
	artifactRunningOnParentCd := 0
	if listingFilterOpts.ParentCdId > 0 {
		// TODO: check if we can fetch LastSuccessfulTriggerOnParent wfr along with last running wf
		parentCdWfrList, err := impl.cdWorkflowRepository.FindArtifactByPipelineIdAndRunnerType(listingFilterOpts.ParentCdId, bean.CD_WORKFLOW_TYPE_DEPLOY, "", 1, []string{argoApplication.Healthy, argoApplication.SUCCEEDED, argoApplication.Progressing})
		if err != nil {
			impl.logger.Errorw("error in getting artifact for parent cd", "parentCdPipelineId", listingFilterOpts.ParentCdId)
			return ciArtifacts, totalCount, err
		}

		if len(parentCdWfrList) != 0 {
			artifactRunningOnParentCd = parentCdWfrList[0].CdWorkflow.CiArtifact.Id
		}
	}

	for _, artifact := range cdArtifacts {
		mInfo, err := bean2.ParseMaterialInfo([]byte(artifact.MaterialInfo), artifact.DataSource)
		if err != nil {
			mInfo = []byte("[]")
			impl.logger.Errorw("Error in parsing artifact material info", "err", err)
		}
		ciArtifact := &bean2.CiArtifactBean{
			Id:           artifact.Id,
			Image:        artifact.Image,
			ImageDigest:  artifact.ImageDigest,
			MaterialInfo: mInfo,
			// TODO:LastSuccessfulTriggerOnParent
			Scanned:                artifact.Scanned,
			ScanEnabled:            artifact.ScanEnabled,
			RunningOnParentCd:      artifact.Id == artifactRunningOnParentCd,
			ExternalCiPipelineId:   artifact.ExternalCiPipelineId,
			ParentCiArtifact:       artifact.ParentCiArtifact,
			CreatedTime:            formatDate(artifact.CreatedOn, bean2.LayoutRFC3339),
			DataSource:             artifact.DataSource,
			CiPipelineId:           artifact.PipelineId,
			CredentialsSourceType:  artifact.CredentialsSourceType,
			CredentialsSourceValue: artifact.CredentialSourceValue,
			Deployed:               artifact.Deployed,
			DeployedTime:           formatDate(artifact.DeployedTime, bean2.LayoutRFC3339),
		}
		if artifact.WorkflowId != nil {
			ciArtifact.CiWorkflowId = *artifact.WorkflowId
		}
		ciArtifacts = append(ciArtifacts, ciArtifact)
	}

	return ciArtifacts, totalCount, nil
}

func (impl *AppArtifactManagerImpl) buildArtifactsForCIParentV2(listingFilterOpts *bean.ArtifactsListFilterOptions, isApprovalNode bool) ([]*bean2.CiArtifactBean, int, error) {

	artifacts, totalCount, err := impl.ciArtifactRepository.GetArtifactsByCDPipelineV3(listingFilterOpts, isApprovalNode)
	if err != nil {
		impl.logger.Errorw("error in getting artifacts for ci", "err", err)
		return nil, totalCount, err
	}

	ciArtifacts := make([]*bean2.CiArtifactBean, 0, len(artifacts))
	for _, artifact := range artifacts {
		mInfo, err := bean2.ParseMaterialInfo([]byte(artifact.MaterialInfo), artifact.DataSource)
		if err != nil {
			mInfo = []byte("[]")
			impl.logger.Errorw("Error in parsing artifact material info", "err", err, "artifact", artifact)
		}
		ciArtifact := &bean2.CiArtifactBean{
			Id:                     artifact.Id,
			Image:                  artifact.Image,
			ImageDigest:            artifact.ImageDigest,
			MaterialInfo:           mInfo,
			ScanEnabled:            artifact.ScanEnabled,
			Scanned:                artifact.Scanned,
			Deployed:               artifact.Deployed,
			DeployedTime:           formatDate(artifact.DeployedTime, bean2.LayoutRFC3339),
			ExternalCiPipelineId:   artifact.ExternalCiPipelineId,
			ParentCiArtifact:       artifact.ParentCiArtifact,
			CreatedTime:            formatDate(artifact.CreatedOn, bean2.LayoutRFC3339),
			DataSource:             artifact.DataSource,
			CiPipelineId:           artifact.PipelineId,
			CredentialsSourceType:  artifact.CredentialsSourceType,
			CredentialsSourceValue: artifact.CredentialSourceValue,
		}
		if artifact.WorkflowId != nil {
			ciArtifact.CiWorkflowId = *artifact.WorkflowId
		}
		ciArtifacts = append(ciArtifacts, ciArtifact)
	}

	return ciArtifacts, totalCount, nil
}

func (impl *AppArtifactManagerImpl) fetchApprovedArtifacts(listingFilterOpts *bean.ArtifactsListFilterOptions, currentRunningArtifactBean *bean2.CiArtifactBean) ([]*bean2.CiArtifactBean, int, error) {
	artifacts, totalCount, err := impl.ciArtifactRepository.FindApprovedArtifactsWithFilter(listingFilterOpts)
	if err != nil {
		impl.logger.Errorw("error in fetching approved image list", "pipelineId", listingFilterOpts.PipelineId, "err", err)
		return nil, totalCount, err
	}
	ciArtifacts := make([]*bean2.CiArtifactBean, 0, len(artifacts))

	// get approval metadata for above ciArtifacts and current running artifact
	// TODO Gireesh: init array with default size and using append is not optimized
	artifactIds := make([]int, 0, len(artifacts)+1)
	for _, item := range artifacts {
		artifactIds = append(artifactIds, item.Id)
	}
	if currentRunningArtifactBean != nil {
		artifactIds = append(artifactIds, currentRunningArtifactBean.Id)
	}

	var userApprovalMetadata map[int]*pipelineConfig.UserApprovalMetadata
	userApprovalMetadata, err = impl.artifactApprovalDataReadService.FetchApprovalDataForArtifacts(artifactIds, listingFilterOpts.PipelineId, listingFilterOpts.ApproversCount) // it will fetch all the request data with nil cd_wfr_rnr_id
	if err != nil {
		impl.logger.Errorw("error occurred while fetching approval data for artifacts", "cdPipelineId", listingFilterOpts.PipelineId, "artifactIds", artifactIds, "err", err)
		return ciArtifacts, totalCount, err
	}

	// TODO Gireesh: this needs refactoring
	for _, artifact := range artifacts {
		mInfo, err := bean2.ParseMaterialInfo([]byte(artifact.MaterialInfo), artifact.DataSource)
		if err != nil {
			mInfo = []byte("[]")
			impl.logger.Errorw("Error in parsing artifact material info", "err", err, "artifact", artifact)
		}
		ciArtifact := &bean2.CiArtifactBean{
			Id:                     artifact.Id,
			Image:                  artifact.Image,
			ImageDigest:            artifact.ImageDigest,
			MaterialInfo:           mInfo,
			ScanEnabled:            artifact.ScanEnabled,
			Scanned:                artifact.Scanned,
			Deployed:               artifact.Deployed,
			DeployedTime:           formatDate(artifact.DeployedTime, bean2.LayoutRFC3339),
			ExternalCiPipelineId:   artifact.ExternalCiPipelineId,
			ParentCiArtifact:       artifact.ParentCiArtifact,
			CreatedTime:            formatDate(artifact.CreatedOn, bean2.LayoutRFC3339),
			CiPipelineId:           artifact.PipelineId,
			DataSource:             artifact.DataSource,
			CredentialsSourceType:  artifact.CredentialsSourceType,
			CredentialsSourceValue: artifact.CredentialSourceValue,
		}
		if artifact.WorkflowId != nil {
			ciArtifact.CiWorkflowId = *artifact.WorkflowId
		}

		if approvalMetadataForArtifact, ok := userApprovalMetadata[artifact.Id]; ok {
			ciArtifact.UserApprovalMetadata = approvalMetadataForArtifact
		}

		ciArtifacts = append(ciArtifacts, ciArtifact)
	}

	if currentRunningArtifactBean != nil {
		if approvalMetadataForArtifact, ok := userApprovalMetadata[currentRunningArtifactBean.Id]; ok {
			currentRunningArtifactBean.UserApprovalMetadata = approvalMetadataForArtifact
		}
		ciArtifacts = append(ciArtifacts, currentRunningArtifactBean)
		totalCount += 1
	}

	return ciArtifacts, totalCount, nil
}

func (impl *AppArtifactManagerImpl) getFilterState(imageTaggingResp []*repository3.ImageTag, filters []*resourceFilter.FilterMetaDataBean, image string, materialInfos []repository.CiMaterialInfo) resourceFilter.FilterState {

	releaseTags := make([]string, 0, len(imageTaggingResp))
	for _, imageTag := range imageTaggingResp {
		if !imageTag.Deleted {
			releaseTags = append(releaseTags, imageTag.TagName)
		}
	}
	filterState, _, err := impl.resourceFilterService.CheckForResource(filters, image, releaseTags, materialInfos)
	if err != nil {
		impl.logger.Errorw("error in evaluating filters for the artifacts", "image", image, "releaseTags", releaseTags)
		// not returning error by choice
	}
	return filterState
}

func (impl *AppArtifactManagerImpl) FetchMaterialForArtifactPromotion(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest, imagePromoterAuth func(*util2.RequestCtx, []string) map[string]bool) (bean2.CiArtifactResponse, error) {

	ciArtifactResponse := bean2.CiArtifactResponse{}

	wfMetadata, err := impl.getWfMetadataForRequest(ctx, request.GetWorkflowId())
	if err != nil {
		impl.logger.Errorw("error in getting wfMetadataForRequest", "workflowId", request.GetWorkflowId(), "err", err)
		return ciArtifactResponse, err
	}

	ciArtifactResponse, err = impl.getPromotionArtifactsForResource(ctx, request, wfMetadata)
	if err != nil {
		impl.logger.Errorw("error in getting ciArtifactResponse", "resource", request.GetResource(), "resourceName", request.GetResourceName(), "err", err)
		return ciArtifactResponse, err
	}

	if len(ciArtifactResponse.CiArtifacts) > 0 {
		ciArtifactResponse.CiArtifacts, err = impl.setAdditionalDataInArtifacts(ciArtifactResponse.CiArtifacts, request.GetAppId())
		if err != nil {
			impl.logger.Errorw("error in setting additional data in artifacts", "appId", request.GetAppId(), "err", err)
			return ciArtifactResponse, err
		}
		ciArtifactResponse.CiArtifacts, err = impl.setDeployedOnEnvironmentsForArtifact(ciArtifactResponse.CiArtifacts, wfMetadata)
		if err != nil {
			impl.logger.Errorw("error in setting environments on which artifact is deployed", "workflowId", request.GetWorkflowId(), "err", err)
			return ciArtifactResponse, err
		}
	}

	appTags, err := impl.imageTaggingService.GetUniqueTagsByAppId(request.GetAppId())
	if err != nil {
		impl.logger.Errorw("service err, GetTagsByAppId", "appId", request.GetAppId(), "err", err)
		return ciArtifactResponse, err
	}

	pipelineIdToRequestMapping, err := impl.artifactPromotionDataReadService.GetPendingRequestMapping(ctx, wfMetadata.GetCdPipelineIds())
	if err != nil {
		impl.logger.Errorw("error in finding deployed artifacts on pipeline", "pipelineIds", wfMetadata.GetCdPipelineIds(), "err", err)
		return bean2.CiArtifactResponse{}, err
	}

	ciArtifactResponse.TagsEditable = wfMetadata.GetHasProdEnv() && request.GetTriggerAccess()
	ciArtifactResponse.AppReleaseTagNames = appTags
	ciArtifactResponse.IsApprovalPendingForPromotion = len(pipelineIdToRequestMapping) > 0
	ciArtifactResponse.RequestedUserId = ctx.GetUserId()

	return ciArtifactResponse, nil
}

func (impl *AppArtifactManagerImpl) getWfMetadataForRequest(ctx *util2.RequestCtx, workflowId int) (*pipelineBean.PromotionRequestWfMetadata, error) {

	cdPipelineIds, _, err := impl.appWorkflowDataReadService.FindCDPipelineIdsAndCdPipelineIdToWfIdMapping([]int{workflowId})
	if err != nil {
		impl.logger.Errorw("error in getting workflow cdPipelineIds and cdPipelineIdToWorkflowIdMapping", "workflowId", workflowId, "err", err)
		return &pipelineBean.PromotionRequestWfMetadata{}, err
	}

	pipelineIdToEnvName, hasProdEnv, err := impl.getWfPipelinesMetadata(cdPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in getting pipelineId to envName mapping", "cdPipelineIds", cdPipelineIds, "err", err)
		return &pipelineBean.PromotionRequestWfMetadata{}, err
	}

	imagePromoterRbacObjects, pipelineIdToRbacObjMapping := impl.enforcerUtil.GetTeamRbacObjectsByPipelineIds(cdPipelineIds)

	authorizedPipelineIds := make([]int, 0, len(cdPipelineIds))

	rbacResults := impl.enforcerUtil.CheckImagePromoterBulkAuth(ctx, imagePromoterRbacObjects)

	for pipelineId, rbacObj := range pipelineIdToRbacObjMapping {
		if authorized := rbacResults[rbacObj]; authorized {
			authorizedPipelineIds = append(authorizedPipelineIds, pipelineId)
		}
	}

	promotionRequestWfMetadata := &pipelineBean.PromotionRequestWfMetadata{}
	promotionRequestWfMetadata = promotionRequestWfMetadata.
		WithCdPipelineIds(cdPipelineIds).
		WithAuthCdPipelineIds(authorizedPipelineIds).
		WithPipelineIdToEnvNameMap(pipelineIdToEnvName).
		WithHasProdEnv(hasProdEnv)

	return promotionRequestWfMetadata, nil

}

func (impl *AppArtifactManagerImpl) getPromotionArtifactsForResource(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest, wfMetadata *pipelineBean.PromotionRequestWfMetadata) (bean2.CiArtifactResponse, error) {

	ciArtifactResponse := bean2.CiArtifactResponse{}
	var err error

	switch request.GetResource() {
	case string(constants.SOURCE_TYPE_CD):

		ciArtifactResponse, err = impl.fetchArtifactsForCDResource(ctx, request)

	case string(constants.SOURCE_TYPE_CI), string(constants.SOURCE_TYPE_LINKED_CI), string(constants.SOURCE_TYPE_LINKED_CD), string(constants.SOURCE_TYPE_JOB_CI):

		ciArtifactResponse, err = impl.fetchArtifactsForCIResource(ctx, request)

	case string(constants.SOURCE_TYPE_WEBHOOK):

		ciArtifactResponse, err = impl.fetchArtifactsForExtCINode(ctx, request)

	case string(constants.PROMOTION_APPROVAL_PENDING_NODE):

		if request.IsPendingForUserRequest() {
			ciArtifactResponse, err = impl.fetchArtifactsPendingForUser(ctx, request, wfMetadata)
		} else {
			ciArtifactResponse, err = impl.fetchArtifactsForPromotionApprovalNode(ctx, request)
		}

	}
	if err != nil {
		impl.logger.Errorw("error in parsing fetch promotion artifact response", "resource", request.GetResource(), "ResourceName", request.GetResourceName(), "err", err)
		return ciArtifactResponse, err
	}

	return ciArtifactResponse, nil
}

func (impl *AppArtifactManagerImpl) fetchArtifactsForCDResource(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest) (bean2.CiArtifactResponse, error) {
	cdPipeline, err := impl.cdPipelineConfigService.GetCdPipelinesByAppAndEnv(request.GetAppId(), 0, request.GetResourceName())
	if err != nil {
		impl.logger.Errorw("error in fetching cd-pipeline by appId and envId", "appId", request.GetAppId(), "environmentId", request.GetResourceName(), "err", err)
		return bean2.CiArtifactResponse{}, util.NewApiError().WithHttpStatusCode(http.StatusUnprocessableEntity).WithUserMessage(constants2.CDPipelineNotFoundErr).WithInternalMessage(constants2.CDPipelineNotFoundErr)
	}

	cdMaterialsRequest := &bean.CdNodeMaterialParams{}
	cdMaterialsRequest = cdMaterialsRequest.WithCDPipelineId(cdPipeline.Pipelines[0].Id).WithListingFilterOptions(request.ListingFilterOptions)

	artifactEntities, totalCount, err := impl.ciArtifactRepository.FindDeployedArtifactsOnPipeline(cdMaterialsRequest)
	if err != nil {
		impl.logger.Errorw("error in fetching artifacts deployed on cdPipeline Node", "cdPipelineId", cdMaterialsRequest.GetCDPipelineId(), "err", err)
		return bean2.CiArtifactResponse{}, err
	}

	artifactResponse := bean2.CiArtifactResponse{
		CiArtifacts: bean2.ConvertArtifactEntityToModel(artifactEntities),
		TotalCount:  totalCount,
	}
	return artifactResponse, nil
}

func (impl *AppArtifactManagerImpl) fetchArtifactsForCIResource(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest) (bean2.CiArtifactResponse, error) {
	ciPipeline, err := impl.ciPipelineConfigService.GetCIPipelineByNameAndAppId(request.GetAppId(), request.GetResourceName())
	if err != nil {
		impl.logger.Errorw("error in fetching ciPipeline by name", "ciPipelineName", request.GetResourceName(), "err", err)
		return bean2.CiArtifactResponse{}, util.NewApiError().WithHttpStatusCode(http.StatusUnprocessableEntity).WithInternalMessage(constants2.CiPipelineNotFoundErr)
	}

	ciNodeRequest := &bean.CiNodeMaterialParams{}
	ciNodeRequest = ciNodeRequest.WithCiPipelineId(ciPipeline.Id).WithListingFilterOptions(request.ListingFilterOptions)

	artifactEntities, totalCount, err := impl.ciArtifactRepository.FindArtifactsByCIPipelineId(ciNodeRequest)
	if err != nil {
		impl.logger.Errorw("error in fetching artifacts deployed on cdPipeline Node", "ciPipelineId", ciNodeRequest.GetCiPipelineId(), "err", err)
		return bean2.CiArtifactResponse{}, err
	}
	artifactResponse := bean2.CiArtifactResponse{
		CiArtifacts: bean2.ConvertArtifactEntityToModel(artifactEntities),
		TotalCount:  totalCount,
	}
	return artifactResponse, nil
}

func (impl *AppArtifactManagerImpl) fetchArtifactsForExtCINode(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest) (bean2.CiArtifactResponse, error) {

	extCiNodeRequest := &bean.ExtCiNodeMaterialParams{}
	extCiNodeRequest = extCiNodeRequest.WithExtCiPipelineId(request.GetResourceId()).WithListingFilterOptions(request.ListingFilterOptions)

	artifactEntities, totalCount, err := impl.ciArtifactRepository.FindArtifactsByExternalCIPipelineId(extCiNodeRequest)
	if err != nil {
		impl.logger.Errorw("error in fetching artifacts deployed on webhook Node", "extCiPipelineId", extCiNodeRequest.GetExtCiPipelineId(), "err", err)
		return bean2.CiArtifactResponse{}, err
	}
	artifactResponse := bean2.CiArtifactResponse{
		CiArtifacts: bean2.ConvertArtifactEntityToModel(artifactEntities),
		TotalCount:  totalCount,
	}
	return artifactResponse, nil
}

func (impl *AppArtifactManagerImpl) fetchArtifactsForPromotionApprovalNode(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest) (bean2.CiArtifactResponse, error) {

	cdPipeline, err := impl.cdPipelineConfigService.FindCdPipelinesByAppAndEnv(request.GetAppId(), 0, request.GetResourceName())
	if err != nil {
		impl.logger.Errorw("error in fetching cd-pipeline by appId and envId", "appId", request.GetAppId(), "environmentId", request.GetResourceName(), "err", err)
		return bean2.CiArtifactResponse{}, err
	}
	if cdPipeline == nil || (cdPipeline != nil && cdPipeline.Id == 0) {
		return bean2.CiArtifactResponse{}, util.NewApiError().WithHttpStatusCode(http.StatusUnprocessableEntity).WithInternalMessage(constants2.CDPipelineNotFoundErr)
	}

	promotionPendingNodeReq := &bean.PromotionPendingNodeMaterialParams{}

	promotionPendingNodeReq = promotionPendingNodeReq.
		WithCDPipelineIds([]int{cdPipeline.Id}).
		WithListingFilterOptions(request.ListingFilterOptions)

	artifactEntities, totalCount, err := impl.ciArtifactRepository.FindArtifactsPendingForPromotion(promotionPendingNodeReq)
	if err != nil {
		impl.logger.Errorw("error in fetching artifacts pending for approval", "cdPipelineIds", promotionPendingNodeReq.GetCDPipelineIds(), "err", err)
		return bean2.CiArtifactResponse{}, err
	}

	imagePromotionApproverEmails, err := impl.getImagePromoterApproverEmails(cdPipeline)
	if err != nil {
		impl.logger.Errorw("error in finding users with image promoter approver access", "pipelineIds", cdPipeline.Id, "err", err)
		return bean2.CiArtifactResponse{}, err
	}

	ciArtifacts := bean2.ConvertArtifactEntityToModel(artifactEntities)
	ciArtifacts, err = impl.setPromotionArtifactMetadata(ctx, ciArtifacts, cdPipeline.Id, constants.AWAITING_APPROVAL)
	if err != nil {
		impl.logger.Errorw("error in fetching promotion approval metadata for artifacts", "cdPipelineIds", cdPipeline.Id, "err", err)
		return bean2.CiArtifactResponse{}, err
	}

	return bean2.CiArtifactResponse{
		CiArtifacts:                  ciArtifacts,
		TotalCount:                   totalCount,
		ImagePromotionApproverEmails: imagePromotionApproverEmails,
	}, nil
}

func (impl *AppArtifactManagerImpl) fetchArtifactsPendingForUser(ctx *util2.RequestCtx, request *bean.PromotionMaterialRequest, wfMetadata *pipelineBean.PromotionRequestWfMetadata) (bean2.CiArtifactResponse, error) {

	imagePromoterAuthCDPipelineIds := wfMetadata.GetAuthCdPipelineIds()

	excludedArtifactIds, err := impl.artifactPromotionDataReadService.GetArtifactsApprovedByUserForPipeline(ctx, imagePromoterAuthCDPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in fetching artifacts pending for current user", "cdPipelineIds", imagePromoterAuthCDPipelineIds, "err", err)
		return bean2.CiArtifactResponse{}, err
	}

	promotionPendingForCurrentUserReq := &bean.PromotionPendingNodeMaterialParams{}
	promotionPendingForCurrentUserReq = promotionPendingForCurrentUserReq.
		WithCDPipelineIds(wfMetadata.GetAuthCdPipelineIds()).
		WithListingFilterOptions(request.ListingFilterOptions).
		WithExcludedArtifactIds(excludedArtifactIds)

	artifactEntities, totalCount, err := impl.ciArtifactRepository.FindArtifactsPendingForPromotion(promotionPendingForCurrentUserReq)
	if err != nil {
		impl.logger.Errorw("error in fetching artifacts pending for approval", "imagePromoterAuthCDPipelineIds", imagePromoterAuthCDPipelineIds, "err", err)
		return bean2.CiArtifactResponse{}, err
	}
	ciArtifacts := bean2.ConvertArtifactEntityToModel(artifactEntities)
	return bean2.CiArtifactResponse{
		CiArtifacts: ciArtifacts,
		TotalCount:  totalCount,
	}, nil
}

func (impl *AppArtifactManagerImpl) getImagePromoterApproverEmails(pipeline *bean2.CDPipelineMinConfig) ([]string, error) {
	teamObj, err := impl.teamService.FetchOne(pipeline.TeamId)
	if err != nil {
		impl.logger.Errorw("error in fetching team by id", "teamId", pipeline.TeamId, "err", err)
		return nil, err
	}
	imagePromotionApproverEmails, err := impl.userService.GetUsersByEnvAndAction(pipeline.AppName, pipeline.EnvironmentIdentifier, teamObj.Name, bean3.ArtifactPromoter)
	if err != nil {
		impl.logger.Errorw("error in finding image promotion approver emails allowed on env", "envName", pipeline.EnvironmentName, "appName", pipeline.AppName, "err", err)
		return nil, err
	}
	return imagePromotionApproverEmails, err
}
