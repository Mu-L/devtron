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

package pipelineConfig

import (
	"context"
	"time"

	"github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/internal/sql/models"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	"github.com/devtron-labs/devtron/internal/sql/repository/appWorkflow"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig/bean/timelineStatus"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig/bean/workflow/cdWorkflow"
	"github.com/devtron-labs/devtron/internal/util"
	util2 "github.com/devtron-labs/devtron/pkg/appStore/util"
	"github.com/devtron-labs/devtron/pkg/cluster/environment/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"k8s.io/utils/pointer"
)

type PipelineType string
type TriggerType string // HOW pipeline should be triggered

func (t TriggerType) ToString() string {
	return string(t)
}

func (t TriggerType) IsManual() bool {
	return t == TRIGGER_TYPE_MANUAL
}

func (t TriggerType) IsAuto() bool {
	return t == TRIGGER_TYPE_AUTOMATIC
}

const TRIGGER_TYPE_AUTOMATIC TriggerType = "AUTOMATIC"
const TRIGGER_TYPE_MANUAL TriggerType = "MANUAL"

type Pipeline struct {
	tableName                     struct{} `sql:"pipeline" pg:",discard_unknown_columns"`
	Id                            int      `sql:"id,pk"`
	AppId                         int      `sql:"app_id,notnull"`
	App                           app.App
	CiPipelineId                  int         `sql:"ci_pipeline_id"`
	TriggerType                   TriggerType `sql:"trigger_type,notnull"` // automatic, manual
	EnvironmentId                 int         `sql:"environment_id"`
	Name                          string      `sql:"pipeline_name,notnull"`
	Deleted                       bool        `sql:"deleted,notnull"`
	PreStageConfig                string      `sql:"pre_stage_config_yaml"`
	PostStageConfig               string      `sql:"post_stage_config_yaml"`
	PreTriggerType                TriggerType `sql:"pre_trigger_type"`                   // automatic, manual; when a pre-cd task doesn't exist/removed in a cd then this field is updated as null
	PostTriggerType               TriggerType `sql:"post_trigger_type"`                  // automatic, manual; when a post-cd task doesn't exist/removed in a cd then this field is updated as null
	PreStageConfigMapSecretNames  string      `sql:"pre_stage_config_map_secret_names"`  // configmap names
	PostStageConfigMapSecretNames string      `sql:"post_stage_config_map_secret_names"` // secret names
	RunPreStageInEnv              bool        `sql:"run_pre_stage_in_env"`               // secret names
	RunPostStageInEnv             bool        `sql:"run_post_stage_in_env"`              // secret names
	DeploymentAppCreated          bool        `sql:"deployment_app_created,notnull"`
	DeploymentAppType             string      `sql:"deployment_app_type,notnull"` // Deprecated;
	DeploymentAppName             string      `sql:"deployment_app_name"`
	DeploymentAppDeleteRequest    bool        `sql:"deployment_app_delete_request,notnull"`
	Environment                   repository.Environment
	sql.AuditLog
}

