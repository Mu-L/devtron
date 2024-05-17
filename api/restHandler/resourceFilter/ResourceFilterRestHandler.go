package resourceFilter

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devtron-labs/devtron/enterprise/pkg/expressionEvaluators"
	"net/http"
	"strconv"

	"github.com/devtron-labs/devtron/api/restHandler/common"
	"github.com/devtron-labs/devtron/enterprise/pkg/resourceFilter"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/devtron-labs/devtron/pkg/auth/user"
	"github.com/devtron-labs/devtron/util/rbac"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/go-playground/validator.v9"
)

const InvalidExpressionsStatusCode = 209

type ResourceFilterRestHandler interface {
	ListFilters(w http.ResponseWriter, r *http.Request)
	GetFilterById(w http.ResponseWriter, r *http.Request)
	UpdateFilter(w http.ResponseWriter, r *http.Request)
	CreateFilter(w http.ResponseWriter, r *http.Request)
	DeleteFilter(w http.ResponseWriter, r *http.Request)
	ValidateExpression(w http.ResponseWriter, r *http.Request)
	GetResourceFilterMetaData(w http.ResponseWriter, r *http.Request)
}

type ResourceFilterRestHandlerImpl struct {
	logger                *zap.SugaredLogger
	userAuthService       user.UserService
	enforcerUtil          rbac.EnforcerUtil
	enforcer              casbin.Enforcer
	resourceFilterService resourceFilter.ResourceFilterService
	celService            expressionEvaluators.CELEvaluatorService
	validator             *validator.Validate
	pipelineRepository    pipelineConfig.PipelineRepository
}

func NewResourceFilterRestHandlerImpl(logger *zap.SugaredLogger,
	userAuthService user.UserService,
	enforcerUtil rbac.EnforcerUtil,
	enforcer casbin.Enforcer,
	celService expressionEvaluators.CELEvaluatorService,
	resourceFilterService resourceFilter.ResourceFilterService,
	validator *validator.Validate,
	pipelineRepository pipelineConfig.PipelineRepository) *ResourceFilterRestHandlerImpl {
	return &ResourceFilterRestHandlerImpl{
		logger:                logger,
		userAuthService:       userAuthService,
		enforcerUtil:          enforcerUtil,
		enforcer:              enforcer,
		resourceFilterService: resourceFilterService,
		celService:            celService,
		validator:             validator,
		pipelineRepository:    pipelineRepository,
	}
}

func (handler *ResourceFilterRestHandlerImpl) ListFilters(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	authorised := handler.applySuperAdminAuth(token)
	if !authorised {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusUnauthorized)
		return
	}

	res, err := handler.resourceFilterService.ListFilters()
	if err != nil {
		handler.logger.Errorw("error in getting active resource filters", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusInternalServerError)
		return
	}
	common.WriteJsonResp(w, nil, res, http.StatusOK)
}

func (handler *ResourceFilterRestHandlerImpl) GetFilterById(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("token")
	authorised := handler.applySuperAdminAuth(token)
	if !authorised {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	filterId, err := strconv.Atoi(vars["id"])
	if err != nil {
		common.WriteJsonResp(w, errors.New(fmt.Sprintf("invalid param Id '%s'", vars["Id"])), nil, http.StatusBadRequest)
		return
	}
	res, err := handler.resourceFilterService.GetFilterById(filterId)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == resourceFilter.FilterNotFound {
			statusCode = http.StatusNotFound
		}
		handler.logger.Errorw("error in getting  resource filter", "err", err, "filterId", filterId)
		common.WriteJsonResp(w, err, nil, statusCode)
		return
	}
	common.WriteJsonResp(w, nil, res, http.StatusOK)
}

