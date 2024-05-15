/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package pipelineConfig

import (
	"context"
	"errors"
	"fmt"
	"github.com/devtron-labs/common-lib/utils/k8s/health"
	"github.com/devtron-labs/devtron/api/bean"
	argoApplication "github.com/devtron-labs/devtron/client/argocdServer/bean"
	"github.com/devtron-labs/devtron/client/gitSensor"
	"github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/sql"
	util4 "github.com/devtron-labs/devtron/util"
	"github.com/go-pg/pg"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"golang.org/x/exp/slices"
	"time"
)

type CdWorkflowRepository interface {
	CheckWorkflowRunnerByReferenceId(referenceId string) (bool, error)
	SaveWorkFlow(ctx context.Context, wf *CdWorkflow) error
	BulkSaveWorkflow(tx *pg.Tx, workflows []*CdWorkflow) error
	UpdateWorkFlow(wf *CdWorkflow) error
	UpdateWorkFlowRunnerWithDeploymentApprovalReqId(wfrId, daRId int) error
	FindById(wfId int) (*CdWorkflow, error)
	FindCdWorkflowMetaByEnvironmentId(appId int, environmentId int, offset int, size int) ([]CdWorkflowRunner, error)
	FindCdWorkflowMetaByPipelineId(pipelineId int, offset int, size int) ([]CdWorkflowRunner, error)
	FindArtifactByPipelineIdAndRunnerType(pipelineId int, runnerType bean.WorkflowType, searchString string, limit int, runnerStatuses []string) ([]CdWorkflowRunner, error)
	SaveWorkFlowRunner(wfr *CdWorkflowRunner) error
	BulkSaveWorkflowRunners(tx *pg.Tx, runners []*CdWorkflowRunner) error
	UpdateWorkFlowRunner(wfr *CdWorkflowRunner) error
	UpdatePreviousQueuedRunnerStatus(cdWfrId, pipelineId int, triggeredBy int32) ([]*CdWorkflowRunner, error)
	UpdateWorkFlowRunnersWithTxn(wfrs []*CdWorkflowRunner, tx *pg.Tx) error
	UpdateWorkFlowRunners(wfr []*CdWorkflowRunner) error
	FindWorkflowRunnerByCdWorkflowId(wfIds []int) ([]*CdWorkflowRunner, error)
	FindWorkflowRunnerByIds(wfrIds []int) ([]*CdWorkflowRunner, error)
	FindWorkflowRunnerByIdsAndStatusesIfPresent(wfrIds []int, statuses map[bean.WorkflowType][]string) ([]*CdWorkflowRunner, error)
	FindBasicWorkflowRunnerWithPipelineIdByIds(wfrIds []int) ([]*CdWorkflowRunner, error)
	FindPreviousCdWfRunnerByStatus(pipelineId int, currentWFRunnerId int, status []string) ([]*CdWorkflowRunner, error)
	FindConfigByPipelineId(pipelineId int) (*CdWorkflowConfig, error)
	FindWorkflowRunnerById(wfrId int) (*CdWorkflowRunner, error)
	FindBasicWorkflowRunnerById(wfrId int) (*CdWorkflowRunner, error)
	FindWorkflowRunnerByIdForApproval(wfrId int) (*CdWorkflowRunner, error)
	FindRetriedWorkflowCountByReferenceId(wfrId int) (int, error)
	FindLatestWfrByAppIdAndEnvironmentId(appId int, environmentId int) (*CdWorkflowRunner, error)
	IsLatestCDWfr(pipelineId, wfrId int) (bool, error)
	FindLatestCdWorkflowRunnerByEnvironmentIdAndRunnerType(appId int, environmentId int, runnerType bean.WorkflowType) (CdWorkflowRunner, error)
	FindAllTriggeredWorkflowCountInLast24Hour() (cdWorkflowCount int, err error)
	GetConnection() *pg.DB

	FindLastPreOrPostTriggeredByPipelineId(pipelineId int) (CdWorkflowRunner, error)
	FindLastPreOrPostTriggeredByEnvironmentId(appId int, environmentId int) (CdWorkflowRunner, error)

	FindByWorkflowIdAndRunnerType(ctx context.Context, wfId int, runnerType bean.WorkflowType) (CdWorkflowRunner, error)
	FindLastStatusByPipelineIdAndRunnerType(pipelineId int, runnerType bean.WorkflowType) (CdWorkflowRunner, error)
	SaveWorkFlows(wfs ...*CdWorkflow) error
	IsLatestWf(pipelineId int, wfId int) (bool, error)
	FindLatestCdWorkflowByPipelineId(pipelineIds []int) (*CdWorkflow, error)
	FindLatestCdWorkflowByPipelineIdV2(pipelineIds []int) ([]*CdWorkflow, error)
	FetchAllCdStagesLatestEntity(pipelineIds []int) ([]*CdWorkflowStatus, error)
	FetchAllCdStagesLatestEntityStatus(wfrIds []int) ([]*CdWorkflowRunner, error)
	ExistsByStatus(status string) (bool, error)

	FetchArtifactsByCdPipelineId(pipelineId int, runnerType bean.WorkflowType, offset, limit int, searchString string) ([]CdWorkflowRunner, error)
	GetLatestTriggersOfHelmPipelinesStuckInNonTerminalStatuses(getPipelineDeployedWithinHours int) ([]*CdWorkflowRunner, error)
	FindLatestRunnerByPipelineIdsAndRunnerType(ctx context.Context, pipelineIds []int, runnerType bean.WorkflowType) ([]CdWorkflowRunner, error)
	IsArtifactDeployedOnStage(ciArtifactId, pipelineId int, runnerType bean.WorkflowType) (bool, error)
	FindLatestSucceededWfsByCDPipelineIds(cdPipelineIds []int) ([]*CdWorkflowMetadata, error)
	GetCdPipelineIdsForRunnerIds(cdWfrIds []int) ([]int, error)
}

type CdWorkflowRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
}

type WorkflowStatus int

const (
	WF_UNKNOWN WorkflowStatus = iota
	REQUEST_ACCEPTED
	ENQUEUED
	QUE_ERROR
	WF_STARTED
	DROPPED_STALE
	DEQUE_ERROR
	TRIGGER_ERROR
)

const (
	WorkflowStarting           = "Starting"
	WorkflowInQueue            = "Queued"
	WorkflowInitiated          = "Initiating"
	WorkflowInProgress         = "Progressing"
	WorkflowAborted            = "Aborted"
	WorkflowFailed             = "Failed"
	WorkflowSucceeded          = "Succeeded"
	WorkflowTimedOut           = "TimedOut"
	WorkflowUnableToFetchState = "UnableToFetch"
	WorkflowTypeDeploy         = "DEPLOY"
	WorkflowTypePre            = "PRE"
	WorkflowTypePost           = "POST"
)

var WfrTerminalStatusList = []string{WorkflowAborted, WorkflowFailed, WorkflowSucceeded, argoApplication.HIBERNATING, string(health.HealthStatusHealthy), string(health.HealthStatusDegraded)}

func (a WorkflowStatus) String() string {
	return [...]string{"WF_UNKNOWN", "REQUEST_ACCEPTED", "ENQUEUED", "QUE_ERROR", "WF_STARTED", "DROPPED_STALE", "DEQUE_ERROR", "TRIGGER_ERROR"}[a]
}

type CdWorkflow struct {
	tableName        struct{}       `sql:"cd_workflow" pg:",discard_unknown_columns"`
	Id               int            `sql:"id,pk"`
	CiArtifactId     int            `sql:"ci_artifact_id"`
	PipelineId       int            `sql:"pipeline_id"`
	WorkflowStatus   WorkflowStatus `sql:"workflow_status,notnull"`
	Pipeline         *Pipeline
	CiArtifact       *repository.CiArtifact
	CdWorkflowRunner []CdWorkflowRunner
	sql.AuditLog
}

type CdWorkflowConfig struct {
	tableName                struct{} `sql:"cd_workflow_config" pg:",discard_unknown_columns"`
	Id                       int      `sql:"id,pk"`
	CdTimeout                int64    `sql:"cd_timeout"`
	MinCpu                   string   `sql:"min_cpu"`
	MaxCpu                   string   `sql:"max_cpu"`
	MinMem                   string   `sql:"min_mem"`
	MaxMem                   string   `sql:"max_mem"`
	MinStorage               string   `sql:"min_storage"`
	MaxStorage               string   `sql:"max_storage"`
	MinEphStorage            string   `sql:"min_eph_storage"`
	MaxEphStorage            string   `sql:"max_eph_storage"`
	CdCacheBucket            string   `sql:"cd_cache_bucket"`
	CdCacheRegion            string   `sql:"cd_cache_region"`
	CdImage                  string   `sql:"cd_image"`
	Namespace                string   `sql:"wf_namespace"`
	CdPipelineId             int      `sql:"cd_pipeline_id"`
	LogsBucket               string   `sql:"logs_bucket"`
	CdArtifactLocationFormat string   `sql:"cd_artifact_location_format"`
}

type WorkflowExecutorType string

const (
	WORKFLOW_EXECUTOR_TYPE_AWF    = "AWF"
	WORKFLOW_EXECUTOR_TYPE_SYSTEM = "SYSTEM"
	NEW_DEPLOYMENT_INITIATED      = "A new deployment was initiated before this deployment completed"
	FOUND_VULNERABILITY           = "Found vulnerability on image"
	GITOPS_REPO_NOT_CONFIGURED    = "GitOps repository is not configured for the app"
)

type CdWorkflowRunnerWithExtraFields struct {
	CdWorkflowRunner
	TotalCount int
}

type CdWorkflowRunner struct {
	tableName                   struct{}             `sql:"cd_workflow_runner" pg:",discard_unknown_columns"`
	Id                          int                  `sql:"id,pk"`
	Name                        string               `sql:"name"`
	WorkflowType                bean.WorkflowType    `sql:"workflow_type"` // pre,post,deploy
	ExecutorType                WorkflowExecutorType `sql:"executor_type"` // awf, system
	Status                      string               `sql:"status"`
	PodStatus                   string               `sql:"pod_status"`
	Message                     string               `sql:"message"`
	StartedOn                   time.Time            `sql:"started_on"`
	FinishedOn                  time.Time            `sql:"finished_on"`
	Namespace                   string               `sql:"namespace"`
	LogLocation                 string               `sql:"log_file_path"`
	TriggeredBy                 int32                `sql:"triggered_by"`
	CdWorkflowId                int                  `sql:"cd_workflow_id"`
	PodName                     string               `sql:"pod_name"`
	BlobStorageEnabled          bool                 `sql:"blob_storage_enabled,notnull"`
	DeploymentApprovalRequestId int                  `sql:"deployment_approval_request_id"`
	RefCdWorkflowRunnerId       int                  `sql:"ref_cd_workflow_runner_id,notnull"`
	HelmReferenceChart          []byte               `sql:""`
	ImagePathReservationIds     []int                `sql:"image_path_reservation_ids" pg:",array,notnull"`
	ReferenceId                 *string              `sql:"reference_id"`
	TriggerMetadata             string               `sql:"-"`

	CdWorkflow                *CdWorkflow
	DeploymentApprovalRequest *DeploymentApprovalRequest
	sql.AuditLog
}

type CdWorkflowMetadata struct {
	Id           int `sql:"id"`
	PipelineId   int `sql:"pipeline_id"`
	CiArtifactId int `sql:"ci_artifact_id"`
}

