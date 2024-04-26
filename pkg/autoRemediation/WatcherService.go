package autoRemediation

import (
	appRepository "github.com/devtron-labs/devtron/internal/sql/repository/app"
	"github.com/devtron-labs/devtron/internal/sql/repository/appWorkflow"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/pkg/autoRemediation/repository"
	repository2 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	"go.uber.org/zap"
	"gopkg.in/square/go-jose.v2/json"
	"strings"
)

type WatcherService interface {
	CreateWatcher(watcherRequest WatcherDto) (int, error)
	GetWatcherById(watcherId int) (*WatcherDto, error)
	DeleteWatcherById(watcherId int) error
	// RetrieveInterceptedEvents() ([]*InterceptedEventsDto, error)
	UpdateWatcherById(watcherId int, watcherRequest WatcherDto) error
	// RetrieveInterceptedEvents(offset int, size int, sortOrder string, searchString string, from time.Time, to time.Time, watchers []string, clusters []string, namespaces []string) (EventsResponse, error)
	FindAllWatchers(offset int, search string, size int, sortOrder string, sortOrderBy string) (WatchersResponse, error)
	GetTriggerByWatcherIds(watcherIds []int) ([]*Trigger, error)
}

type WatcherServiceImpl struct {
	watcherRepository            repository.WatcherRepository
	triggerRepository            repository.TriggerRepository
	interceptedEventsRepository  repository.InterceptedEventsRepository
	appRepository                appRepository.AppRepository
	ciPipelineRepository         pipelineConfig.CiPipelineRepository
	environmentRepository        repository2.EnvironmentRepository
	appWorkflowMappingRepository appWorkflow.AppWorkflowRepository
	clusterRepository            repository2.ClusterRepository
	logger                       *zap.SugaredLogger
}

