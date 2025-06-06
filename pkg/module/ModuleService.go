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

package module

import (
	"context"
	"errors"
	"github.com/caarlos0/env/v6"
	"github.com/devtron-labs/devtron/api/helm-app/gRPC"
	client "github.com/devtron-labs/devtron/api/helm-app/service"
	clientErrors "github.com/devtron-labs/devtron/pkg/errors"
	"github.com/devtron-labs/devtron/pkg/module/bean"
	moduleRepo "github.com/devtron-labs/devtron/pkg/module/repo"
	moduleUtil "github.com/devtron-labs/devtron/pkg/module/util"
	"github.com/devtron-labs/devtron/pkg/policyGovernance/security/scanTool"
	"github.com/devtron-labs/devtron/pkg/server"
	serverBean "github.com/devtron-labs/devtron/pkg/server/bean"
	serverEnvConfig "github.com/devtron-labs/devtron/pkg/server/config"
	serverDataStore "github.com/devtron-labs/devtron/pkg/server/store"
	util2 "github.com/devtron-labs/devtron/util"
	"github.com/go-pg/pg"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"reflect"
	"strings"
	"time"
)

type ModuleService interface {
	GetModuleInfo(name string) (*bean.ModuleInfoDto, error)
	GetModuleConfig(name string) (*bean.ModuleConfigDto, error)
	HandleModuleAction(userId int32, moduleName string, moduleActionRequest *bean.ModuleActionRequestDto) (*bean.ActionResponse, error)
	GetAllModuleInfo() ([]bean.ModuleInfoDto, error)
	EnableModule(moduleName, version string) (*bean.ActionResponse, error)
}

type ModuleServiceImpl struct {
	logger                         *zap.SugaredLogger
	serverEnvConfig                *serverEnvConfig.ServerEnvConfig
	moduleRepository               moduleRepo.ModuleRepository
	moduleActionAuditLogRepository ModuleActionAuditLogRepository
	helmAppService                 client.HelmAppService
	serverDataStore                *serverDataStore.ServerDataStore
	// no need to inject serverCacheService, moduleCacheService and cronService, but not generating in wire_gen (not triggering cache work in constructor) if not injecting. hence injecting
	// serverCacheService should be injected first as it changes serverEnvConfig in its constructor, which is used by moduleCacheService and moduleCronService
	serverCacheService             server.ServerCacheService
	moduleCacheService             ModuleCacheService
	moduleCronService              ModuleCronService
	moduleServiceHelper            ModuleServiceHelper
	moduleResourceStatusRepository moduleRepo.ModuleResourceStatusRepository
	scanToolMetadataService        scanTool.ScanToolMetadataService
	globalEnvVariables             *util2.GlobalEnvVariables
	moduleEnvConfig                *bean.ModuleEnvConfig
	installedModulesMap            map[string]bool
}

func NewModuleServiceImpl(logger *zap.SugaredLogger, serverEnvConfig *serverEnvConfig.ServerEnvConfig, moduleRepository moduleRepo.ModuleRepository,
	moduleActionAuditLogRepository ModuleActionAuditLogRepository, helmAppService client.HelmAppService, serverDataStore *serverDataStore.ServerDataStore, serverCacheService server.ServerCacheService, moduleCacheService ModuleCacheService, moduleCronService ModuleCronService,
	moduleServiceHelper ModuleServiceHelper, moduleResourceStatusRepository moduleRepo.ModuleResourceStatusRepository,
	scanToolMetadataService scanTool.ScanToolMetadataService, envVariables *util2.EnvironmentVariables, moduleEnvConfig *bean.ModuleEnvConfig) *ModuleServiceImpl {
	installedModulesMap := make(map[string]bool)
	for _, module := range moduleEnvConfig.InstalledModules {
		installedModulesMap[module] = true
	}
	return &ModuleServiceImpl{
		logger:                         logger,
		serverEnvConfig:                serverEnvConfig,
		moduleRepository:               moduleRepository,
		moduleActionAuditLogRepository: moduleActionAuditLogRepository,
		helmAppService:                 helmAppService,
		serverDataStore:                serverDataStore,
		serverCacheService:             serverCacheService,
		moduleCacheService:             moduleCacheService,
		moduleCronService:              moduleCronService,
		moduleServiceHelper:            moduleServiceHelper,
		moduleResourceStatusRepository: moduleResourceStatusRepository,
		scanToolMetadataService:        scanToolMetadataService,
		globalEnvVariables:             envVariables.GlobalEnvVariables,
		moduleEnvConfig:                moduleEnvConfig,
		installedModulesMap:            installedModulesMap,
	}
}

