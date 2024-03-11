package deploymentWindow

import (
	"encoding/json"
	mapset "github.com/deckarep/golang-set"
	bean2 "github.com/devtron-labs/devtron/pkg/globalPolicy/bean"
	"github.com/devtron-labs/devtron/pkg/resourceQualifiers"
	"github.com/devtron-labs/devtron/pkg/timeoutWindow"
	"github.com/devtron-labs/devtron/pkg/timeoutWindow/repository"
	"github.com/devtron-labs/devtron/pkg/variables/utils"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
	"strings"
	"time"
)

func (impl DeploymentWindowServiceImpl) GetActiveProfileForAppEnv(targetTime time.Time, appId int, envId int, userId int32) (*DeploymentWindowProfile, UserActionState, error) {
	stateResponse, err := impl.GetDeploymentWindowProfileState(targetTime, appId, []int{envId}, 0, userId)
	if err != nil {
		return nil, Allowed, err
	}

	var appliedProfile *DeploymentWindowProfile
	actionState := Allowed
	if state, ok := stateResponse.EnvironmentStateMap[envId]; ok {
		actionState = state.UserActionState
		appliedProfile = state.AppliedProfile.DeploymentWindowProfile
	}
	return appliedProfile, actionState, nil
}

func (impl DeploymentWindowServiceImpl) GetDeploymentWindowProfileStateAppGroup(targetTime time.Time, selectors []AppEnvSelector, filterForDays int, userId int32) (*DeploymentWindowAppGroupResponse, error) {

	appIdsToOverview, err := impl.GetDeploymentWindowProfileOverviewBulk(selectors)
	if err != nil {
		return nil, err
	}
	//superAdmins, err := impl.userService.GetSuperAdmins()
	//if err != nil {
	//	superAdmins = make([]int32, 0)
	//}

	appGroupData := make([]AppData, 0)
	for appId, overview := range appIdsToOverview {

		envResponse, err := impl.calculateStateForEnvironments(targetTime, overview, filterForDays, userId)
		if err != nil {
			return nil, err
		}

		appGroupData = append(appGroupData, AppData{
			AppId:                 appId,
			DeploymentProfileList: envResponse,
		})
	}
	return &DeploymentWindowAppGroupResponse{AppData: appGroupData}, nil
}

