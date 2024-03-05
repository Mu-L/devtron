package resourceQualifiers

import (
	"errors"
	"fmt"
	helper2 "github.com/devtron-labs/devtron/internal/sql/repository/helper"
	"github.com/devtron-labs/devtron/pkg/devtronResource/bean"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"go.uber.org/zap"
)

type QualifiersMappingRepository interface {
	// transaction util funcs
	sql.TransactionWrapper
	CreateQualifierMappings(qualifierMappings []*QualifierMapping, tx *pg.Tx) ([]*QualifierMapping, error)
	GetQualifierMappings(resourceType ResourceType, scope *Scope, searchableIdMap map[bean.DevtronResourceSearchableKeyName]int, resourceIds []int) ([]*QualifierMapping, error)
	GetQualifierMappingsForFilter(scope Scope, searchableIdMap map[bean.DevtronResourceSearchableKeyName]int) ([]*QualifierMapping, error)
	GetQualifierMappingsForFilterById(resourceId int) ([]*QualifierMapping, error)
	GetQualifierMappingsByResourceType(resourceType ResourceType) ([]*QualifierMapping, error)
	DeleteAllQualifierMappings(ResourceType, sql.AuditLog, *pg.Tx) error
	DeleteAllQualifierMappingsByResourceTypeAndId(resourceType ResourceType, resourceId int, auditLog sql.AuditLog, tx *pg.Tx) error
	DeleteByResourceTypeIdentifierKeyAndValue(resourceType ResourceType, identifierKey int, identifierValue int, auditLog sql.AuditLog, tx *pg.Tx) error
	DeleteAllByResourceTypeAndQualifierId(resourceType ResourceType, resourceId int, qualifierIds []int, auditLog sql.AuditLog, tx *pg.Tx) error
	DeleteAllByIds(qualifierMappingIds []int, auditLog sql.AuditLog, tx *pg.Tx) error
	GetDbConnection() *pg.DB
	DeleteGivenQualifierMappingsByResourceType(resourceType ResourceType, identifierKey int, identifierValueInts []int, auditLog sql.AuditLog, tx *pg.Tx) error
	GetActiveIdentifierCountPerResource(resourceType ResourceType, resourceIds []int, identifierKey int, identifierValueIntSpaceQuery string) ([]ResourceIdentifierCount, error)
	GetActiveMappingsCount(resourceType ResourceType, excludeIdentifiersQuery string, identifierKey int) (int, error)
	GetIdentifierIdsByResourceTypeAndIds(resourceType ResourceType, resourceIds []int, identifierKey int) ([]int, error)
	GetMappingsByResourceTypeAndIdsAndQualifierId(resourceType ResourceType, resourceIds []int, qualifier int) ([]*QualifierMapping, error)
	GetResourceIdsByIdentifier(resourceType ResourceType, identifierKey int, identifierId int) ([]int, error)
	GetQualifierMappingsWithIdentifierFilter(resourceType ResourceType, resourceId, identifierKey int, identifierValueStringLike, identifierValueSortOrder string, includeIdentifiersQuery string, limit, offset int, needTotalCount bool) ([]*QualifierMappingWithExtraColumns, error)
	GetQualifierMappingsForListOfQualifierValues(resourceType ResourceType, valuesMap map[Qualifier][][]int, searchableIdMap map[bean.DevtronResourceSearchableKeyName]int, resourceIds []int) ([]*QualifierMapping, error)
}

type QualifiersMappingRepositoryImpl struct {
	dbConnection *pg.DB
	*sql.TransactionUtilImpl
	logger *zap.SugaredLogger
}

func NewQualifiersMappingRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger) (*QualifiersMappingRepositoryImpl, error) {
	return &QualifiersMappingRepositoryImpl{
		dbConnection:        dbConnection,
		logger:              logger,
		TransactionUtilImpl: sql.NewTransactionUtilImpl(dbConnection),
	}, nil
}

func (repo *QualifiersMappingRepositoryImpl) CreateQualifierMappings(qualifierMappings []*QualifierMapping, tx *pg.Tx) ([]*QualifierMapping, error) {
	err := tx.Insert(&qualifierMappings)
	if err != nil {
		return nil, err
	}
	return qualifierMappings, nil
}

