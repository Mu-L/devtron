/*
 * Copyright (c) 2020-2024. Devtron Inc.
 */

package restHandler

import (
	"github.com/devtron-labs/devtron/api/restHandler/common"
	chartService "github.com/devtron-labs/devtron/pkg/chart"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/chartRef"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type ChartRefRestHandler interface {
	ChartRefAutocomplete(w http.ResponseWriter, r *http.Request)
	ChartRefAutocompleteForApp(w http.ResponseWriter, r *http.Request)
	ChartRefAutocompleteForEnv(w http.ResponseWriter, r *http.Request)
	ChartRefAutocompleteByChartId(w http.ResponseWriter, r *http.Request)
}

type ChartRefRestHandlerImpl struct {
	logger          *zap.SugaredLogger
	chartRefService chartRef.ChartRefService
	chartService    chartService.ChartService
}

func NewChartRefRestHandlerImpl(logger *zap.SugaredLogger, chartRefService chartRef.ChartRefService,
	chartService chartService.ChartService) *ChartRefRestHandlerImpl {
	handler := &ChartRefRestHandlerImpl{logger: logger, chartRefService: chartRefService, chartService: chartService}
	return handler
}

func (handler ChartRefRestHandlerImpl) ChartRefAutocomplete(w http.ResponseWriter, r *http.Request) {
	result, err := handler.chartRefService.ChartRefAutocomplete()
	if err != nil {
		handler.logger.Errorw("service err, ChartRefAutocomplete", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, result, http.StatusOK)
}

func (handler ChartRefRestHandlerImpl) ChartRefAutocompleteForApp(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.logger.Errorw("request err, ChartRefAutocompleteForApp", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	result, err := handler.chartService.ChartRefAutocompleteForAppOrEnv(appId, 0)
	if err != nil {
		handler.logger.Errorw("service err, ChartRefAutocompleteForApp", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, result, http.StatusOK)
}

func (handler ChartRefRestHandlerImpl) ChartRefAutocompleteForEnv(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	appId, err := strconv.Atoi(vars["appId"])
	if err != nil {
		handler.logger.Errorw("request err, ChartRefAutocompleteForEnv", "err", err, "appId", appId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	environmentId, err := strconv.Atoi(vars["environmentId"])
	if err != nil {
		handler.logger.Errorw("request err, ChartRefAutocompleteForEnv", "err", err, "appId", appId, "environmentId", environmentId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	result, err := handler.chartService.ChartRefAutocompleteForAppOrEnv(appId, environmentId)
	if err != nil {
		handler.logger.Errorw("service err, ChartRefAutocompleteForEnv", "err", err, "appId", appId, "environmentId", environmentId)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, result, http.StatusOK)
}

func (handler ChartRefRestHandlerImpl) ChartRefAutocompleteByChartId(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chartRefId, err := strconv.Atoi(vars["chartRefId"])
	if err != nil {
		handler.logger.Errorw("request err, ChartRefAutocompleteByChartId", "err", err, "appId", chartRefId)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	result, _, err := handler.chartRefService.GetAppOverrideForDefaultTemplate(chartRefId)
	if err != nil {
		handler.logger.Errorw("service err, ChartRefAutocompleteByChartId", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}

	common.WriteJsonResp(w, err, result, http.StatusOK)
}