// TODO: move from here to adapter
func GetTriggerMetricsFromRunnerObj(runner *CdWorkflowRunner) util4.CDMetrics {
	return util4.CDMetrics{
		AppName:         runner.CdWorkflow.Pipeline.DeploymentAppName,
		Status:          runner.Status,
		DeploymentType:  runner.CdWorkflow.Pipeline.DeploymentAppType,
		EnvironmentName: runner.CdWorkflow.Pipeline.Environment.Name,
		Time:            time.Since(runner.StartedOn).Seconds() - time.Since(runner.FinishedOn).Seconds(),
	}
}

func (c *CdWorkflowRunner) IsStatusSucceeded() bool {
	return slices.Contains([]string{WorkflowSucceeded, string(health.HealthStatusHealthy)}, c.Status)
}

func (c *CdWorkflowRunner) IsExternalRun() bool {
	var isExtCluster bool
	if c.WorkflowType == WorkflowTypePre {
		isExtCluster = c.CdWorkflow.Pipeline.RunPreStageInEnv
	} else if c.WorkflowType == WorkflowTypePost {
		isExtCluster = c.CdWorkflow.Pipeline.RunPostStageInEnv
	}
	return isExtCluster
}

type CiPipelineMaterialResponse struct {
	Id              int                    `json:"id"`
	GitMaterialId   int                    `json:"gitMaterialId"`
	GitMaterialUrl  string                 `json:"gitMaterialUrl"`
	GitMaterialName string                 `json:"gitMaterialName"`
	Type            string                 `json:"type"`
	Value           string                 `json:"value"`
	Active          bool                   `json:"active"`
	History         []*gitSensor.GitCommit `json:"history,omitempty"`
	LastFetchTime   time.Time              `json:"lastFetchTime"`
	IsRepoError     bool                   `json:"isRepoError"`
	RepoErrorMsg    string                 `json:"repoErrorMsg"`
	IsBranchError   bool                   `json:"isBranchError"`
	BranchErrorMsg  string                 `json:"branchErrorMsg"`
	Url             string                 `json:"url"`
	Regex           string                 `json:"regex"`
}

type TriggerWorkflowStatus struct {
	CdWorkflowStatus []*CdWorkflowStatus `json:"cdWorkflowStatus"`
	CiWorkflowStatus []*CiWorkflowStatus `json:"ciWorkflowStatus"`
}

type CdWorkflowStatus struct {
	CiPipelineId               int    `json:"ci_pipeline_id"`
	PipelineId                 int    `json:"pipeline_id"`
	PipelineName               string `json:"pipeline_name,omitempty"`
	DeployStatus               string `json:"deploy_status"`
	PreStatus                  string `json:"pre_status"`
	PostStatus                 string `json:"post_status"`
	WorkflowType               string `json:"workflow_type,omitempty"`
	WfrId                      int    `json:"wfr_id,omitempty"`
	DeploymentAppDeleteRequest bool   `json:"deploymentAppDeleteRequest"`
}

type CiWorkflowStatus struct {
	CiPipelineId      int    `json:"ciPipelineId"`
	CiPipelineName    string `json:"ciPipelineName,omitempty"`
	CiStatus          string `json:"ciStatus"`
	StorageConfigured bool   `json:"storageConfigured"`
	CiWorkflowId      int    `json:"ciWorkflowId,omitempty"`
}

type AppDeploymentStatus struct {
	AppId        int    `json:"appId"`
	PipelineId   int    `json:"pipelineId"`
	DeployStatus string `json:"deployStatus"`
	WfrId        int    `json:"wfrId,omitempty"`
}

func NewCdWorkflowRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger) *CdWorkflowRepositoryImpl {
	return &CdWorkflowRepositoryImpl{
		dbConnection: dbConnection,
		logger:       logger,
	}
}

func (impl *CdWorkflowRepositoryImpl) FindPreviousCdWfRunnerByStatus(pipelineId int, currentWFRunnerId int, status []string) ([]*CdWorkflowRunner, error) {
	var runner []*CdWorkflowRunner
	err := impl.dbConnection.
		Model(&runner).
		Column("cd_workflow_runner.*", "CdWorkflow").
		Where("cd_workflow.pipeline_id = ?", pipelineId).
		Where("cd_workflow_runner.id < ?", currentWFRunnerId).
		Where("workflow_type = ? ", bean.CD_WORKFLOW_TYPE_DEPLOY).
		Where("cd_workflow_runner.status not in (?) ", pg.In(status)).
		Order("cd_workflow_runner.id DESC").
		Select()
	return runner, err
}

func (impl *CdWorkflowRepositoryImpl) SaveWorkFlow(ctx context.Context, wf *CdWorkflow) error {
	_, span := otel.Tracer("orchestrator").Start(ctx, "cdWorkflowRepository.SaveWorkFlow")
	defer span.End()
	err := impl.dbConnection.Insert(wf)
	return err
}
func (impl *CdWorkflowRepositoryImpl) SaveWorkFlows(wfs ...*CdWorkflow) error {
	err := impl.dbConnection.Insert(&wfs)
	return err
}

func (impl *CdWorkflowRepositoryImpl) BulkSaveWorkflow(tx *pg.Tx, workflows []*CdWorkflow) error {
	if len(workflows) == 0 {
		//sending nil to handle cases of bulk deployment not blocking other deployments
		return nil
	}
	err := tx.Insert(&workflows)
	return err
}

func (impl *CdWorkflowRepositoryImpl) UpdateWorkFlow(wf *CdWorkflow) error {
	_, err := impl.dbConnection.Model(wf).WherePK().UpdateNotNull()
	return err
}