func (impl *QualifiersMappingRepositoryImpl) addScopeWhereClauseForFilter(query *orm.Query, scope Scope, searchableKeyNameIdMap map[bean.DevtronResourceSearchableKeyName]int) *orm.Query {
	return query.Where(
		"((identifier_key = ? AND identifier_value_int = ?) "+
			"OR (identifier_key = ? AND identifier_value_int IN (?)) "+
			"OR (identifier_key = ? AND identifier_value_int = ?) "+
			"OR (identifier_key = ? AND identifier_value_int IN (?)))",
		searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_APP_ID], scope.AppId,
		searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_PROJECT_ID], pg.In([]int{scope.ProjectId, AllProjectsInt}),
		searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_ENV_ID], scope.EnvId,
		searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_CLUSTER_ID], pg.In([]int{scope.ClusterId, GetEnvIdentifierValue(scope)}),
	)
}

func (repo *QualifiersMappingRepositoryImpl) GetQualifierMappingsForFilter(scope Scope, searchableIdMap map[bean.DevtronResourceSearchableKeyName]int) ([]*QualifierMapping, error) {
	var qualifierMappings []*QualifierMapping
	query := repo.dbConnection.Model(&qualifierMappings).
		Where("active = ?", true).
		Where("resource_type = ?", Filter)

	query = repo.addScopeWhereClauseForFilter(query, scope, searchableIdMap)
	err := query.Select()
	if err == pg.ErrNoRows {
		repo.logger.Errorw("no qualifier mappings found", "scope", scope)
		err = nil
	}
	return qualifierMappings, err
}
func (repo *QualifiersMappingRepositoryImpl) GetQualifierMappingsForFilterById(resourceId int) ([]*QualifierMapping, error) {
	var qualifierMappings []*QualifierMapping
	err := repo.dbConnection.Model(&qualifierMappings).
		Where("active = ?", true).
		Where("resource_type = ?", Filter).
		Where("resource_id = ?", resourceId).
		Select()
	if err == pg.ErrNoRows {
		return qualifierMappings, errors.New(fmt.Sprintf("no qualifier mappings found for given filter id %v", resourceId))
	}
	return qualifierMappings, nil
}

const appEnvCondition = "(((identifier_key = ? AND identifier_value_int in (?)) OR (identifier_key = ? AND identifier_value_int in (?))) AND qualifier_id = ?)"
const condition = "(qualifier_id = ? AND identifier_key = ? AND identifier_value_int in (?))"

func addCond(query *orm.Query, qualifier Qualifier, valuesMap map[Qualifier][][]int, identifierKey int) *orm.Query {
	if _, ok := valuesMap[qualifier]; ok {
		if len(valuesMap[qualifier][0]) > 0 {
			query = query.WhereOr(condition,
				qualifier, identifierKey, pg.In(valuesMap[qualifier][0]),
			)
		}
	}
	return query
}

func (repo *QualifiersMappingRepositoryImpl) addScopeWhereClauseBatch(q *orm.Query, valuesMap map[Qualifier][][]int, drs map[bean.DevtronResourceSearchableKeyName]int) *orm.Query {

	q = q.WhereGroup(func(query *orm.Query) (*orm.Query, error) {
		if _, ok := valuesMap[APP_AND_ENV_QUALIFIER]; ok {
			if len(valuesMap[APP_AND_ENV_QUALIFIER][0]) > 0 && len(valuesMap[APP_AND_ENV_QUALIFIER][1]) > 0 {
				query = query.WhereOr(appEnvCondition,
					drs[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_APP_ID], pg.In(valuesMap[APP_AND_ENV_QUALIFIER][0]),
					drs[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_ENV_ID], pg.In(valuesMap[APP_AND_ENV_QUALIFIER][1]),
					APP_AND_ENV_QUALIFIER,
				)
			}
		}
		query = addCond(query, APP_QUALIFIER, valuesMap, drs[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_APP_ID])
		query = addCond(query, ENV_QUALIFIER, valuesMap, drs[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_ENV_ID])
		query = addCond(query, CLUSTER_QUALIFIER, valuesMap, drs[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_CLUSTER_ID])
		query = addCond(query, PIPELINE_QUALIFIER, valuesMap, drs[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_PIPELINE_ID])
		query = query.WhereOr("(qualifier_id = ?)", GLOBAL_QUALIFIER)
		return query, nil
	})
	return q
}

