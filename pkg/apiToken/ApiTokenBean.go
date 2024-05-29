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

package apiToken

import (
	"github.com/devtron-labs/devtron/pkg/bean"
	"github.com/golang-jwt/jwt/v4"
)

type ApiTokenCustomClaims struct {
	Email   string `json:"email"`
	Version string `json:"version"`
	jwt.RegisteredClaims
}
type TokenCustomClaimsForNotification struct {
	DraftId           int                         `json:"draftId"`
	DraftVersionId    int                         `json:"draftVersionId"`
	ApprovalRequestId int                         `json:"approvalRequestId"`
	ArtifactId        int                         `json:"artifactId"`
	PipelineId        int                         `json:"pipelineId"`
	ActionType        bean.UserApprovalActionType `json:"actionType" validate:"required"`
	AppId             int                         `json:"appId" validate:"required"`
	EnvId             int                         `json:"envId"`
	UserId            int32                       `json:"userId"`
	ApiTokenCustomClaims
}

type ArtifactPromotionApprovalNotificationClaims struct {
	AppId           int      `json:"appId"`
	AppName         string   `json:"appName"`
	EnvId           int      `json:"envId"`
	EnvName         string   `json:"envName"`
	Image           string   `json:"image"`
	ImageTags       []string `json:"imageTags"`
	ImageComment    string   `json:"imageComment"`
	PromotionSource string   `json:"promotionSourceType"`
	ArtifactId      int      `json:"artifactId"`
	WorkflowId      int      `json:"workflowId"`
	UserId          int32    `json:"userId"`
	ApiTokenCustomClaims
}

type DraftApprovalRequest struct {
	DraftId        int `json:"draftId"`
	DraftVersionId int `json:"draftVersionId"`
	NotificationApprovalRequest
}
type DeploymentApprovalRequest struct {
	ApprovalRequestId int `json:"approvalRequestId"`
	ArtifactId        int `json:"artifactId"`
	PipelineId        int `json:"pipelineId"`
	NotificationApprovalRequest
}
type NotificationApprovalRequest struct {
	AppId   int    `json:"appId" validate:"required"`
	EnvId   int    `json:"envId"`
	EmailId string `json:"email"`
	UserId  int32  `json:"userId"`
}

func GetDraftApprovalRequest(envId int, appId int, draftId int, draftVersionId int, userId int32) DraftApprovalRequest {
	return DraftApprovalRequest{
		DraftId:        draftId,
		DraftVersionId: draftVersionId,
		NotificationApprovalRequest: NotificationApprovalRequest{
			AppId:  appId,
			EnvId:  envId,
			UserId: userId,
		},
	}
}

func (claims *TokenCustomClaimsForNotification) setRegisteredClaims(registeredClaims jwt.RegisteredClaims) {
	claims.RegisteredClaims = registeredClaims
}

func (draftReq *DraftApprovalRequest) GetClaimsForDraftApprovalRequest() *TokenCustomClaimsForNotification {
	return &TokenCustomClaimsForNotification{
		DraftId:        draftReq.DraftId,
		DraftVersionId: draftReq.DraftVersionId,
		AppId:          draftReq.NotificationApprovalRequest.AppId,
		EnvId:          draftReq.NotificationApprovalRequest.EnvId,
		UserId:         draftReq.UserId,
		ApiTokenCustomClaims: ApiTokenCustomClaims{
			Email: draftReq.NotificationApprovalRequest.EmailId,
		},
	}
}

func (depReq *DeploymentApprovalRequest) GetClaimsForDeploymentApprovalRequest() *TokenCustomClaimsForNotification {
	return &TokenCustomClaimsForNotification{
		ApprovalRequestId: depReq.ApprovalRequestId,
		ArtifactId:        depReq.ArtifactId,
		PipelineId:        depReq.PipelineId,
		AppId:             depReq.NotificationApprovalRequest.AppId,
		EnvId:             depReq.NotificationApprovalRequest.EnvId,
		UserId:            depReq.UserId,
		ApiTokenCustomClaims: ApiTokenCustomClaims{
			Email: depReq.NotificationApprovalRequest.EmailId,
		},
	}
}

func (depReq *DeploymentApprovalRequest) CreateApprovalActionRequest() bean.UserApprovalActionRequest {
	return bean.UserApprovalActionRequest{
		AppId:             depReq.AppId,
		ActionType:        bean.APPROVAL_APPROVE_ACTION,
		ApprovalRequestId: depReq.ApprovalRequestId,
		PipelineId:        depReq.PipelineId,
		ArtifactId:        depReq.ArtifactId,
	}
}
