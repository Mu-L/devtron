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

package app

import (
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	repository2 "github.com/devtron-labs/devtron/pkg/appStore/deployment/repository"
	"github.com/devtron-labs/devtron/pkg/bean"
	"github.com/devtron-labs/devtron/pkg/user/repository"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/validation"
	"strconv"
	"strings"
	"time"
)

type AppCrudOperationService interface {
	Create(request *bean.AppLabelDto, tx *pg.Tx) (*bean.AppLabelDto, error)
	FindById(id int) (*bean.AppLabelDto, error)
	FindAll() ([]*bean.AppLabelDto, error)
	GetAppMetaInfo(appId int) (*bean.AppMetaInfoDto, error)
	GetHelmAppMetaInfo(appId string) (*bean.AppMetaInfoDto, error)
	GetLabelsByAppIdForDeployment(appId int) ([]byte, error)
	GetLabelsByAppId(appId int) (map[string]string, error)
	UpdateApp(request *bean.CreateAppDTO) (*bean.CreateAppDTO, error)
	UpdateProjectForApps(request *bean.UpdateProjectBulkAppsRequest) (*bean.UpdateProjectBulkAppsRequest, error)
	GetAppMetaInfoByAppName(appName string) (*bean.AppMetaInfoDto, error)
	GetAppListByTeamIds(teamIds []int, appType string) ([]*TeamAppBean, error)
}
type AppCrudOperationServiceImpl struct {
	logger                 *zap.SugaredLogger
	appLabelRepository     pipelineConfig.AppLabelRepository
	appRepository          app.AppRepository
	userRepository         repository.UserRepository
	installedAppRepository repository2.InstalledAppRepository
}

func NewAppCrudOperationServiceImpl(appLabelRepository pipelineConfig.AppLabelRepository,
	logger *zap.SugaredLogger, appRepository app.AppRepository, userRepository repository.UserRepository, installedAppRepository repository2.InstalledAppRepository) *AppCrudOperationServiceImpl {
	return &AppCrudOperationServiceImpl{
		appLabelRepository:     appLabelRepository,
		logger:                 logger,
		appRepository:          appRepository,
		userRepository:         userRepository,
		installedAppRepository: installedAppRepository,
	}
}

type AppBean struct {
	Id     int    `json:"id"`
	Name   string `json:"name,notnull"`
	TeamId int    `json:"teamId,omitempty"`
}

type TeamAppBean struct {
	ProjectId   int        `json:"projectId"`
	ProjectName string     `json:"projectName"`
	AppList     []*AppBean `json:"appList"`
}

func (impl AppCrudOperationServiceImpl) UpdateApp(request *bean.CreateAppDTO) (*bean.CreateAppDTO, error) {
	dbConnection := impl.appRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return nil, err
	}
	// Rollback tx on error.
	defer tx.Rollback()

	app, err := impl.appRepository.FindById(request.Id)
	if err != nil {
		impl.logger.Errorw("error in fetching app", "error", err)
		return nil, err
	}
	app.TeamId = request.TeamId
	app.UpdatedOn = time.Now()
	app.UpdatedBy = request.UserId
	err = impl.appRepository.Update(app)
	if err != nil {
		impl.logger.Errorw("error in updating app", "error", err)
		return nil, err
	}

	_, err = impl.UpdateLabelsInApp(request, tx)
	if err != nil {
		impl.logger.Errorw("error in updating app labels", "error", err)
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("error in commit db transaction", "error", err)
		return nil, err
	}

	return request, nil
}

func (impl AppCrudOperationServiceImpl) UpdateProjectForApps(request *bean.UpdateProjectBulkAppsRequest) (*bean.UpdateProjectBulkAppsRequest, error) {
	dbConnection := impl.appRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return nil, err
	}
	// Rollback tx on error.
	defer tx.Rollback()
	apps, err := impl.appRepository.FindAppsByTeamId(request.TeamId)
	if err != nil {
		impl.logger.Errorw("error in fetching apps", "error", err)
		return nil, err
	}
	for _, app := range apps {
		app.TeamId = request.TeamId
		app.UpdatedOn = time.Now()
		app.UpdatedBy = request.UserId
		err = impl.appRepository.Update(app)
		if err != nil {
			impl.logger.Errorw("error in updating app", "error", err)
			return nil, err
		}
	}
	err = tx.Commit()
	if err != nil {
		impl.logger.Errorw("error in commit db transaction", "error", err)
		return nil, err
	}
	return nil, nil
}

