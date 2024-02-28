package user

import (
	"fmt"
	bean3 "github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin/bean"
	"github.com/devtron-labs/devtron/pkg/auth/user/adapter"
	"github.com/devtron-labs/devtron/pkg/auth/user/helper"
	helper2 "github.com/devtron-labs/devtron/pkg/auth/user/repository/helper"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	"github.com/devtron-labs/authenticator/middleware"
	"github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	bean2 "github.com/devtron-labs/devtron/pkg/auth/user/bean"
	"github.com/devtron-labs/devtron/pkg/auth/user/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

type UserCommonService interface {
	GetPValUpdateMap(team, entityName, env, entity, cluster, namespace, group, kind, resource string, approver bool, workflow string) map[repository.PValUpdateKey]string
	GetRenderedRoleData(defaultRoleData repository.RoleCacheDetailObj, pValUpdateMap map[repository.PValUpdateKey]string) *repository.RoleModel
	GetRenderedPolicy(defaultPolicy repository.PolicyCacheDetailObj, pValUpdateMap map[repository.PValUpdateKey]string) []bean3.Policy
	CreateDefaultPoliciesForAllTypes(team, entityName, env, entity, cluster, namespace, group, kind, resource, actionType, accessType string, approver bool, workflow string, userId int32) (bool, error, []bean3.Policy)
	RemoveRolesAndReturnEliminatedPolicies(userInfo *bean.UserInfo, existingRoleIds map[int]repository.UserRoleModel, eliminatedRoleIds map[int]*repository.UserRoleModel, tx *pg.Tx, token string, managerAuth func(resource, token, object string) bool) ([]bean3.Policy, error)
	RemoveRolesAndReturnEliminatedPoliciesForGroups(request *bean.RoleGroup, existingRoles map[int]*repository.RoleGroupRoleMapping, eliminatedRoles map[int]*repository.RoleGroupRoleMapping, tx *pg.Tx, token string, managerAuth func(resource string, token string, object string) bool) ([]bean3.Policy, error)
	CheckRbacForClusterEntity(cluster, namespace, group, kind, resource, token string, managerAuth func(resource, token, object string) bool) bool
	GetCapacityForRoleFilter(roleFilters []bean.RoleFilter) (int, map[int]int)
	MergeCustomRoleFilters(roleFilters []bean.RoleFilter) []bean.RoleFilter
	BuildRoleFilterKeyForCluster(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string)
	BuildRoleFilterKeyForJobs(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string)
	BuildRoleFilterKeyForOtherEntity(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string)
	BuildRoleFilterForAllTypes(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string)
	GetUniqueKeyForAllEntity(role repository.RoleModel) string
	GetUniqueKeyForAllEntityWithTimeAndStatus(role repository.RoleModel, status bean.Status, timeout time.Time) string
	GetUniqueKeyForRoleFilter(roleFilter bean.RoleFilter) string
	SetDefaultValuesIfNotPresent(request *bean.ListingRequest, isRoleGroup bool)
	DeleteRoleForUserFromCasbin(mappings map[string][]bean3.GroupPolicy) bool
	DeleteUserForRoleFromCasbin(mappings map[string][]bean3.GroupPolicy) bool
}

type UserCommonServiceImpl struct {
	userAuthRepository          repository.UserAuthRepository
	logger                      *zap.SugaredLogger
	userRepository              repository.UserRepository
	roleGroupRepository         repository.RoleGroupRepository
	sessionManager2             *middleware.SessionManager
	defaultRbacDataCacheFactory repository.RbacDataCacheFactory
	userRbacConfig              *UserRbacConfig
}

func NewUserCommonServiceImpl(userAuthRepository repository.UserAuthRepository,
	logger *zap.SugaredLogger,
	userRepository repository.UserRepository,
	userGroupRepository repository.RoleGroupRepository,
	sessionManager2 *middleware.SessionManager,
	defaultRbacDataCacheFactory repository.RbacDataCacheFactory) *UserCommonServiceImpl {
	userConfig := &UserRbacConfig{}
	err := env.Parse(userConfig)
	if err != nil {
		logger.Fatal("error occurred while parsing user config", err)
	}
	serviceImpl := &UserCommonServiceImpl{
		userAuthRepository:          userAuthRepository,
		logger:                      logger,
		userRepository:              userRepository,
		roleGroupRepository:         userGroupRepository,
		sessionManager2:             sessionManager2,
		defaultRbacDataCacheFactory: defaultRbacDataCacheFactory,
		userRbacConfig:              userConfig,
	}
	cStore = sessions.NewCookieStore(randKey())
	defaultRbacDataCacheFactory.SyncPolicyCache()
	defaultRbacDataCacheFactory.SyncRoleDataCache()
	return serviceImpl
}

type UserRbacConfig struct {
	UseRbacCreationV2 bool `env:"USE_RBAC_CREATION_V2" envDefault:"true"`
}

func (impl UserCommonServiceImpl) CreateDefaultPoliciesForAllTypes(team, entityName, env, entity, cluster, namespace, group, kind, resource, actionType, accessType string, approver bool, workflow string, userId int32) (bool, error, []bean3.Policy) {
	if impl.userRbacConfig.UseRbacCreationV2 {
		impl.logger.Debugw("using rbac creation v2 for creating default policies")
		return impl.CreateDefaultPoliciesForAllTypesV2(team, entityName, env, entity, cluster, namespace, group, kind, resource, actionType, accessType, approver, workflow)
	} else {
		return impl.userAuthRepository.CreateDefaultPoliciesForAllTypes(team, entityName, env, entity, cluster, namespace, group, kind, resource, actionType, accessType, approver, userId)
	}
}

func (impl UserCommonServiceImpl) CreateDefaultPoliciesForAllTypesV2(team, entityName, env, entity, cluster, namespace, group, kind, resource, actionType, accessType string, approver bool, workflow string) (bool, error, []bean3.Policy) {
	//TODO: below txn is making this process slow, need to do bulk operation for role creation.
	//For detail - https://github.com/devtron-labs/devtron/blob/main/pkg/user/benchmarking-results

	renderedRole, renderedPolicyDetails, err := impl.getRenderedRoleAndPolicy(team, entityName, env, entity, cluster, namespace, group, kind, resource, actionType, accessType, approver, workflow)
	if err != nil {
		return false, err, nil
	}
	_, err = impl.userAuthRepository.CreateRole(renderedRole)
	if err != nil && strings.Contains("duplicate key value violates unique constraint", err.Error()) {
		return false, err, nil
	}
	return true, nil, renderedPolicyDetails
}

func (impl UserCommonServiceImpl) getRenderedRoleAndPolicy(team, entityName, env, entity, cluster, namespace, group, kind, resource, actionType, accessType string, approver bool, workflow string) (*repository.RoleModel, []bean3.Policy, error) {
	//getting map of values to be used for rendering
	pValUpdateMap := impl.GetPValUpdateMap(team, entityName, env, entity, cluster, namespace, group, kind, resource, approver, workflow)

	//getting default role data and policy
	defaultRoleData, defaultPolicy, err := impl.getDefaultRbacRoleAndPolicyByRoleFilter(entity, accessType, actionType)
	if err != nil {
		return nil, nil, err
	}
	//getting rendered role and policy data
	renderedRoleData := impl.GetRenderedRoleData(defaultRoleData, pValUpdateMap)
	renderedPolicy := impl.GetRenderedPolicy(defaultPolicy, pValUpdateMap)

	return renderedRoleData, renderedPolicy, nil
}

func (impl UserCommonServiceImpl) getDefaultRbacRoleAndPolicyByRoleFilter(entity, accessType, action string) (repository.RoleCacheDetailObj, repository.PolicyCacheDetailObj, error) {
	//getting default role and policy data from cache
	return impl.defaultRbacDataCacheFactory.
		GetDefaultRoleDataAndPolicyByEntityAccessTypeAndRoleType(entity, accessType, action)
}

func (impl UserCommonServiceImpl) GetRenderedRoleData(defaultRoleData repository.RoleCacheDetailObj, pValUpdateMap map[repository.PValUpdateKey]string) *repository.RoleModel {
	renderedRoleData := &repository.RoleModel{
		Role:        getResolvedValueFromPValDetailObject(defaultRoleData.Role, pValUpdateMap).String(),
		Entity:      getResolvedValueFromPValDetailObject(defaultRoleData.Entity, pValUpdateMap).String(),
		EntityName:  getResolvedValueFromPValDetailObject(defaultRoleData.EntityName, pValUpdateMap).String(),
		Team:        getResolvedValueFromPValDetailObject(defaultRoleData.Team, pValUpdateMap).String(),
		Environment: getResolvedValueFromPValDetailObject(defaultRoleData.Environment, pValUpdateMap).String(),
		AccessType:  getResolvedValueFromPValDetailObject(defaultRoleData.AccessType, pValUpdateMap).String(),
		Action:      getResolvedValueFromPValDetailObject(defaultRoleData.Action, pValUpdateMap).String(),
		Cluster:     getResolvedValueFromPValDetailObject(defaultRoleData.Cluster, pValUpdateMap).String(),
		Namespace:   getResolvedValueFromPValDetailObject(defaultRoleData.Namespace, pValUpdateMap).String(),
		Group:       getResolvedValueFromPValDetailObject(defaultRoleData.Group, pValUpdateMap).String(),
		Kind:        getResolvedValueFromPValDetailObject(defaultRoleData.Kind, pValUpdateMap).String(),
		Resource:    getResolvedValueFromPValDetailObject(defaultRoleData.Resource, pValUpdateMap).String(),
		Approver:    getResolvedValueFromPValDetailObject(defaultRoleData.Approver, pValUpdateMap).Boolean(),
		Workflow:    getResolvedValueFromPValDetailObject(defaultRoleData.Workflow, pValUpdateMap).String(),
		AuditLog: sql.AuditLog{ //not storing user information because this role can be mapped to other users in future and hence can lead to confusion
			CreatedOn: time.Now(),
			UpdatedOn: time.Now(),
		},
	}
	return renderedRoleData
}

func (impl UserCommonServiceImpl) GetRenderedPolicy(defaultPolicy repository.PolicyCacheDetailObj, pValUpdateMap map[repository.PValUpdateKey]string) []bean3.Policy {
	renderedPolicies := make([]bean3.Policy, 0, len(defaultPolicy.ResActObjSet))
	policyType := getResolvedValueFromPValDetailObject(defaultPolicy.Type, pValUpdateMap)
	policySub := getResolvedValueFromPValDetailObject(defaultPolicy.Sub, pValUpdateMap)
	for _, v := range defaultPolicy.ResActObjSet {
		policyRes := getResolvedValueFromPValDetailObject(v.Res, pValUpdateMap)
		policyAct := getResolvedValueFromPValDetailObject(v.Act, pValUpdateMap)
		policyObj := getResolvedValueFromPValDetailObject(v.Obj, pValUpdateMap)
		renderedPolicy := bean3.Policy{
			Type: bean3.PolicyType(policyType.String()),
			Sub:  bean3.Subject(policySub.String()),
			Res:  bean3.Resource(policyRes.String()),
			Act:  bean3.Action(policyAct.String()),
			Obj:  bean3.Object(policyObj.String()),
		}
		renderedPolicies = append(renderedPolicies, renderedPolicy)
	}
	return renderedPolicies
}

func getResolvedValueFromPValDetailObject(pValDetailObj repository.PValDetailObj, pValUpdateMap map[repository.PValUpdateKey]string) repository.PValResolvedValue {
	if len(pValDetailObj.IndexKeyMap) == 0 {
		return repository.NewPValResolvedValue(pValDetailObj.Value)
	}
	pValBytes := []byte(pValDetailObj.Value)
	var resolvedValueInBytes []byte
	for i, pValByte := range pValBytes {
		if pValByte == '%' {
			valUpdateKey := pValDetailObj.IndexKeyMap[i]
			val := pValUpdateMap[valUpdateKey]
			resolvedValueInBytes = append(resolvedValueInBytes, []byte(val)...)
		} else {
			resolvedValueInBytes = append(resolvedValueInBytes, pValByte)
		}
	}
	return repository.NewPValResolvedValue(string(resolvedValueInBytes))
}

func (impl UserCommonServiceImpl) GetPValUpdateMap(team, entityName, env, entity, cluster,
	namespace, group, kind, resource string, approver bool, workflow string) map[repository.PValUpdateKey]string {
	pValUpdateMap := make(map[repository.PValUpdateKey]string)
	pValUpdateMap[repository.EntityPValUpdateKey] = entity
	if entity == bean.CLUSTER_ENTITIY {
		pValUpdateMap[repository.ClusterPValUpdateKey] = cluster
		pValUpdateMap[repository.NamespacePValUpdateKey] = namespace
		pValUpdateMap[repository.GroupPValUpdateKey] = group
		pValUpdateMap[repository.KindPValUpdateKey] = kind
		pValUpdateMap[repository.ResourcePValUpdateKey] = resource
		pValUpdateMap[repository.ClusterObjPValUpdateKey] = getResolvedPValMapValue(cluster)
		pValUpdateMap[repository.NamespaceObjPValUpdateKey] = getResolvedPValMapValue(namespace)
		pValUpdateMap[repository.GroupObjPValUpdateKey] = getResolvedPValMapValue(group)
		pValUpdateMap[repository.KindObjPValUpdateKey] = getResolvedPValMapValue(kind)
		pValUpdateMap[repository.ResourceObjPValUpdateKey] = getResolvedPValMapValue(resource)
	} else {
		pValUpdateMap[repository.EntityNamePValUpdateKey] = entityName
		pValUpdateMap[repository.TeamPValUpdateKey] = team
		pValUpdateMap[repository.AppPValUpdateKey] = entityName
		pValUpdateMap[repository.EnvPValUpdateKey] = env
		pValUpdateMap[repository.TeamObjPValUpdateKey] = getResolvedPValMapValue(team)
		pValUpdateMap[repository.AppObjPValUpdateKey] = getResolvedPValMapValue(entityName)
		pValUpdateMap[repository.EnvObjPValUpdateKey] = getResolvedPValMapValue(env)
		pValUpdateMap[repository.ApproverPValUpdateKey] = strconv.FormatBool(approver)
		if entity == bean2.EntityJobs {
			pValUpdateMap[repository.WorkflowPValUpdateKey] = workflow
			pValUpdateMap[repository.WorkflowObjPValUpdateKey] = getResolvedPValMapValue(workflow)
		}
	}
	return pValUpdateMap
}

func getResolvedPValMapValue(rawValue string) string {
	resolvedVal := rawValue
	if rawValue == "" {
		resolvedVal = "*"
	}
	return resolvedVal
}

func (impl UserCommonServiceImpl) RemoveRolesAndReturnEliminatedPolicies(userInfo *bean.UserInfo,
	existingRoleIds map[int]repository.UserRoleModel, eliminatedRoleIds map[int]*repository.UserRoleModel,
	tx *pg.Tx, token string, managerAuth func(resource, token, object string) bool) ([]bean3.Policy, error) {
	var eliminatedPolicies []bean3.Policy
	// this map keeps the role id vs bool value for storing if existing role is given with different timeoutWindowConfiguration, handling multiple same rows.
	// for eg . user has (p1,e1,a1,admin ,active) combination  and multiple rows come in request for (p1,e1,a1,admin, inactive) so this maps handles this.
	timeoutChangedMap := make(map[int]bool)
	// DELETE Removed Items
	for _, roleFilter := range userInfo.RoleFilters {
		roleFilterStatus := helper.GetActualStatusFromExpressionAndStatus(roleFilter.Status, roleFilter.TimeoutWindowExpression)
		if roleFilter.Entity == bean.CLUSTER_ENTITIY {
			namespaces := strings.Split(roleFilter.Namespace, ",")
			groups := strings.Split(roleFilter.Group, ",")
			kinds := strings.Split(roleFilter.Kind, ",")
			resources := strings.Split(roleFilter.Resource, ",")
			accessType := roleFilter.AccessType
			actionType := roleFilter.Action
			for _, namespace := range namespaces {
				for _, group := range groups {
					for _, kind := range kinds {
						for _, resource := range resources {
							isValidAuth := impl.CheckRbacForClusterEntity(roleFilter.Cluster, namespace, group, kind, resource, token, managerAuth)
							if !isValidAuth {
								continue
							}
							roleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(roleFilter.Entity, "", "", "", "", roleFilter.Approver, accessType, roleFilter.Cluster, namespace, group, kind, resource, actionType, false, "")
							if err != nil {
								impl.logger.Errorw("Error in fetching roles by filter", "roleFilter", roleFilter)
								return nil, err
							}
							if roleModel.Id == 0 {
								impl.logger.Warnw("no role found for given filter", "filter", roleFilter)
								continue
							}
							if val, ok := existingRoleIds[roleModel.Id]; ok {
								hasTimeChanged := helper.HasTimeWindowChanged(roleFilterStatus, roleFilter.TimeoutWindowExpression, val.TimeoutWindowConfiguration)
								if hasTimeChanged {
									timeoutChangedMap[roleModel.Id] = true
								} else {
									delete(eliminatedRoleIds, roleModel.Id)
								}
							}
						}
					}
				}
			}
		} else if roleFilter.Entity == bean2.EntityJobs {
			if len(roleFilter.Team) > 0 { // check auth only for apps permission, skip for chart group
				rbacObject := fmt.Sprintf("%s", roleFilter.Team)
				isValidAuth := managerAuth(casbin.ResourceUser, token, rbacObject)
				if !isValidAuth {
					continue
				}
			}
			entityNames := strings.Split(roleFilter.EntityName, ",")
			environments := strings.Split(roleFilter.Environment, ",")
			workflows := strings.Split(roleFilter.Workflow, ",")
			actionType := roleFilter.Action
			accessType := roleFilter.AccessType
			for _, environment := range environments {
				for _, entityName := range entityNames {
					for _, workflow := range workflows {
						roleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(roleFilter.Entity, roleFilter.Team, entityName, environment, actionType, false, accessType, "", "", "", "", "", actionType, false, workflow)
						if err != nil {
							impl.logger.Errorw("Error in fetching roles by filter", "user", userInfo)
							return nil, err
						}
						if roleModel.Id == 0 {
							impl.logger.Debugw("no role found for given filter", "filter", roleFilter)
							userInfo.Status = "role not fount for any given filter: " + roleFilter.Team + "," + environment + "," + entityName + "," + roleFilter.Action
							continue
						}
						if val, ok := existingRoleIds[roleModel.Id]; ok {
							hasTimeChanged := helper.HasTimeWindowChanged(roleFilterStatus, roleFilter.TimeoutWindowExpression, val.TimeoutWindowConfiguration)
							if hasTimeChanged {
								timeoutChangedMap[roleModel.Id] = true
							} else {
								delete(eliminatedRoleIds, roleModel.Id)
							}
						}
					}
				}
			}
		} else {
			if len(roleFilter.Team) > 0 { // check auth only for apps permission, skip for chart group
				rbacObject := fmt.Sprintf("%s", roleFilter.Team)
				isValidAuth := managerAuth(casbin.ResourceUser, token, rbacObject)
				if !isValidAuth {
					continue
				}
			}
			entityNames := strings.Split(roleFilter.EntityName, ",")
			environments := strings.Split(roleFilter.Environment, ",")
			actions := strings.Split(roleFilter.Action, ",")
			accessType := roleFilter.AccessType
			for _, environment := range environments {
				for _, entityName := range entityNames {
					for _, actionType := range actions {
						roleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(roleFilter.Entity, roleFilter.Team, entityName, environment, actionType, roleFilter.Approver, accessType, "", "", "", "", "", actionType, false, "")
						if err != nil {
							impl.logger.Errorw("Error in fetching roles by filter", "user", userInfo)
							return nil, err
						}
						oldRoleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(roleFilter.Entity, roleFilter.Team, entityName, environment, actionType, roleFilter.Approver, accessType, "", "", "", "", "", actionType, true, "")
						if err != nil {
							return nil, err
						}
						if roleModel.Id == 0 {
							impl.logger.Debugw("no role found for given filter", "filter", roleFilter)
							userInfo.Status = "role not fount for any given filter: " + roleFilter.Team + "," + environment + "," + entityName + "," + roleFilter.Action
							continue
						}
						if val, ok := existingRoleIds[roleModel.Id]; ok {
							hasTimeChanged := helper.HasTimeWindowChanged(roleFilterStatus, roleFilter.TimeoutWindowExpression, val.TimeoutWindowConfiguration)
							if hasTimeChanged {
								timeoutChangedMap[roleModel.Id] = true
							} else {
								delete(eliminatedRoleIds, roleModel.Id)
							}
						}
						isChartGroupEntity := roleFilter.Entity == bean.CHART_GROUP_ENTITY
						if _, ok := existingRoleIds[oldRoleModel.Id]; ok && !isChartGroupEntity {
							//delete old role mapping from existing but not from eliminated roles (so that it gets deleted)
							delete(existingRoleIds, oldRoleModel.Id)
						}
					}
				}
			}
		}
	}
	// deleting from existingRoleIds map if timeout has changed
	for id, _ := range timeoutChangedMap {
		delete(existingRoleIds, id)
	}

	// delete remaining Ids from casbin role mapping table in orchestrator and casbin policy db
	// which are existing but not provided in this request

	for _, userRoleModel := range eliminatedRoleIds {
		role, err := impl.userAuthRepository.GetRoleById(userRoleModel.RoleId)
		if err != nil {
			return nil, err
		}
		if len(role.Team) > 0 {
			rbacObject := fmt.Sprintf("%s", role.Team)
			isValidAuth := managerAuth(casbin.ResourceUser, token, rbacObject)
			if !isValidAuth {
				continue
			}
		}
		if role.Entity == bean.CLUSTER_ENTITIY {
			isValidAuth := impl.CheckRbacForClusterEntity(role.Cluster, role.Namespace, role.Group, role.Kind, role.Resource, token, managerAuth)
			if !isValidAuth {
				continue
			}
		}
		_, err = impl.userAuthRepository.DeleteUserRoleMapping(userRoleModel, tx)
		if err != nil {
			impl.logger.Errorw("Error in delete user role mapping", "user", userInfo)
			return nil, err
		}
		timeExpression, expressionFormat := helper2.GetCasbinFormattedTimeAndFormat(userRoleModel.TimeoutWindowConfiguration)

		casbinPolicy := adapter.GetCasbinGroupPolicy(userInfo.EmailId, role.Role, timeExpression, expressionFormat)
		eliminatedPolicies = append(eliminatedPolicies, casbinPolicy)
	}
	// DELETE ENDS
	return eliminatedPolicies, nil
}

func (impl UserCommonServiceImpl) RemoveRolesAndReturnEliminatedPoliciesForGroups(request *bean.RoleGroup, existingRoles map[int]*repository.RoleGroupRoleMapping, eliminatedRoles map[int]*repository.RoleGroupRoleMapping, tx *pg.Tx, token string, managerAuth func(resource string, token string, object string) bool) ([]bean3.Policy, error) {
	// Filter out removed items in current request
	//var policies []casbin.Policy
	for _, roleFilter := range request.RoleFilters {
		entity := roleFilter.Entity
		if entity == bean.CLUSTER_ENTITIY {
			namespaces := strings.Split(roleFilter.Namespace, ",")
			groups := strings.Split(roleFilter.Group, ",")
			kinds := strings.Split(roleFilter.Kind, ",")
			resources := strings.Split(roleFilter.Resource, ",")
			actionType := roleFilter.Action
			accessType := roleFilter.AccessType
			for _, namespace := range namespaces {
				for _, group := range groups {
					for _, kind := range kinds {
						for _, resource := range resources {
							isValidAuth := impl.CheckRbacForClusterEntity(roleFilter.Cluster, namespace, group, kind, resource, token, managerAuth)
							if !isValidAuth {
								continue
							}
							roleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(entity, "", "", "", "", roleFilter.Approver, accessType, roleFilter.Cluster, namespace, group, kind, resource, actionType, false, "")
							if err != nil {
								impl.logger.Errorw("Error in fetching roles by filter", "user", request)
								return nil, err
							}
							oldRoleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(entity, "", "", "", "", roleFilter.Approver, accessType, roleFilter.Cluster, namespace, group, kind, resource, actionType, true, "")
							if err != nil {
								impl.logger.Errorw("Error in fetching roles by filter", "user", request)
								return nil, err
							}
							if roleModel.Id == 0 && oldRoleModel.Id == 0 {
								impl.logger.Warnw("no role found for given filter", "filter", roleFilter)
								continue
							}
							if _, ok := existingRoles[roleModel.Id]; ok {
								delete(eliminatedRoles, roleModel.Id)
							}
							if _, ok := existingRoles[oldRoleModel.Id]; ok {
								//delete old role mapping from existing but not from eliminated roles (so that it gets deleted)
								delete(existingRoles, oldRoleModel.Id)
							}
						}
					}
				}
			}
		} else if entity == bean2.EntityJobs {
			if len(roleFilter.Team) > 0 { // check auth only for apps permission, skip for chart group
				rbacObject := fmt.Sprintf("%s", roleFilter.Team)
				isValidAuth := managerAuth(casbin.ResourceUser, token, rbacObject)
				if !isValidAuth {
					continue
				}
			}
			entityNames := strings.Split(roleFilter.EntityName, ",")
			environments := strings.Split(roleFilter.Environment, ",")
			workflows := strings.Split(roleFilter.Workflow, ",")
			accessType := roleFilter.AccessType
			actionType := roleFilter.Action
			for _, environment := range environments {
				for _, entityName := range entityNames {
					for _, workflow := range workflows {
						roleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(roleFilter.Entity, roleFilter.Team, entityName, environment, actionType, false, accessType, "", "", "", "", "", "", false, workflow)
						if err != nil {
							impl.logger.Errorw("Error in fetching roles by filter", "user", request)
							return nil, err
						}
						if roleModel.Id == 0 {
							impl.logger.Warnw("no role found for given filter", "filter", roleFilter)
							request.Status = "role not fount for any given filter: " + roleFilter.Team + "," + environment + "," + entityName + "," + actionType
							continue
						}
						if _, ok := existingRoles[roleModel.Id]; ok {
							delete(eliminatedRoles, roleModel.Id)
						}
					}
				}
			}
		} else {
			if len(roleFilter.Team) > 0 { // check auth only for apps permission, skip for chart group
				rbacObject := fmt.Sprintf("%s", roleFilter.Team)
				isValidAuth := managerAuth(casbin.ResourceUser, token, rbacObject)
				if !isValidAuth {
					continue
				}
			}
			entityNames := strings.Split(roleFilter.EntityName, ",")
			environments := strings.Split(roleFilter.Environment, ",")
			actions := strings.Split(roleFilter.Action, ",")
			accessType := roleFilter.AccessType
			for _, environment := range environments {
				for _, entityName := range entityNames {
					for _, actionType := range actions {
						roleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(roleFilter.Entity, roleFilter.Team, entityName, environment, actionType, roleFilter.Approver, accessType, "", "", "", "", "", "", false, "")
						if err != nil {
							impl.logger.Errorw("Error in fetching roles by filter", "user", request)
							return nil, err
						}
						oldRoleModel, err := impl.userAuthRepository.GetRoleByFilterForAllTypes(roleFilter.Entity, roleFilter.Team, entityName, environment, actionType, roleFilter.Approver, accessType, "", "", "", "", "", "", true, "")
						if err != nil {
							impl.logger.Errorw("Error in fetching roles by filter by old values", "user", request)
							return nil, err
						}
						if roleModel.Id == 0 && oldRoleModel.Id == 0 {
							impl.logger.Warnw("no role found for given filter", "filter", roleFilter)
							request.Status = "role not fount for any given filter: " + roleFilter.Team + "," + environment + "," + entityName + "," + actionType
							continue
						}
						if _, ok := existingRoles[roleModel.Id]; ok {
							delete(eliminatedRoles, roleModel.Id)
						}
						isChartGroupEntity := roleFilter.Entity == bean.CHART_GROUP_ENTITY
						if _, ok := existingRoles[oldRoleModel.Id]; ok && !isChartGroupEntity {
							//delete old role mapping from existing but not from eliminated roles (so that it gets deleted)
							delete(existingRoles, oldRoleModel.Id)
						}
					}
				}
			}
		}
	}

	//delete remaining Ids from casbin role mapping table in orchestrator and casbin policy db
	// which are existing but not provided in this request
	var eliminatedPolicies []bean3.Policy
	for _, model := range eliminatedRoles {
		role, err := impl.userAuthRepository.GetRoleById(model.RoleId)
		if err != nil {
			return nil, err
		}
		if len(role.Team) > 0 {
			rbacObject := fmt.Sprintf("%s", role.Team)
			isValidAuth := managerAuth(casbin.ResourceUser, token, rbacObject)
			if !isValidAuth {
				continue
			}
		}
		if role.Entity == bean.CLUSTER_ENTITIY {
			isValidAuth := impl.CheckRbacForClusterEntity(role.Cluster, role.Namespace, role.Group, role.Kind, role.Resource, token, managerAuth)
			if !isValidAuth {
				continue
			}
		}
		_, err = impl.roleGroupRepository.DeleteRoleGroupRoleMapping(model, tx)
		if err != nil {
			return nil, err
		}
		policyGroup, err := impl.roleGroupRepository.GetRoleGroupById(model.RoleGroupId)
		if err != nil {
			return nil, err
		}
		eliminatedPolicies = append(eliminatedPolicies, bean3.Policy{Type: "g", Sub: bean3.Subject(policyGroup.CasbinName), Obj: bean3.Object(role.Role)})
	}
	return eliminatedPolicies, nil
}

func containsArr(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (impl UserCommonServiceImpl) CheckRbacForClusterEntity(cluster, namespace, group, kind, resource, token string, managerAuth func(resource, token, object string) bool) bool {
	if namespace == "NONE" {
		namespace = ""
	}
	if group == "NONE" {
		group = ""
	}
	if kind == "NONE" {
		kind = ""
	}
	if resource == "NONE" {
		resource = ""
	}
	namespaceObj := namespace
	groupObj := group
	kindObj := kind
	resourceObj := resource
	if namespace == "" {
		namespaceObj = "*"
	}
	if group == "" {
		groupObj = "*"
	}
	if kind == "" {
		kindObj = "*"
	}
	if resource == "" {
		resourceObj = "*"
	}

	rbacResource := fmt.Sprintf("%s/%s/%s", strings.ToLower(cluster), strings.ToLower(namespaceObj), casbin.ResourceUser)
	resourcesArray := strings.Split(resourceObj, ",")
	for _, resourceVal := range resourcesArray {
		rbacObject := fmt.Sprintf("%s/%s/%s", groupObj, kindObj, resourceVal)
		allowed := managerAuth(rbacResource, token, rbacObject)
		if !allowed {
			return false
		}
	}
	return true
}

func (impl UserCommonServiceImpl) GetCapacityForRoleFilter(roleFilters []bean.RoleFilter) (int, map[int]int) {
	capacity := 0

	m := make(map[int]int)
	for index, roleFilter := range roleFilters {
		namespaces := strings.Split(roleFilter.Namespace, ",")
		groups := strings.Split(roleFilter.Group, ",")
		kinds := strings.Split(roleFilter.Kind, ",")
		resources := strings.Split(roleFilter.Resource, ",")
		entityNames := strings.Split(roleFilter.EntityName, ",")
		environments := strings.Split(roleFilter.Environment, ",")
		actions := strings.Split(roleFilter.Action, ",")
		workflows := strings.Split(roleFilter.Workflow, ",")
		value := math.Max(float64(len(namespaces)*len(groups)*len(kinds)*len(resources)*2), math.Max(float64(len(entityNames)*len(environments)*len(actions)*6), float64(len(entityNames)*len(environments)*len(workflows)*8)))
		m[index] = int(value)
		capacity += int(value)
	}
	return capacity, m
}

func (impl UserCommonServiceImpl) MergeCustomRoleFilters(roleFilters []bean.RoleFilter) []bean.RoleFilter {
	// it will merge custom roles belong to same team, env & app structure
	var updatedRoleFilters []bean.RoleFilter
	roleFilterMap := make(map[string]bean.RoleFilter)
	for _, roleFilter := range roleFilters {
		team := roleFilter.Team
		if len(team) == 0 {
			updatedRoleFilters = append(updatedRoleFilters, roleFilter)
		} else {
			roleKey := fmt.Sprintf("%s_%s_%s", roleFilter.Team, roleFilter.Environment, roleFilter.EntityName)
			if filter, found := roleFilterMap[roleKey]; found {
				filter.Action = fmt.Sprintf("%s,%s", filter.Action, roleFilter.Action)
				roleFilterMap[roleKey] = filter
			} else {
				roleFilterMap[roleKey] = roleFilter
			}
		}
	}
	for _, roleFilter := range roleFilterMap {
		updatedRoleFilters = append(updatedRoleFilters, roleFilter)
	}
	return updatedRoleFilters
}

func (impl UserCommonServiceImpl) BuildRoleFilterForAllTypes(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string) {
	switch role.Entity {
	case bean.CLUSTER_ENTITIY:
		{
			impl.BuildRoleFilterKeyForCluster(roleFilterMap, role, key)
		}
	case bean2.EntityJobs:
		{
			impl.BuildRoleFilterKeyForJobs(roleFilterMap, role, key)
		}
	default:
		{
			impl.BuildRoleFilterKeyForOtherEntity(roleFilterMap, role, key)
		}
	}
}

func (impl UserCommonServiceImpl) BuildRoleFilterKeyForCluster(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string) {
	namespaceArr := strings.Split(roleFilterMap[key].Namespace, ",")
	if containsArr(namespaceArr, AllNamespace) {
		roleFilterMap[key].Namespace = AllNamespace
	} else if !containsArr(namespaceArr, role.Namespace) {
		roleFilterMap[key].Namespace = fmt.Sprintf("%s,%s", roleFilterMap[key].Namespace, role.Namespace)
	}
	groupArr := strings.Split(roleFilterMap[key].Group, ",")
	if containsArr(groupArr, AllGroup) {
		roleFilterMap[key].Group = AllGroup
	} else if !containsArr(groupArr, role.Group) {
		roleFilterMap[key].Group = fmt.Sprintf("%s,%s", roleFilterMap[key].Group, role.Group)
	}
	kindArr := strings.Split(roleFilterMap[key].Kind, ",")
	if containsArr(kindArr, AllKind) {
		roleFilterMap[key].Kind = AllKind
	} else if !containsArr(kindArr, role.Kind) {
		roleFilterMap[key].Kind = fmt.Sprintf("%s,%s", roleFilterMap[key].Kind, role.Kind)
	}
	resourceArr := strings.Split(roleFilterMap[key].Resource, ",")
	if containsArr(resourceArr, AllResource) {
		roleFilterMap[key].Resource = AllResource
	} else if !containsArr(resourceArr, role.Resource) {
		roleFilterMap[key].Resource = fmt.Sprintf("%s,%s", roleFilterMap[key].Resource, role.Resource)
	}
}

func (impl UserCommonServiceImpl) BuildRoleFilterKeyForJobs(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string) {
	envArr := strings.Split(roleFilterMap[key].Environment, ",")
	if containsArr(envArr, AllEnvironment) {
		roleFilterMap[key].Environment = AllEnvironment
	} else if !containsArr(envArr, role.Environment) {
		roleFilterMap[key].Environment = fmt.Sprintf("%s,%s", roleFilterMap[key].Environment, role.Environment)
	}
	entityArr := strings.Split(roleFilterMap[key].EntityName, ",")
	if !containsArr(entityArr, role.EntityName) {
		roleFilterMap[key].EntityName = fmt.Sprintf("%s,%s", roleFilterMap[key].EntityName, role.EntityName)
	}
	workflowArr := strings.Split(roleFilterMap[key].Workflow, ",")
	if containsArr(workflowArr, AllWorkflow) {
		roleFilterMap[key].Workflow = AllWorkflow
	} else if !containsArr(workflowArr, role.Workflow) {
		roleFilterMap[key].Workflow = fmt.Sprintf("%s,%s", roleFilterMap[key].Workflow, role.Workflow)
	}
}

func (impl UserCommonServiceImpl) BuildRoleFilterKeyForOtherEntity(roleFilterMap map[string]*bean.RoleFilter, role repository.RoleModel, key string) {
	envArr := strings.Split(roleFilterMap[key].Environment, ",")
	if containsArr(envArr, AllEnvironment) {
		roleFilterMap[key].Environment = AllEnvironment
	} else if !containsArr(envArr, role.Environment) {
		roleFilterMap[key].Environment = fmt.Sprintf("%s,%s", roleFilterMap[key].Environment, role.Environment)
	}
	entityArr := strings.Split(roleFilterMap[key].EntityName, ",")
	if !containsArr(entityArr, role.EntityName) {
		roleFilterMap[key].EntityName = fmt.Sprintf("%s,%s", roleFilterMap[key].EntityName, role.EntityName)
	}
}

func (impl UserCommonServiceImpl) GetUniqueKeyForAllEntityWithTimeAndStatus(role repository.RoleModel, status bean.Status, timeout time.Time) string {
	key := impl.GetUniqueKeyForAllEntity(role)
	return fmt.Sprintf("%s_%s_%s", key, status, timeout)
}

func (impl UserCommonServiceImpl) GetUniqueKeyForAllEntity(role repository.RoleModel) string {
	key := ""
	if len(role.Team) > 0 && role.Entity != bean2.EntityJobs {
		key = fmt.Sprintf("%s_%s_%s_%t", role.Team, role.Action, role.AccessType, role.Approver)
	} else if role.Entity == bean2.EntityJobs {
		key = fmt.Sprintf("%s_%s_%s_%s", role.Team, role.Action, role.AccessType, role.Entity)
	} else if len(role.Entity) > 0 {
		if role.Entity == bean.CLUSTER_ENTITIY {
			key = fmt.Sprintf("%s_%s_%s_%s_%s_%s", role.Entity, role.Action, role.Cluster,
				role.Namespace, role.Group, role.Kind)
		} else {
			key = fmt.Sprintf("%s_%s", role.Entity, role.Action)
		}
	}
	return key
}

func (impl UserCommonServiceImpl) SetDefaultValuesIfNotPresent(request *bean.ListingRequest, isRoleGroup bool) {
	if len(request.SortBy) == 0 {
		if isRoleGroup {
			request.SortBy = bean2.GroupName
		} else {
			request.SortBy = bean2.Email
		}
	}
	if request.Size == 0 {
		request.Size = bean2.DefaultSize
	}
}

func (impl UserCommonServiceImpl) DeleteRoleForUserFromCasbin(mappings map[string][]bean3.GroupPolicy) bool {
	successful := true
	for v0, v1s := range mappings {
		for _, v1 := range v1s {
			flag := casbin.DeleteRoleForUserV2(v0, v1.Role, v1.TimeoutWindowExpression, v1.ExpressionFormat)
			if flag == false {
				impl.logger.Warnw("unable to delete role:", "v0", v0, "v1", v1)
				successful = false
				return successful
			}
		}
	}
	return successful
}

func (impl UserCommonServiceImpl) DeleteUserForRoleFromCasbin(mappings map[string][]bean3.GroupPolicy) bool {
	successful := true
	for v1, v0s := range mappings {
		for _, v0 := range v0s {
			flag := casbin.DeleteRoleForUserV2(v0.User, v1, v0.TimeoutWindowExpression, v0.ExpressionFormat)
			if flag == false {
				impl.logger.Warnw("unable to delete role:", "v0", v0, "v1", v1)
				successful = false
				return successful
			}
		}
	}
	return successful
}

func (impl UserCommonServiceImpl) GetUniqueKeyForRoleFilter(roleFilter bean.RoleFilter) string {
	key := fmt.Sprintf("%s-%s-%s-%s-%s-%s-%t-%s-%s-%s-%s-%s-%s", roleFilter.Entity, roleFilter.Team, roleFilter.Environment,
		roleFilter.EntityName, roleFilter.Action, roleFilter.AccessType, roleFilter.Approver, roleFilter.Cluster, roleFilter.Namespace, roleFilter.Group, roleFilter.Kind, roleFilter.Resource, roleFilter.Workflow)
	return key
}