type PipelineRepository interface {
	Save(pipeline []*Pipeline, tx *pg.Tx) error
	Update(pipeline *Pipeline, tx *pg.Tx) error
	FindActiveByAppId(appId int) (pipelines []*Pipeline, err error)
	Delete(id int, userId int32, tx *pg.Tx) error
	MarkPartiallyDeleted(id int, userId int32, tx *pg.Tx) error
	FindByName(pipelineName string) (pipeline *Pipeline, err error)
	PipelineExists(pipelineName string) (bool, error)
	FindById(id int) (pipeline *Pipeline, err error)
	FindByIdEvenIfInactive(id int) (pipeline *Pipeline, err error)
	GetPostStageConfigById(id int) (pipeline *Pipeline, err error)
	FindAppAndEnvDetailsByPipelineId(id int) (pipeline *Pipeline, err error)
	FindActiveByEnvIdAndDeploymentType(environmentId int, deploymentAppType string, exclusionList []int, includeApps []int) ([]*Pipeline, error)
	FindByIdsIn(ids []int) ([]*Pipeline, error)
	FindByCiPipelineIdsIn(ciPipelineIds []int) ([]*Pipeline, error)

	FindAutomaticByCiPipelineId(ciPipelineId int) (pipelines []*Pipeline, err error)
	GetByEnvOverrideId(envOverrideId int) ([]Pipeline, error)
	GetByEnvOverrideIdAndEnvId(envOverrideId, envId int) (Pipeline, error)
	FindActiveByAppIdAndEnvironmentId(appId int, environmentId int) (pipelines []*Pipeline, err error)
	FindOneByAppIdAndEnvId(appId int, envId int) (*Pipeline, error)
	UniqueAppEnvironmentPipelines() ([]*Pipeline, error)
	FindByCiPipelineId(ciPipelineId int) (pipelines []*Pipeline, err error)
	FindByParentCiPipelineId(ciPipelineId int) (pipelines []*Pipeline, err error)
	FindByPipelineTriggerGitHash(gitHash string) (pipeline *Pipeline, err error)
	FindByIdsInAndEnvironment(ids []int, environmentId int) ([]*Pipeline, error)
	FindActiveByAppIdAndEnvironmentIdV2() (pipelines []*Pipeline, err error)
	GetConnection() *pg.DB
	FindAllPipelineCreatedCountInLast24Hour() (pipelineCount int, err error)
	FindAllDeletedPipelineCountInLast24Hour() (pipelineCount int, err error)
	FindActiveByEnvId(envId int) (pipelines []*Pipeline, err error)
	FindActivePipelineAppIdsByEnvId(envId int) ([]int, error)
	FindActivePipelineByEnvId(envId int) (pipelines []*Pipeline, err error)
	FindActiveByEnvIds(envId []int) (pipelines []*Pipeline, err error)
	FindActiveByInFilter(envId int, appIdIncludes []int) (pipelines []*Pipeline, err error)
	FindActivePipelineAppIdsByInFilter(envId int, appIdIncludes []int) ([]int, error)
	FindActiveByNotFilter(envId int, appIdExcludes []int) (pipelines []*Pipeline, err error)
	FindAllPipelinesWithoutOverriddenCharts(appId int) (pipelineIds []int, err error)
	FindActiveByAppIdAndPipelineId(appId int, pipelineId int) ([]*Pipeline, error)
	FindActiveByAppIdAndEnvId(appId int, envId int) (*Pipeline, error)
	SetDeploymentAppCreatedInPipeline(deploymentAppCreated bool, pipelineId int, userId int32) error
	UpdateCdPipelineDeploymentAppInFilter(deploymentAppType string, cdPipelineIdIncludes []int, userId int32, deploymentAppCreated bool, delete bool) error
	UpdateCdPipelineAfterDeployment(deploymentAppType string, cdPipelineIdIncludes []int, userId int32, delete bool) error
	FindNumberOfAppsWithCdPipeline(appIds []int) (count int, err error)
	GetAppAndEnvDetailsForDeploymentAppTypePipeline(deploymentAppType string, clusterIds []int) ([]*Pipeline, error)
	GetArgoPipelinesHavingTriggersStuckInLastPossibleNonTerminalTimelines(pendingSinceSeconds int, timeForDegradation int) ([]*Pipeline, error)
	GetArgoPipelinesHavingLatestTriggerStuckInNonTerminalStatuses(deployedBeforeMinutes int, getPipelineDeployedWithinHours int) ([]*Pipeline, error)
	FindIdsByAppIdsAndEnvironmentIds(appIds, environmentIds []int) (ids []int, err error)
	FindIdsByProjectIdsAndEnvironmentIds(projectIds, environmentIds []int) ([]int, error)

	GetArgoPipelineByArgoAppName(argoAppName string) ([]Pipeline, error)
	FindActiveByAppIds(appIds []int) (pipelines []*Pipeline, err error)
	FindAppAndEnvironmentAndProjectByPipelineIds(pipelineIds []int) (pipelines []*Pipeline, err error)
	FilterDeploymentDeleteRequestedPipelineIds(cdPipelineIds []int) (map[int]bool, error)
	FindDeploymentTypeByPipelineIds(cdPipelineIds []int) (map[int]DeploymentObject, error)
	UpdateCiPipelineId(tx *pg.Tx, pipelineIds []int, ciPipelineId int) error
	UpdateOldCiPipelineIdToNewCiPipelineId(tx *pg.Tx, oldCiPipelineId, newCiPipelineId int) error
	// FindWithEnvironmentByCiIds Possibility of duplicate environment names when filtered by unique pipeline ids
	FindWithEnvironmentByCiIds(ctx context.Context, cIPipelineIds []int) ([]*Pipeline, error)
	FindDeploymentAppTypeByAppIdAndEnvId(appId, envId int) (string, error)
	FindByAppIdToEnvIdsMapping(appIdToEnvIds map[int][]int) ([]*Pipeline, error)
	FindDeploymentAppTypeByIds(ids []int) (pipelines []*Pipeline, err error)
	GetAllAppsByClusterAndDeploymentAppType(clusterIds []int, deploymentAppName string) ([]*PipelineDeploymentConfigObj, error)
	GetAllArgoAppInfoByDeploymentAppNames(deploymentAppNames []string) ([]*PipelineDeploymentConfigObj, error)
	FindEnvIdsByIdsInIncludingDeleted(ids []int) ([]int, error)
	GetPipelineCountByDeploymentType(deploymentType string) (int, error)
}

type CiArtifactDTO struct {
	Id           int    `json:"id"`
	PipelineId   int    `json:"pipelineId"` //id of the ci pipeline from which this webhook was triggered
	Image        string `json:"image"`
	ImageDigest  string `json:"imageDigest"`
	MaterialInfo string `json:"materialInfo"` //git material metadata json array string
	DataSource   string `json:"dataSource"`
	WorkflowId   *int   `json:"workflowId"`
}

type DeploymentObject struct {
	DeploymentType models.DeploymentType `sql:"deployment_type"`
	PipelineId     int                   `sql:"pipeline_id"`
	Status         string                `sql:"status"`
}

type PipelineDeploymentConfigObj struct {
	DeploymentAppName string `json:"deployment_app_name"`
	AppId             int    `json:"app_id"`
	ClusterId         int    `json:"cluster_id"`
	EnvironmentId     int    `json:"environment_id"`
	Namespace         string `json:"namespace"`
}

type PipelineRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
}

func NewPipelineRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger) *PipelineRepositoryImpl {
	return &PipelineRepositoryImpl{dbConnection: dbConnection, logger: logger}
}

func (impl *PipelineRepositoryImpl) GetConnection() *pg.DB {
	return impl.dbConnection
}

