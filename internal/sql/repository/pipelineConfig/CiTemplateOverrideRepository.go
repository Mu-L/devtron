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

package pipelineConfig

import (
	"github.com/devtron-labs/devtron/internal/sql/repository/dockerRegistry"
	repository2 "github.com/devtron-labs/devtron/pkg/build/git/gitMaterial/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"go.uber.org/zap"
)

type CiTemplateOverride struct {
	tableName                 struct{} `sql:"ci_template_override" pg:",discard_unknown_columns"`
	Id                        int      `sql:"id"`
	CiPipelineId              int      `sql:"ci_pipeline_id"`
	DockerRegistryId          string   `sql:"docker_registry_id"`
	DockerRepository          string   `sql:"docker_repository"`
	DockerfilePath            string   `sql:"dockerfile_path"`
	GitMaterialId             int      `sql:"git_material_id"`
	BuildContextGitMaterialId int      `sql:"build_context_git_material_id"`
	Active                    bool     `sql:"active,notnull"`
	CiBuildConfigId           int      `sql:"ci_build_config_id"`
	sql.AuditLog
	GitMaterial    *repository2.GitMaterial
	DockerRegistry *repository.DockerArtifactStore
	CiBuildConfig  *CiBuildConfig
}

type CiTemplateOverrideRepository interface {
	Save(templateOverrideConfig *CiTemplateOverride) (*CiTemplateOverride, error)
	Update(templateOverrideConfig *CiTemplateOverride) (*CiTemplateOverride, error)
	FindByAppId(appId int) ([]*CiTemplateOverride, error)
	FindByCiPipelineIds(ciPipelineIds []int) ([]*CiTemplateOverride, error)
	FindByCiPipelineId(ciPipelineId int) (*CiTemplateOverride, error)
	FindIfTemplateOverrideExistsByCiPipelineIdsAndGitMaterialId(ciPipelineIds []int, gitMaterialId int) (bool, error)
}

type CiTemplateOverrideRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
}

func NewCiTemplateOverrideRepositoryImpl(dbConnection *pg.DB,
	logger *zap.SugaredLogger) *CiTemplateOverrideRepositoryImpl {
	return &CiTemplateOverrideRepositoryImpl{
		dbConnection: dbConnection,
		logger:       logger,
	}
}

func (repo *CiTemplateOverrideRepositoryImpl) Save(templateOverrideConfig *CiTemplateOverride) (*CiTemplateOverride, error) {
	err := repo.dbConnection.Insert(templateOverrideConfig)
	if err != nil {
		repo.logger.Errorw("error in saving templateOverrideConfig", "err", err)
		return nil, err
	}
	return templateOverrideConfig, nil
}

func (repo *CiTemplateOverrideRepositoryImpl) Update(templateOverrideConfig *CiTemplateOverride) (*CiTemplateOverride, error) {
	err := repo.dbConnection.Update(templateOverrideConfig)
	if err != nil {
		repo.logger.Errorw("error in updating templateOverrideConfig", "err", err)
		return nil, err
	}
	return templateOverrideConfig, nil
}

func (repo *CiTemplateOverrideRepositoryImpl) FindByAppId(appId int) ([]*CiTemplateOverride, error) {
	var ciTemplateOverrides []*CiTemplateOverride
	err := repo.dbConnection.Model(&ciTemplateOverrides).
		Column("ci_template_override.*", "CiBuildConfig").
		Join("INNER JOIN ci_pipeline cp on cp.id=ci_template_override.ci_pipeline_id").
		Join("INNER JOIN ci_build_config cbc on cbc.id=ci_template_override.ci_build_config_id").
		Where("app_id = ?", appId).
		Where("is_docker_config_overridden = ?", true).
		Where("ci_template_override.active = ?", true).
		Where("cp.deleted = ?", false).
		Select()
	if err != nil {
		repo.logger.Errorw("error in getting ciTemplateOverride by appId", "err", err, "appId", appId)
		return nil, err
	}
	return ciTemplateOverrides, nil
}

func (repo *CiTemplateOverrideRepositoryImpl) FindByCiPipelineIds(ciPipelineIds []int) ([]*CiTemplateOverride, error) {
	var ciTemplateOverrides []*CiTemplateOverride
	err := repo.dbConnection.Model(&ciTemplateOverrides).
		Column("ci_template_override.*", "CiBuildConfig").
		Where("ci_template_override.ci_pipeline_id in (?)", pg.In(ciPipelineIds)).
		Where("ci_template_override.active = ?", true).
		Select()
	if err != nil {
		repo.logger.Errorw("error in getting ciTemplateOverride by appId", "err", err, "ciPipelineIds", ciPipelineIds)
		return nil, err
	}
	return ciTemplateOverrides, nil
}

func (repo *CiTemplateOverrideRepositoryImpl) FindByCiPipelineId(ciPipelineId int) (*CiTemplateOverride, error) {
	ciTemplateOverride := &CiTemplateOverride{}
	err := repo.dbConnection.Model(ciTemplateOverride).
		Column("ci_template_override.*", "GitMaterial", "DockerRegistry", "CiBuildConfig").
		Where("ci_pipeline_id = ?", ciPipelineId).
		Where("ci_template_override.active = ?", true).
		Select()
	if err != nil {
		repo.logger.Errorw("error in getting ciTemplateOverride by ciPipelineId", "err", err, "ciPipelineId", ciPipelineId)
		return ciTemplateOverride, err
	}
	return ciTemplateOverride, nil
}

func (repo *CiTemplateOverrideRepositoryImpl) FindIfTemplateOverrideExistsByCiPipelineIdsAndGitMaterialId(ciPipelineIds []int, gitMaterialId int) (bool, error) {
	if len(ciPipelineIds) == 0 {
		return false, nil
	}
	count, err := repo.dbConnection.Model((*CiTemplateOverride)(nil)).
		Where("ci_pipeline_id in (?)", pg.In(ciPipelineIds)).
		WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			return q.Where("git_material_id = ?", gitMaterialId).WhereOr("build_context_git_material_id = ?", gitMaterialId), nil
		}).
		Where("active = ?", true).
		Count()
	if err != nil {
		repo.logger.Errorw("error in checking if template override exists", "ciPipelineIds", ciPipelineIds, "gitMaterialId", gitMaterialId, "err", err)
		return false, err
	}
	return count > 0, nil

}