func (repo *QualifiersMappingRepositoryImpl) addScopeWhereClause(query *orm.Query, scope *Scope, searchableKeyNameIdMap map[bean.DevtronResourceSearchableKeyName]int) *orm.Query {
	return query.Where(
		"(((identifier_key = ? AND identifier_value_int = ?) OR (identifier_key = ? AND identifier_value_int = ?)) AND qualifier_id = ?) "+
			"OR (qualifier_id = ? AND identifier_key = ? AND identifier_value_int = ?) "+
			"OR (qualifier_id = ? AND identifier_key = ? AND identifier_value_int = ?) "+
			"OR (qualifier_id = ? AND identifier_key = ? AND identifier_value_int = ?) "+
			"OR (qualifier_id = ? AND identifier_key = ? AND identifier_value_int = ?) "+
			"OR (qualifier_id = ?)",
		searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_APP_ID], scope.AppId, searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_ENV_ID], scope.EnvId, APP_AND_ENV_QUALIFIER,
		APP_QUALIFIER, searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_APP_ID], scope.AppId,
		ENV_QUALIFIER, searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_ENV_ID], scope.EnvId,
		CLUSTER_QUALIFIER, searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_CLUSTER_ID], scope.ClusterId,
		PIPELINE_QUALIFIER, searchableKeyNameIdMap[bean.DEVTRON_RESOURCE_SEARCHABLE_KEY_PIPELINE_ID], scope.PipelineId,
		GLOBAL_QUALIFIER,
	)
}

func (repo *QualifiersMappingRepositoryImpl) GetQualifierMappings(resourceType ResourceType, scope *Scope, searchableIdMap map[bean.DevtronResourceSearchableKeyName]int, resourceIds []int) ([]*QualifierMapping, error) {
	var qualifierMappings []*QualifierMapping
	query := repo.dbConnection.Model(&qualifierMappings).
		Where("active = ?", true).
		Where("resource_type = ?", resourceType)

	if len(resourceIds) > 0 {
		query = query.Where("resource_id IN (?)", pg.In(resourceIds))
	}

	// Enterprise Only
	if scope != nil {
		query = repo.addScopeWhereClause(query, scope, searchableIdMap)
	}

	err := query.Select()
	if err != nil {
		return nil, err
	}
	return qualifierMappings, nil
}

func (repo *QualifiersMappingRepositoryImpl) GetQualifierMappingsForListOfQualifierValues(resourceType ResourceType, valuesMap map[Qualifier][][]int, searchableIdMap map[bean.DevtronResourceSearchableKeyName]int, resourceIds []int) ([]*QualifierMapping, error) {
	var qualifierMappings []*QualifierMapping
	query := repo.dbConnection.Model(&qualifierMappings).
		Where("active = ?", true).
		Where("resource_type = ?", resourceType)

	if len(resourceIds) > 0 {
		query = query.Where("resource_id IN (?)", pg.In(resourceIds))
	}

	// Enterprise Only
	if valuesMap != nil {
		query = repo.addScopeWhereClauseBatch(query, valuesMap, searchableIdMap)
	}

	err := query.Select()
	if err != nil && err != pg.ErrNoRows {
		return nil, err
	}
	return qualifierMappings, nil
}

func (repo *QualifiersMappingRepositoryImpl) GetQualifierMappingsByResourceType(resourceType ResourceType) ([]*QualifierMapping, error) {
	var qualifierMappings []*QualifierMapping
	query := repo.dbConnection.Model(&qualifierMappings).
		Where("active = ?", true).
		Where("resource_type = ?", resourceType)
	err := query.Select()
	if err != nil {
		return nil, err
	}
	return qualifierMappings, nil
}

