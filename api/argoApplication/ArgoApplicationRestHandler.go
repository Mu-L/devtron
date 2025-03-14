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

package argoApplication

import (
	"context"
	"errors"
	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/pkg/argoApplication"
	"github.com/devtron-labs/devtron/pkg/argoApplication/read"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
)

type ArgoApplicationRestHandler interface {
	ListApplications(w http.ResponseWriter, r *http.Request)
	GetApplicationDetail(w http.ResponseWriter, r *http.Request)
}

type ArgoApplicationRestHandlerImpl struct {
	argoApplicationService argoApplication.ArgoApplicationService
	readService            read.ArgoApplicationReadService
	logger                 *zap.SugaredLogger
	enforcer               casbin.Enforcer
}

func NewArgoApplicationRestHandlerImpl(argoApplicationService argoApplication.ArgoApplicationService,
	readService read.ArgoApplicationReadService, logger *zap.SugaredLogger, enforcer casbin.Enforcer) *ArgoApplicationRestHandlerImpl {
	return &ArgoApplicationRestHandlerImpl{
		argoApplicationService: argoApplicationService,
		readService:            readService,
		logger:                 logger,
		enforcer:               enforcer,
	}

}

func (handler *ArgoApplicationRestHandlerImpl) ListApplications(w http.ResponseWriter, r *http.Request) {
	// handle super-admin RBAC
	token := r.Header.Get("token")
	if ok := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionGet, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	v := r.URL.Query()
	clusterIdString := v.Get("clusterIds")
	var clusterIds []int
	if clusterIdString != "" {
		clusterIdSlices := strings.Split(clusterIdString, ",")
		for _, clusterId := range clusterIdSlices {
			id, err := strconv.Atoi(clusterId)
			if err != nil {
				handler.logger.Errorw("error in converting clusterId", "err", err, "clusterIdString", clusterIdString)
				common.WriteJsonResp(w, err, "please send valid cluster Ids", http.StatusBadRequest)
				return
			}
			clusterIds = append(clusterIds, id)
		}
	}
	resp, err := handler.argoApplicationService.ListApplications(clusterIds)
	if err != nil {
		handler.logger.Errorw("error in listing all argo applications", "err", err, "clusterIds", clusterIds)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, resp, http.StatusOK)
}

func (handler *ArgoApplicationRestHandlerImpl) GetApplicationDetail(w http.ResponseWriter, r *http.Request) {
	// handle super-admin RBAC
	token := r.Header.Get("token")
	if ok := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionGet, "*"); !ok {
		common.WriteJsonResp(w, errors.New("unauthorized"), nil, http.StatusForbidden)
		return
	}
	ctx := r.Context()
	ctx = context.WithValue(ctx, "token", token)

	var err error
	v := r.URL.Query()
	resourceName := v.Get("name")
	namespace := v.Get("namespace")
	clusterIdString := v.Get("clusterId")

	var clusterId int
	if clusterIdString != "" {
		clusterId, err = strconv.Atoi(clusterIdString)
		if err != nil {
			handler.logger.Errorw("error in converting clusterId", "err", err, "clusterIdString", clusterIdString)
			common.WriteJsonResp(w, err, "please send valid cluster Ids", http.StatusBadRequest)
			return
		}
	}
	resp, err := handler.readService.GetAppDetailEA(ctx, resourceName, namespace, clusterId)
	if err != nil {
		handler.logger.Errorw("error in getting argo application app detail", "err", err, "resourceName", resourceName, "clusterId", clusterId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, resp, http.StatusOK)
}