func (impl AppCrudOperationServiceImpl) Create(request *bean.AppLabelDto, tx *pg.Tx) (*bean.AppLabelDto, error) {
	_, err := impl.appLabelRepository.FindByAppIdAndKeyAndValue(request.AppId, request.Key, request.Value)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching app label", "error", err)
		return nil, err
	}
	if err == pg.ErrNoRows {
		model := &pipelineConfig.AppLabel{
			Key:   request.Key,
			Value: request.Value,
			AppId: request.AppId,
		}
		model.CreatedBy = request.UserId
		model.UpdatedBy = request.UserId
		model.CreatedOn = time.Now()
		model.UpdatedOn = time.Now()
		_, err = impl.appLabelRepository.Create(model, tx)
		if err != nil {
			impl.logger.Errorw("error in creating new app labels", "error", err)
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("duplicate key found for app %d, %s", request.AppId, request.Key)
	}
	return request, nil
}

func (impl AppCrudOperationServiceImpl) UpdateLabelsInApp(request *bean.CreateAppDTO, tx *pg.Tx) (*bean.CreateAppDTO, error) {
	appLabels, err := impl.appLabelRepository.FindAllByAppId(request.Id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching app label", "error", err)
		return nil, err
	}
	appLabelMap := make(map[string]*pipelineConfig.AppLabel)
	for _, appLabel := range appLabels {
		uniqueLabelExists := fmt.Sprintf("%s:%s", appLabel.Key, appLabel.Value)
		if _, ok := appLabelMap[uniqueLabelExists]; !ok {
			appLabelMap[uniqueLabelExists] = appLabel
		}
	}

	for _, label := range request.AppLabels {
		uniqueLabelRequest := fmt.Sprintf("%s:%s", label.Key, label.Value)
		if _, ok := appLabelMap[uniqueLabelRequest]; !ok {
			// create new
			model := &pipelineConfig.AppLabel{
				Key:   label.Key,
				Value: label.Value,
				AppId: request.Id,
			}
			model.CreatedBy = request.UserId
			model.UpdatedBy = request.UserId
			model.CreatedOn = time.Now()
			model.UpdatedOn = time.Now()
			_, err = impl.appLabelRepository.Create(model, tx)
			if err != nil {
				impl.logger.Errorw("error in creating new app labels", "error", err)
				return nil, err
			}
		} else {
			// delete from map so that item remain live, all other item will be delete from this app
			delete(appLabelMap, uniqueLabelRequest)
		}
	}
	for _, appLabel := range appLabelMap {
		err = impl.appLabelRepository.Delete(appLabel, tx)
		if err != nil {
			impl.logger.Errorw("error in delete app label", "error", err)
			return nil, err
		}
	}
	return request, nil
}

func (impl AppCrudOperationServiceImpl) FindById(id int) (*bean.AppLabelDto, error) {
	model, err := impl.appLabelRepository.FindById(id)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching app labels", "error", err)
		return nil, err
	}
	if err == pg.ErrNoRows {
		return &bean.AppLabelDto{}, nil
	}
	label := &bean.AppLabelDto{
		Key:   model.Key,
		Value: model.Value,
		AppId: model.AppId,
	}
	return label, nil
}

func (impl AppCrudOperationServiceImpl) FindAll() ([]*bean.AppLabelDto, error) {
	results := make([]*bean.AppLabelDto, 0)
	models, err := impl.appLabelRepository.FindAll()
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching FindAll app labels", "error", err)
		return nil, err
	}
	if err == pg.ErrNoRows {
		return results, nil
	}
	for _, model := range models {
		dto := &bean.AppLabelDto{
			AppId: model.AppId,
			Key:   model.Key,
			Value: model.Value,
		}
		results = append(results, dto)
	}
	return results, nil
}