func (impl DeploymentWindowServiceImpl) GetDeploymentWindowProfileState(targetTime time.Time, appId int, envIds []int, filterForDays int, userId int32) (*DeploymentWindowResponse, error) {
	overview, err := impl.GetDeploymentWindowProfileOverview(appId, envIds)
	if err != nil {
		return nil, err
	}

	response, err := impl.calculateStateForEnvironments(targetTime, overview, filterForDays, userId)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (impl DeploymentWindowServiceImpl) calculateStateForEnvironments(targetTime time.Time, overview *DeploymentWindowResponse, filterForDays int, userId int32) (*DeploymentWindowResponse, error) {
	envIdToProfileStates := lo.GroupBy(overview.Profiles, func(item ProfileState) int {
		return item.EnvId
	})

	envIdToEnvironmentState := make(map[int]EnvironmentState)
	resultProfiles := make([]ProfileState, 0)
	for envId, profileStates := range envIdToProfileStates {
		filteredProfileStates, appliedProfile, excludedUsers, excludedUsersEmail, canDeploy, err := impl.getAppliedProfileAndCalculateStates(targetTime, profileStates, filterForDays)
		if err != nil {
			return nil, err
		}
		envState := EnvironmentState{
			ExcludedUsers:      excludedUsers,
			ExcludedUserEmails: excludedUsersEmail,
			AppliedProfile:     appliedProfile,
			UserActionState:    getUserActionStateForUser(canDeploy, excludedUsers, userId),
		}
		envIdToEnvironmentState[envId] = envState
		resultProfiles = append(resultProfiles, filteredProfileStates...)
	}
	response := &DeploymentWindowResponse{
		EnvironmentStateMap: envIdToEnvironmentState,
		Profiles:            resultProfiles,
		//SuperAdmins:         superAdmins,
	}
	return response, nil
}

func getUserActionStateForUser(canDeploy bool, excludedUsers []int32, userId int32) UserActionState {
	userActionState := Allowed
	if !canDeploy {
		if slices.Contains(excludedUsers, userId) {
			userActionState = Partial
		} else {
			userActionState = Blocked
		}
	}
	return userActionState
}

func (impl DeploymentWindowServiceImpl) getAppliedProfileAndCalculateStates(targetTime time.Time, profileStates []ProfileState, filterForDays int) ([]ProfileState, *ProfileState, []int32, []string, bool, error) {

	var appliedProfile *ProfileState
	var combinedExcludedUsers, allUserIds []int32
	combinedExcludedUserEmails := make([]string, 0)

	superAdmins, err := impl.userService.GetSuperAdminIds()
	if err != nil {
		return nil, appliedProfile, combinedExcludedUsers, combinedExcludedUserEmails, false, err
	}

	filteredBlackoutProfiles, _, isBlackoutActive, err := impl.calculateStateForProfiles(targetTime, profileStates, Blackout, filterForDays)
	if err != nil {
		return nil, appliedProfile, combinedExcludedUsers, combinedExcludedUserEmails, false, err
	}

	filteredMaintenanceProfiles, isMaintenanceActive, _, err := impl.calculateStateForProfiles(targetTime, profileStates, Maintenance, filterForDays)
	if err != nil {
		return nil, appliedProfile, combinedExcludedUsers, combinedExcludedUserEmails, false, err
	}

	if len(filteredBlackoutProfiles) == 0 && len(filteredMaintenanceProfiles) == 0 {
		return nil, appliedProfile, combinedExcludedUsers, combinedExcludedUserEmails, true, nil
	}

	canDeploy := !isBlackoutActive && isMaintenanceActive
	allProfiles := append(filteredBlackoutProfiles, filteredMaintenanceProfiles...)
	var isSuperAdminExcluded bool
	if isBlackoutActive && isMaintenanceActive { //deployment is blocked, restriction through blackout
		// if both are active then blackout takes precedence in overall calculation
		appliedProfile = impl.getLongestEndingProfile(filteredBlackoutProfiles)
		combinedExcludedUsers, allUserIds, isSuperAdminExcluded = impl.getCombinedUserIds(filteredBlackoutProfiles)

	} else if !isBlackoutActive && !isMaintenanceActive { //deployment is blocked, restriction through maintenance
		// if nothing is active then earliest starting maintenance will be shown
		appliedProfile = impl.getEarliestStartingProfile(filteredMaintenanceProfiles)
		combinedExcludedUsers, allUserIds, isSuperAdminExcluded = impl.getCombinedUserIds(filteredMaintenanceProfiles)
	} else if isBlackoutActive && !isMaintenanceActive { //deployment is blocked, restriction through both
		// longest of restrictions coming from both blackout and maintenance
		appliedProfile = impl.getLongestEndingProfile(allProfiles)
		combinedExcludedUsers, allUserIds, isSuperAdminExcluded = impl.getCombinedUserIds(allProfiles)

	} else if !isBlackoutActive && isMaintenanceActive { //deployment not blocked
		// applied profile here would be the longest running maintenance profile even if a blackout starts before that
		appliedProfile = impl.getLongestEndingProfile(filteredMaintenanceProfiles)
		if appliedProfile == nil {
			appliedProfile = impl.getEarliestStartingProfile(filteredBlackoutProfiles)
		}
	}

	if isSuperAdminExcluded {
		combinedExcludedUsers = lo.Uniq(append(combinedExcludedUsers, superAdmins...))
		allUserIds = lo.Uniq(append(allUserIds, superAdmins...))
	}

	allUserInfo, err := impl.userService.GetByIds(allUserIds)
	if err != nil {
		return nil, appliedProfile, combinedExcludedUsers, combinedExcludedUserEmails, true, nil
	}
	userInfoMap := make(map[int32]string, 0)
	for _, user := range allUserInfo {
		if strings.Contains(user.EmailId, "@") {
			userInfoMap[user.Id] = user.EmailId
		}
	}

	for i, profile := range allProfiles {

		excludedIds := make([]int32, 0)
		if len(profile.DeploymentWindowProfile.ExcludedUsersList) > 0 {
			excludedIds = profile.DeploymentWindowProfile.ExcludedUsersList
		}

		if profile.DeploymentWindowProfile.IsSuperAdminExcluded {
			excludedIds = lo.Uniq(append(excludedIds, superAdmins...))
		}
		emails := make([]string, 0)
		for _, id := range excludedIds {
			if email, ok := userInfoMap[id]; ok {
				emails = append(emails, email)
			}
		}
		allProfiles[i].AllExcludedUsers = emails

		if profile.DeploymentWindowProfile.Id == appliedProfile.DeploymentWindowProfile.Id {
			appliedProfile.AllExcludedUsers = emails
		}
	}
	emails := make([]string, 0)
	for _, userId := range combinedExcludedUsers {
		if email, ok := userInfoMap[userId]; ok {
			emails = append(emails, email)
		}
	}
	combinedExcludedUserEmails = emails

	return allProfiles, appliedProfile, combinedExcludedUsers, combinedExcludedUserEmails, canDeploy, nil
}

func (impl DeploymentWindowServiceImpl) getCombinedUserIds(profiles []ProfileState) ([]int32, []int32, bool) {

	if len(profiles) == 0 {
		return []int32{}, []int32{}, false
	}
	userSet := mapset.NewSet()
	allUsersSet := mapset.NewSet()

	if len(profiles[0].DeploymentWindowProfile.ExcludedUsersList) > 0 {
		//userSet.Add(profiles[0].DeploymentWindowProfile.ExcludedUsersList)
		userSet = mapset.NewSet(profiles[0].DeploymentWindowProfile.ExcludedUsersList)
	}

	isSuperAdminExcluded := false
	lo.ForEach(profiles, func(profile ProfileState, index int) {
		var users []int32
		if profile.DeploymentWindowProfile.IsUserExcluded {
			users = profile.DeploymentWindowProfile.ExcludedUsersList
		}

		isSuperAdminExcluded = profile.DeploymentWindowProfile.IsSuperAdminExcluded

		profileUserSet := mapset.NewSet()
		if len(users) > 0 {
			profileUserSet = mapset.NewSet(users)
			allUsersSet = allUsersSet.Union(profileUserSet)
		}

		userSet = userSet.Intersect(profileUserSet)
	})

	//if isSuperAdminExcluded && len(superAdmins) > 0 {
	//	userSet = userSet.Union(mapset.NewSet(superAdmins))
	//}

	return utils.ToInt32Array(userSet.ToSlice()), utils.ToInt32Array(allUsersSet.ToSlice()), isSuperAdminExcluded
}

func (impl DeploymentWindowServiceImpl) getLongestEndingProfile(profiles []ProfileState) *ProfileState {

	if len(profiles) == 0 {
		return nil
	}

	profile := lo.Reduce(profiles, func(profile ProfileState, item ProfileState, index int) ProfileState {
		if item.CalculatedTimestamp.After(profile.CalculatedTimestamp) {
			return item
		}
		return profile
	}, profiles[0])
	return &profile
}

func (impl DeploymentWindowServiceImpl) getEarliestStartingProfile(profiles []ProfileState) *ProfileState {
	if len(profiles) == 0 {
		return nil
	}

	profile := lo.Reduce(profiles, func(profile ProfileState, item ProfileState, index int) ProfileState {
		if item.CalculatedTimestamp.Before(profile.CalculatedTimestamp) {
			return item
		}
		return profile
	}, profiles[0])
	return &profile
}

func (impl DeploymentWindowServiceImpl) calculateStateForProfiles(targetTime time.Time, profileStates []ProfileState, profileType DeploymentWindowType, filterForDays int) ([]ProfileState, bool, bool, error) {

	filteredProfiles := lo.Filter(profileStates, func(item ProfileState, index int) bool {
		return item.DeploymentWindowProfile.Type == profileType
	})

	allActive := true
	oneActive := false
	finalProfileStates := make([]ProfileState, 0)

	for _, profile := range filteredProfiles {
		loc, err := impl.getTimeZoneData(profile.DeploymentWindowProfile.TimeZone)
		if err != nil {
			return nil, false, false, err
		}
		timeWithZone := targetTime.In(loc)
		isActive, windowTimeStamp, window := impl.timeWindowService.GetActiveWindow(timeWithZone, profile.DeploymentWindowProfile.DeploymentWindowList)

		if window == nil {
			// doing nothing if no window is returned
			// this means that no relevant window in the profile was found therefore skipping this profile
			continue
		}

		if filterForDays > 0 && windowTimeStamp.Sub(timeWithZone) > time.Duration(filterForDays)*time.Hour*24 {
			continue
		}

		profile.IsActive = isActive
		profile.CalculatedTimestamp = windowTimeStamp
		profile.DeploymentWindowProfile.DeploymentWindowList = []*timeoutWindow.TimeWindow{window}

		if !oneActive && isActive {
			oneActive = true
		}
		if allActive && !isActive {
			allActive = false
		}
		finalProfileStates = append(finalProfileStates, profile)
	}
	return finalProfileStates, allActive, oneActive, nil
}

func (impl DeploymentWindowServiceImpl) CreateDeploymentWindowProfile(profile *DeploymentWindowProfile, userId int32) (*DeploymentWindowProfile, error) {
	tx, err := impl.StartATransaction()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// create policy
	policy, err := profile.convertToPolicyDataModel(userId)
	if err != nil {
		return nil, err
	}
	policy, err = impl.globalPolicyManager.CreatePolicy(policy, tx)
	if err != nil {
		return nil, err
	}
	profile.Id = policy.Id

	err = impl.timeWindowService.UpdateWindowMappings(profile.DeploymentWindowList, userId, err, tx, policy.Id)
	if err != nil {
		return nil, err
	}
	err = impl.CommitATransaction(tx)
	if err != nil {
		return nil, err
	}

	return profile, err
}

func (impl DeploymentWindowServiceImpl) UpdateDeploymentWindowProfile(profile *DeploymentWindowProfile, userId int32) (*DeploymentWindowProfile, error) {
	tx, err := impl.StartATransaction()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// create policy
	policy, err := profile.convertToPolicyDataModel(userId)
	if err != nil {
		return nil, err
	}
	policy, err = impl.globalPolicyManager.UpdatePolicy(policy, tx)
	if err != nil {
		return nil, err
	}
	err = impl.timeWindowService.UpdateWindowMappings(profile.DeploymentWindowList, userId, err, tx, policy.Id)
	if err != nil {
		return nil, err
	}
	err = impl.CommitATransaction(tx)
	if err != nil {
		return nil, err
	}
	return profile, err
}

func (impl DeploymentWindowServiceImpl) DeleteDeploymentWindowProfileForId(profileId int, userId int32) error {
	tx, err := impl.StartATransaction()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = impl.globalPolicyManager.DeletePolicyById(tx, profileId, userId)
	if err != nil {
		return err
	}
	err = impl.timeWindowService.UpdateWindowMappings([]*timeoutWindow.TimeWindow{}, userId, err, tx, profileId)
	if err != nil {
		return err
	}
	err = impl.CommitATransaction(tx)
	if err != nil {
		return err
	}

	return err
}

func (impl DeploymentWindowServiceImpl) GetDeploymentWindowProfileForId(profileId int) (*DeploymentWindowProfile, error) {
	//get policy
	policyModel, err := impl.globalPolicyManager.GetPolicyById(profileId)
	if err != nil {
		return nil, err
	}

	idToWindows, err := impl.timeWindowService.GetWindowsForResources([]int{profileId}, repository.DeploymentWindowProfile)
	if err != nil {
		return nil, err
	}

	windows, ok := idToWindows[profileId]
	if !ok {
		return nil, nil
	}
	profilePolicy, err := impl.getPolicyFromModel(policyModel)
	if err != nil {
		return nil, err
	}

	return profilePolicy.toDeploymentWindowProfile(policyModel, windows), nil
}

func (impl DeploymentWindowServiceImpl) getPolicyFromModel(policyModel *bean2.GlobalPolicyBaseModel) (*DeploymentWindowProfilePolicy, error) {
	profilePolicy := &DeploymentWindowProfilePolicy{}
	err := json.Unmarshal([]byte(policyModel.JsonData), &profilePolicy)
	if err != nil {
		return nil, err
	}
	return profilePolicy, nil
}

func (impl DeploymentWindowServiceImpl) ListDeploymentWindowProfiles() ([]*DeploymentWindowProfileMetadata, error) {
	//get policy
	policyModels, err := impl.globalPolicyManager.GetAllActiveByType(bean2.GLOBAL_POLICY_TYPE_DEPLOYMENT_WINDOW)
	if err != nil {
		return nil, err
	}

	return lo.Map(policyModels, func(model *bean2.GlobalPolicyBaseModel, index int) *DeploymentWindowProfileMetadata {
		policy, err := impl.getPolicyFromModel(model)
		if err != nil {
			return nil
		}
		return &DeploymentWindowProfileMetadata{
			Description: model.Description,
			Id:          model.Id,
			Name:        model.Name,
			Type:        policy.Type,
		}
	}), nil
}

func (impl DeploymentWindowServiceImpl) GetDeploymentWindowProfileOverview(appId int, envIds []int) (*DeploymentWindowResponse, error) {

	resources, profileIdToProfile, err := impl.getProfileMappingsForApp(appId, envIds)
	if err != nil {
		return nil, err
	}

	envIdToMappings := lo.GroupBy(resources, func(item ProfileMapping) int {
		return item.EnvId
	})
	profileStates := impl.getProfileStates(envIdToMappings, profileIdToProfile)

	return &DeploymentWindowResponse{
		Profiles: profileStates,
	}, nil
}

func (impl DeploymentWindowServiceImpl) getProfileMappingsForApp(appId int, envIds []int) ([]ProfileMapping, map[int]*DeploymentWindowProfile, error) {
	selections := lo.Map(envIds, func(envId int, index int) *resourceQualifiers.SelectionIdentifier {
		return &resourceQualifiers.SelectionIdentifier{
			AppId: appId,
			EnvId: envId,
		}
	})

	resources, profileIdToProfile, err := impl.getResourcesAndProfilesForSelections(selections)
	if err != nil {
		return nil, nil, err
	}
	return resources, profileIdToProfile, nil
}

func (impl DeploymentWindowServiceImpl) getProfileStates(envIdToMappings map[int][]ProfileMapping, profileIdToProfile map[int]*DeploymentWindowProfile) []ProfileState {
	profileStates := make([]ProfileState, 0)
	for envId, mappings := range envIdToMappings {
		for _, mapping := range mappings {
			profile := profileIdToProfile[mapping.ProfileId]
			if !profile.Enabled {
				continue
			}
			profileStates = append(profileStates, ProfileState{
				DeploymentWindowProfile: profile,
				EnvId:                   envId,
			})
		}
	}
	return profileStates
}

func (impl DeploymentWindowServiceImpl) getProfileIdToProfile(profileIds []int) (map[int]*DeploymentWindowProfile, error) {

	models, err := impl.globalPolicyManager.GetPolicyByIds(profileIds)
	if err != nil {
		return nil, err
	}
	profileIdToModel := make(map[int]*bean2.GlobalPolicyBaseModel)
	for _, model := range models {
		profileIdToModel[model.Id] = model
	}

	profileIdToWindows, err := impl.timeWindowService.GetWindowsForResources(profileIds, repository.DeploymentWindowProfile)
	if err != nil {
		return nil, err
	}

	profileIdToProfile := make(map[int]*DeploymentWindowProfile)
	for _, profileId := range profileIds {

		windows := profileIdToWindows[profileId]

		profilePolicy, err := impl.getPolicyFromModel(profileIdToModel[profileId])
		if err != nil {
			return nil, err
		}
		deploymentProfile := profilePolicy.toDeploymentWindowProfile(profileIdToModel[profileId], windows)
		profileIdToProfile[profileId] = deploymentProfile
	}
	return profileIdToProfile, nil
}

func (impl DeploymentWindowServiceImpl) GetDeploymentWindowProfileOverviewBulk(appEnvSelectors []AppEnvSelector) (map[int]*DeploymentWindowResponse, error) {

	profileIdToProfile, appIdToMappings, err := impl.getMappedResourcesForAppgroups(appEnvSelectors)
	if err != nil {
		return nil, err
	}

	appIdToResponse := make(map[int]*DeploymentWindowResponse)
	for appId, mappings := range appIdToMappings {
		envIdToMappings := lo.GroupBy(mappings, func(item ProfileMapping) int {
			return item.EnvId
		})
		profileStates := impl.getProfileStates(envIdToMappings, profileIdToProfile)
		appIdToResponse[appId] = &DeploymentWindowResponse{
			Profiles: profileStates,
		}

	}
	return appIdToResponse, nil
}

func (impl DeploymentWindowServiceImpl) getMappedResourcesForAppgroups(appEnvSelectors []AppEnvSelector) (map[int]*DeploymentWindowProfile, map[int][]ProfileMapping, error) {
	selections := lo.Map(appEnvSelectors, func(appEnv AppEnvSelector, index int) *resourceQualifiers.SelectionIdentifier {
		return &resourceQualifiers.SelectionIdentifier{
			AppId: appEnv.AppId,
			EnvId: appEnv.EnvId,
		}
	})
	mappings, profileIdToProfile, err := impl.getResourcesAndProfilesForSelections(selections)
	if err != nil {
		return nil, nil, err
	}
	appIdToMappings := lo.GroupBy(mappings, func(item ProfileMapping) int {
		return item.AppId
	})
	return profileIdToProfile, appIdToMappings, nil
}

func (impl DeploymentWindowServiceImpl) getResourcesAndProfilesForSelections(selections []*resourceQualifiers.SelectionIdentifier) ([]ProfileMapping, map[int]*DeploymentWindowProfile, error) {
	resources, err := impl.resourceMappingService.GetResourceMappingsForSelections(resourceQualifiers.DeploymentWindowProfile, resourceQualifiers.ApplicationEnvironmentSelector, selections)
	if err != nil {
		return nil, nil, err
	}

	profileIds := lo.Map(resources, func(mapping resourceQualifiers.ResourceQualifierMappings, index int) int {
		return mapping.ResourceId
	})
	profileIdToProfile, err := impl.getProfileIdToProfile(profileIds)
	if err != nil {
		return nil, nil, err
	}

	mappings := lo.Map(resources, func(item resourceQualifiers.ResourceQualifierMappings, index int) ProfileMapping {
		return ProfileMapping{
			ProfileId: item.ResourceId,
			AppId:     item.SelectionIdentifier.AppId,
			EnvId:     item.SelectionIdentifier.EnvId,
		}
	})

	return mappings, profileIdToProfile, nil
}
