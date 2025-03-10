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

package deploymentConfig

import (
	"github.com/devtron-labs/devtron/internal/sql/repository/helper"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
)

type DeploymentAppType string

const (
	Argo DeploymentAppType = "argo_cd"
	Helm DeploymentAppType = "helm"
)

type ConfigType string

const (
	Custom          ConfigType = "custom"
	SystemGenerated            = "system_generated"
)

type DeploymentConfig struct {
	tableName         struct{} `sql:"deployment_config" pg:",discard_unknown_columns"`
	Id                int      `sql:"id,pk"`
	AppId             int      `sql:"app_id"`
	EnvironmentId     int      `sql:"environment_id"`
	DeploymentAppType string   `sql:"deployment_app_type"`
	ConfigType        string   `sql:"config_type"`
	RepoUrl           string   `sql:"repo_url"`
	RepoName          string   `sql:"repo_name"`
	ReleaseMode       string   `sql:"release_mode"`
	Active            bool     `sql:"active,notnull"`
	sql.AuditLog
}

type Repository interface {
	Save(tx *pg.Tx, config *DeploymentConfig) (*DeploymentConfig, error)
	SaveAll(tx *pg.Tx, configs []*DeploymentConfig) ([]*DeploymentConfig, error)
	Update(tx *pg.Tx, config *DeploymentConfig) (*DeploymentConfig, error)
	UpdateAll(tx *pg.Tx, config []*DeploymentConfig) ([]*DeploymentConfig, error)
	GetById(id int) (*DeploymentConfig, error)
	GetByAppIdAndEnvId(appId, envId int) (*DeploymentConfig, error)
	GetAppLevelConfigForDevtronApps(appId int) (*DeploymentConfig, error)
	GetAppLevelConfigByAppIds(appIds []int) ([]*DeploymentConfig, error)
	GetAppAndEnvLevelConfigsInBulk(appIdToEnvIdsMap map[int][]int) ([]*DeploymentConfig, error)
	GetByAppIdAndEnvIdEvenIfInactive(appId, envId int) (*DeploymentConfig, error)
	UpdateRepoUrlByAppIdAndEnvId(repoUrl string, appId, envId int) error
	GetConfigByAppIds(appIds []int) ([]*DeploymentConfig, error)
	GetDeploymentAppTypeForChartStoreAppByAppId(appId int) (string, error)
}

type RepositoryImpl struct {
	dbConnection *pg.DB
}

func NewRepositoryImpl(dbConnection *pg.DB) *RepositoryImpl {
	return &RepositoryImpl{dbConnection: dbConnection}
}

func (impl *RepositoryImpl) Save(tx *pg.Tx, config *DeploymentConfig) (*DeploymentConfig, error) {
	var err error
	if tx != nil {
		err = tx.Insert(config)
	} else {
		err = impl.dbConnection.Insert(config)
	}
	return config, err
}

func (impl *RepositoryImpl) SaveAll(tx *pg.Tx, configs []*DeploymentConfig) ([]*DeploymentConfig, error) {
	var err error
	if tx != nil {
		err = tx.Insert(&configs)
	} else {
		err = impl.dbConnection.Insert(&configs)
	}
	return configs, err
}

func (impl *RepositoryImpl) Update(tx *pg.Tx, config *DeploymentConfig) (*DeploymentConfig, error) {
	var err error
	if tx != nil {
		err = tx.Update(config)
	} else {
		err = impl.dbConnection.Update(config)
	}
	return config, err
}

func (impl *RepositoryImpl) UpdateAll(tx *pg.Tx, configs []*DeploymentConfig) ([]*DeploymentConfig, error) {
	var err error
	for _, config := range configs {
		if tx != nil {
			_, err = tx.Model(config).WherePK().UpdateNotNull()
		} else {
			_, err = impl.dbConnection.Model(&config).UpdateNotNull()
		}
		if err != nil {
			return nil, err
		}
	}
	return configs, err
}