func NewWatcherServiceImpl(watcherRepository repository.WatcherRepository,
	triggerRepository repository.TriggerRepository,
	interceptedEventsRepository repository.InterceptedEventsRepository,
	appRepository appRepository.AppRepository,
	ciPipelineRepository pipelineConfig.CiPipelineRepository,
	environmentRepository repository2.EnvironmentRepository,
	appWorkflowMappingRepository appWorkflow.AppWorkflowRepository,
	clusterRepository repository2.ClusterRepository,
	logger *zap.SugaredLogger) *WatcherServiceImpl {
	return &WatcherServiceImpl{
		watcherRepository:            watcherRepository,
		triggerRepository:            triggerRepository,
		interceptedEventsRepository:  interceptedEventsRepository,
		appRepository:                appRepository,
		ciPipelineRepository:         ciPipelineRepository,
		environmentRepository:        environmentRepository,
		appWorkflowMappingRepository: appWorkflowMappingRepository,
		clusterRepository:            clusterRepository,
		logger:                       logger,
	}
}
func (impl *WatcherServiceImpl) CreateWatcher(watcherRequest WatcherDto) (int, error) {

	var gvks []string
	for _, res := range watcherRequest.EventConfiguration.K8sResources {
		jsonString, _ := json.Marshal(res)
		gvks = append(gvks, string(jsonString))
	}
	strings.Join(gvks, ",")
	watcher := &repository.Watcher{
		Name:             watcherRequest.Name,
		Desc:             watcherRequest.Description,
		FilterExpression: watcherRequest.EventConfiguration.EventExpression,
		Gvks:             gvks,
	}
	tx, err := impl.watcherRepository.StartTx()
	if err != nil {
		impl.logger.Errorw("error in creating watcher", "error", err)
		return 0, err
	}
	defer impl.watcherRepository.RollbackTx(tx)
	watcher, err = impl.watcherRepository.Save(watcher, tx)
	if err != nil {
		impl.logger.Errorw("error in saving watcher", "error", err)
		return 0, err
	}
	err = impl.createTriggerForWatcher(watcherRequest, watcher.Id)
	if err != nil {
		impl.logger.Errorw("error in saving triggers", "error", err)
		return 0, err
	}
	return watcher.Id, nil
}
func (impl *WatcherServiceImpl) createTriggerForWatcher(watcherRequest WatcherDto, watcherId int) error {
	var jsonData []byte
	var jobNames, envNames, pipelineNames []string
	for _, res := range watcherRequest.Triggers {
		jobNames = append(jobNames, res.Data.JobName)
		envNames = append(envNames, res.Data.ExecutionEnvironment)
		pipelineNames = append(pipelineNames, res.Data.PipelineName)
	}
	apps, err := impl.appRepository.FetchAppByDisplayNamesForJobs(jobNames)
	if err != nil {
		impl.logger.Errorw("error in fetching apps", "error", err)
		return err
	}
	var jobIds []int
	for _, app := range apps {
		jobIds = append(jobIds, app.Id)
	}
	pipelines, err := impl.ciPipelineRepository.FindByNames(pipelineNames, jobIds)
	if err != nil {
		impl.logger.Errorw("error in fetching pipelines", "error", err)
		return err
	}
	envs, err := impl.environmentRepository.FindByNames(envNames)
	if err != nil {
		impl.logger.Errorw("error in fetching environment", "error", err)
		return err
	}
	displayNameToId := make(map[string]int)
	for _, app := range apps {
		displayNameToId[app.DisplayName] = app.Id
	}
	pipelineNameToId := make(map[string]int)
	for _, pipeline := range pipelines {
		pipelineNameToId[pipeline.Name] = pipeline.Id
	}
	envNameToId := make(map[string]int)
	for _, env := range envs {
		envNameToId[env.Name] = env.Id
	}
	for _, res := range watcherRequest.Triggers {
		triggerData := TriggerData{
			RuntimeParameters:      res.Data.RuntimeParameters,
			JobId:                  displayNameToId[res.Data.JobName],
			JobName:                res.Data.JobName,
			PipelineId:             pipelineNameToId[res.Data.PipelineName],
			PipelineName:           res.Data.PipelineName,
			ExecutionEnvironment:   res.Data.ExecutionEnvironment,
			ExecutionEnvironmentId: envNameToId[res.Data.ExecutionEnvironment],
		}
		jsonData, err = json.Marshal(triggerData)
		if err != nil {
			impl.logger.Errorw("error in trigger data ", "error", err)
			return err
		}
		trigger := &repository.Trigger{
			WatcherId: watcherId,
			Data:      jsonData,
		}
		if res.IdentifierType == repository.DEVTRON_JOB {
			trigger.Type = repository.DEVTRON_JOB
		}
		tx, err := impl.triggerRepository.StartTx()
		if err != nil {
			impl.logger.Errorw("error in creating trigger", "error", err)
			return err
		}
		defer impl.triggerRepository.RollbackTx(tx)
		_, err = impl.triggerRepository.Save(trigger, tx)
		if err != nil {
			impl.logger.Errorw("error in saving trigger", "error", err)
			return err
		}
	}
	return nil
}

func (impl *WatcherServiceImpl) GetWatcherById(watcherId int) (*WatcherDto, error) {
	watcher, err := impl.watcherRepository.GetWatcherById(watcherId)
	if err != nil {
		impl.logger.Errorw("error in getting watcher", "error", err)
		return nil, err
	}
	var k8sResources []K8sResource
	for _, gvksString := range watcher.Gvks {
		var res K8sResource
		if err := json.Unmarshal([]byte(gvksString), &res); err != nil {
			impl.logger.Errorw("error in unmarshalling gvks", "error", err)
			return nil, err
		}
		k8sResources = append(k8sResources, res)
	}
	watcherResponse := WatcherDto{
		Name:        watcher.Name,
		Description: watcher.Desc,
		EventConfiguration: EventConfiguration{
			K8sResources:    k8sResources,
			EventExpression: watcher.FilterExpression,
		},
	}
	triggers, err := impl.triggerRepository.GetTriggerByWatcherId(watcherId)
	if err != nil {
		impl.logger.Errorw("error in getting trigger for watcher id", "watcherId", watcherId, "error", err)
		return &WatcherDto{}, err
	}
	for _, trigger := range triggers {
		var triggerResp Trigger
		if err := json.Unmarshal(trigger.Data, &triggerResp); err != nil {
			impl.logger.Errorw("error in unmarshalling trigger data", "error", err)
			return nil, err
		}
		triggerResp.IdentifierType = trigger.Type
		watcherResponse.Triggers = append(watcherResponse.Triggers, triggerResp)
	}
	return &watcherResponse, nil

}