func (impl AppCrudOperationServiceImpl) GetAppMetaInfo(appId int) (*bean.AppMetaInfoDto, error) {
	app, err := impl.appRepository.FindAppAndProjectByAppId(appId)
	if err != nil {
		impl.logger.Errorw("error in fetching GetAppMetaInfo", "error", err)
		return nil, err
	}
	labels := make([]*bean.Label, 0)
	models, err := impl.appLabelRepository.FindAllByAppId(appId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching GetAppMetaInfo", "error", err)
		return nil, err
	}
	if err == pg.ErrNoRows {
		impl.logger.Infow("no labels found for app", "app", app)
	} else {
		for _, model := range models {
			dto := &bean.Label{
				Key:   model.Key,
				Value: model.Value,
			}
			labels = append(labels, dto)
		}
	}

	user, err := impl.userRepository.GetByIdIncludeDeleted(app.CreatedBy)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching user for app meta info", "error", err)
		return nil, err
	}
	userEmailId := ""
	if user != nil && user.Id > 0 {
		if user.Active {
			userEmailId = fmt.Sprintf(user.EmailId)
		} else {
			userEmailId = fmt.Sprintf("%s (inactive)", user.EmailId)
		}
	}
	info := &bean.AppMetaInfoDto{
		AppId:       app.Id,
		AppName:     app.AppName,
		ProjectId:   app.TeamId,
		ProjectName: app.Team.Name,
		CreatedBy:   userEmailId,
		CreatedOn:   app.CreatedOn,
		Labels:      labels,
		Active:      app.Active,
	}
	return info, nil
}

func (impl AppCrudOperationServiceImpl) GetHelmAppMetaInfo(appId string) (*bean.AppMetaInfoDto, error) {

	// adding separate function for helm apps because for CLI helm apps, apps can be of form "1|clusterName|releaseName"
	// In this case app details can be fetched using app name / release Name.

	appIdSplitted := strings.Split(appId, "|")

	app := &app.App{}
	var err error

	if len(appIdSplitted) > 1 {
		appName := appIdSplitted[2]

		app, err = impl.appRepository.FindAppAndProjectByAppName(appName)

		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in fetching app meta data", "err", err)
			return nil, err
		}

		if app.Id == 0 {
			app.AppName = appName
		}

	} else {

		appIdInt, err := strconv.Atoi(appId)

		if err != nil {
			impl.logger.Errorw("error in converting appId to integer", "err", err)
			return nil, err
		}

		installedAppVersion, err := impl.installedAppRepository.GetInstalledAppVersion(appIdInt)

		app.Id = installedAppVersion.InstalledApp.AppId
		app.AppName = installedAppVersion.InstalledApp.App.AppName
		app.TeamId = installedAppVersion.InstalledApp.App.TeamId
		app.Team.Name = installedAppVersion.InstalledApp.App.Team.Name
		app.CreatedBy = installedAppVersion.InstalledApp.App.CreatedBy
		app.Active = installedAppVersion.InstalledApp.App.Active

		if err != nil {
			impl.logger.Errorw("error in fetching App Meta Info", "error", err)
			return nil, err
		}
	}

	user, err := impl.userRepository.GetByIdIncludeDeleted(app.CreatedBy)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in fetching user for app meta info", "error", err)
		return nil, err
	}
	userEmailId := ""
	if user != nil && user.Id > 0 {
		if user.Active {
			userEmailId = fmt.Sprintf(user.EmailId)
		} else {
			userEmailId = fmt.Sprintf("%s (inactive)", user.EmailId)
		}
	}
	info := &bean.AppMetaInfoDto{
		AppId:       app.Id,
		AppName:     app.AppName,
		ProjectId:   app.TeamId,
		ProjectName: app.Team.Name,
		CreatedBy:   userEmailId,
		CreatedOn:   app.CreatedOn,
		Active:      app.Active,
	}
	return info, nil
}