func (repo *QualifiersMappingRepositoryImpl) DeleteAllQualifierMappings(resourceType ResourceType, auditLog sql.AuditLog, tx *pg.Tx) error {
	_, err := repo.getQualifierMappingDeleteQuery(resourceType, tx, auditLog).
		Update()
	return err
}
func (repo *QualifiersMappingRepositoryImpl) DeleteAllQualifierMappingsByResourceTypeAndId(resourceType ResourceType, resourceId int, auditLog sql.AuditLog, tx *pg.Tx) error {
	_, err := tx.Model(&QualifierMapping{}).
		Set("updated_by = ?", auditLog.UpdatedBy).
		Set("updated_on = ?", auditLog.UpdatedOn).
		Set("active = ?", false).
		Where("active = ?", true).
		Where("resource_type = ?", resourceType).
		Where("resource_id = ?", resourceId).
		Update()
	return err
}

func (repo *QualifiersMappingRepositoryImpl) DeleteByResourceTypeIdentifierKeyAndValue(resourceType ResourceType, identifierKey int, identifierValue int, auditLog sql.AuditLog, tx *pg.Tx) error {
	_, err := repo.getQualifierMappingDeleteQuery(resourceType, tx, auditLog).
		Where("identifier_key = ?", identifierKey).
		Where("identifier_value_int = ?", identifierValue).
		Update()
	return err
}

func (repo *QualifiersMappingRepositoryImpl) DeleteAllByResourceTypeAndQualifierId(resourceType ResourceType, resourceId int, qualifierIds []int, auditLog sql.AuditLog, tx *pg.Tx) error {
	_, err := tx.Model(&QualifierMapping{}).
		Set("updated_by = ?", auditLog.UpdatedBy).
		Set("updated_on = ?", auditLog.UpdatedOn).
		Set("active = ?", false).
		Where("active = ?", true).
		Where("resource_type = ?", resourceType).
		Where("resource_id = ?", resourceId).
		Where("qualifier_id in (?)", pg.In(qualifierIds)).
		Update()
	return err
}

func (repo *QualifiersMappingRepositoryImpl) DeleteAllByIds(qualifierMappingIds []int, auditLog sql.AuditLog, tx *pg.Tx) error {
	_, err := tx.Model(&QualifierMapping{}).
		Set("updated_by = ?", auditLog.UpdatedBy).
		Set("updated_on = ?", auditLog.UpdatedOn).
		Set("active = ?", false).
		Where("active = ?", true).
		Where("id in (?)", pg.In(qualifierMappingIds)).
		Update()
	return err
}

func (repo *QualifiersMappingRepositoryImpl) getQualifierMappingDeleteQuery(resourceType ResourceType, tx *pg.Tx, auditLog sql.AuditLog) *orm.Query {
	return tx.Model(&QualifierMapping{}).
		Set("updated_by = ?", auditLog.UpdatedBy).
		Set("updated_on = ?", auditLog.UpdatedOn).
		Set("active = ?", false).
		Where("active = ?", true).
		Where("resource_type = ? ", resourceType)
}

func (repo *QualifiersMappingRepositoryImpl) GetDbConnection() *pg.DB {
	return repo.dbConnection
}

func (repo *QualifiersMappingRepositoryImpl) DeleteGivenQualifierMappingsByResourceType(resourceType ResourceType, identifierKey int, identifierValueInts []int, auditLog sql.AuditLog, tx *pg.Tx) error {
	_, err := tx.Model(&QualifierMapping{}).
		Set("updated_by=?", auditLog.UpdatedBy).
		Set("updated_on=?", auditLog.UpdatedOn).
		Set("active=?", false).
		Where("active=?", true).
		Where("resource_type=?", resourceType).
		Where("identifier_value_int IN (?)", pg.In(identifierValueInts)).
		Where("identifier_key=?", identifierKey).
		Update()
	return err
}

func (repo *QualifiersMappingRepositoryImpl) GetActiveIdentifierCountPerResource(resourceType ResourceType, resourceIds []int, identifierKey int, identifierValueIntSpaceQuery string) ([]ResourceIdentifierCount, error) {
	query := " SELECT COUNT(DISTINCT identifier_value_int) as identifier_count, resource_id" +
		" FROM resource_qualifier_mapping " +
		" WHERE resource_type = ? AND identifier_key = ? AND active=true "
	if identifierValueIntSpaceQuery != "" {
		query += " AND identifier_value_int IN (" + identifierValueIntSpaceQuery + ") "
	}
	if len(resourceIds) > 0 {
		query += fmt.Sprintf(" AND resource_id IN (%s) ", helper2.GetCommaSepratedString(resourceIds))
	}

	query += " GROUP BY resource_id"
	counts := make([]ResourceIdentifierCount, 0)
	_, err := repo.dbConnection.Query(&counts, query, resourceType, identifierKey)
	return counts, err
}