func (impl *CdWorkflowRepositoryImpl) FindById(wfId int) (*CdWorkflow, error) {
	ddWorkflow := &CdWorkflow{}
	err := impl.dbConnection.Model(ddWorkflow).
		Column("cd_workflow.*, CdWorkflowRunner").Where("id = ?", wfId).Select()
	return ddWorkflow, err
}

func (impl *CdWorkflowRepositoryImpl) FindConfigByPipelineId(pipelineId int) (*CdWorkflowConfig, error) {
	cdWorkflowConfig := &CdWorkflowConfig{}
	err := impl.dbConnection.Model(cdWorkflowConfig).Where("cd_pipeline_id = ?", pipelineId).Select()
	return cdWorkflowConfig, err
}

func (impl *CdWorkflowRepositoryImpl) FindLatestCdWorkflowByPipelineId(pipelineIds []int) (*CdWorkflow, error) {
	cdWorkflow := &CdWorkflow{}
	err := impl.dbConnection.Model(cdWorkflow).Where("pipeline_id in (?)", pg.In(pipelineIds)).Order("id DESC").Limit(1).Select()
	return cdWorkflow, err
}

func (impl *CdWorkflowRepositoryImpl) FindLatestCdWorkflowByPipelineIdV2(pipelineIds []int) ([]*CdWorkflow, error) {
	var cdWorkflow []*CdWorkflow
	// err := impl.dbConnection.Model(&cdWorkflow).Where("pipeline_id in (?)", pg.In(pipelineIds)).Order("id DESC").Select()
	query := "SELECT cdw.pipeline_id, cdw.workflow_status, MAX(id) as id from cd_workflow cdw" +
		" WHERE cdw.pipeline_id in(?)" +
		" GROUP by cdw.pipeline_id, cdw.workflow_status ORDER by id desc;"
	_, err := impl.dbConnection.Query(&cdWorkflow, query, pg.In(pipelineIds))
	if err != nil {
		return cdWorkflow, err
	}
	// TODO - Group By Environment And Pipeline will get latest pipeline from top
	return cdWorkflow, err
}
func (impl *CdWorkflowRepositoryImpl) FindAllTriggeredWorkflowCountInLast24Hour() (cdWorkflowCount int, err error) {
	cnt, err := impl.dbConnection.
		Model(&CdWorkflow{}).
		ColumnExpr("DISTINCT pipeline_id").
		Join("JOIN cd_workflow_runner ON cd_workflow.id = cd_workflow_runner.cd_workflow_id").
		Where("cd_workflow_runner.workflow_type = ? AND cd_workflow_runner.started_on > ?", bean.CD_WORKFLOW_TYPE_DEPLOY, time.Now().AddDate(0, 0, -1)).
		Group("cd_workflow.pipeline_id").
		Count()
	if err != nil {
		impl.logger.Errorw("error occurred while fetching cd workflow", "err", err)
	}
	return cnt, err
}
func (impl *CdWorkflowRepositoryImpl) FindCdWorkflowMetaByEnvironmentId(appId int, environmentId int, offset int, limit int) ([]CdWorkflowRunner, error) {
	var wfrList []CdWorkflowRunner
	err := impl.dbConnection.
		Model(&wfrList).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("p.environment_id = ?", environmentId).
		Where("p.app_id = ?", appId).
		Where("p.deleted = ?", false).
		Order("cd_workflow_runner.id DESC").
		Join("inner join cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		Join("inner join ci_artifact cia on cia.id = wf.ci_artifact_id").
		Join("inner join pipeline p on p.id = wf.pipeline_id").
		// Join("left join users u on u.id = wfr.triggered_by").
		Offset(offset).Limit(limit).
		Select()
	if err != nil {
		return nil, err
	}
	return wfrList, err
}

func (impl *CdWorkflowRepositoryImpl) FindLatestCdWorkflowRunnerByEnvironmentIdAndRunnerType(appId int, environmentId int, runnerType bean.WorkflowType) (CdWorkflowRunner, error) {
	var wfr CdWorkflowRunner
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline").
		Where("p.environment_id = ?", environmentId).
		Where("p.app_id = ?", appId).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Join("inner join cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		Join("inner join pipeline p on p.id = wf.pipeline_id").
		Order("cd_workflow_runner.id DESC").Limit(1).
		Select()
	if err != nil {
		impl.logger.Errorw("error in getting cdWfr by appId, envId and runner type", "appId", appId, "envId", environmentId, "runnerType", runnerType)
		return wfr, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindLatestCdWorkflowRunnerByEnvironmentIdAndRunnerTypeAndStatus(appId int, environmentId int, runnerType bean.WorkflowType) (CdWorkflowRunner, error) {
	var wfr CdWorkflowRunner
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline").
		Where("p.environment_id = ?", environmentId).
		Where("p.app_id = ?", appId).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Where("").
		Join("inner join cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		Join("inner join pipeline p on p.id = wf.pipeline_id").
		Order("cd_workflow_runner.id DESC").Limit(1).
		Select()
	if err != nil {
		impl.logger.Errorw("error in getting cdWfr by appId, envId and runner type", "appId", appId, "envId", environmentId, "runnerType", runnerType)
		return wfr, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) GetConnection() *pg.DB {
	return impl.dbConnection
}

func (impl *CdWorkflowRepositoryImpl) FindCdWorkflowMetaByPipelineId(pipelineId int, offset int, limit int) ([]CdWorkflowRunner, error) {
	var wfrList []CdWorkflowRunner
	err := impl.dbConnection.
		Model(&wfrList).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("cd_workflow.pipeline_id = ?", pipelineId).
		Order("cd_workflow_runner.id DESC").
		// Join("inner join cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		// Join("inner join ci_artifact cia on cia.id = wf.ci_artifact_id").
		// Join("inner join pipeline p on p.id = wf.pipeline_id").
		// Join("left join users u on u.id = wfr.triggered_by").
		// Order("ORDER BY cd_workflow_runner.started_on DESC").
		Offset(offset).Limit(limit).
		Select()

	if err != nil {
		return nil, err
	}
	return wfrList, err
}

func (impl *CdWorkflowRepositoryImpl) FindArtifactByPipelineIdAndRunnerType(pipelineId int, runnerType bean.WorkflowType, searchString string, limit int, runnerStatuses []string) ([]CdWorkflowRunner, error) {
	var wfrList []CdWorkflowRunner
	searchStringFinal := "%" + searchString + "%"
	query := impl.dbConnection.
		Model(&wfrList).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("cd_workflow.pipeline_id = ?", pipelineId).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Where("cd_workflow__ci_artifact.image LIKE ?", searchStringFinal)
	if len(runnerStatuses) > 0 {
		query.Where("cd_workflow_runner.status IN (?)", pg.In(runnerStatuses))
	}
	err := query.
		Order("cd_workflow_runner.id DESC").
		Limit(limit).
		Select()
	if err != nil {
		return nil, err
	}
	return wfrList, err
}

func (impl *CdWorkflowRepositoryImpl) FindLastPreOrPostTriggeredByPipelineId(pipelineId int) (CdWorkflowRunner, error) {
	wfr := CdWorkflowRunner{}
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("cd_workflow.pipeline_id = ?", pipelineId).
		Where("cd_workflow_runner.workflow_type != ?", bean.CD_WORKFLOW_TYPE_DEPLOY).
		Order("cd_workflow_runner.id DESC").
		Limit(1).
		Select()
	if err != nil {
		return wfr, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindLatestWfrByAppIdAndEnvironmentId(appId int, environmentId int) (*CdWorkflowRunner, error) {
	wfr := &CdWorkflowRunner{}
	err := impl.dbConnection.
		Model(wfr).
		Column("cd_workflow_runner.*", "CdWorkflow.Pipeline.id", "CdWorkflow.Pipeline.deployment_app_delete_request", "CdWorkflow.Pipeline.deployment_app_type").
		Where("p.environment_id = ?", environmentId).
		Where("p.app_id = ?", appId).
		Where("cd_workflow_runner.workflow_type = ?", bean.CD_WORKFLOW_TYPE_DEPLOY).
		Order("cd_workflow_runner.id DESC").
		Join("inner join cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		Join("inner join pipeline p on p.id = wf.pipeline_id").
		Limit(1).
		Select()
	if err != nil {
		return wfr, err
	}
	return wfr, nil
}

func (impl *CdWorkflowRepositoryImpl) IsLatestCDWfr(pipelineId, wfrId int) (bool, error) {
	wfr := &CdWorkflowRunner{}
	ifAnySuccessorWfrExists, err := impl.dbConnection.
		Model(wfr).
		Column("cd_workflow_runner.*", "CdWorkflow").
		Where("wf.pipeline_id = ?", pipelineId).
		Where("cd_workflow_runner.workflow_type = ?", bean.CD_WORKFLOW_TYPE_DEPLOY).
		Order("cd_workflow_runner.id DESC").
		Join("inner join cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		Where("cd_workflow_runner.id > ?", wfrId).
		Exists()
	return !ifAnySuccessorWfrExists, err
}

func (impl *CdWorkflowRepositoryImpl) FindLastPreOrPostTriggeredByEnvironmentId(appId int, environmentId int) (CdWorkflowRunner, error) {
	wfr := CdWorkflowRunner{}
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("p.environment_id = ?", environmentId).
		Where("p.app_id = ?", appId).
		Where("cd_workflow_runner.workflow_type != ?", bean.CD_WORKFLOW_TYPE_DEPLOY).
		Order("cd_workflow_runner.id DESC").
		Join("inner join cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		Join("inner join ci_artifact cia on cia.id = wf.ci_artifact_id").
		Join("inner join pipeline p on p.id = wf.pipeline_id").
		Limit(1).
		Select()
	if err != nil {
		return wfr, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) SaveWorkFlowRunner(wfr *CdWorkflowRunner) error {
	err := impl.dbConnection.Insert(wfr)
	return err
}

func (impl *CdWorkflowRepositoryImpl) BulkSaveWorkflowRunners(tx *pg.Tx, runners []*CdWorkflowRunner) error {
	if len(runners) == 0 {
		//sending nil to handle cases of bulk deployment not blocking other deployments
		return nil
	}
	err := tx.Insert(&runners)
	return err
}

func (impl *CdWorkflowRepositoryImpl) UpdateWorkFlowRunner(wfr *CdWorkflowRunner) error {
	wfr.Message = util.GetTruncatedMessage(wfr.Message, 1000)
	err := impl.dbConnection.Update(wfr)
	return err
}

func (impl *CdWorkflowRepositoryImpl) UpdateWorkFlowRunnerWithDeploymentApprovalReqId(wfrId, daRId int) error {
	var wfr CdWorkflowRunner
	_, err := impl.dbConnection.Model(&wfr).
		Set("deployment_approval_request_id = ?", daRId).
		Where("id = ?", wfrId).
		Update()
	return err
}

func (impl *CdWorkflowRepositoryImpl) UpdatePreviousQueuedRunnerStatus(cdWfrId, pipelineId int, triggeredBy int32) ([]*CdWorkflowRunner, error) {
	var wfr []*CdWorkflowRunner
	_, err := impl.dbConnection.Model(&wfr).
		Set("status = ?", WorkflowFailed).
		Set("finished_on = ?", time.Now()).
		Set("updated_on = ?", time.Now()).
		Set("updated_by = ?", triggeredBy).
		Set("message = ?", NEW_DEPLOYMENT_INITIATED).
		Where("workflow_type = ?", bean.CD_WORKFLOW_TYPE_DEPLOY).
		Where("cd_workflow_id in (SELECT id from cd_workflow WHERE pipeline_id = ?)", pipelineId).
		Where("id < ?", cdWfrId).
		Where("status = ?", WorkflowInQueue).
		Update()
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) UpdateWorkFlowRunnersWithTxn(wfrs []*CdWorkflowRunner, tx *pg.Tx) error {
	_, err := tx.Model(&wfrs).Column("finished_on", "message", "status", "updated_on", "updated_by").Update()
	return err
}

func (impl *CdWorkflowRepositoryImpl) UpdateWorkFlowRunners(wfrs []*CdWorkflowRunner) error {
	for _, wfr := range wfrs {
		err := impl.dbConnection.Update(wfr)
		if err != nil {
			impl.logger.Errorw("error in updating wfr", "err", err)
			return err
		}
	}
	return nil
}
func (impl *CdWorkflowRepositoryImpl) FindWorkflowRunnerByCdWorkflowId(wfIds []int) ([]*CdWorkflowRunner, error) {
	var wfr []*CdWorkflowRunner
	err := impl.dbConnection.Model(&wfr).Where("cd_workflow_id in (?)", pg.In(wfIds)).Select()
	if err != nil {
		return nil, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindWorkflowRunnerByIds(wfrIds []int) ([]*CdWorkflowRunner, error) {
	if len(wfrIds) == 0 {
		return nil, pg.ErrNoRows
	}
	var wfr []*CdWorkflowRunner
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline").
		Where("cd_workflow_runner.id in (?)", pg.In(wfrIds)).
		Order("id ASC").
		Select()
	if err != nil {
		return nil, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindWorkflowRunnerByIdsAndStatusesIfPresent(wfrIds []int, statuses map[bean.WorkflowType][]string) ([]*CdWorkflowRunner, error) {
	if len(wfrIds) == 0 {
		return nil, pg.ErrNoRows
	}
	var wfr []*CdWorkflowRunner
	query := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline").
		Where("cd_workflow_runner.id in (?)", pg.In(wfrIds)).
		Order("id ASC")
	//if len(statuses) > 0 {
	//	query = query.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
	//		for workflowType, status := range statuses {
	//			q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
	//				q = q.Where("workflow_type = ? ", workflowType).
	//					Where("status in (?)", pg.In(status))
	//				return q, nil
	//			})
	//		}
	//		return q, nil
	//	})
	//}//TODO: need to evaluate once
	err := query.Select()
	if err != nil {
		return nil, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindBasicWorkflowRunnerWithPipelineIdByIds(wfrIds []int) ([]*CdWorkflowRunner, error) {
	if len(wfrIds) == 0 {
		return nil, pg.ErrNoRows
	}
	var wfr []*CdWorkflowRunner
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.status", "cd_workflow_runner.id", "cd_workflow_runner.workflow_type", "CdWorkflow.pipeline_id").
		Where("cd_workflow_runner.id in (?)", pg.In(wfrIds)).
		Order("id ASC").
		Select()
	if err != nil {
		return nil, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindWorkflowRunnerById(wfrId int) (*CdWorkflowRunner, error) {
	wfr := &CdWorkflowRunner{}
	err := impl.dbConnection.Model(wfr).Column("cd_workflow_runner.*", "CdWorkflow", "DeploymentApprovalRequest", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact", "CdWorkflow.Pipeline.Environment", "CdWorkflow.Pipeline.App").
		Where("cd_workflow_runner.id = ?", wfrId).Select()
	return wfr, err
}
func (impl *CdWorkflowRepositoryImpl) FindWorkflowRunnerByIdForApproval(wfrId int) (*CdWorkflowRunner, error) {
	wfr := &CdWorkflowRunner{}
	err := impl.dbConnection.Model(wfr).Column("cd_workflow_runner.*", "DeploymentApprovalRequest").
		Where("cd_workflow_runner.id = ?", wfrId).Select()
	return wfr, err
}
func (impl *CdWorkflowRepositoryImpl) FindBasicWorkflowRunnerById(wfrId int) (*CdWorkflowRunner, error) {
	wfr := &CdWorkflowRunner{}
	err := impl.dbConnection.Model(wfr).
		Where("cd_workflow_runner.id = ?", wfrId).Select()
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindRetriedWorkflowCountByReferenceId(wfrId int) (int, error) {
	retryCount := 0
	query := fmt.Sprintf("select count(id) "+
		"from cd_workflow_runner where ref_cd_workflow_runner_id = %v", wfrId)

	_, err := impl.dbConnection.Query(&retryCount, query)
	return retryCount, err
}

func (impl *CdWorkflowRepositoryImpl) FindByWorkflowIdAndRunnerType(ctx context.Context, wfId int, runnerType bean.WorkflowType) (CdWorkflowRunner, error) {
	var wfr CdWorkflowRunner
	_, span := otel.Tracer("orchestrator").Start(ctx, "cdWorkflowRepository.FindByWorkflowIdAndRunnerType")
	defer span.End()
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("cd_workflow.id = ?", wfId).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Select()
	if err != nil {
		return wfr, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) FindLastStatusByPipelineIdAndRunnerType(pipelineId int, runnerType bean.WorkflowType) (CdWorkflowRunner, error) {
	wfr := CdWorkflowRunner{}
	err := impl.dbConnection.
		Model(&wfr).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("cd_workflow.pipeline_id = ?", pipelineId).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Order("cd_workflow_runner.id DESC").
		Limit(1).
		Select()
	if err != nil {
		return wfr, err
	}
	return wfr, err
}

func (impl *CdWorkflowRepositoryImpl) IsLatestWf(pipelineId int, wfId int) (bool, error) {
	exists, err := impl.dbConnection.Model(&CdWorkflow{}).
		Where("pipeline_id =?", pipelineId).
		Where("id > ?", wfId).
		Exists()
	return !exists, err
}

func (impl *CdWorkflowRepositoryImpl) FetchAllCdStagesLatestEntity(pipelineIds []int) ([]*CdWorkflowStatus, error) {
	var cdWorkflowStatus []*CdWorkflowStatus
	if len(pipelineIds) == 0 {
		return cdWorkflowStatus, nil
	}
	query := "select p.ci_pipeline_id, wf.pipeline_id, wfr.workflow_type, max(wfr.id) as wfr_id from cd_workflow_runner wfr" +
		" inner join cd_workflow wf on wf.id=wfr.cd_workflow_id" +
		" inner join pipeline p on p.id = wf.pipeline_id" +
		" where wf.pipeline_id in (" + sqlIntSeq(pipelineIds) + ")" +
		" group by p.ci_pipeline_id, wf.pipeline_id, wfr.workflow_type order by wfr_id desc;"
	_, err := impl.dbConnection.Query(&cdWorkflowStatus, query)
	if err != nil {
		impl.logger.Error("err", err)
		return cdWorkflowStatus, err
	}
	return cdWorkflowStatus, nil
}

func (impl *CdWorkflowRepositoryImpl) FetchAllCdStagesLatestEntityStatus(wfrIds []int) ([]*CdWorkflowRunner, error) {
	var wfrList []*CdWorkflowRunner
	err := impl.dbConnection.Model(&wfrList).Column("cd_workflow_runner.*").
		Where("cd_workflow_runner.id in (?)", pg.In(wfrIds)).Select()
	return wfrList, err
}

func (impl *CdWorkflowRepositoryImpl) ExistsByStatus(status string) (bool, error) {
	exists, err := impl.dbConnection.Model(&CdWorkflowRunner{}).
		Where("status =?", status).
		Exists()
	return exists, err
}

func (impl *CdWorkflowRepositoryImpl) FetchArtifactsByCdPipelineId(pipelineId int, runnerType bean.WorkflowType, offset, limit int, searchString string) ([]CdWorkflowRunner, error) {
	var wfrList []CdWorkflowRunner
	searchStringFinal := "%" + searchString + "%"
	err := impl.dbConnection.
		Model(&wfrList).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("cd_workflow.pipeline_id = ?", pipelineId).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Where("cd_workflow__ci_artifact.image LIKE ?", searchStringFinal).
		Order("cd_workflow_runner.id DESC").
		Limit(limit).Offset(offset).
		Select()
	if err != nil {
		impl.logger.Errorw("error in getting Wfrs and ci artifacts by pipelineId", "err", err, "pipelineId", pipelineId)
		return nil, err
	}
	return wfrList, err
}

func (impl *CdWorkflowRepositoryImpl) FetchArtifactsByCdPipelineIdV2(listingFilterOptions bean.ArtifactsListFilterOptions) ([]CdWorkflowRunner, int, error) {
	var wfrList []CdWorkflowRunner
	query := impl.dbConnection.
		Model(&wfrList).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline", "CdWorkflow.CiArtifact").
		Where("cd_workflow.pipeline_id = ?", listingFilterOptions.PipelineId).
		Where("cd_workflow_runner.workflow_type = ?", listingFilterOptions.StageType).
		Where("cd_workflow__ci_artifact.image LIKE ?", listingFilterOptions.SearchString)

	if len(listingFilterOptions.ExcludeArtifactIds) > 0 {
		query = query.Where("cd_workflow__ci_artifact.id NOT IN (?)", pg.In(listingFilterOptions.ExcludeArtifactIds))
	}
	totalCount, err := query.Count()
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting Wfrs count and ci artifacts by pipelineId", "err", err, "pipelineId", listingFilterOptions.PipelineId)
		return nil, totalCount, err
	}

	query = query.Order("cd_workflow_runner.id DESC").
		Limit(listingFilterOptions.Limit).
		Offset(listingFilterOptions.Offset)

	err = query.Select()
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting Wfrs and ci artifacts by pipelineId", "err", err, "pipelineId", listingFilterOptions.PipelineId)
		return nil, totalCount, err
	}
	return wfrList, totalCount, nil
}

func (impl *CdWorkflowRepositoryImpl) GetLatestTriggersOfHelmPipelinesStuckInNonTerminalStatuses(getPipelineDeployedWithinHours int) ([]*CdWorkflowRunner, error) {
	var wfrList []*CdWorkflowRunner
	excludedStatusList := WfrTerminalStatusList
	excludedStatusList = append(excludedStatusList, WorkflowInitiated, WorkflowInQueue, WorkflowStarting)
	err := impl.dbConnection.
		Model(&wfrList).
		Column("cd_workflow_runner.*", "CdWorkflow.id", "CdWorkflow.pipeline_id", "CdWorkflow.Pipeline.id", "CdWorkflow.Pipeline.deployment_app_name", "CdWorkflow.Pipeline.deployment_app_type", "CdWorkflow.Pipeline.deleted", "CdWorkflow.Pipeline.Environment").
		Join("INNER JOIN cd_workflow wf on wf.id = cd_workflow_runner.cd_workflow_id").
		Join("INNER JOIN pipeline p on p.id = wf.pipeline_id").
		Join("INNER JOIN environment e on e.id = p.environment_id").
		Where("cd_workflow_runner.workflow_type=?", bean.CD_WORKFLOW_TYPE_DEPLOY).
		Where("cd_workflow_runner.status not in (?)", pg.In(excludedStatusList)).
		Where("cd_workflow_runner.cd_workflow_id in"+
			" (SELECT max(cd_workflow.id) as id from cd_workflow"+
			" INNER JOIN cd_workflow_runner on cd_workflow.id = cd_workflow_runner.cd_workflow_id"+
			" WHERE cd_workflow_runner.status != ?"+
			" GROUP BY cd_workflow.pipeline_id"+
			" ORDER BY cd_workflow.pipeline_id desc)", WorkflowInQueue).
		Where("p.deployment_app_type = ?", util.PIPELINE_DEPLOYMENT_TYPE_HELM).
		Where("cd_workflow_runner.started_on > NOW() - INTERVAL '? hours'", getPipelineDeployedWithinHours).
		Where("p.deleted=?", false).
		Order("cd_workflow_runner.id DESC").
		Select()
	if err != nil {
		impl.logger.Errorw("error,GetLatestTriggersOfHelmPipelinesStuckInNonTerminalStatuses ", "err", err)
		return nil, err
	}
	return wfrList, err
}

func (impl *CdWorkflowRepositoryImpl) CheckWorkflowRunnerByReferenceId(referenceId string) (bool, error) {
	exists, err := impl.dbConnection.Model((*CdWorkflowRunner)(nil)).
		Where("cd_workflow_runner.reference_id = ?", referenceId).
		Exists()
	if errors.Is(err, pg.ErrNoRows) {
		return false, nil
	}
	return exists, err
}

func (impl *CdWorkflowRepositoryImpl) IsArtifactDeployedOnStage(ciArtifactId, pipelineId int, runnerType bean.WorkflowType) (bool, error) {
	exists, err := impl.dbConnection.Model((*CdWorkflowRunner)(nil)).
		Join("INNER JOIN cd_workflow cdw on cdw.id = cd_workflow_runner.cd_workflow_id").
		Where("cdw.pipeline_id = ?", pipelineId).
		Where("cdw.ci_artifact_id = ?", ciArtifactId).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Where("cd_workflow_runner.status IN (?)", pg.In([]string{"Healthy", "Succeeded"})).
		Exists()
	if errors.Is(err, pg.ErrNoRows) {
		return false, nil
	}
	return exists, err
}

func (impl *CdWorkflowRepositoryImpl) FindLatestSucceededWfsByCDPipelineIds(cdPipelineIds []int) ([]*CdWorkflowMetadata, error) {
	var cdWorkflow []*CdWorkflowMetadata
	if len(cdPipelineIds) == 0 {
		return cdWorkflow, nil
	}
	query := `
			WITH workflow AS (
				SELECT MAX(cw.id) AS cdw_id
				FROM cd_workflow cw
				INNER JOIN cd_workflow_runner cwr ON cw.id = cwr.cd_workflow_id
				WHERE (
					cw.pipeline_id IN (?)
					AND cwr.workflow_type = 'DEPLOY'
					AND cwr.status IN ('Succeeded', 'Healthy')
				)
				GROUP BY cw.pipeline_id
			)
			SELECT id, pipeline_id, ci_artifact_id
			FROM cd_workflow
			WHERE id IN (SELECT cdw_id FROM workflow)
			`
	_, err := impl.dbConnection.Query(&cdWorkflow, query, pg.In(cdPipelineIds))
	if err != nil {
		impl.logger.Errorw("error in finding all workflows for given artifactIds and cdPipelineIds", "cdPipelineIds", cdPipelineIds, "err", err)
		return nil, err
	}
	return cdWorkflow, nil
}

func (impl *CdWorkflowRepositoryImpl) FindLatestRunnerByPipelineIdsAndRunnerType(ctx context.Context, pipelineIds []int, runnerType bean.WorkflowType) ([]CdWorkflowRunner, error) {
	_, span := otel.Tracer("orchestrator").Start(ctx, "FindLatestRunnerByPipelineIdsAndRunnerType")
	defer span.End()
	if pipelineIds == nil || len(pipelineIds) == 0 {
		return nil, pg.ErrNoRows
	}
	var latestWfrs []CdWorkflowRunner
	err := impl.dbConnection.
		Model(&latestWfrs).
		Column("cd_workflow_runner.*", "CdWorkflow", "CdWorkflow.Pipeline").
		ColumnExpr("MAX(cd_workflow_runner.id)").
		Where("cd_workflow.pipeline_id IN (?)", pg.In(pipelineIds)).
		Where("cd_workflow_runner.workflow_type = ?", runnerType).
		Where("cd_workflow__pipeline.deleted = ?", false).
		Group("cd_workflow_runner.id", "cd_workflow.id", "cd_workflow__pipeline.id").
		Select()
	if err != nil {
		impl.logger.Errorw("error in getting cdWfr by appId, envId and runner type", "pipelineIds", pipelineIds, "runnerType", runnerType)
		return nil, err
	}
	return latestWfrs, err
}

func (impl *CdWorkflowRepositoryImpl) GetCdPipelineIdsForRunnerIds(cdWfrIds []int) ([]int, error) {
	if len(cdWfrIds) == 0 {
		return nil, pg.ErrNoRows
	}
	var cdPipelineIds []int
	err := impl.dbConnection.
		Model().
		Table("cd_workflow_runner").
		Join("INNER JOIN cd_workflow cdwf").
		JoinOn("cdwf.id  = cd_workflow_runner.cd_workflow_id").
		Join("INNER JOIN pipeline p").
		JoinOn("p.id = cdwf.pipeline_id").
		ColumnExpr("DISTINCT p.id").
		Where("cd_workflow_runner.id IN (?)", pg.In(cdWfrIds)).
		Where("p.deleted = ?", false).
		Select(&cdPipelineIds)
	if err != nil {
		impl.logger.Errorw("error in getting pipeline ids by cdWfrIds", "cdWfrIds", cdWfrIds, "err", err)
		return nil, err
	}
	return cdPipelineIds, err
}