func (impl *PipelineRepositoryImpl) FindByIdsIn(ids []int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	err := impl.dbConnection.Model(&pipelines).
		Column("pipeline.*", "App", "Environment", "Environment.Cluster").
		Join("inner join app a on pipeline.app_id = a.id").
		Join("inner join environment e on pipeline.environment_id = e.id").
		Join("inner join cluster c on c.id = e.cluster_id").
		Where("pipeline.id in (?)", pg.In(ids)).
		Where("pipeline.deleted = false").
		Select()
	if err != nil {
		impl.logger.Errorw("error on fetching pipelines", "ids", ids)
	}
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindByIdsInAndEnvironment(ids []int, environmentId int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	err := impl.dbConnection.Model(&pipelines).
		Where("id in (?)", pg.In(ids)).
		Where("environment_id = ?", environmentId).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindByCiPipelineIdsIn(ciPipelineIds []int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	err := impl.dbConnection.Model(&pipelines).
		Where("ci_pipeline_id in (?)", pg.In(ciPipelineIds)).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) Save(pipeline []*Pipeline, tx *pg.Tx) error {
	var v []interface{}
	for _, i := range pipeline {
		v = append(v, i)
	}
	_, err := tx.Model(v...).Insert()
	return err
}

func (impl *PipelineRepositoryImpl) Update(pipeline *Pipeline, tx *pg.Tx) error {
	err := tx.Update(pipeline)
	return err
}

func (impl *PipelineRepositoryImpl) FindAutomaticByCiPipelineId(ciPipelineId int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).
		Where("ci_pipeline_id =?", ciPipelineId).
		Where("trigger_type =?", TRIGGER_TYPE_AUTOMATIC).
		Where("deleted =?", false).
		Select()
	if err != nil && util.IsErrNoRows(err) {
		return make([]*Pipeline, 0), nil
	} else if err != nil {
		return nil, err
	}
	return pipelines, nil
}

func (impl *PipelineRepositoryImpl) FindByCiPipelineId(ciPipelineId int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).
		Where("ci_pipeline_id =?", ciPipelineId).
		Where("deleted =?", false).
		Select()
	if err != nil && util.IsErrNoRows(err) {
		return make([]*Pipeline, 0), nil
	} else if err != nil {
		return nil, err
	}
	return pipelines, nil
}

func (impl *PipelineRepositoryImpl) FindByParentCiPipelineId(ciPipelineId int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).
		Column("pipeline.*").
		Join("INNER JOIN app_workflow_mapping awm on awm.component_id = pipeline.id").
		Where("pipeline.ci_pipeline_id =?", ciPipelineId).
		Where("awm.parent_type =?", appWorkflow.CIPIPELINE).
		Where("pipeline.deleted =?", false).
		Select()
	if err != nil && util.IsErrNoRows(err) {
		return make([]*Pipeline, 0), nil
	} else if err != nil {
		return nil, err
	}
	return pipelines, nil
}

func (impl *PipelineRepositoryImpl) FindActiveByAppId(appId int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).
		Column("pipeline.*", "Environment").
		Where("app_id = ?", appId).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindActiveByAppIdAndEnvironmentId(appId int, environmentId int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).
		Column("pipeline.*", "Environment", "App").
		Where("app_id = ?", appId).
		Where("deleted = ?", false).
		Where("environment_id = ? ", environmentId).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindOneByAppIdAndEnvId(appId int, envId int) (*Pipeline, error) {
	pipeline := Pipeline{}
	err := impl.dbConnection.Model(&pipeline).
		Column("pipeline.*").
		Where("app_id = ?", appId).
		Where("deleted = ?", false).
		Where("environment_id = ? ", envId).
		Select()
	return &pipeline, err
}

func (impl *PipelineRepositoryImpl) FindActiveByAppIdAndEnvironmentIdV2() (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) Delete(id int, userId int32, tx *pg.Tx) error {
	pipeline := &Pipeline{}
	r, err := tx.Model(pipeline).Set("deleted =?", true).Set("deployment_app_created =?", false).
		Set("updated_on = ?", time.Now()).Set("updated_by = ?", userId).Where("id =?", id).Update()
	impl.logger.Debugw("update result", "r-affected", r.RowsAffected(), "r-return", r.RowsReturned(), "model", r.Model())
	return err
}

func (impl *PipelineRepositoryImpl) MarkPartiallyDeleted(id int, userId int32, tx *pg.Tx) error {
	pipeline := &Pipeline{}
	_, err := tx.Model(pipeline).
		Set("deployment_app_delete_request = ?", true).
		Set("updated_on = ?", time.Now()).
		Set("updated_by = ?", userId).
		Where("deleted = ?", false).
		Where("id = ?", id).
		Update()
	return err
}

func (impl *PipelineRepositoryImpl) FindByName(pipelineName string) (pipeline *Pipeline, err error) {
	pipeline = &Pipeline{}
	err = impl.dbConnection.Model(pipeline).
		Where("pipeline_name = ?", pipelineName).
		Select()
	return pipeline, err
}

func (impl *PipelineRepositoryImpl) PipelineExists(pipelineName string) (bool, error) {
	pipeline := &Pipeline{}
	exists, err := impl.dbConnection.Model(pipeline).
		Where("pipeline_name = ?", pipelineName).
		Where("deleted =? ", false).
		Exists()
	return exists, err
}

func (impl *PipelineRepositoryImpl) FindById(id int) (pipeline *Pipeline, err error) {
	pipeline = &Pipeline{}
	err = impl.dbConnection.
		Model(pipeline).
		Column("pipeline.*", "App", "Environment").
		Join("inner join app a on pipeline.app_id = a.id").
		Where("pipeline.id = ?", id).
		Where("deleted = ?", false).
		Select()
	return pipeline, err
}

func (impl *PipelineRepositoryImpl) FindByIdEvenIfInactive(id int) (pipeline *Pipeline, err error) {
	pipeline = &Pipeline{}
	err = impl.dbConnection.
		Model(pipeline).
		Column("pipeline.*", "App", "Environment").
		Join("inner join app a on pipeline.app_id = a.id").
		Where("pipeline.id = ?", id).
		Select()
	return pipeline, err
}

func (impl *PipelineRepositoryImpl) GetPostStageConfigById(id int) (pipeline *Pipeline, err error) {
	pipeline = &Pipeline{}
	err = impl.dbConnection.
		Model(pipeline).
		Column("pipeline.post_stage_config_yaml").
		Where("pipeline.id = ?", id).
		Where("deleted = ?", false).
		Select()
	return pipeline, err
}

