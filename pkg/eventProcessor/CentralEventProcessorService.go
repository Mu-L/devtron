package eventProcessor

import (
	"github.com/devtron-labs/devtron/pkg/eventProcessor/in"
	"go.uber.org/zap"
)

type CentralEventProcessor struct {
	logger                                *zap.SugaredLogger
	workflowEventProcessor                *in.WorkflowEventProcessorImpl
	ciPipelineEventProcessor              *in.CIPipelineEventProcessorImpl
	cdPipelineEventProcessor              *in.CDPipelineEventProcessorImpl
	deployedApplicationEventProcessorImpl *in.DeployedApplicationEventProcessorImpl
	appStoreAppsEventProcessorImpl        *in.AppStoreAppsEventProcessorImpl
	chartScanEventProcessorImpl           *in.ChartScanEventProcessorImpl
}

func NewCentralEventProcessor(logger *zap.SugaredLogger,
	workflowEventProcessor *in.WorkflowEventProcessorImpl,
	ciPipelineEventProcessor *in.CIPipelineEventProcessorImpl,
	cdPipelineEventProcessor *in.CDPipelineEventProcessorImpl,
	deployedApplicationEventProcessorImpl *in.DeployedApplicationEventProcessorImpl,
	appStoreAppsEventProcessorImpl *in.AppStoreAppsEventProcessorImpl,
	chartScanEventProcessorImpl *in.ChartScanEventProcessorImpl,
) (*CentralEventProcessor, error) {
	cep := &CentralEventProcessor{
		logger:                                logger,
		workflowEventProcessor:                workflowEventProcessor,
		ciPipelineEventProcessor:              ciPipelineEventProcessor,
		cdPipelineEventProcessor:              cdPipelineEventProcessor,
		deployedApplicationEventProcessorImpl: deployedApplicationEventProcessorImpl,
		appStoreAppsEventProcessorImpl:        appStoreAppsEventProcessorImpl,
		chartScanEventProcessorImpl:           chartScanEventProcessorImpl,
	}
	err := cep.SubscribeAll()
	if err != nil {
		return nil, err
	}
	return cep, nil
}

func (impl *CentralEventProcessor) SubscribeAll() error {
	var err error

	//CI pipeline event starts
	err = impl.ciPipelineEventProcessor.SubscribeNewCIMaterialEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeNewCIMaterialEvent", "err", err)
		return err
	}
	//CI pipeline event ends

	//CD pipeline event starts

	err = impl.cdPipelineEventProcessor.SubscribeCDBulkTriggerTopic()
	if err != nil {
		impl.logger.Errorw("error, SubscribeCDBulkTriggerTopic", "err", err)
		return err
	}

	err = impl.cdPipelineEventProcessor.SubscribeArgoTypePipelineSyncEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeArgoTypePipelineSyncEvent", "err", err)
		return err
	}

	//CD pipeline event ends

	//Workflow event starts

	err = impl.workflowEventProcessor.SubscribeDeployStageSuccessEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeDeployStageSuccessEvent", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeCDStageCompleteEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeCDStageCompleteEvent", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeImageScanningSuccessEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeImageScanningSuccessEvent", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeTriggerBulkAction()
	if err != nil {
		impl.logger.Errorw("error, SubscribeTriggerBulkAction", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeHibernateBulkAction()
	if err != nil {
		impl.logger.Errorw("error, SubscribeHibernateBulkAction", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeCIWorkflowStatusUpdate()
	if err != nil {
		impl.logger.Errorw("error, SubscribeCIWorkflowStatusUpdate", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeCDWorkflowStatusUpdate()
	if err != nil {
		impl.logger.Errorw("error, SubscribeCDWorkflowStatusUpdate", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeCICompleteEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeCICompleteEvent", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeDevtronAsyncHelmInstallRequest()
	if err != nil {
		impl.logger.Errorw("error, SubscribeDevtronAsyncHelmInstallRequest", "err", err)
		return err
	}
	err = impl.workflowEventProcessor.SubscribeCDPipelineDeleteEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeCDPipelineDeleteEvent", "err", err)
		return err
	}

	//Workflow event ends

	//Deployed application status event starts (currently only argo)

	err = impl.deployedApplicationEventProcessorImpl.SubscribeArgoAppUpdate()
	if err != nil {
		impl.logger.Errorw("error, SubscribeArgoAppUpdate", "err", err)
		return err
	}
	err = impl.deployedApplicationEventProcessorImpl.SubscribeArgoAppDeleteStatus()
	if err != nil {
		impl.logger.Errorw("error, SubscribeArgoAppDeleteStatus", "err", err)
		return err
	}

	//Deployed application status event ends (currently only argo)

	//AppStore apps event starts

	err = impl.appStoreAppsEventProcessorImpl.SubscribeAppStoreAppsBulkDeployEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeAppStoreAppsBulkDeployEvent", "err", err)
		return err
	}

	err = impl.appStoreAppsEventProcessorImpl.SubscribeHelmInstallStatusEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeHelmInstallStatusEvent", "err", err)
		return err
	}

	err = impl.chartScanEventProcessorImpl.SubscribeChartScanEvent()
	if err != nil {
		impl.logger.Errorw("error, SubscribeChartScanEvent", "err", err)
		return err
	}

	//AppStore apps event ends

	return nil
}