func (impl AppCrudOperationServiceImpl) GetLabelsByAppIdForDeployment(appId int) ([]byte, error) {
	appLabelJson := &bean.AppLabelsJsonForDeployment{}
	labels, err := impl.appLabelRepository.FindAllByAppId(appId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting app labels by appId", "err", err, "appId", appId)
		return nil, err
	}
	labelsDto := make(map[string]string)
	for _, label := range labels {
		labelKey := strings.TrimSpace(label.Key)
		labelValue := strings.TrimSpace(label.Value)

		// if labelKey or labelValue is empty then don't add in labels
		if len(labelKey) == 0 || len(labelValue) == 0 {
			impl.logger.Warnw("Ignoring label to propagate to app level", "labelKey", labelKey, "labelValue", labelValue, "appId", appId)
			continue
		}

		// if labelKey is not satisfying the label key criteria don't add in labels
		// label key must be a 'qualified name' (https://github.com/kubernetes/website/issues/17969)
		errs := validation.IsQualifiedName(labelKey)
		if len(errs) > 0 {
			impl.logger.Warnw("Ignoring label to propagate to app level", "message", fmt.Sprintf("Validation error - label key - %s is not satisfying the label key criteria", labelKey), "appId", appId)
			continue
		}

		// if labelValue is not satisfying the label value criteria don't add in labels
		errs = validation.IsValidLabelValue(labelValue)
		if len(errs) > 0 {
			impl.logger.Warnw("Ignoring label to propagate to app level", "message", fmt.Sprintf("Validation error - label value - %s is not satisfying the label value criteria", labelValue), "appId", appId)
			continue
		}

		labelsDto[labelKey] = labelValue
	}
	appLabelJson.Labels = labelsDto
	appLabelByte, err := json.Marshal(appLabelJson)
	if err != nil {
		impl.logger.Errorw("error in marshaling appLabels json", "err", err, "appLabelJson", appLabelJson)
		return nil, err
	}
	return appLabelByte, nil
}
func (impl AppCrudOperationServiceImpl) GetLabelsByAppId(appId int) (map[string]string, error) {
	labels, err := impl.appLabelRepository.FindAllByAppId(appId)
	if err != nil {
		if err != pg.ErrNoRows {
			impl.logger.Errorw("error in getting app labels by appId", "err", err, "appId", appId)
			return nil, err
		} else {
			return nil, nil
		}
	}
	labelsDto := make(map[string]string)
	for _, label := range labels {
		labelsDto[label.Key] = label.Value
	}
	return labelsDto, nil
}

func (impl AppCrudOperationServiceImpl) GetAppMetaInfoByAppName(appName string) (*bean.AppMetaInfoDto, error) {
	app, err := impl.appRepository.FindAppAndProjectByAppName(appName)
	if err != nil {
		impl.logger.Errorw("error in fetching GetAppMetaInfoByAppName", "error", err)
		return nil, err
	}
	info := &bean.AppMetaInfoDto{
		AppId:       app.Id,
		AppName:     app.AppName,
		ProjectId:   app.TeamId,
		ProjectName: app.Team.Name,
		CreatedOn:   app.CreatedOn,
		Active:      app.Active,
	}
	return info, nil
}

func (impl AppCrudOperationServiceImpl) GetAppListByTeamIds(teamIds []int, appType string) ([]*TeamAppBean, error) {
	var appsRes []*TeamAppBean
	teamMap := make(map[int]*TeamAppBean)
	if len(teamIds) == 0 {
		return appsRes, nil
	}
	apps, err := impl.appRepository.FindAppsByTeamIds(teamIds, appType)
	if err != nil {
		impl.logger.Errorw("error while fetching app", "err", err)
		return nil, err
	}
	for _, app := range apps {
		if _, ok := teamMap[app.TeamId]; ok {
			teamMap[app.TeamId].AppList = append(teamMap[app.TeamId].AppList, &AppBean{Id: app.Id, Name: app.AppName})
		} else {

			teamMap[app.TeamId] = &TeamAppBean{ProjectId: app.Team.Id, ProjectName: app.Team.Name}
			teamMap[app.TeamId].AppList = append(teamMap[app.TeamId].AppList, &AppBean{Id: app.Id, Name: app.AppName})
		}
	}

	for _, v := range teamMap {
		if len(v.AppList) == 0 {
			v.AppList = make([]*AppBean, 0)
		}
		appsRes = append(appsRes, v)
	}

	if len(appsRes) == 0 {
		appsRes = make([]*TeamAppBean, 0)
	}

	return appsRes, err
}
