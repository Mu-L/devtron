package adapter

import (
	"github.com/devtron-labs/devtron/api/bean"
	bean2 "github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin/bean"
	bean3 "github.com/devtron-labs/devtron/pkg/auth/common/bean"
	"github.com/devtron-labs/devtron/pkg/auth/user/repository"
	"strings"
	"time"
)

func GetLastLoginTime(model repository.UserModel) time.Time {
	lastLoginTime := time.Time{}
	if model.UserAudit != nil {
		lastLoginTime = model.UserAudit.UpdatedOn
	}
	return lastLoginTime
}

func GetCasbinGroupPolicy(emailId string, role string, expression string, expressionFormat string) bean2.Policy {
	return bean2.Policy{
		Type: "g",
		Sub:  bean2.Subject(emailId),
		Res:  bean2.Resource(expression),
		Act:  bean2.Action(expressionFormat),
		Obj:  bean2.Object(role),
	}
}

func GetCasbinGroupPolicyForEmailAndRoleOnly(emailId string, role string) bean2.Policy {
	return bean2.Policy{
		Type: "g",
		Sub:  bean2.Subject(emailId),
		Obj:  bean2.Object(role),
	}
}

func GetBasicRoleGroupDetailsAdapter(name, description string, id int32, casbinName string) *bean.RoleGroup {
	roleGroup := &bean.RoleGroup{
		Id:          id,
		Name:        name,
		Description: description,
		CasbinName:  casbinName,
	}
	return roleGroup
}

func GetUserRoleGroupAdapter(group *bean.RoleGroup, status bean.Status, timeoutExpression time.Time) bean.UserRoleGroup {
	return bean.UserRoleGroup{
		RoleGroup:               group,
		Status:                  status,
		TimeoutWindowExpression: timeoutExpression,
	}
}

func CreateRestrictedGroup(roleGroupName string, hasSuperAdminPermission bool) bean.RestrictedGroup {
	trimmedGroup := strings.TrimPrefix(roleGroupName, bean3.GroupPrefix)
	return bean.RestrictedGroup{
		Group:                   trimmedGroup,
		HasSuperAdminPermission: hasSuperAdminPermission,
	}
}
