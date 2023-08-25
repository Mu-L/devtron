// Code generated by MockGen. DO NOT EDIT.
// Source: util/rbac/EnforcerUtil.go

// Package mock_rbac is a generated GoMock package.
package mock_rbac

import (
	reflect "reflect"

	application "github.com/devtron-labs/devtron/util/k8s"
	pipelineConfig "github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	bean "github.com/devtron-labs/devtron/pkg/bean"
	gomock "github.com/golang/mock/gomock"
)

// MockEnforcerUtil is a mock of EnforcerUtil interface.
type MockEnforcerUtil struct {
	ctrl     *gomock.Controller
	recorder *MockEnforcerUtilMockRecorder
}

// MockEnforcerUtilMockRecorder is the mock recorder for MockEnforcerUtil.
type MockEnforcerUtilMockRecorder struct {
	mock *MockEnforcerUtil
}

// NewMockEnforcerUtil creates a new mock instance.
func NewMockEnforcerUtil(ctrl *gomock.Controller) *MockEnforcerUtil {
	mock := &MockEnforcerUtil{ctrl: ctrl}
	mock.recorder = &MockEnforcerUtilMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEnforcerUtil) EXPECT() *MockEnforcerUtilMockRecorder {
	return m.recorder
}

// GetAllActiveTeamNames mocks base method.
func (m *MockEnforcerUtil) GetAllActiveTeamNames() ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllActiveTeamNames")
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllActiveTeamNames indicates an expected call of GetAllActiveTeamNames.
func (mr *MockEnforcerUtilMockRecorder) GetAllActiveTeamNames() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllActiveTeamNames", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAllActiveTeamNames))
}

// GetAppAndEnvObjectByDbPipeline mocks base method.
func (m *MockEnforcerUtil) GetAppAndEnvObjectByDbPipeline(cdPipelines []*pipelineConfig.Pipeline) map[int][]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppAndEnvObjectByDbPipeline", cdPipelines)
	ret0, _ := ret[0].(map[int][]string)
	return ret0
}

// GetAppAndEnvObjectByDbPipeline indicates an expected call of GetAppAndEnvObjectByDbPipeline.
func (mr *MockEnforcerUtilMockRecorder) GetAppAndEnvObjectByDbPipeline(cdPipelines interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppAndEnvObjectByDbPipeline", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppAndEnvObjectByDbPipeline), cdPipelines)
}

// GetAppAndEnvObjectByPipeline mocks base method.
func (m *MockEnforcerUtil) GetAppAndEnvObjectByPipeline(cdPipelines []*bean.CDPipelineConfigObject) map[int][]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppAndEnvObjectByPipeline", cdPipelines)
	ret0, _ := ret[0].(map[int][]string)
	return ret0
}

// GetAppAndEnvObjectByPipeline indicates an expected call of GetAppAndEnvObjectByPipeline.
func (mr *MockEnforcerUtilMockRecorder) GetAppAndEnvObjectByPipeline(cdPipelines interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppAndEnvObjectByPipeline", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppAndEnvObjectByPipeline), cdPipelines)
}

// GetAppAndEnvObjectByPipelineIds mocks base method.
func (m *MockEnforcerUtil) GetAppAndEnvObjectByPipelineIds(cdPipelineIds []int) map[int][]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppAndEnvObjectByPipelineIds", cdPipelineIds)
	ret0, _ := ret[0].(map[int][]string)
	return ret0
}

// GetAppAndEnvObjectByPipelineIds indicates an expected call of GetAppAndEnvObjectByPipelineIds.
func (mr *MockEnforcerUtilMockRecorder) GetAppAndEnvObjectByPipelineIds(cdPipelineIds interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppAndEnvObjectByPipelineIds", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppAndEnvObjectByPipelineIds), cdPipelineIds)
}

// GetAppObjectByCiPipelineIds mocks base method.
func (m *MockEnforcerUtil) GetAppObjectByCiPipelineIds(ciPipelineIds []int) map[int]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppObjectByCiPipelineIds", ciPipelineIds)
	ret0, _ := ret[0].(map[int]string)
	return ret0
}

// GetAppObjectByCiPipelineIds indicates an expected call of GetAppObjectByCiPipelineIds.
func (mr *MockEnforcerUtilMockRecorder) GetAppObjectByCiPipelineIds(ciPipelineIds interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppObjectByCiPipelineIds", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppObjectByCiPipelineIds), ciPipelineIds)
}