func (handler *ResourceFilterRestHandlerImpl) CreateFilter(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	token := r.Header.Get("token")
	authorised := handler.applySuperAdminAuth(token)
	if !authorised {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(r.Body)
	req := &resourceFilter.FilterRequestResponseBean{}
	err = decoder.Decode(req)
	if err != nil {
		handler.logger.Errorw("request err, Save", "error", err, "request", req)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	err = handler.validator.Struct(*req)
	if err != nil {
		handler.logger.Errorw("request err, Save", "error", err, "request", req)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	res, err := handler.resourceFilterService.CreateFilter(userId, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		handler.logger.Errorw("error in creating resource filters", "err", err)
		if err.Error() == resourceFilter.AppAndEnvSelectorRequiredMessage {
			statusCode = http.StatusBadRequest
		} else if err.Error() == resourceFilter.InvalidExpressions {
			err = nil
			statusCode = InvalidExpressionsStatusCode
		}
		common.WriteJsonResp(w, err, res, statusCode)
		return
	}
	common.WriteJsonResp(w, nil, res, http.StatusOK)
}

func (handler *ResourceFilterRestHandlerImpl) UpdateFilter(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	token := r.Header.Get("token")
	authorised := handler.applySuperAdminAuth(token)
	if !authorised {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	filterId, err := strconv.Atoi(vars["id"])
	if err != nil {
		common.WriteJsonResp(w, errors.New(fmt.Sprintf("invalid param Id '%s'", vars["Id"])), nil, http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	req := &resourceFilter.FilterRequestResponseBean{}
	err = decoder.Decode(req)
	if err != nil {
		handler.logger.Errorw("request err, Save", "error", err, "request", req)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	err = handler.validator.Struct(*req)
	if err != nil {
		handler.logger.Errorw("request err, Save", "error", err, "request", req)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	req.Id = filterId
	res, err := handler.resourceFilterService.UpdateFilter(userId, req)
	if err != nil {
		statusCode := http.StatusBadRequest
		handler.logger.Errorw("error in updating resource filters", "err", err)
		if err.Error() == resourceFilter.AppAndEnvSelectorRequiredMessage {
			statusCode = http.StatusBadRequest
		} else if err.Error() == resourceFilter.InvalidExpressions {
			err = nil
			statusCode = InvalidExpressionsStatusCode
		}
		common.WriteJsonResp(w, err, res, statusCode)
		return
	}
	common.WriteJsonResp(w, nil, nil, http.StatusOK)
}

func (handler *ResourceFilterRestHandlerImpl) DeleteFilter(w http.ResponseWriter, r *http.Request) {
	userId, err := handler.userAuthService.GetLoggedInUser(r)
	if err != nil {
		common.WriteJsonResp(w, err, "Unauthorized User", http.StatusUnauthorized)
		return
	}
	token := r.Header.Get("token")
	authorised := handler.applySuperAdminAuth(token)
	if !authorised {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	filterId, err := strconv.Atoi(vars["id"])
	if err != nil {
		handler.logger.Errorw("error in getting active resource filters", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	err = handler.resourceFilterService.DeleteFilter(userId, filterId)
	if err != nil {
		handler.logger.Errorw("error in deleting resource filters", "err", err)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}
	common.WriteJsonResp(w, nil, nil, http.StatusOK)
}

func (handler *ResourceFilterRestHandlerImpl) ValidateExpression(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	token := r.Header.Get("token")
	authorised := handler.applySuperAdminAuth(token)
	if !authorised {
		common.WriteJsonResp(w, fmt.Errorf("unauthorized user"), "Unauthorized User", http.StatusUnauthorized)
		return
	}
	var request expressionEvaluators.ValidateRequestResponse
	err := decoder.Decode(&request)
	if err != nil {
		handler.logger.Errorw("request err, UpdateRoleGroup", "err", err, "payload", request)
		common.WriteJsonResp(w, err, nil, http.StatusBadRequest)
		return
	}

	response, _ := handler.celService.ValidateCELRequest(request)
	common.WriteJsonResp(w, err, response, http.StatusOK)
}

func (handler *ResourceFilterRestHandlerImpl) applySuperAdminAuth(token string) bool {
	isSuperAdmin := handler.enforcer.Enforce(token, casbin.ResourceGlobal, casbin.ActionCreate, "*")

	return isSuperAdmin
}

func (handler *ResourceFilterRestHandlerImpl) GetResourceFilterMetaData(w http.ResponseWriter, r *http.Request) {
	res := resourceFilter.FILTER_CRITERIA
	common.WriteJsonResp(w, nil, res, http.StatusOK)
}