func (impl *PipelineRepositoryImpl) FindAppAndEnvDetailsByPipelineId(id int) (pipeline *Pipeline, err error) {
	pipeline = &Pipeline{}
	err = impl.dbConnection.
		Model(pipeline).
		Column("App.id", "App.app_name", "App.app_type", "Environment.id", "Environment.cluster_id").
		Join("inner join app a on pipeline.app_id = a.id").
		Join("inner join environment e on pipeline.environment_id = e.id").
		Where("pipeline.id = ?", id).
		Where("deleted = ?", false).
		Select()
	return pipeline, err
}

// FindActiveByEnvIdAndDeploymentType takes in environment id and current deployment app type
// and fetches and returns a list of pipelines matching the same excluding given app ids.
func (impl *PipelineRepositoryImpl) FindActiveByEnvIdAndDeploymentType(environmentId int,
	deploymentAppType string, exclusionList []int, includeApps []int) ([]*Pipeline, error) {

	// NOTE: PG query throws error with slice of integer
	exclusionListString := util2.ConvertIntArrayToStringArray(exclusionList)

	inclusionListString := util2.ConvertIntArrayToStringArray(includeApps)

	var pipelines []*Pipeline

	query := impl.dbConnection.
		Model(&pipelines).
		Column("pipeline.*", "App", "Environment").
		Join("inner join app a on pipeline.app_id = a.id").
		Join("LEFT JOIN deployment_config dc on dc.active=true and dc.app_id = pipeline.app_id and dc.environment_id=pipeline.environment_id").
		Where("pipeline.environment_id = ?", environmentId).
		Where("(pipeline.deployment_app_type=? or dc.deployment_app_type=?)", deploymentAppType, deploymentAppType).
		Where("pipeline.deleted = ?", false)

	if len(exclusionListString) > 0 {
		query.Where("pipeline.app_id not in (?)", pg.In(exclusionListString))
	}

	if len(inclusionListString) > 0 {
		query.Where("pipeline.app_id in (?)", pg.In(inclusionListString))
	}

	err := query.Select()
	return pipelines, err
}

// Deprecated:
func (impl *PipelineRepositoryImpl) FindByEnvOverrideId(envOverrideId int) (pipeline []Pipeline, err error) {
	var pipelines []Pipeline
	err = impl.dbConnection.
		Model(&pipelines).
		Column("pipeline.*").
		Join("INNER JOIN pipeline_config_override pco on pco.pipeline_id = pipeline.id").
		Where("pco.env_config_override_id = ?", envOverrideId).Group("pipeline.id, pipeline.pipeline_name").
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) GetByEnvOverrideId(envOverrideId int) ([]Pipeline, error) {
	var pipelines []Pipeline
	query := "" +
		" SELECT p.*" +
		" FROM chart_env_config_override ceco" +
		" INNER JOIN charts ch on ch.id = ceco.chart_id" +
		" INNER JOIN environment env on env.id = ceco.target_environment" +
		" INNER JOIN app ap on ap.id = ch.app_id" +
		" INNER JOIN pipeline p on p.app_id = ap.id" +
		" WHERE ceco.id=?;"
	_, err := impl.dbConnection.Query(&pipelines, query, envOverrideId)

	if err != nil {
		return nil, err
	}
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) GetByEnvOverrideIdAndEnvId(envOverrideId, envId int) (Pipeline, error) {
	var pipeline Pipeline
	query := "" +
		" SELECT p.*" +
		" FROM chart_env_config_override ceco" +
		" INNER JOIN charts ch on ch.id = ceco.chart_id" +
		" INNER JOIN environment env on env.id = ceco.target_environment" +
		" INNER JOIN app ap on ap.id = ch.app_id" +
		" INNER JOIN pipeline p on p.app_id = ap.id" +
		" WHERE ceco.id=? and p.environment_id=?;"
	_, err := impl.dbConnection.Query(&pipeline, query, envOverrideId, envId)

	if err != nil {
		return pipeline, err
	}
	return pipeline, err
}