// GetAppRBACByAppIdAndPipelineId mocks base method.
func (m *MockEnforcerUtil) GetAppRBACByAppIdAndPipelineId(appId, pipelineId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppRBACByAppIdAndPipelineId", appId, pipelineId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAppRBACByAppIdAndPipelineId indicates an expected call of GetAppRBACByAppIdAndPipelineId.
func (mr *MockEnforcerUtilMockRecorder) GetAppRBACByAppIdAndPipelineId(appId, pipelineId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppRBACByAppIdAndPipelineId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppRBACByAppIdAndPipelineId), appId, pipelineId)
}

// GetAppRBACByAppNameAndEnvId mocks base method.
func (m *MockEnforcerUtil) GetAppRBACByAppNameAndEnvId(appName string, envId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppRBACByAppNameAndEnvId", appName, envId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAppRBACByAppNameAndEnvId indicates an expected call of GetAppRBACByAppNameAndEnvId.
func (mr *MockEnforcerUtilMockRecorder) GetAppRBACByAppNameAndEnvId(appName, envId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppRBACByAppNameAndEnvId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppRBACByAppNameAndEnvId), appName, envId)
}

// GetAppRBACName mocks base method.
func (m *MockEnforcerUtil) GetAppRBACName(appName string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppRBACName", appName)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAppRBACName indicates an expected call of GetAppRBACName.
func (mr *MockEnforcerUtilMockRecorder) GetAppRBACName(appName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppRBACName", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppRBACName), appName)
}

// GetAppRBACNameByAppId mocks base method.
func (m *MockEnforcerUtil) GetAppRBACNameByAppId(appId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppRBACNameByAppId", appId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAppRBACNameByAppId indicates an expected call of GetAppRBACNameByAppId.
func (mr *MockEnforcerUtilMockRecorder) GetAppRBACNameByAppId(appId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppRBACNameByAppId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppRBACNameByAppId), appId)
}

// GetAppRBACNameByTeamIdAndAppId mocks base method.
func (m *MockEnforcerUtil) GetAppRBACNameByTeamIdAndAppId(teamId, appId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAppRBACNameByTeamIdAndAppId", teamId, appId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAppRBACNameByTeamIdAndAppId indicates an expected call of GetAppRBACNameByTeamIdAndAppId.
func (mr *MockEnforcerUtilMockRecorder) GetAppRBACNameByTeamIdAndAppId(teamId, appId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAppRBACNameByTeamIdAndAppId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetAppRBACNameByTeamIdAndAppId), teamId, appId)
}

// GetEnvRBACArrayByAppId mocks base method.
func (m *MockEnforcerUtil) GetEnvRBACArrayByAppId(appId int) []string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEnvRBACArrayByAppId", appId)
	ret0, _ := ret[0].([]string)
	return ret0
}

// GetEnvRBACArrayByAppId indicates an expected call of GetEnvRBACArrayByAppId.
func (mr *MockEnforcerUtilMockRecorder) GetEnvRBACArrayByAppId(appId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEnvRBACArrayByAppId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetEnvRBACArrayByAppId), appId)
}

// GetEnvRBACNameByAppId mocks base method.
func (m *MockEnforcerUtil) GetEnvRBACNameByAppId(appId, envId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEnvRBACNameByAppId", appId, envId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetEnvRBACNameByAppId indicates an expected call of GetEnvRBACNameByAppId.
func (mr *MockEnforcerUtilMockRecorder) GetEnvRBACNameByAppId(appId, envId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEnvRBACNameByAppId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetEnvRBACNameByAppId), appId, envId)
}

// GetEnvRBACNameByCdPipelineIdAndEnvId mocks base method.
func (m *MockEnforcerUtil) GetEnvRBACNameByCdPipelineIdAndEnvId(cdPipelineId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEnvRBACNameByCdPipelineIdAndEnvId", cdPipelineId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetEnvRBACNameByCdPipelineIdAndEnvId indicates an expected call of GetEnvRBACNameByCdPipelineIdAndEnvId.
func (mr *MockEnforcerUtilMockRecorder) GetEnvRBACNameByCdPipelineIdAndEnvId(cdPipelineId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEnvRBACNameByCdPipelineIdAndEnvId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetEnvRBACNameByCdPipelineIdAndEnvId), cdPipelineId)
}

// GetEnvRBACNameByCiPipelineIdAndEnvId mocks base method.
func (m *MockEnforcerUtil) GetEnvRBACNameByCiPipelineIdAndEnvId(ciPipelineId, envId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEnvRBACNameByCiPipelineIdAndEnvId", ciPipelineId, envId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetEnvRBACNameByCiPipelineIdAndEnvId indicates an expected call of GetEnvRBACNameByCiPipelineIdAndEnvId.
func (mr *MockEnforcerUtilMockRecorder) GetEnvRBACNameByCiPipelineIdAndEnvId(ciPipelineId, envId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEnvRBACNameByCiPipelineIdAndEnvId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetEnvRBACNameByCiPipelineIdAndEnvId), ciPipelineId, envId)
}