func (impl ModuleServiceImpl) GetModuleInfo(name string) (*bean.ModuleInfoDto, error) {
	impl.logger.Debugw("getting module info", "name", name)

	moduleInfoDto := &bean.ModuleInfoDto{
		Name: name,
	}

	// fetch from DB
	module, err := impl.moduleRepository.FindOne(name)
	if err != nil {
		if err == pg.ErrNoRows {
			status, moduleType, flagForMarkingActiveTool, err := impl.handleModuleNotFoundStatus(name)
			if err != nil {
				impl.logger.Errorw("error in handling module not found status ", "name", name, "err", err)
			}
			if flagForMarkingActiveTool {
				toolVersion := bean.TRIVY_V1
				if name == bean.ModuleNameSecurityClair {
					toolVersion = bean.CLAIR_V4
				}
				_, err = impl.EnableModule(name, toolVersion)
				if err != nil {
					impl.logger.Errorw("error in enabling module", "err", err, "module", name)
				}
			}
			moduleInfoDto.Status = status
			moduleInfoDto.Moduletype = moduleType
			return moduleInfoDto, err
		}
		// otherwise some error case
		impl.logger.Errorw("error in getting module from DB ", "name", name, "err", err)
		return nil, err
	}

	// now this is the case when data found in DB
	// if module is in installing state, then trigger module status check and override module model
	if module.Status == bean.ModuleStatusInstalling {
		impl.moduleCronService.HandleModuleStatusIfNotInProgress(module.Name)
		// override module model
		module, err = impl.moduleRepository.FindOne(name)
		if err != nil {
			impl.logger.Errorw("error in getting module from DB ", "name", name, "err", err)
			return nil, err
		}
	}
	// Handling for previous Modules
	flagForEnablingState := false
	if module.ModuleType != bean.MODULE_TYPE_SECURITY && module.Status == bean.ModuleStatusInstalled {
		flagForEnablingState = true
		err = impl.moduleRepository.MarkModuleAsEnabled(name)
		if err != nil {
			impl.logger.Errorw("error in updating module as active ", "moduleName", name, "err", err)
			return nil, err
		}
	}
	// send DB status
	moduleInfoDto.Status = module.Status
	// Enabled State Assignment
	moduleInfoDto.Enabled = module.Enabled || flagForEnablingState
	moduleInfoDto.Moduletype = module.ModuleType
	// handle module resources status data
	moduleId := module.Id
	moduleResourcesStatusFromDb, err := impl.moduleResourceStatusRepository.FindAllActiveByModuleId(moduleId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting module resources status from DB ", "moduleId", moduleId, "moduleName", name, "err", err)
		return nil, err
	}
	if moduleResourcesStatusFromDb != nil {
		var moduleResourcesStatus []*bean.ModuleResourceStatusDto
		for _, moduleResourceStatusFromDb := range moduleResourcesStatusFromDb {
			moduleResourcesStatus = append(moduleResourcesStatus, &bean.ModuleResourceStatusDto{
				Group:         moduleResourceStatusFromDb.Group,
				Version:       moduleResourceStatusFromDb.Version,
				Kind:          moduleResourceStatusFromDb.Kind,
				Name:          moduleResourceStatusFromDb.Name,
				HealthStatus:  moduleResourceStatusFromDb.HealthStatus,
				HealthMessage: moduleResourceStatusFromDb.HealthMessage,
			})
		}
		moduleInfoDto.ModuleResourcesStatus = moduleResourcesStatus
	}

	return moduleInfoDto, nil
}

func (impl ModuleServiceImpl) GetModuleConfig(name string) (*bean.ModuleConfigDto, error) {
	moduleConfig := &bean.ModuleConfigDto{}
	if name == bean.BlobStorage {
		blobStorageConfig := &bean.BlobStorageConfig{}
		env.Parse(blobStorageConfig)
		moduleConfig.Enabled = blobStorageConfig.Enabled
	}
	return moduleConfig, nil
}