func (impl *RepositoryImpl) GetById(id int) (*DeploymentConfig, error) {
	result := &DeploymentConfig{}
	err := impl.dbConnection.Model(result).Where("id = ?", id).Where("active = ?", true).Select()
	return result, err
}

func (impl *RepositoryImpl) GetByAppIdAndEnvId(appId, envId int) (*DeploymentConfig, error) {
	result := &DeploymentConfig{}
	err := impl.dbConnection.Model(result).
		Where("app_id = ?", appId).
		Where("environment_id = ? ", envId).
		Where("active = ?", true).
		Order("id DESC").Limit(1).
		Select()
	return result, err
}

func (impl *RepositoryImpl) GetAppLevelConfigForDevtronApps(appId int) (*DeploymentConfig, error) {
	result := &DeploymentConfig{}
	err := impl.dbConnection.Model(result).
		Where("app_id = ? ", appId).
		Where("environment_id is NULL").
		Where("active = ?", true).
		Select()
	return result, err
}

func (impl *RepositoryImpl) GetAppLevelConfigByAppIds(appIds []int) ([]*DeploymentConfig, error) {
	var result []*DeploymentConfig
	err := impl.dbConnection.Model(&result).
		Where("app_id in (?) and environment_id is NULL ", pg.In(appIds)).
		Where("active = ?", true).
		Select()
	return result, err
}

func (impl *RepositoryImpl) GetAppAndEnvLevelConfigsInBulk(appIdToEnvIdsMap map[int][]int) ([]*DeploymentConfig, error) {
	var result []*DeploymentConfig
	err := impl.dbConnection.Model(&result).
		WhereOrGroup(func(query *orm.Query) (*orm.Query, error) {
			for appId, envIds := range appIdToEnvIdsMap {
				if len(envIds) == 0 {
					continue
				}
				query = query.Where("app_id = ?", appId).Where("environment_id in (?)", pg.In((envIds))).Where("active = ?", true)
			}
			return query, nil
		}).Select()
	return result, err
}

func (impl *RepositoryImpl) GetByAppIdAndEnvIdEvenIfInactive(appId, envId int) (*DeploymentConfig, error) {
	result := &DeploymentConfig{}
	err := impl.dbConnection.Model(result).
		WhereGroup(func(query *orm.Query) (*orm.Query, error) {
			query = query.Where("app_id = ?", appId)
			if envId == 0 {
				query = query.Where("environment_id is NULL")
			} else {
				query = query.Where("environment_id = ? ", envId)
			}
			return query, nil
		}).
		Order("id DESC").Limit(1).
		Select()
	return result, err
}

func (impl *RepositoryImpl) UpdateRepoUrlByAppIdAndEnvId(repoUrl string, appId, envId int) error {
	_, err := impl.dbConnection.
		Model((*DeploymentConfig)(nil)).
		Set("repo_url = ? ", repoUrl).
		Where("app_id = ? and environment_id = ? ", appId, envId).
		Update()
	return err
}

func (impl *RepositoryImpl) GetConfigByAppIds(appIds []int) ([]*DeploymentConfig, error) {
	var results []*DeploymentConfig
	err := impl.dbConnection.Model(&results).
		Where("app_id in (?) ", pg.In(appIds)).
		Where("active = ?", true).
		Select()
	return results, err
}

func (impl *RepositoryImpl) GetDeploymentAppTypeForChartStoreAppByAppId(appId int) (string, error) {
	result := &DeploymentConfig{}
	err := impl.dbConnection.Model(result).
		Join("inner join app a").
		JoinOn("deployment_config.app_id = a.id").
		Where("deployment_config.app_id = ? ", appId).
		Where("deployment_config.active = ?", true).
		Where("a.active = ?", true).
		Where("a.app_type = ? ", helper.ChartStoreApp).
		Select()
	return result.DeploymentAppType, err
}