// GetHelmObject mocks base method.
func (m *MockEnforcerUtil) GetHelmObject(appId, envId int) (string, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHelmObject", appId, envId)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// GetHelmObject indicates an expected call of GetHelmObject.
func (mr *MockEnforcerUtilMockRecorder) GetHelmObject(appId, envId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHelmObject", reflect.TypeOf((*MockEnforcerUtil)(nil).GetHelmObject), appId, envId)
}

// GetHelmObjectByAppNameAndEnvId mocks base method.
func (m *MockEnforcerUtil) GetHelmObjectByAppNameAndEnvId(appName string, envId int) (string, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHelmObjectByAppNameAndEnvId", appName, envId)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// GetHelmObjectByAppNameAndEnvId indicates an expected call of GetHelmObjectByAppNameAndEnvId.
func (mr *MockEnforcerUtilMockRecorder) GetHelmObjectByAppNameAndEnvId(appName, envId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHelmObjectByAppNameAndEnvId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetHelmObjectByAppNameAndEnvId), appName, envId)
}

// GetHelmObjectByProjectIdAndEnvId mocks base method.
func (m *MockEnforcerUtil) GetHelmObjectByProjectIdAndEnvId(teamId, envId int) (string, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHelmObjectByProjectIdAndEnvId", teamId, envId)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// GetHelmObjectByProjectIdAndEnvId indicates an expected call of GetHelmObjectByProjectIdAndEnvId.
func (mr *MockEnforcerUtilMockRecorder) GetHelmObjectByProjectIdAndEnvId(teamId, envId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHelmObjectByProjectIdAndEnvId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetHelmObjectByProjectIdAndEnvId), teamId, envId)
}

// GetProjectAdminRBACNameBYAppName mocks base method.
func (m *MockEnforcerUtil) GetProjectAdminRBACNameBYAppName(appName string) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProjectAdminRBACNameBYAppName", appName)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetProjectAdminRBACNameBYAppName indicates an expected call of GetProjectAdminRBACNameBYAppName.
func (mr *MockEnforcerUtilMockRecorder) GetProjectAdminRBACNameBYAppName(appName interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProjectAdminRBACNameBYAppName", reflect.TypeOf((*MockEnforcerUtil)(nil).GetProjectAdminRBACNameBYAppName), appName)
}

// GetRBACNameForClusterEntity mocks base method.
func (m *MockEnforcerUtil) GetRBACNameForClusterEntity(clusterName string, resourceIdentifier application.ResourceIdentifier) (string, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRBACNameForClusterEntity", clusterName, resourceIdentifier)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// GetRBACNameForClusterEntity indicates an expected call of GetRBACNameForClusterEntity.
func (mr *MockEnforcerUtilMockRecorder) GetRBACNameForClusterEntity(clusterName, resourceIdentifier interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRBACNameForClusterEntity", reflect.TypeOf((*MockEnforcerUtil)(nil).GetRBACNameForClusterEntity), clusterName, resourceIdentifier)
}

// GetRbacObjectsByAppIds mocks base method.
func (m *MockEnforcerUtil) GetRbacObjectsByAppIds(appIds []int) map[int]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRbacObjectsByAppIds", appIds)
	ret0, _ := ret[0].(map[int]string)
	return ret0
}

// GetRbacObjectsByAppIds indicates an expected call of GetRbacObjectsByAppIds.
func (mr *MockEnforcerUtilMockRecorder) GetRbacObjectsByAppIds(appIds interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRbacObjectsByAppIds", reflect.TypeOf((*MockEnforcerUtil)(nil).GetRbacObjectsByAppIds), appIds)
}

// GetRbacObjectsForAllApps mocks base method.
func (m *MockEnforcerUtil) GetRbacObjectsForAllApps() map[int]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRbacObjectsForAllApps")
	ret0, _ := ret[0].(map[int]string)
	return ret0
}

// GetRbacObjectsForAllApps indicates an expected call of GetRbacObjectsForAllApps.
func (mr *MockEnforcerUtilMockRecorder) GetRbacObjectsForAllApps() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRbacObjectsForAllApps", reflect.TypeOf((*MockEnforcerUtil)(nil).GetRbacObjectsForAllApps))
}

// GetRbacObjectsForAllAppsAndEnvironments mocks base method.
func (m *MockEnforcerUtil) GetRbacObjectsForAllAppsAndEnvironments() (map[int]string, map[string]string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRbacObjectsForAllAppsAndEnvironments")
	ret0, _ := ret[0].(map[int]string)
	ret1, _ := ret[1].(map[string]string)
	return ret0, ret1
}

