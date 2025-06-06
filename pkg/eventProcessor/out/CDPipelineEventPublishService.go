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

package out

import (
	"encoding/json"
	pubsub "github.com/devtron-labs/common-lib/pubsub-lib"
	bean2 "github.com/devtron-labs/devtron/api/bean"
	bean3 "github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"github.com/devtron-labs/devtron/pkg/eventProcessor/bean"
	"go.uber.org/zap"
)

type CDPipelineEventPublishService interface {
	PublishBulkTriggerTopicEvent(pipelineId, appId,
		artifactId int, userMetadata *bean3.UserMetadata) error

	PublishArgoTypePipelineSyncEvent(pipelineId, installedAppVersionId int,
		userId int32, isAppStoreApplication bool) error
}

type CDPipelineEventPublishServiceImpl struct {
	logger       *zap.SugaredLogger
	pubSubClient *pubsub.PubSubClientServiceImpl
}

func NewCDPipelineEventPublishServiceImpl(logger *zap.SugaredLogger,
	pubSubClient *pubsub.PubSubClientServiceImpl) *CDPipelineEventPublishServiceImpl {
	return &CDPipelineEventPublishServiceImpl{
		logger:       logger,
		pubSubClient: pubSubClient,
	}
}

func (impl *CDPipelineEventPublishServiceImpl) PublishBulkTriggerTopicEvent(pipelineId, appId,
	artifactId int, userMetadata *bean3.UserMetadata) error {
	event := &bean.BulkCDDeployEvent{
		ValuesOverrideRequest: &bean2.ValuesOverrideRequest{
			PipelineId:     pipelineId,
			AppId:          appId,
			CiArtifactId:   artifactId,
			UserId:         userMetadata.UserId,
			CdWorkflowType: bean2.CD_WORKFLOW_TYPE_DEPLOY,
		},
		UserId:       userMetadata.UserId,
		UserMetadata: userMetadata,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		impl.logger.Errorw("failed to marshal cd bulk deploy event request", "request", event, "err", err)
		return err
	}
	err = impl.pubSubClient.Publish(pubsub.CD_BULK_DEPLOY_TRIGGER_TOPIC, string(payload))
	if err != nil {
		impl.logger.Errorw("failed to publish trigger request event", "topic", pubsub.CD_BULK_DEPLOY_TRIGGER_TOPIC,
			"err", err, "request", event)
		return err
	}
	return nil
}

func (impl *CDPipelineEventPublishServiceImpl) PublishArgoTypePipelineSyncEvent(pipelineId, installedAppVersionId int,
	userId int32, isAppStoreApplication bool) error {
	statusUpdateEvent := bean.ArgoPipelineStatusSyncEvent{
		PipelineId:            pipelineId,
		InstalledAppVersionId: installedAppVersionId,
		UserId:                userId,
		IsAppStoreApplication: isAppStoreApplication,
	}
	data, err := json.Marshal(statusUpdateEvent)
	if err != nil {
		impl.logger.Errorw("error while writing cd pipeline delete event to nats", "err", err, "req", statusUpdateEvent)
		return err
	} else {
		err = impl.pubSubClient.Publish(pubsub.ARGO_PIPELINE_STATUS_UPDATE_TOPIC, string(data))
		if err != nil {
			impl.logger.Errorw("error, PublishArgoTypePipelineSyncEvent", "topic", pubsub.ARGO_PIPELINE_STATUS_UPDATE_TOPIC, "error", err, "data", data)
			return err
		}
	}
	return nil
}
