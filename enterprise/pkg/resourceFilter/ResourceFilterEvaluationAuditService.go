package resourceFilter

import (
	"fmt"
	"github.com/devtron-labs/devtron/enterprise/pkg/expressionEvaluators"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type FilterHistoryObject struct {
	FilterHistoryId int                              `json:"filter_history_id"`
	State           expressionEvaluators.FilterState `json:"state"`
	Message         string                           `json:"message"`
}

type FilterEvaluationAuditService interface {
	CreateFilterEvaluation(subjectType SubjectType, subjectId int, refType ReferenceType, refId int, filters []*FilterMetaDataBean, filterIdVsState map[int]expressionEvaluators.FilterState) (*ResourceFilterEvaluationAudit, error)
	UpdateFilterEvaluationAuditRef(id int, refType ReferenceType, refId int) error
	GetLastEvaluationFilterHistoryDataBySubjects(subjectType SubjectType, subjectIds []int, referenceId int, referenceType ReferenceType) (map[int]map[int]time.Time, map[int]expressionEvaluators.FilterState, error)
	GetLastEvaluationFilterHistoryDataBySubjectsAndReferences(subjectType SubjectType, subjectIds []int, referenceIds []int, referenceType ReferenceType) (map[string]map[int]time.Time, error)
	CreateFilterEvaluationAuditCustom(subjectType SubjectType, subjectId int, refType ReferenceType, refId int, filterHistoryObjectsStr string, filterType ResourceFilterType) (*ResourceFilterEvaluationAudit, error)
	GetLatestByRefAndMultiSubjectAndFilterType(referenceType ReferenceType, referenceId int, subjectType SubjectType, subjectIds []int, filterType ResourceFilterType) ([]*ResourceFilterEvaluationAudit, error)
	SaveFilterEvaluationAudit(tx *pg.Tx, subjectType SubjectType, subjectId int, referenceId int, referenceType ReferenceType, userId int32, filterHistoryObjects string, filterType ResourceFilterType) (*ResourceFilterEvaluationAudit, error)
	GetByIds(ids []int) ([]*ResourceFilterEvaluationAudit, error)
}

type FilterEvaluationAuditServiceImpl struct {
	logger                    *zap.SugaredLogger
	filterEvaluationAuditRepo FilterEvaluationAuditRepository
	filterAuditRepo           FilterAuditRepository
}

func NewFilterEvaluationAuditServiceImpl(logger *zap.SugaredLogger,
	filterEvaluationAuditRepo FilterEvaluationAuditRepository,
	filterAuditRepo FilterAuditRepository) *FilterEvaluationAuditServiceImpl {
	return &FilterEvaluationAuditServiceImpl{
		logger:                    logger,
		filterEvaluationAuditRepo: filterEvaluationAuditRepo,
		filterAuditRepo:           filterAuditRepo,
	}
}

// todo: complete this
func (impl *FilterEvaluationAuditServiceImpl) GetByIds(ids []int) ([]*ResourceFilterEvaluationAudit, error) {
	return impl.filterEvaluationAuditRepo.GetByIds(ids)
}

func (impl *FilterEvaluationAuditServiceImpl) SaveFilterEvaluationAudit(tx *pg.Tx, subjectType SubjectType, subjectId int, referenceId int, referenceType ReferenceType, userId int32, filterHistoryObjects string, filterType ResourceFilterType) (*ResourceFilterEvaluationAudit, error) {
	evaluationAudit := NewResourceFilterEvaluationAudit(&referenceType, referenceId, filterHistoryObjects, &subjectType, subjectId, sql.NewDefaultAuditLog(userId), filterType)
	createdEvaluationAudit, err := impl.filterEvaluationAuditRepo.Create(tx, &evaluationAudit)
	if err != nil {
		impl.logger.Errorw("error in saving evaluation audit", "evaluationAudit", evaluationAudit, "err", err)
		return nil, err
	}
	return createdEvaluationAudit, nil
}

func (impl *FilterEvaluationAuditServiceImpl) CreateFilterEvaluationAuditCustom(subjectType SubjectType, subjectId int, refType ReferenceType, refId int, filterHistoryObjectsStr string, filterType ResourceFilterType) (*ResourceFilterEvaluationAudit, error) {

	currentTime := time.Now()
	auditLog := sql.AuditLog{
		CreatedOn: currentTime,
		CreatedBy: 1,
	}

	filterEvaluationAudit := NewResourceFilterEvaluationAudit(&refType, refId, filterHistoryObjectsStr, &subjectType, subjectId, auditLog, filterType)
	savedFilterEvaluationAudit, err := impl.filterEvaluationAuditRepo.Create(nil, &filterEvaluationAudit)
	if err != nil {
		impl.logger.Errorw("error in saving resource filter evaluation result in resource_filter_evaluation_audit table", "err", err, "filterEvaluationAudit", filterEvaluationAudit)
		return savedFilterEvaluationAudit, err
	}
	return savedFilterEvaluationAudit, nil
}

func (impl *FilterEvaluationAuditServiceImpl) CreateFilterEvaluation(subjectType SubjectType, subjectId int, refType ReferenceType, refId int, filters []*FilterMetaDataBean, filterIdVsState map[int]expressionEvaluators.FilterState) (*ResourceFilterEvaluationAudit, error) {
	filterHistoryObjectsStr, err := impl.extractFilterHistoryObjects(filters, filterIdVsState)
	if err != nil {
		impl.logger.Errorw("error in extracting filter history objects", "err", err, "filters", filters, "filterIdVsState", filterIdVsState)
		return nil, err
	}

	currentTime := time.Now()
	auditLog := sql.AuditLog{
		CreatedOn: currentTime,
		CreatedBy: 1,
	}

	filterEvaluationAudit := NewResourceFilterEvaluationAudit(&refType, refId, filterHistoryObjectsStr, &subjectType, subjectId, auditLog, FILTER_CONDITION)
	savedFilterEvaluationAudit, err := impl.filterEvaluationAuditRepo.Create(nil, &filterEvaluationAudit)
	if err != nil {
		impl.logger.Errorw("error in saving resource filter evaluation result in resource_filter_evaluation_audit table", "err", err, "filterEvaluationAudit", filterEvaluationAudit)
		return savedFilterEvaluationAudit, err
	}
	return savedFilterEvaluationAudit, nil
}

func (impl *FilterEvaluationAuditServiceImpl) UpdateFilterEvaluationAuditRef(id int, refType ReferenceType, refId int) error {
	return impl.filterEvaluationAuditRepo.UpdateRefTypeAndRefId(id, refType, refId)
}

func (impl *FilterEvaluationAuditServiceImpl) GetLastEvaluationFilterHistoryDataBySubjects(subjectType SubjectType, subjectIds []int, referenceId int, referenceType ReferenceType) (map[int]map[int]time.Time, map[int]expressionEvaluators.FilterState, error) {

	// find the evaluation audit
	resourceFilterEvaluationAudits, err := impl.filterEvaluationAuditRepo.GetByRefAndMultiSubject(referenceType, referenceId, subjectType, subjectIds)
	if err != nil {
		impl.logger.Errorw("error in finding resource filters evaluation audit data", "referenceType", referenceType, "referenceId", referenceId, "subjectType", subjectType, "subjectIds", subjectIds, "err", err)
		return nil, nil, err
	}

	subjectIdVsfilterHistoryIdVsEvaluatedTimeMap := make(map[int]map[int]time.Time)
	subjectIdVsState := make(map[int]expressionEvaluators.FilterState)
	for _, resourceFilterEvaluationAudit := range resourceFilterEvaluationAudits {
		filterHistoryIdVsEvaluatedTimeMap, ok := subjectIdVsfilterHistoryIdVsEvaluatedTimeMap[resourceFilterEvaluationAudit.SubjectId]
		if !ok {
			filterHistoryIdVsEvaluatedTimeMap = make(map[int]time.Time)
		}
		filterHistoryObjects, err := getFilterHistoryObjectsFromJsonString(resourceFilterEvaluationAudit.FilterHistoryObjects)
		if err != nil {
			impl.logger.Errorw("error in extracting filter history objects from json string", "err", err, "jsonString", resourceFilterEvaluationAudit.FilterHistoryObjects)
			return nil, nil, err
		}

		subjectIdVsState[resourceFilterEvaluationAudit.SubjectId] = expressionEvaluators.ALLOW
		filterStateForSubject := true
		for _, filterHistoryObject := range filterHistoryObjects {
			filterHistoryIdVsEvaluatedTimeMap[filterHistoryObject.FilterHistoryId] = resourceFilterEvaluationAudit.CreatedOn
			filterStateForSubject = filterStateForSubject && (filterHistoryObject.State == expressionEvaluators.ALLOW)
		}
		subjectIdVsfilterHistoryIdVsEvaluatedTimeMap[resourceFilterEvaluationAudit.SubjectId] = filterHistoryIdVsEvaluatedTimeMap
		if !filterStateForSubject {
			subjectIdVsState[resourceFilterEvaluationAudit.SubjectId] = expressionEvaluators.BLOCK
		}
	}
	return subjectIdVsfilterHistoryIdVsEvaluatedTimeMap, subjectIdVsState, nil
}

func (impl *FilterEvaluationAuditServiceImpl) GetLastEvaluationFilterHistoryDataBySubjectsAndReferences(subjectType SubjectType, subjectIds []int, referenceIds []int, referenceType ReferenceType) (map[string]map[int]time.Time, error) {
	// find the evaluation audit
	resourceFilterEvaluationAudits, err := impl.filterEvaluationAuditRepo.GetByMultiRefAndMultiSubject(referenceType, referenceIds, subjectType, subjectIds)
	if err != nil {
		impl.logger.Errorw("error in finding resource filters evaluation audit data", "referenceType", referenceType, "referenceIds", referenceIds, "subjectType", subjectType, "subjectIds", subjectIds)
		return nil, err
	}

	subjectIdVsfilterHistoryIdVsEvaluatedTimeMap := make(map[string]map[int]time.Time)

	for _, resourceFilterEvaluationAudit := range resourceFilterEvaluationAudits {
		subjectAndRefKey := fmt.Sprintf("%v-%v", resourceFilterEvaluationAudit.SubjectId, resourceFilterEvaluationAudit.ReferenceId)
		filterHistoryIdVsEvaluatedTimeMap, ok := subjectIdVsfilterHistoryIdVsEvaluatedTimeMap[subjectAndRefKey]
		if !ok {
			filterHistoryIdVsEvaluatedTimeMap = make(map[int]time.Time)
		}
		filterHistoryObjects, err := getFilterHistoryObjectsFromJsonString(resourceFilterEvaluationAudit.FilterHistoryObjects)
		if err != nil {
			impl.logger.Errorw("error in extracting filter history objects from json string", "err", err, "jsonString", resourceFilterEvaluationAudit.FilterHistoryObjects)
			return nil, err
		}
		for _, filterHistoryObject := range filterHistoryObjects {
			filterHistoryIdVsEvaluatedTimeMap[filterHistoryObject.FilterHistoryId] = resourceFilterEvaluationAudit.CreatedOn
		}
		subjectIdVsfilterHistoryIdVsEvaluatedTimeMap[subjectAndRefKey] = filterHistoryIdVsEvaluatedTimeMap
	}
	return subjectIdVsfilterHistoryIdVsEvaluatedTimeMap, nil
}

func (impl *FilterEvaluationAuditServiceImpl) extractFilterHistoryObjects(filters []*FilterMetaDataBean, filterIdVsState map[int]expressionEvaluators.FilterState) (string, error) {
	filterIds := make([]int, 0)
	// store filtersMap here, later will help to identify filters that doesn't have filterAudit
	filtersMap := make(map[int]*FilterMetaDataBean)
	filterHistoryObjectMap := make(map[int]*FilterHistoryObject)
	for _, filter := range filters {
		filterIds = append(filterIds, filter.Id)
		filtersMap[filter.Id] = filter
		message := ""
		for _, condition := range filter.Conditions {
			message = fmt.Sprintf("\n%s conditionType : %v , errorMsg : %v", message, condition.ConditionType, condition.ErrorMsg)
		}
		filterHistoryObjectMap[filter.Id] = &FilterHistoryObject{
			State:   filterIdVsState[filter.Id],
			Message: message,
		}
	}

	resourceFilterEvaluationAudits, err := impl.filterAuditRepo.GetLatestResourceFilterAuditByFilterIds(filterIds)
	if err != nil {
		impl.logger.Errorw("error in getting latest resource filter audits for given filter id's", "filterIds", filterIds, "err", err)
		return "", err
	}

	for _, resourceFilterEvaluationAudit := range resourceFilterEvaluationAudits {
		if filterHistoryObject, ok := filterHistoryObjectMap[resourceFilterEvaluationAudit.FilterId]; ok {
			filterHistoryObject.FilterHistoryId = resourceFilterEvaluationAudit.Id

			// delete filter from filtersMap for which we found filter audit
			delete(filtersMap, resourceFilterEvaluationAudit.FilterId)
		}
	}

	// if filtersMap is not empty ,there are some filters for which we never stored audit entry, so create filter audit for those
	if len(filtersMap) > 0 {
		filterHistoryObjectMap, err = impl.createFilterAuditForMissingFilters(filtersMap, filterHistoryObjectMap)
		if err != nil {
			impl.logger.Errorw("error in creating filter audit data for missing filters", "missingFiltersMap", filtersMap, "err", err)
			return "", err
		}
	}

	filterHistoryObjects := make([]*FilterHistoryObject, 0, len(filterHistoryObjectMap))
	for _, val := range filterHistoryObjectMap {
		filterHistoryObjects = append(filterHistoryObjects, val)
	}
	jsonStr, err := getJsonStringFromFilterHistoryObjects(filterHistoryObjects)
	if err != nil {
		impl.logger.Errorw("error in getting json string for filter history objects", "filterHistoryObjects", filterHistoryObjects, "err", err)
		return "", err
	}
	return jsonStr, err

}

// createFilterAuditForMissingFilters will create snapshot of filter data in filter audit table and gets updated filterHistoryObjectMap.
// this function exists because filter auditing is added later, so there is possibility that filters exist without any auditing data
func (impl *FilterEvaluationAuditServiceImpl) createFilterAuditForMissingFilters(filtersMap map[int]*FilterMetaDataBean, filterHistoryObjectMap map[int]*FilterHistoryObject) (map[int]*FilterHistoryObject, error) {
	tx, err := impl.filterAuditRepo.StartTx()
	if err != nil {
		impl.logger.Errorw("error in starting db transaction", "err", err)
		return filterHistoryObjectMap, err
	}

	defer impl.filterAuditRepo.RollbackTx(tx)

	for _, filter := range filtersMap {
		conditionsStr, err := getJsonStringFromResourceCondition(filter.Conditions)
		if err != nil {
			impl.logger.Errorw("error in getting json string from filter conditions", "err", err, "filterConditions", filter.Conditions)
			return filterHistoryObjectMap, err
		}
		action := Create
		userId := int32(1) // system user
		filterAudit := NewResourceFilterAudit(filter.Id, conditionsStr, filter.TargetObject, &action, userId)
		savedFilterAudit, err := impl.filterAuditRepo.CreateResourceFilterAudit(tx, &filterAudit)
		if err != nil {
			impl.logger.Errorw("error in creating filter audit for missing filters", "err", err, "filterAudit", filterAudit)
			return filterHistoryObjectMap, err
		}

		if filterHistoryObject, ok := filterHistoryObjectMap[savedFilterAudit.FilterId]; ok {
			filterHistoryObject.FilterHistoryId = savedFilterAudit.Id
		}

	}
	err = impl.filterAuditRepo.CommitTx(tx)
	if err != nil {
		impl.logger.Errorw("error in committing db transaction", "err", err)
		return filterHistoryObjectMap, err
	}

	return filterHistoryObjectMap, err
}

func (impl *FilterEvaluationAuditServiceImpl) GetLatestByRefAndMultiSubjectAndFilterType(referenceType ReferenceType, referenceId int, subjectType SubjectType, subjectIds []int, filterType ResourceFilterType) ([]*ResourceFilterEvaluationAudit, error) {
	return impl.filterEvaluationAuditRepo.GetLatestByRefAndMultiSubjectAndFilterType(referenceType, referenceId, subjectType, subjectIds, filterType)
}
