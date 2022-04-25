package plugin

import (
	"github.com/devtron-labs/devtron/pkg/plugin/repository"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type GlobalVariable struct {
	Name   string `json:"name"`
	Value  string `json:"value,omitempty"`
	Format string `json:"format"`
}

type GlobalPluginService interface {
	GetAllGlobalVariables() ([]*GlobalVariable, error)
	ListAllPlugins() ([]*PluginMetadataDto, error)
	GetPluginDetailById(pluginId int) (*PluginDetailDto, error)
}

func NewGlobalPluginService(logger *zap.SugaredLogger, globalPluginRepository repository.GlobalPluginRepository) *GlobalPluginServiceImpl {
	return &GlobalPluginServiceImpl{
		logger:                 logger,
		globalPluginRepository: globalPluginRepository,
	}
}

type GlobalPluginServiceImpl struct {
	logger                 *zap.SugaredLogger
	globalPluginRepository repository.GlobalPluginRepository
}

func (impl *GlobalPluginServiceImpl) GetAllGlobalVariables() ([]*GlobalVariable, error) {
	globalVariableNames := []string{
		//"GIT_REPOSITORY",
		//"GIT_BRANCH",
		//"GIT_HASH",
		//"GIT_TAG",
		//"LATEST_COMMIT_AUTHOR",
		//"PIPELINE_NAME",
		"WORKING_DIRECTORY",
		"DOCKER_IMAGE_TAG",
		"DOCKER_REPOSITORY",
		"DOCKER_REGISTRY_URL",
		"DOCKER_IMAGE",
		"APP_NAME",
		"TRIGGER_BY_AUTHOR",
	}
	var globalVariables []*GlobalVariable
	for _, globalVariableName := range globalVariableNames {
		globalVariable := &GlobalVariable{
			Name:   globalVariableName,
			Format: string(repository.PLUGIN_VARIABLE_FORMAT_TYPE_STRING),
		}
		globalVariables = append(globalVariables, globalVariable)
	}
	return globalVariables, nil
}

func (impl *GlobalPluginServiceImpl) ListAllPlugins() ([]*PluginMetadataDto, error) {
	impl.logger.Infow("request received, ListAllPlugins")

	var plugins []*PluginMetadataDto

	//getting all plugins metadata(without tags)
	pluginsMetadata, err := impl.globalPluginRepository.GetMetaDataForAllPlugins()
	if err != nil {
		impl.logger.Errorw("error in getting plugins", "err", err)
		return nil, err
	}
	//getting all plugin tags
	pluginTags, err := impl.globalPluginRepository.GetAllPluginTags()
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error in getting all plugin tags", "err", err)
		return nil, err
	}
	tagIdNameMap := make(map[int]string)
	for _, tag := range pluginTags {
		tagIdNameMap[tag.Id] = tag.Name
	}
	//getting plugin-tag relations
	relations, err := impl.globalPluginRepository.GetAllPluginTagRelations()
	if err != nil {
		impl.logger.Errorw("error in getting all plugin-tag relations", "err", err)
		return nil, err
	}
	pluginIdTagsMap := make(map[int][]string)
	for _, relation := range relations {
		tag, ok := tagIdNameMap[relation.TagId]
		if ok {
			tags, ok2 := pluginIdTagsMap[relation.PluginId]
			if ok2 {
				tags = append(tags, tag)
			} else {
				tags = []string{tag}
			}
			pluginIdTagsMap[relation.PluginId] = tags
		}
	}
	for _, pluginMetadata := range pluginsMetadata {
		plugin := &PluginMetadataDto{
			Id:          pluginMetadata.Id,
			Name:        pluginMetadata.Name,
			Type:        string(pluginMetadata.Type),
			Description: pluginMetadata.Description,
			Icon:        pluginMetadata.Icon,
		}
		tags, ok := pluginIdTagsMap[pluginMetadata.Id]
		if ok {
			plugin.Tags = tags
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

func (impl *GlobalPluginServiceImpl) GetPluginDetailById(pluginId int) (*PluginDetailDto, error) {
	impl.logger.Infow("request received, GetPluginDetail", "pluginId", pluginId)

	//getting metadata
	pluginMetadata, err := impl.globalPluginRepository.GetMetaDataByPluginId(pluginId)
	if err != nil {
		impl.logger.Errorw("error in getting plugins", "err", err, "pluginId", pluginId)
		return nil, err
	}
	metadataDto := &PluginMetadataDto{
		Id:          pluginMetadata.Id,
		Name:        pluginMetadata.Name,
		Type:        string(pluginMetadata.Type),
		Description: pluginMetadata.Description,
		Icon:        pluginMetadata.Icon,
	}
	pluginDetail := &PluginDetailDto{
		Metadata: metadataDto,
	}

	//getting exposed variables
	pluginVariables, err := impl.globalPluginRepository.GetExposedVariablesByPluginId(pluginId)
	if err != nil {
		impl.logger.Errorw("error in getting pluginVariables by pluginId", "err", err, "pluginId", pluginId)
		return nil, err
	}

	var inputVariablesDto []*PluginVariableDto
	var outputVariablesDto []*PluginVariableDto

	for _, pluginVariable := range pluginVariables {
		variableDto := &PluginVariableDto{
			Id:                    pluginVariable.Id,
			Name:                  pluginVariable.Name,
			Format:                pluginVariable.Format,
			Description:           pluginVariable.Description,
			IsExposed:             pluginVariable.IsExposed,
			AllowEmptyValue:       pluginVariable.AllowEmptyValue,
			DefaultValue:          pluginVariable.DefaultValue,
			Value:                 pluginVariable.Value,
			ValueType:             pluginVariable.ValueType,
			PreviousStepIndex:     pluginVariable.PreviousStepIndex,
			ReferenceVariableName: pluginVariable.ReferenceVariableName,
		}
		if pluginVariable.VariableType == repository.PLUGIN_VARIABLE_TYPE_INPUT {
			inputVariablesDto = append(inputVariablesDto, variableDto)
		} else if pluginVariable.VariableType == repository.PLUGIN_VARIABLE_TYPE_OUTPUT {
			outputVariablesDto = append(outputVariablesDto, variableDto)
		}
	}
	pluginDetail.InputVariables = inputVariablesDto
	pluginDetail.OutputVariables = outputVariablesDto
	return pluginDetail, nil
}