func (impl *PipelineRepositoryImpl) UniqueAppEnvironmentPipelines() ([]*Pipeline, error) {
	var pipelines []*Pipeline

	err := impl.dbConnection.
		Model(&pipelines).
		ColumnExpr("DISTINCT app_id, environment_id").
		Where("deleted = ?", false).
		Select()
	if err != nil {
		return nil, err
	}
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindByPipelineTriggerGitHash(gitHash string) (pipeline *Pipeline, err error) {
	var pipelines *Pipeline
	err = impl.dbConnection.
		Model(&pipelines).
		Column("pipeline.*").
		Join("INNER JOIN pipeline_config_override pco on pco.pipeline_id = pipeline.id").
		Where("pco.git_hash = ?", gitHash).Order(" ORDER BY pco.created_on DESC").Limit(1).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindAllPipelineCreatedCountInLast24Hour() (pipelineCount int, err error) {
	pipelineCount, err = impl.dbConnection.Model(&Pipeline{}).
		Where("created_on > ?", time.Now().AddDate(0, 0, -1)).
		Count()
	return pipelineCount, err
}

func (impl *PipelineRepositoryImpl) FindAllDeletedPipelineCountInLast24Hour() (pipelineCount int, err error) {
	pipelineCount, err = impl.dbConnection.Model(&Pipeline{}).
		Where("created_on > ? and deleted=?", time.Now().AddDate(0, 0, -1), true).
		Count()
	return pipelineCount, err
}

func (impl *PipelineRepositoryImpl) FindActiveByEnvId(envId int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).Column("pipeline.*", "App", "Environment").
		Where("environment_id = ?", envId).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindActivePipelineAppIdsByEnvId(envId int) ([]int, error) {
	var appIds []int
	err := impl.dbConnection.Model((*Pipeline)(nil)).Column("app_id").
		Where("environment_id = ?", envId).
		Where("deleted = ?", false).
		Select(&appIds)
	return appIds, err
}

func (impl *PipelineRepositoryImpl) FindActivePipelineByEnvId(envId int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).Column("pipeline.*", "App", "Environment").
		Where("environment_id = ?", envId).
		Where("deleted = ?", false).
		Where("deployment_app_delete_request = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindActiveByEnvIds(envIds []int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).Column("pipeline.*").
		Where("environment_id in (?)", pg.In(envIds)).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindActiveByInFilter(envId int, appIdIncludes []int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).Column("pipeline.*", "App", "Environment").
		Where("environment_id = ?", envId).
		Where("app_id in (?)", pg.In(appIdIncludes)).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindActivePipelineAppIdsByInFilter(envId int, appIdIncludes []int) ([]int, error) {
	var appIds []int
	err := impl.dbConnection.Model((*Pipeline)(nil)).Column("app_id").
		Where("environment_id = ?", envId).
		Where("app_id in (?)", pg.In(appIdIncludes)).
		Where("deleted = ?", false).Select(&appIds)
	return appIds, err
}

func (impl *PipelineRepositoryImpl) FindActiveByNotFilter(envId int, appIdExcludes []int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).Column("pipeline.*", "App", "Environment").
		Where("environment_id = ?", envId).
		Where("app_id not in (?)", pg.In(appIdExcludes)).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindAllPipelinesWithoutOverriddenCharts(appId int) (pipelineIds []int, err error) {
	err = impl.dbConnection.Model().Table("pipeline").Column("pipeline.id").
		Where("pipeline.deleted = ?", false).Where("pipeline.app_id = ?", appId).
		Where(`pipeline.environment_id NOT IN (
				SELECT ceco.target_environment FROM chart_env_config_override ceco 
				INNER JOIN charts ON charts.id = ceco.chart_id
				WHERE charts.app_id = ? AND charts.active = ? AND ceco.is_override = ?
				AND ceco.active = ? AND ceco.latest = ?)`, appId, true, true, true, true).
		Select(&pipelineIds)
	return pipelineIds, err
}

func (impl *PipelineRepositoryImpl) FindActiveByAppIdAndPipelineId(appId int, pipelineId int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	err := impl.dbConnection.Model(&pipelines).
		Where("app_id = ?", appId).
		Where("ci_pipeline_id = ?", pipelineId).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindActiveByAppIdAndEnvId(appId int, envId int) (*Pipeline, error) {
	var pipeline Pipeline
	err := impl.dbConnection.Model(&pipeline).
		Where("app_id = ?", appId).
		Where("environment_id = ?", envId).
		Where("deleted = ?", false).
		Select()
	return &pipeline, err
}

func (impl *PipelineRepositoryImpl) SetDeploymentAppCreatedInPipeline(deploymentAppCreated bool, pipelineId int, userId int32) error {
	query := "update pipeline set deployment_app_created=?, updated_on=?, updated_by=? where id=?;"
	var pipeline *Pipeline
	_, err := impl.dbConnection.Query(pipeline, query, deploymentAppCreated, time.Now(), userId, pipelineId)
	return err
}

// UpdateCdPipelineDeploymentAppInFilter takes in deployment app type and list of cd pipeline ids and
// updates the deploymentAppType and sets deployment_app_created to false in the table for given ids.
func (impl *PipelineRepositoryImpl) UpdateCdPipelineDeploymentAppInFilter(deploymentAppType string,
	cdPipelineIdIncludes []int, userId int32, deploymentAppCreated bool, isDeleted bool) error {
	query := "update pipeline set deployment_app_created = ?, deployment_app_type = ?, " +
		"updated_by = ?, updated_on = ?, deployment_app_delete_request = ? where id in (?);"
	var pipeline *Pipeline
	_, err := impl.dbConnection.Query(pipeline, query, deploymentAppCreated, deploymentAppType, userId, time.Now(), isDeleted, pg.In(cdPipelineIdIncludes))

	return err
}

func (impl *PipelineRepositoryImpl) UpdateCdPipelineAfterDeployment(deploymentAppType string,
	cdPipelineIdIncludes []int, userId int32, isDeleted bool) error {
	query := "update pipeline set deployment_app_type = ?, " +
		"updated_by = ?, updated_on = ?, deployment_app_delete_request = ? where id in (?);"
	var pipeline *Pipeline
	_, err := impl.dbConnection.Query(pipeline, query, deploymentAppType, userId, time.Now(), isDeleted, pg.In(cdPipelineIdIncludes))

	return err
}

func (impl *PipelineRepositoryImpl) FindNumberOfAppsWithCdPipeline(appIds []int) (count int, err error) {
	var pipelines []*Pipeline
	count, err = impl.dbConnection.
		Model(&pipelines).
		ColumnExpr("DISTINCT app_id").
		Where("app_id in (?)", pg.In(appIds)).
		Count()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (impl *PipelineRepositoryImpl) GetAppAndEnvDetailsForDeploymentAppTypePipeline(deploymentAppType string, clusterIds []int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	err := impl.dbConnection.
		Model(&pipelines).
		Column("pipeline.id", "App.app_name", "pipeline.deployment_app_name", "Environment.cluster_id", "Environment.namespace", "Environment.environment_name").
		Join("inner join app a on pipeline.app_id = a.id").
		Join("inner join environment e on pipeline.environment_id = e.id").
		Join("LEFT JOIN deployment_config dc on dc.active=true and dc.app_id = pipeline.app_id and dc.environment_id=pipeline.environment_id").
		Where("e.cluster_id in (?)", pg.In(clusterIds)).
		Where("a.active = ?", true).
		Where("pipeline.deleted = ?", false).
		Where("(pipeline.deployment_app_type=? or dc.deployment_app_type=?)", deploymentAppType, deploymentAppType).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) GetArgoPipelinesHavingTriggersStuckInLastPossibleNonTerminalTimelines(pendingSinceSeconds int, timeForDegradation int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	queryString := `select p.* from pipeline p inner join cd_workflow cw on cw.pipeline_id = p.id
    inner join cd_workflow_runner cwr on cwr.cd_workflow_id=cw.id
    left join deployment_config dc on dc.active=true and dc.app_id = p.app_id and dc.environment_id=p.environment_id
    where cwr.id in (select cd_workflow_runner_id from pipeline_status_timeline
					where id in
						(select DISTINCT ON (cd_workflow_runner_id) max(id) as id from pipeline_status_timeline
							group by cd_workflow_runner_id, id order by cd_workflow_runner_id,id desc)
					and status in (?) and status_time < NOW() - INTERVAL '? seconds')
    and cwr.started_on > NOW() - INTERVAL '? minutes' and (p.deployment_app_type=? or dc.deployment_app_type=?) and p.deleted=?;`
	_, err := impl.dbConnection.Query(&pipelines, queryString,
		pg.In([]timelineStatus.TimelineStatus{timelineStatus.TIMELINE_STATUS_KUBECTL_APPLY_SYNCED,
			timelineStatus.TIMELINE_STATUS_FETCH_TIMED_OUT, timelineStatus.TIMELINE_STATUS_UNABLE_TO_FETCH_STATUS}),
		pendingSinceSeconds, timeForDegradation, util.PIPELINE_DEPLOYMENT_TYPE_ACD, util.PIPELINE_DEPLOYMENT_TYPE_ACD, false)
	if err != nil {
		impl.logger.Errorw("error in GetArgoPipelinesHavingTriggersStuckInLastPossibleNonTerminalTimelines", "err", err)
		return nil, err
	}
	return pipelines, nil
}

func (impl *PipelineRepositoryImpl) GetArgoPipelinesHavingLatestTriggerStuckInNonTerminalStatuses(getPipelineDeployedBeforeMinutes int, getPipelineDeployedWithinHours int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	queryString := `select p.id from pipeline p inner join cd_workflow cw on cw.pipeline_id = p.id
    inner join cd_workflow_runner cwr on cwr.cd_workflow_id=cw.id
    left join deployment_config dc on dc.active=true and dc.app_id = p.app_id and dc.environment_id=p.environment_id
    where cwr.id in (select id from cd_workflow_runner
                     	where started_on < NOW() - INTERVAL '? minutes' and started_on > NOW() - INTERVAL '? hours' and status not in (?)
                     	and workflow_type=? and cd_workflow_id in (select DISTINCT ON (pipeline_id) max(id) as id from cd_workflow
                     	  group by pipeline_id, id order by pipeline_id, id desc))
    and (p.deployment_app_type=? or dc.deployment_app_type=?) and p.deleted=?;`
	_, err := impl.dbConnection.Query(&pipelines, queryString, getPipelineDeployedBeforeMinutes, getPipelineDeployedWithinHours,
		pg.In(append(cdWorkflow.WfrTerminalStatusList, cdWorkflow.WorkflowInitiated, cdWorkflow.WorkflowInQueue)),
		bean.CD_WORKFLOW_TYPE_DEPLOY, util.PIPELINE_DEPLOYMENT_TYPE_ACD, util.PIPELINE_DEPLOYMENT_TYPE_ACD, false)
	if err != nil {
		impl.logger.Errorw("error in GetArgoPipelinesHavingLatestTriggerStuckInNonTerminalStatuses", "err", err)
		return nil, err
	}
	return pipelines, nil
}

func (impl *PipelineRepositoryImpl) FindIdsByAppIdsAndEnvironmentIds(appIds, environmentIds []int) ([]int, error) {
	var pipelineIds []int
	query := "select id from pipeline where app_id in (?) and environment_id in (?) and deleted = ?;"
	_, err := impl.dbConnection.Query(&pipelineIds, query, pg.In(appIds), pg.In(environmentIds), false)
	if err != nil {
		impl.logger.Errorw("error in getting pipelineIds by appIds and envIds", "err", err, "appIds", appIds, "envIds", environmentIds)
		return pipelineIds, err
	}
	return pipelineIds, err
}

func (impl *PipelineRepositoryImpl) FindIdsByProjectIdsAndEnvironmentIds(projectIds, environmentIds []int) ([]int, error) {
	var pipelineIds []int
	query := "select p.id from pipeline p inner join app a on a.id=p.app_id where a.team_id in (?) and p.environment_id in (?) and p.deleted = ? and a.active = ?;"
	_, err := impl.dbConnection.Query(&pipelineIds, query, pg.In(projectIds), pg.In(environmentIds), false, true)
	if err != nil {
		impl.logger.Errorw("error in getting pipelineIds by projectIds and envIds", "err", err, "projectIds", projectIds, "envIds", environmentIds)
		return pipelineIds, err
	}
	return pipelineIds, err
}

func (impl *PipelineRepositoryImpl) GetArgoPipelineByArgoAppName(argoAppName string) ([]Pipeline, error) {
	var pipeline []Pipeline
	err := impl.dbConnection.Model(&pipeline).
		Join("LEFT JOIN deployment_config dc on dc.app_id = pipeline.app_id and dc.environment_id=pipeline.environment_id and dc.active=true").
		Column("pipeline.*", "Environment").
		Where("deployment_app_name = ?", argoAppName).
		Where("(pipeline.deployment_app_type=? or dc.deployment_app_type=?)", util.PIPELINE_DEPLOYMENT_TYPE_ACD, util.PIPELINE_DEPLOYMENT_TYPE_ACD).
		Where("deleted = ?", false).
		Select()
	if err != nil {
		impl.logger.Errorw("error in getting pipeline by argoAppName", "err", err, "argoAppName", argoAppName)
		return pipeline, err
	}
	return pipeline, nil
}

func (impl *PipelineRepositoryImpl) FindActiveByAppIds(appIds []int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).
		Column("pipeline.*", "App", "Environment").
		Where("app_id in(?)", pg.In(appIds)).
		Where("deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindAppAndEnvironmentAndProjectByPipelineIds(pipelineIds []int) (pipelines []*Pipeline, err error) {
	if len(pipelineIds) == 0 {
		return pipelines, nil
	}
	err = impl.dbConnection.Model(&pipelines).Column("pipeline.*", "App", "Environment", "App.Team").
		Where("pipeline.id in(?)", pg.In(pipelineIds)).
		Where("pipeline.deleted = ?", false).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FilterDeploymentDeleteRequestedPipelineIds(cdPipelineIds []int) (map[int]bool, error) {
	var pipelineIds []int
	pipelineIdsMap := make(map[int]bool)
	query := "select pipeline.id from pipeline where pipeline.id in (?) and pipeline.deployment_app_delete_request = ?;"
	_, err := impl.dbConnection.Query(&pipelineIds, query, pg.In(cdPipelineIds), true)
	if err != nil {
		return pipelineIdsMap, err
	}
	for _, pipelineId := range pipelineIds {
		pipelineIdsMap[pipelineId] = true
	}
	return pipelineIdsMap, nil
}

func (impl *PipelineRepositoryImpl) FindDeploymentTypeByPipelineIds(cdPipelineIds []int) (map[int]DeploymentObject, error) {

	pipelineIdsMap := make(map[int]DeploymentObject)

	var deploymentType []DeploymentObject
	//query := "with pcos as(select max(id) as id from pipeline_config_override where pipeline_id in (?) " +
	//	"group by pipeline_id) select pco.deployment_type,pco.pipeline_id, aps.status from pipeline_config_override " +
	//	"pco inner join pcos on pcos.id=pco.id" +
	//	" inner join pipeline p on p.id=pco.pipeline_id left join app_status aps on aps.app_id=p.app_id " +
	//	"and aps.env_id=p.environment_id;"
	query := " WITH pcos AS " +
		" (SELECT max(p.id) AS id FROM pipeline_config_override p " +
		" INNER JOIN cd_workflow_runner cdwr ON cdwr.cd_workflow_id = p.cd_workflow_id " +
		//                  pipeline ids               deploy type         initiated,queued,failed
		" WHERE p.pipeline_id in (?) AND cdwr.workflow_type = ? AND cdwr.status NOT IN (?) " +
		" GROUP BY p.pipeline_id) select pco.deployment_type,pco.pipeline_id, aps.status from pipeline_config_override " +
		"pco inner join pcos on pcos.id=pco.id" +
		" inner join pipeline p on p.id=pco.pipeline_id left join app_status aps on aps.app_id=p.app_id " +
		"and aps.env_id=p.environment_id;"
	_, err := impl.dbConnection.Query(&deploymentType, query, pg.In(cdPipelineIds), bean.CD_WORKFLOW_TYPE_DEPLOY, pg.In([]string{cdWorkflow.WorkflowInitiated, cdWorkflow.WorkflowInQueue, cdWorkflow.WorkflowFailed}))
	if err != nil {
		return pipelineIdsMap, err
	}

	for _, v := range deploymentType {
		pipelineIdsMap[v.PipelineId] = v
	}

	return pipelineIdsMap, nil
}

func (impl *PipelineRepositoryImpl) UpdateOldCiPipelineIdToNewCiPipelineId(tx *pg.Tx, oldCiPipelineId, newCiPipelineId int) error {
	newCiPipId := pointer.Int(newCiPipelineId)
	if newCiPipelineId == 0 {
		newCiPipId = nil
	}
	_, err := tx.Model((*Pipeline)(nil)).Set("ci_pipeline_id = ?", newCiPipId).
		Where("ci_pipeline_id = ? ", oldCiPipelineId).
		Where("deleted = ?", false).Update()
	return err
}

func (impl *PipelineRepositoryImpl) UpdateCiPipelineId(tx *pg.Tx, pipelineIds []int, ciPipelineId int) error {
	if len(pipelineIds) == 0 {
		return nil
	}
	_, err := tx.Model((*Pipeline)(nil)).Set("ci_pipeline_id = ?", ciPipelineId).
		Where("id IN (?) ", pg.In(pipelineIds)).
		Where("deleted = ?", false).Update()
	return err
}

func (impl *PipelineRepositoryImpl) FindWithEnvironmentByCiIds(ctx context.Context, cIPipelineIds []int) ([]*Pipeline, error) {
	_, span := otel.Tracer("orchestrator").Start(ctx, "FindWithEnvironmentByCiIds")
	defer span.End()
	var cDPipelines []*Pipeline
	err := impl.dbConnection.Model(&cDPipelines).
		Column("pipeline.*", "Environment").
		Where("ci_pipeline_id in (?)", pg.In(cIPipelineIds)).
		Select()
	if err != nil {
		return nil, err
	}
	return cDPipelines, nil
}

func (impl *PipelineRepositoryImpl) FindDeploymentAppTypeByAppIdAndEnvId(appId, envId int) (string, error) {
	var deploymentAppType string
	err := impl.dbConnection.Model((*Pipeline)(nil)).
		Column("deployment_app_type").
		Where("app_id = ? and environment_id=? and deleted=false", appId, envId).
		Select(&deploymentAppType)
	return deploymentAppType, err
}

func (impl *PipelineRepositoryImpl) FindByAppIdToEnvIdsMapping(appIdToEnvIds map[int][]int) ([]*Pipeline, error) {
	var pipelines []*Pipeline
	err := impl.dbConnection.Model(&pipelines).
		WhereGroup(func(query *orm.Query) (*orm.Query, error) {
			for appId, envIds := range appIdToEnvIds {
				if len(envIds) == 0 {
					continue
				}
				query = query.WhereOr("app_id = ? and environment_id in (?) and deleted=false", appId, pg.In(envIds))
			}
			return query, nil
		}).
		Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) FindDeploymentAppTypeByIds(ids []int) (pipelines []*Pipeline, err error) {
	err = impl.dbConnection.Model(&pipelines).Column("id", "app_id", "env_id", "deployment_app_type").
		Where("id in (?)", pg.In(ids)).Where("deleted = ?", false).Select()
	return pipelines, err
}

func (impl *PipelineRepositoryImpl) GetAllAppsByClusterAndDeploymentAppType(clusterIds []int, deploymentAppName string) ([]*PipelineDeploymentConfigObj, error) {
	result := make([]*PipelineDeploymentConfigObj, 0)
	if len(clusterIds) == 0 {
		return result, nil
	}
	err := impl.dbConnection.Model().
		Table("pipeline").
		ColumnExpr("pipeline.deployment_app_name AS deployment_app_name").
		ColumnExpr("pipeline.app_id AS app_id").
		ColumnExpr("pipeline.environment_id AS environment_id").
		ColumnExpr("environment.cluster_id AS cluster_id").
		ColumnExpr("environment.namespace AS namespace").
		// inner join with app
		Join("INNER JOIN app").
		JoinOn("pipeline.app_id = app.id").
		// inner join with environment
		Join("INNER JOIN environment").
		JoinOn("pipeline.environment_id = environment.id").
		// left join with deployment_config
		Join("LEFT JOIN deployment_config").
		JoinOn("pipeline.app_id = deployment_config.app_id").
		JoinOn("pipeline.environment_id = deployment_config.environment_id").
		JoinOn("deployment_config.active = ?", true).
		// where conditions
		Where("environment.cluster_id in (?)", pg.In(clusterIds)).
		Where("pipeline.deleted = ?", false).
		Where("app.active = ?", true).
		Where("environment.active = ?", true).
		WhereGroup(func(query *orm.Query) (*orm.Query, error) {
			return query.WhereOr("pipeline.deployment_app_type = ?", deploymentAppName).
				WhereOr("deployment_config.deployment_app_type = ?", deploymentAppName), nil
		}).
		Select(&result)
	return result, err
}

func (impl *PipelineRepositoryImpl) GetAllArgoAppInfoByDeploymentAppNames(deploymentAppNames []string) ([]*PipelineDeploymentConfigObj, error) {
	result := make([]*PipelineDeploymentConfigObj, 0)
	if len(deploymentAppNames) == 0 {
		return result, nil
	}
	err := impl.dbConnection.Model().
		Table("pipeline").
		ColumnExpr("pipeline.deployment_app_name AS deployment_app_name").
		ColumnExpr("pipeline.app_id AS app_id").
		ColumnExpr("pipeline.environment_id AS environment_id").
		ColumnExpr("environment.cluster_id AS cluster_id").
		ColumnExpr("environment.namespace AS namespace").
		// inner join with app
		Join("INNER JOIN app").
		JoinOn("pipeline.app_id = app.id").
		// inner join with environment
		Join("INNER JOIN environment").
		JoinOn("pipeline.environment_id = environment.id").
		// left join with deployment_config
		Join("LEFT JOIN deployment_config").
		JoinOn("pipeline.app_id = deployment_config.app_id").
		JoinOn("pipeline.environment_id = deployment_config.environment_id").
		JoinOn("deployment_config.active = ?", true).
		// where conditions
		Where("pipeline.deployment_app_name in (?)", pg.In(deploymentAppNames)).
		Where("pipeline.deleted = ?", false).
		Where("app.active = ?", true).
		Where("environment.active = ?", true).
		WhereGroup(func(query *orm.Query) (*orm.Query, error) {
			return query.WhereOr("pipeline.deployment_app_type = ?", util.PIPELINE_DEPLOYMENT_TYPE_ACD).
				WhereOr("deployment_config.deployment_app_type = ?", util.PIPELINE_DEPLOYMENT_TYPE_ACD), nil
		}).
		Select(&result)
	return result, err
}

func (impl *PipelineRepositoryImpl) FindEnvIdsByIdsInIncludingDeleted(ids []int) ([]int, error) {
	var envIds []int
	if len(ids) == 0 {
		return envIds, nil
	}
	err := impl.dbConnection.Model(&Pipeline{}).
		Column("pipeline.environment_id").
		Where("pipeline.id in (?)", pg.In(ids)).
		Select(&envIds)
	if err != nil {
		impl.logger.Errorw("error on fetching pipelines", "ids", ids, "err", err)
	}
	return envIds, err
}

func (impl *PipelineRepositoryImpl) GetPipelineCountByDeploymentType(deploymentType string) (int, error) {
	var count int
	// Count pipelines by deployment type, considering both pipeline table and deployment_config table
	// The deployment_config table can override the deployment type from the pipeline table
	query := `
		SELECT COUNT(DISTINCT p.id)
		FROM pipeline p
		LEFT JOIN deployment_config dc ON dc.active = true
			AND dc.app_id = p.app_id
			AND dc.environment_id = p.environment_id
		WHERE p.deleted = false
			AND (p.deployment_app_type = ? OR dc.deployment_app_type = ?)
	`
	_, err := impl.dbConnection.Query(&count, query, deploymentType, deploymentType)
	if err != nil {
		impl.logger.Errorw("error getting pipeline count by deployment type", "deploymentType", deploymentType, "err", err)
		return 0, err
	}
	return count, nil
}