func (impl *WatcherServiceImpl) DeleteWatcherById(watcherId int) error {
	err := impl.triggerRepository.DeleteTriggerByWatcherId(watcherId)
	if err != nil {
		impl.logger.Errorw("error in deleting trigger by watcher id", "watcherId", watcherId, "error", err)
		return err
	}
	err = impl.watcherRepository.DeleteWatcherById(watcherId)
	if err != nil {
		impl.logger.Errorw("error in deleting watcher by its id", watcherId, "error", err)
		return err
	}
	return nil
}

// func (impl *WatcherServiceImpl) RetrieveInterceptedEvents() ([]*InterceptedEventsDto, error) {
// 	// message type?
// 	var interceptedEventsResponse []*InterceptedEventsDto
// 	interceptedEvents, err := impl.interceptedEventsRepository.GetAllInterceptedEvents()
// 	if err != nil {
// 		impl.logger.Errorw("error in retrieving intercepted events", "error", err)
// 		return nil, err
// 	}
// 	for _, interceptedEvent := range interceptedEvents {
// 		cluster, err := impl.clusterRepository.FindById(interceptedEvent.ClusterId)
// 		if err != nil {
// 			impl.logger.Errorw("error in retrieving cluster name ", "error", err)
// 			return nil, err
// 		}
// 		interceptedEventResponse := &InterceptedEventsDto{
// 			Message:         interceptedEvent.Message,
// 			MessageType:     interceptedEvent.MessageType,
// 			Event:           interceptedEvent.Event,
// 			InvolvedObject:  interceptedEvent.InvolvedObject,
// 			ClusterName:     cluster.ClusterName,
// 			Namespace:       interceptedEvent.Namespace,
// 			InterceptedTime: (interceptedEvent.InterceptedAt).String(),
// 			ExecutionStatus: interceptedEvent.Status,
// 			TriggerId:       interceptedEvent.TriggerId,
// 		}
// 		triggerResp := Trigger{}
// 		trigger, err := impl.triggerRepository.GetTriggerById(interceptedEventResponse.TriggerId)
// 		if err != nil {
// 			impl.logger.Errorw("error in retrieving intercepted events", "error", err)
// 			return nil, err
// 		}
// 		triggerResp.Id = trigger.Id
// 		triggerResp.IdentifierType = trigger.Type
// 		triggerRespData := TriggerData{}
// 		if err := json.Unmarshal(trigger.Data, &triggerRespData); err != nil {
// 			impl.logger.Errorw("error in unmarshalling trigger data", "error", err)
// 			return nil, err
// 		}
// 		triggerResp.Data.JobName = triggerRespData.JobName
// 		triggerResp.Data.PipelineName = triggerRespData.PipelineName
// 		triggerResp.Data.RuntimeParameters = triggerRespData.RuntimeParameters
// 		triggerResp.Data.ExecutionEnvironment = triggerRespData.ExecutionEnvironment
// 		triggerResp.Data.PipelineId = triggerRespData.PipelineId
// 		triggerResp.Data.JobId = triggerRespData.JobId
// 		triggerResp.Data.ExecutionEnvironmentId = triggerRespData.ExecutionEnvironmentId
// 		interceptedEventResponse.Trigger = triggerResp
// 		interceptedEventsResponse = append(interceptedEventsResponse, interceptedEventResponse)
// 	}
// 	return interceptedEventsResponse, nil
// }

func (impl *WatcherServiceImpl) UpdateWatcherById(watcherId int, watcherRequest WatcherDto) error {
	watcher, err := impl.watcherRepository.GetWatcherById(watcherId)
	if err != nil {
		impl.logger.Errorw("error in retrieving watcher by id", watcherId, "error", err)
		return err
	}
	var gvks []string
	for _, res := range watcherRequest.EventConfiguration.K8sResources {
		jsonString, _ := json.Marshal(res)
		gvks = append(gvks, string(jsonString))
	}
	strings.Join(gvks, ",")
	watcher.Name = watcherRequest.Name
	watcher.Desc = watcherRequest.Description
	watcher.FilterExpression = watcherRequest.EventConfiguration.EventExpression
	watcher.Gvks = gvks
	err = impl.triggerRepository.DeleteTriggerByWatcherId(watcher.Id)
	if err != nil {
		impl.logger.Errorw("error in deleting trigger by watcher id", watcherId, "error", err)
		return err
	}
	err = impl.createTriggerForWatcher(watcherRequest, watcherId)
	if err != nil {
		impl.logger.Errorw("error in creating trigger by watcher id", watcherId, "error", err)
		return err
	}
	return nil
}

