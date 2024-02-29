package deploymentWindow

import (
	"encoding/json"
	"github.com/devtron-labs/devtron/enterprise/pkg/app/blackbox"
	"github.com/devtron-labs/devtron/pkg/globalPolicy/bean"
	"github.com/samber/lo"
	"time"
)

type DeploymentWindowProfilePolicy struct {
	//DeploymentWindowList []*TimeWindow `json:"deploymentWindowList,omitempty"`
	//Enabled              bool          `json:"enabled" searchFieldType:"boolean"`
	TimeZone             string               `json:"timeZone"`
	DisplayMessage       string               `json:"displayMessage"`
	ExcludedUsersList    []int32              `json:"excludedUsersList"`
	IsSuperAdminExcluded bool                 `json:"isSuperAdminExcluded"`
	IsUserExcluded       bool                 `json:"isUserExcluded"`
	Type                 DeploymentWindowType `json:"type" isSearchField:"true"`
}

func (profile DeploymentWindowProfile) toPolicy() DeploymentWindowProfilePolicy {
	return DeploymentWindowProfilePolicy{
		TimeZone:             profile.TimeZone,
		DisplayMessage:       profile.DisplayMessage,
		ExcludedUsersList:    profile.ExcludedUsersList,
		IsSuperAdminExcluded: profile.IsSuperAdminExcluded,
		IsUserExcluded:       profile.IsUserExcluded,
		Type:                 profile.Type,
	}
}

func (profile DeploymentWindowProfile) convertToPolicyDataModel(userId int32) (*bean.GlobalPolicyDataModel, error) {

	policyBytes, err := json.Marshal(profile.toPolicy())
	if err != nil {
		return nil, err
	}
	return &bean.GlobalPolicyDataModel{
		GlobalPolicyBaseModel: bean.GlobalPolicyBaseModel{
			Name:          profile.Name,
			Description:   profile.Description,
			Enabled:       profile.Enabled,
			PolicyOf:      bean.GLOBAL_POLICY_TYPE_DEPLOYMENT_WINDOW,
			PolicyVersion: bean.GLOBAL_POLICY_VERSION_V1,
			JsonData:      string(policyBytes),
			Active:        true,
			UserId:        userId,
		},
		SearchableFields: GetSearchableFields(profile),
	}, nil
}

func (profilePolicy DeploymentWindowProfilePolicy) toDeploymentWindowProfile(policyModel *bean.GlobalPolicyBaseModel, windows []*TimeWindow) *DeploymentWindowProfile {
	return &DeploymentWindowProfile{
		DeploymentWindowList: windows,
		Enabled:              policyModel.Enabled,
		TimeZone:             profilePolicy.TimeZone,
		DisplayMessage:       profilePolicy.DisplayMessage,
		ExcludedUsersList:    profilePolicy.ExcludedUsersList,
		IsSuperAdminExcluded: profilePolicy.IsSuperAdminExcluded,
		IsUserExcluded:       profilePolicy.IsUserExcluded,
		DeploymentWindowProfileMetadata: DeploymentWindowProfileMetadata{
			Description: policyModel.Description,
			Id:          policyModel.Id,
			Name:        policyModel.Name,
			Type:        profilePolicy.Type,
		},
	}
}

func (timeWindow *TimeWindow) toTimeRange() blackbox.TimeRange {
	return blackbox.TimeRange{
		TimeFrom:       timeWindow.TimeFrom,
		TimeTo:         timeWindow.TimeTo,
		HourMinuteFrom: timeWindow.HourMinuteFrom,
		HourMinuteTo:   timeWindow.HourMinuteTo,
		DayFrom:        timeWindow.DayFrom,
		DayTo:          timeWindow.DayTo,
		WeekdayFrom:    timeWindow.WeekdayFrom.toWeekday(),
		WeekdayTo:      timeWindow.WeekdayTo.toWeekday(),
		Weekdays:       lo.Map(timeWindow.Weekdays, func(item DayOfWeek, index int) time.Weekday { return item.toWeekday() }),
		Frequency:      timeWindow.Frequency.toTimeRangeFrequency(),
	}
}

func (f Frequency) toTimeRangeFrequency() blackbox.Frequency {
	switch f {
	case Fixed:
		return blackbox.FIXED
	case Daily:
		return blackbox.DAILY
	case Weekly:
		return blackbox.WEEKLY
	case WeeklyRange:
		return blackbox.WEEKLY_RANGE
	case Monthly:
		return blackbox.MONTHLY
	}
	return ""
}