func (impl ModuleServiceImpl) extractModuleTypeFromModule(moduleName string) (string, error) {
	var moduleType string
	if len(impl.installedModulesMap) > 0 {
		if strings.Contains(moduleName, bean.MODULE_TYPE_SECURITY) {
			moduleType = bean.MODULE_TYPE_SECURITY
		}
	} else {
		// central-api call
		moduleMetaData, err := impl.moduleServiceHelper.GetModuleMetadata(moduleName)
		if err != nil {
			impl.logger.Errorw("Error in getting module metadata", "moduleName", moduleName, "err", err)
			return "", err
		}
		moduleType = gjson.Get(string(moduleMetaData), "result.moduleType").String()
	}
	return moduleType, nil
}

func (impl ModuleServiceImpl) handleModuleNotFoundStatus(moduleName string) (bean.ModuleStatus, string, bool, error) {
	// if entry is not found in database, then check if that module is legacy or not
	// if enterprise user -> if legacy -> then mark as installed in db and return as installed, if not legacy -> return as not installed
	// if non-enterprise user->  fetch helm release enable Key. if true -> then mark as installed in db and return as installed. if false ->
	//// (continuation of above line) if legacy -> check if cicd is installed with <= 0.5.3 from DB and moduleName != argo-cd -> then mark as installed in db and return as installed. otherwise return as not installed
	moduleType, err := impl.extractModuleTypeFromModule(moduleName)
	if err != nil {
		impl.logger.Errorw("error in handleModuleNotFoundStatus", "moduleName", moduleName, "err", err)
		return bean.ModuleStatusNotInstalled, "", false, err
	}

	flagForEnablingState := false
	flagForActiveTool := false
	if moduleType == bean.MODULE_TYPE_SECURITY {
		err := impl.moduleRepository.FindByModuleTypeAndStatus(moduleType, bean.ModuleStatusInstalled)
		if err != nil {
			if err == pg.ErrNoRows {
				flagForEnablingState = true
				flagForActiveTool = true
			} else {
				impl.logger.Errorw("error in getting module by type", "moduleName", moduleName, "err", err)
				return bean.ModuleStatusNotInstalled, moduleType, false, err
			}
		}
	} else {
		flagForEnablingState = true
	}

	if len(impl.installedModulesMap) == 0 {
		devtronHelmAppIdentifier := impl.helmAppService.GetDevtronHelmAppIdentifier()
		releaseInfo, err := impl.helmAppService.GetValuesYaml(context.Background(), devtronHelmAppIdentifier)
		if err != nil {
			impl.logger.Errorw("Error in getting values yaml for devtron operator helm release", "moduleName", moduleName, "err", err)
			apiError := clientErrors.ConvertToApiError(err)
			if apiError != nil {
				err = apiError
			}
			return bean.ModuleStatusNotInstalled, moduleType, false, err
		}
		releaseValues := releaseInfo.MergedValues

		// if check non-cicd module status
		if moduleName != bean.ModuleNameCiCd {
			isEnabled := gjson.Get(releaseValues, moduleUtil.BuildModuleEnableKey(impl.serverEnvConfig.DevtronOperatorBasePath, moduleName)).Bool()
			if isEnabled {
				status, err := impl.saveModuleAsInstalled(moduleName, moduleType, flagForEnablingState)
				return status, moduleType, flagForActiveTool, err
			}
		} else if util2.IsBaseStack() {
			// check if cicd is in installing state
			// if devtron is installed with cicd module, then cicd module should be shown as installing
			installerModulesIface := gjson.Get(releaseValues, impl.serverEnvConfig.DevtronInstallerModulesPath).Value()
			if installerModulesIface != nil {
				installerModulesIfaceKind := reflect.TypeOf(installerModulesIface).Kind()
				if installerModulesIfaceKind == reflect.Slice {
					installerModules := installerModulesIface.([]interface{})
					for _, installerModule := range installerModules {
						if installerModule == moduleName {
							status, err := impl.saveModule(moduleName, bean.ModuleStatusInstalling, moduleType, flagForEnablingState)
							return status, moduleType, false, err
						}
					}
				} else {
					impl.logger.Warnw("Invalid installerModulesIfaceKind expected slice", "installerModulesIfaceKind", installerModulesIfaceKind, "val", installerModulesIface)
				}
			}
		}
	} else {
		if _, ok := impl.installedModulesMap[moduleName]; !ok {
			return bean.ModuleStatusNotInstalled, moduleType, false, nil
		}
		if moduleName != bean.ModuleNameCiCd {
			status, err := impl.saveModuleAsInstalled(moduleName, moduleType, flagForEnablingState)
			return status, moduleType, flagForActiveTool, err
		} else if util2.IsBaseStack() {
			status, err := impl.saveModule(moduleName, bean.ModuleStatusInstalling, moduleType, flagForEnablingState)
			return status, moduleType, false, err
		}
	}

	return bean.ModuleStatusNotInstalled, moduleType, false, nil

}