func (impl *WatcherServiceImpl) FindAllWatchers(offset int, search string, size int, sortOrder string, sortOrderBy string) (WatchersResponse, error) {
	search = strings.ToLower(search)
	params := WatcherQueryParams{
		Offset:      offset,
		Size:        size,
		Search:      search,
		SortOrderBy: sortOrderBy,
		SortOrder:   sortOrder,
	}
	watchers, err := impl.watcherRepository.FindAllWatchersByQueryName(params)
	if err != nil {
		impl.logger.Errorw("error in retrieving watchers ", "error", err)
		return WatchersResponse{}, err
	}
	var watcherIds []int
	for _, watcher := range watchers {
		watcherIds = append(watcherIds, watcher.Id)
	}
	triggers, err := impl.triggerRepository.GetTriggerByWatcherIds(watcherIds)
	if err != nil {
		impl.logger.Errorw("error in retrieving triggers ", "error", err)
		return WatchersResponse{}, err
	}
	var triggerIds []int
	watcherIdToTrigger := make(map[int]repository.Trigger)
	for _, trigger := range triggers {
		triggerIds = append(triggerIds, trigger.Id)
		watcherIdToTrigger[trigger.WatcherId] = *trigger
	}

	watcherResponses := WatchersResponse{
		Size:   params.Size,
		Offset: params.Offset,
		Total:  len(watchers),
	}
	var pipelineIds []int
	for _, watcher := range watchers {
		var triggerResp TriggerData
		if err := json.Unmarshal(watcherIdToTrigger[watcher.Id].Data, &triggerResp); err != nil {
			impl.logger.Errorw("error in unmarshalling trigger data", "error", err)
			return WatchersResponse{}, err
		}
		pipelineIds = append(pipelineIds, triggerResp.PipelineId)
		watcherResponses.List = append(watcherResponses.List, WatcherItem{
			Name:            watcher.Name,
			Description:     watcher.Desc,
			JobPipelineName: triggerResp.PipelineName,
			JobPipelineId:   triggerResp.PipelineId,
		})
	}
	workflows, err := impl.appWorkflowMappingRepository.FindWFCIMappingByCIPipelineIds(pipelineIds)
	if err != nil {
		impl.logger.Errorw("error in retrieving workflows ", "error", err)
		return WatchersResponse{}, err
	}
	var pipelineIdtoAppworkflow map[int]int
	for _, workflow := range workflows {
		pipelineIdtoAppworkflow[workflow.ComponentId] = workflow.AppWorkflowId
	}
	for _, watcherList := range watcherResponses.List {
		watcherList.WorkflowId = pipelineIdtoAppworkflow[watcherList.JobPipelineId]
	}

	return watcherResponses, nil
}

func (impl *WatcherServiceImpl) GetTriggerByWatcherIds(watcherIds []int) ([]*Trigger, error) {
	triggers, err := impl.triggerRepository.GetTriggerByWatcherIds(watcherIds)
	if err != nil {
		impl.logger.Errorw("error in getting triggers by watcher ids", "watcherIds", watcherIds, "err", err)
		return nil, err
	}

	triggersResult := make([]*Trigger, 0, len(triggers))
	for _, trigger := range triggers {
		triggerResp := Trigger{}
		triggerResp.Id = trigger.Id
		triggerResp.IdentifierType = trigger.Type
		triggerData := TriggerData{}
		if err := json.Unmarshal(trigger.Data, &triggerData); err != nil {
			impl.logger.Errorw("error in unmarshalling trigger data", "error", err)
			return nil, err
		}
		triggerResp.Data.JobName = triggerData.JobName
		triggerResp.Data.PipelineName = triggerData.PipelineName
		triggerResp.Data.RuntimeParameters = triggerData.RuntimeParameters
		triggerResp.Data.ExecutionEnvironment = triggerData.ExecutionEnvironment
		triggerResp.Data.PipelineId = triggerData.PipelineId
		triggerResp.Data.JobId = triggerData.JobId
		triggerResp.Data.ExecutionEnvironmentId = triggerData.ExecutionEnvironmentId

		triggersResult = append(triggersResult, &triggerResp)
	}

	return triggersResult, nil
}