// GetRbacObjectsForAllAppsAndEnvironments indicates an expected call of GetRbacObjectsForAllAppsAndEnvironments.
func (mr *MockEnforcerUtilMockRecorder) GetRbacObjectsForAllAppsAndEnvironments() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRbacObjectsForAllAppsAndEnvironments", reflect.TypeOf((*MockEnforcerUtil)(nil).GetRbacObjectsForAllAppsAndEnvironments))
}

// GetRbacObjectsForAllAppsWithMatchingAppName mocks base method.
func (m *MockEnforcerUtil) GetRbacObjectsForAllAppsWithMatchingAppName(appNameMatch string) map[int]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRbacObjectsForAllAppsWithMatchingAppName", appNameMatch)
	ret0, _ := ret[0].(map[int]string)
	return ret0
}

// GetRbacObjectsForAllAppsWithMatchingAppName indicates an expected call of GetRbacObjectsForAllAppsWithMatchingAppName.
func (mr *MockEnforcerUtilMockRecorder) GetRbacObjectsForAllAppsWithMatchingAppName(appNameMatch interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRbacObjectsForAllAppsWithMatchingAppName", reflect.TypeOf((*MockEnforcerUtil)(nil).GetRbacObjectsForAllAppsWithMatchingAppName), appNameMatch)
}

// GetRbacObjectsForAllAppsWithTeamID mocks base method.
func (m *MockEnforcerUtil) GetRbacObjectsForAllAppsWithTeamID(teamID int) map[int]string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRbacObjectsForAllAppsWithTeamID", teamID)
	ret0, _ := ret[0].(map[int]string)
	return ret0
}

// GetRbacObjectsForAllAppsWithTeamID indicates an expected call of GetRbacObjectsForAllAppsWithTeamID.
func (mr *MockEnforcerUtilMockRecorder) GetRbacObjectsForAllAppsWithTeamID(teamID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRbacObjectsForAllAppsWithTeamID", reflect.TypeOf((*MockEnforcerUtil)(nil).GetRbacObjectsForAllAppsWithTeamID), teamID)
}

// GetTeamAndEnvironmentRbacObjectByCDPipelineId mocks base method.
func (m *MockEnforcerUtil) GetTeamAndEnvironmentRbacObjectByCDPipelineId(pipelineId int) (string, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTeamAndEnvironmentRbacObjectByCDPipelineId", pipelineId)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(string)
	return ret0, ret1
}

// GetTeamAndEnvironmentRbacObjectByCDPipelineId indicates an expected call of GetTeamAndEnvironmentRbacObjectByCDPipelineId.
func (mr *MockEnforcerUtilMockRecorder) GetTeamAndEnvironmentRbacObjectByCDPipelineId(pipelineId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTeamAndEnvironmentRbacObjectByCDPipelineId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetTeamAndEnvironmentRbacObjectByCDPipelineId), pipelineId)
}

// GetTeamEnvRBACNameByAppId mocks base method.
func (m *MockEnforcerUtil) GetTeamEnvRBACNameByAppId(appId, envId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTeamEnvRBACNameByAppId", appId, envId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetTeamEnvRBACNameByAppId indicates an expected call of GetTeamEnvRBACNameByAppId.
func (mr *MockEnforcerUtilMockRecorder) GetTeamEnvRBACNameByAppId(appId, envId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTeamEnvRBACNameByAppId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetTeamEnvRBACNameByAppId), appId, envId)
}

// GetTeamRBACByCiPipelineId mocks base method.
func (m *MockEnforcerUtil) GetTeamRBACByCiPipelineId(pipelineId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTeamRBACByCiPipelineId", pipelineId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetTeamRBACByCiPipelineId indicates an expected call of GetTeamRBACByCiPipelineId.
func (mr *MockEnforcerUtilMockRecorder) GetTeamRBACByCiPipelineId(pipelineId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTeamRBACByCiPipelineId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetTeamRBACByCiPipelineId), pipelineId)
}

// GetTeamRbacObjectByCiPipelineId mocks base method.
func (m *MockEnforcerUtil) GetTeamRbacObjectByCiPipelineId(ciPipelineId int) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTeamRbacObjectByCiPipelineId", ciPipelineId)
	ret0, _ := ret[0].(string)
	return ret0
}

// GetTeamRbacObjectByCiPipelineId indicates an expected call of GetTeamRbacObjectByCiPipelineId.
func (mr *MockEnforcerUtilMockRecorder) GetTeamRbacObjectByCiPipelineId(ciPipelineId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTeamRbacObjectByCiPipelineId", reflect.TypeOf((*MockEnforcerUtil)(nil).GetTeamRbacObjectByCiPipelineId), ciPipelineId)
}
