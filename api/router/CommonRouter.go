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

package router

import (
	"github.com/devtron-labs/devtron/api/restHandler"
	"github.com/gorilla/mux"
)

type CommonRouter interface {
	InitCommonRouter(router *mux.Router)
}
type CommonRouterImpl struct {
	commonRestHandler restHandler.CommonRestHandler
}

func NewCommonRouterImpl(commonRestHandler restHandler.CommonRestHandler) *CommonRouterImpl {
	return &CommonRouterImpl{commonRestHandler: commonRestHandler}
}
func (impl CommonRouterImpl) InitCommonRouter(router *mux.Router) {
	router.Path("/checklist").
		HandlerFunc(impl.commonRestHandler.GlobalChecklist).
		Methods("GET")
	router.Path("/environment-variables").
		HandlerFunc(impl.commonRestHandler.EnvironmentVariableList).
		Methods("GET")
}