func (impl ModuleServiceImpl) HandleModuleAction(userId int32, moduleName string, moduleActionRequest *bean.ModuleActionRequestDto) (*bean.ActionResponse, error) {
	impl.logger.Debugw("handling module action request", "moduleName", moduleName, "userId", userId, "payload", moduleActionRequest)

	//check if can update server
	if impl.serverEnvConfig.DevtronInstallationType != serverBean.DevtronInstallationTypeOssHelm {
		return nil, errors.New("module installation is not allowed")
	}

	// insert into audit table
	moduleActionAuditLog := &ModuleActionAuditLog{
		ModuleName: moduleName,
		Version:    moduleActionRequest.Version,
		Action:     moduleActionRequest.Action,
		CreatedOn:  time.Now(),
		CreatedBy:  userId,
	}
	err := impl.moduleActionAuditLogRepository.Save(moduleActionAuditLog)
	if err != nil {
		impl.logger.Errorw("error in saving into audit log for module action ", "err", err)
		return nil, err
	}

	// get module by name
	// if error, throw error
	// if module not found, then insert entry
	// if module found, then update entry
	module, err := impl.moduleRepository.FindOne(moduleName)
	moduleFound := true
	if err != nil {
		// either error or no data found
		if err == pg.ErrNoRows {
			// in case of entry not found, update variable
			moduleFound = false
			// initialise module to save in DB
			module = &moduleRepo.Module{
				Name: moduleName,
			}
		} else {
			// otherwise some error case
			impl.logger.Errorw("error in getting module ", "moduleName", moduleName, "err", err)
			return nil, err
		}
	} else {
		// case of data found from DB
		// check if module is already installed or installing
		currentModuleStatus := module.Status
		if currentModuleStatus == bean.ModuleStatusInstalling || currentModuleStatus == bean.ModuleStatusInstalled {
			return nil, errors.New("module is already in installing/installed state")
		}

	}

	// since the request can only come for install, hence update the DB with installing status
	module.Status = bean.ModuleStatusInstalling
	module.Version = moduleActionRequest.Version
	module.UpdatedOn = time.Now()
	tx, err := impl.moduleRepository.GetConnection().Begin()
	if err != nil {
		impl.logger.Errorw("error in  opening an transaction", "err", err)
		return nil, err
	}
	defer tx.Rollback()
	flagForEnablingState := false
	if moduleActionRequest.ModuleType == bean.MODULE_TYPE_SECURITY {
		res := strings.Split(moduleName, ".")
		if len(res) < 2 {
			impl.logger.Errorw("error in getting toolname from module name as len is less than 2", "err", err, "moduleName", moduleName)
			return nil, errors.New("error in getting tool name from module name as len is less than 2")
		}
		toolName := strings.ToUpper(res[1])
		// Finding the Module by type and status, if no module exists of current type marking current module as active and enabled by default.
		err = impl.moduleRepository.FindByModuleTypeAndStatus(moduleActionRequest.ModuleType, bean.ModuleStatusInstalled)
		if err != nil {
			if err == pg.ErrNoRows {
				var toolversion string
				if moduleName == bean.ModuleNameSecurityClair {
					// Handled for V4 for CLAIR as we are not using CLAIR V2 anymore.
					toolversion = bean.CLAIR_V4
				} else if moduleName == bean.ModuleNameSecurityTrivy {
					toolversion = bean.TRIVY_V1
				}
				err2 := impl.scanToolMetadataService.MarkToolAsActive(toolName, toolversion, tx)
				if err2 != nil {
					impl.logger.Errorw("error in marking tool as active ", "err", err2)
					return nil, err2
				}
				flagForEnablingState = true
			} else {
				impl.logger.Errorw("error in getting module by type", "moduleName", moduleName, "err", err)
				return nil, err
			}
		}
	} else {
		flagForEnablingState = true
	}
	module.ModuleType = moduleActionRequest.ModuleType
	if moduleFound {
		err = impl.moduleRepository.UpdateWithTransaction(module, tx)
	} else {
		err = impl.moduleRepository.SaveWithTransaction(module, tx)
	}
	if err != nil {
		impl.logger.Errorw("error in saving/updating module ", "moduleName", moduleName, "err", err)
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	// HELM_OPERATION Starts
	devtronHelmAppIdentifier := impl.helmAppService.GetDevtronHelmAppIdentifier()
	chartRepository := &gRPC.ChartRepository{
		Name: impl.serverEnvConfig.DevtronHelmRepoName,
		Url:  impl.serverEnvConfig.DevtronHelmRepoUrl,
	}

	extraValues := make(map[string]interface{})
	extraValues[impl.serverEnvConfig.DevtronInstallerReleasePath] = moduleActionRequest.Version
	extraValues[impl.serverEnvConfig.DevtronInstallerModulesPath] = []interface{}{moduleName}
	alreadyInstalledModuleNames, err := impl.moduleRepository.GetInstalledModuleNames()
	if err != nil {
		impl.logger.Errorw("error in getting modules with installed status ", "err", err)
		return nil, err
	}
	moduleEnableKeys := moduleUtil.BuildAllModuleEnableKeys(impl.serverEnvConfig.DevtronOperatorBasePath, moduleName)
	for _, moduleEnableKey := range moduleEnableKeys {
		extraValues[moduleEnableKey] = true
	}
	for _, alreadyInstalledModuleName := range alreadyInstalledModuleNames {
		if alreadyInstalledModuleName != moduleName {
			alreadyInstalledModuleEnableKeys := moduleUtil.BuildAllModuleEnableKeys(impl.serverEnvConfig.DevtronOperatorBasePath, alreadyInstalledModuleName)
			for _, alreadyInstalledModuleEnableKey := range alreadyInstalledModuleEnableKeys {
				extraValues[alreadyInstalledModuleEnableKey] = true
			}
		}
	}
	extraValuesYamlUrl := util2.BuildDevtronBomUrl(impl.serverEnvConfig.DevtronBomUrl, moduleActionRequest.Version)

	updateResponse, err := impl.helmAppService.UpdateApplicationWithChartInfoWithExtraValues(context.Background(), devtronHelmAppIdentifier, chartRepository, extraValues, extraValuesYamlUrl, true)
	if err != nil {
		impl.logger.Errorw("error in updating helm release ", "err", err)
		apiError := clientErrors.ConvertToApiError(err)
		if apiError != nil {
			err = apiError
		}
		module.Status = bean.ModuleStatusInstallFailed
		impl.moduleRepository.Update(module)
		return nil, err
	}
	if !updateResponse.GetSuccess() {
		module.Status = bean.ModuleStatusInstallFailed
		impl.moduleRepository.Update(module)
		return nil, errors.New("success is false from helm")
	}
	// HELM_OPERATION Ends
	if flagForEnablingState {
		err = impl.moduleRepository.MarkModuleAsEnabled(moduleName)
		if err != nil {
			impl.logger.Errorw("error in updating module as active ", "moduleName", moduleName, "err", err)
			return nil, err
		}
	}
	return &bean.ActionResponse{
		Success: true,
	}, nil
}
func (impl ModuleServiceImpl) EnableModule(moduleName, version string) (*bean.ActionResponse, error) {

	// get module by name
	module, err := impl.moduleRepository.FindOne(moduleName)
	if err != nil {
		impl.logger.Errorw("error in getting module ", "moduleName", moduleName, "err", err)
		return nil, err
	}
	dbConnection := impl.moduleRepository.GetConnection()
	tx, err := dbConnection.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	res := strings.Split(moduleName, ".")
	// Handling for future tools if integrated
	if len(res) < 2 {
		impl.logger.Errorw("error in getting toolName from modulename as module Length is smaller than 2")
		return nil, errors.New("error in getting tool name from module name as len is less than 2")
	}
	// Extracting out toolName for security module for now
	toolName := strings.ToUpper(res[1])
	err = impl.moduleRepository.MarkModuleAsEnabledWithTransaction(moduleName, tx)
	if err != nil {
		impl.logger.Errorw("error in updating module as active ", "moduleName", moduleName, "err", err, "moduleName", module.Name)
		return nil, err
	}
	err = impl.scanToolMetadataService.MarkToolAsActive(toolName, version, tx)
	if err != nil {
		impl.logger.Errorw("error in marking tool as active ", "err", err, "moduleName", module.Name)
		return nil, err
	}
	err = impl.scanToolMetadataService.MarkOtherToolsInActive(toolName, tx, version)
	if err != nil {
		impl.logger.Errorw("error in marking other tools inactive ", "err", err, "moduleName", module.Name)
		return nil, err
	}
	// Currently Supporting one tool at a time
	err = impl.moduleRepository.MarkOtherModulesDisabledOfSameType(moduleName, module.ModuleType, tx)
	if err != nil {
		impl.logger.Errorw("error in marking other modules of same module type inactive ", "err", err, "moduleName", module.Name)
		return nil, err
	}
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return &bean.ActionResponse{
		Success: true,
	}, nil
}

func (impl ModuleServiceImpl) saveModuleAsInstalled(moduleName string, moduleType string, moduleEnabled bool) (bean.ModuleStatus, error) {
	return impl.saveModule(moduleName, bean.ModuleStatusInstalled, moduleType, moduleEnabled)
}

func (impl ModuleServiceImpl) saveModule(moduleName string, moduleStatus bean.ModuleStatus, moduleType string, moduleEnabled bool) (bean.ModuleStatus, error) {
	module := &moduleRepo.Module{
		Name:       moduleName,
		Version:    impl.serverDataStore.CurrentVersion,
		Status:     moduleStatus,
		UpdatedOn:  time.Now(),
		ModuleType: moduleType,
		Enabled:    moduleEnabled,
	}
	err := impl.moduleRepository.Save(module)
	if err != nil {
		impl.logger.Errorw("error in saving module status ", "moduleName", moduleName, "moduleStatus", moduleStatus, "err", err)
		return bean.ModuleStatusNotInstalled, err
	}
	return moduleStatus, nil
}

func (impl ModuleServiceImpl) GetAllModuleInfo() ([]bean.ModuleInfoDto, error) {
	// fetch from DB
	modules, err := impl.moduleRepository.FindAll()
	if err != nil {
		if err == pg.ErrNoRows {
			impl.logger.Errorw("no installed modules found ", "err", err)
			return nil, err
		}
		// otherwise some error case
		impl.logger.Errorw("error in getting modules from DB ", "err", err)
		return nil, err
	}
	var installedModules []bean.ModuleInfoDto
	// now this is the case when data found in DB
	for _, module := range modules {
		moduleInfoDto := bean.ModuleInfoDto{
			Name:       module.Name,
			Status:     module.Status,
			Moduletype: module.ModuleType,
			Enabled:    module.Enabled,
		}
		enabled := false
		if module.ModuleType != bean.MODULE_TYPE_SECURITY && module.Status == bean.ModuleStatusInstalled {
			module.Enabled = true
			enabled = true
			err := impl.moduleRepository.Update(&module)
			if err != nil {
				impl.logger.Errorw("error in updating installed module to enabled for previous modules", "err", err, "module", module.Name)
			}
		}
		moduleInfoDto.Enabled = enabled || module.Enabled
		moduleId := module.Id
		moduleResourcesStatusFromDb, err := impl.moduleResourceStatusRepository.FindAllActiveByModuleId(moduleId)
		if err != nil && err != pg.ErrNoRows {
			impl.logger.Errorw("error in getting module resources status from DB ", "moduleId", moduleId, "moduleName", module.Name, "err", err)
			return nil, err
		}
		if moduleResourcesStatusFromDb != nil {
			var moduleResourcesStatus []*bean.ModuleResourceStatusDto
			for _, moduleResourceStatusFromDb := range moduleResourcesStatusFromDb {
				moduleResourcesStatus = append(moduleResourcesStatus, &bean.ModuleResourceStatusDto{
					Group:         moduleResourceStatusFromDb.Group,
					Version:       moduleResourceStatusFromDb.Version,
					Kind:          moduleResourceStatusFromDb.Kind,
					Name:          moduleResourceStatusFromDb.Name,
					HealthStatus:  moduleResourceStatusFromDb.HealthStatus,
					HealthMessage: moduleResourceStatusFromDb.HealthMessage,
				})
			}
			moduleInfoDto.ModuleResourcesStatus = moduleResourcesStatus
		}
		installedModules = append(installedModules, moduleInfoDto)
	}

	return installedModules, nil
}