func (repo *QualifiersMappingRepositoryImpl) GetMappingsByResourceTypeAndIdsAndQualifierId(resourceType ResourceType, resourceIds []int, qualifier int) ([]*QualifierMapping, error) {
	mappings := make([]*QualifierMapping, 0)
	if len(resourceIds) == 0 {
		return mappings, nil
	}
	err := repo.dbConnection.Model(&mappings).
		Where("active = ?", true).
		Where("resource_type = ?", resourceType).
		Where("resource_id IN (?)", pg.In(resourceIds)).
		Where("qualifier_id = ?", qualifier).
		Select()
	if err == pg.ErrNoRows {
		err = nil
	}
	return mappings, err
}

func (repo *QualifiersMappingRepositoryImpl) GetIdentifierIdsByResourceTypeAndIds(resourceType ResourceType, resourceIds []int, identifierKey int) ([]int, error) {
	if len(resourceIds) == 0 {
		return nil, nil
	}

	var identifierIds []int
	query := "SELECT DISTINCT identifier_value_int " +
		" FROM resource_qualifier_mapping " +
		" WHERE resource_type = ? " +
		" AND identifier_key = ? " +
		" AND resource_id IN (?) " +
		" AND active=true"
	_, err := repo.dbConnection.Query(&identifierIds, query, resourceType, identifierKey, pg.In(resourceIds))
	return identifierIds, err
}

func (repo *QualifiersMappingRepositoryImpl) GetActiveMappingsCount(resourceType ResourceType, includeIdentifiersQuery string, identifierKey int) (int, error) {
	count, err := repo.dbConnection.Model(&QualifierMapping{}).
		Where("active = ?", true).
		Where("resource_type = ?", resourceType).
		Where("identifier_key = ?", identifierKey).
		Where("identifier_value_int IN (" + includeIdentifiersQuery + ")").
		Count()
	return count, err
}

func (repo *QualifiersMappingRepositoryImpl) GetQualifierMappingsWithIdentifierFilter(resourceType ResourceType, resourceId, identifierKey int, identifierValueStringLike, identifierValueSortOrder string, includeIdentifiersQuery string, limit, offset int, needTotalCount bool) ([]*QualifierMappingWithExtraColumns, error) {
	query := "SELECT identifier_value_int , identifier_value_string , resource_id "
	if needTotalCount {
		query += ",COUNT(id) OVER() AS total_count "
	}
	query += " FROM resource_qualifier_mapping "

	whereClause := fmt.Sprintf("WHERE resource_type = %d AND identifier_key = %d AND active=true ", resourceType, identifierKey)
	if resourceId > 0 {
		whereClause += fmt.Sprintf(" AND resource_id = %d ", resourceId)
	}
	if identifierValueStringLike != "" {
		whereClause += " AND identifier_value_string LIKE '%" + identifierValueStringLike + "%' "
	}

	if includeIdentifiersQuery != "" {
		whereClause += " AND identifier_value_int IN (" + includeIdentifiersQuery + ") "
	}

	query += whereClause

	if identifierValueSortOrder != "" {
		query += fmt.Sprintf(" ORDER BY identifier_value_string %s ", identifierValueSortOrder)
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d ", limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d ", offset)
	}

	var qualifierMappings []*QualifierMappingWithExtraColumns
	_, err := repo.dbConnection.Query(&qualifierMappings, query)
	return qualifierMappings, err
}

func (repo *QualifiersMappingRepositoryImpl) GetResourceIdsByIdentifier(resourceType ResourceType, identifierKey int, identifierId int) ([]int, error) {
	resourceIds := make([]int, 0)
	err := repo.dbConnection.Model((*QualifierMapping)(nil)).
		Column("resource_id").
		Where("active=?", true).
		Where("resource_type=?", resourceType).
		Where("identifier_key=?", identifierKey).
		Where("identifier_value_int=?", identifierId).
		Select(&resourceIds)
	return resourceIds, err
}
