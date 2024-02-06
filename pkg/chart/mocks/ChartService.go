// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	chart "github.com/devtron-labs/devtron/pkg/chart"

	json "encoding/json"

	mock "github.com/stretchr/testify/mock"

	util "github.com/devtron-labs/devtron/internal/util"
)

// ChartService is an autogenerated mock type for the ChartService type
type ChartService struct {
	mock.Mock
}

// AppMetricsEnableDisable provides a mock function with given fields: appMetricRequest
func (_m *ChartService) AppMetricsEnableDisable(appMetricRequest chart.AppMetricEnableDisableRequest) (*chart.AppMetricEnableDisableRequest, error) {
	ret := _m.Called(appMetricRequest)

	var r0 *chart.AppMetricEnableDisableRequest
	if rf, ok := ret.Get(0).(func(chart.AppMetricEnableDisableRequest) *chart.AppMetricEnableDisableRequest); ok {
		r0 = rf(appMetricRequest)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.AppMetricEnableDisableRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(chart.AppMetricEnableDisableRequest) error); ok {
		r1 = rf(appMetricRequest)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChartRefAutocomplete provides a mock function with given fields:
func (_m *ChartService) ChartRefAutocomplete() ([]chart.ChartRef, error) {
	ret := _m.Called()

	var r0 []chart.ChartRef
	if rf, ok := ret.Get(0).(func() []chart.ChartRef); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]chart.ChartRef)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChartRefAutocompleteForAppOrEnv provides a mock function with given fields: appId, envId
func (_m *ChartService) ChartRefAutocompleteForAppOrEnv(appId int, envId int) (*chart.ChartRefResponse, error) {
	ret := _m.Called(appId, envId)

	var r0 *chart.ChartRefResponse
	if rf, ok := ret.Get(0).(func(int, int) *chart.ChartRefResponse); ok {
		r0 = rf(appId, envId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.ChartRefResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(appId, envId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChartRefIdsCompatible provides a mock function with given fields: oldChartRefId, newChartRefId
func (_m *ChartService) ChartRefIdsCompatible(oldChartRefId int, newChartRefId int) (bool, string, string) {
	ret := _m.Called(oldChartRefId, newChartRefId)

	var r0 bool
	if rf, ok := ret.Get(0).(func(int, int) bool); ok {
		r0 = rf(oldChartRefId, newChartRefId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(int, int) string); ok {
		r1 = rf(oldChartRefId, newChartRefId)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 string
	if rf, ok := ret.Get(2).(func(int, int) string); ok {
		r2 = rf(oldChartRefId, newChartRefId)
	} else {
		r2 = ret.Get(2).(string)
	}

	return r0, r1, r2
}

// CheckChartExists provides a mock function with given fields: chartRefId
func (_m *ChartService) CheckChartExists(chartRefId int) error {
	ret := _m.Called(chartRefId)

	var r0 error
	if rf, ok := ret.Get(0).(func(int) error); ok {
		r0 = rf(chartRefId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CheckCustomChartByAppId provides a mock function with given fields: id
func (_m *ChartService) CheckCustomChartByAppId(id int) (bool, error) {
	ret := _m.Called(id)

	var r0 bool
	if rf, ok := ret.Get(0).(func(int) bool); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckCustomChartByChartId provides a mock function with given fields: id
func (_m *ChartService) CheckCustomChartByChartId(id int) (bool, error) {
	ret := _m.Called(id)

	var r0 bool
	if rf, ok := ret.Get(0).(func(int) bool); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CheckIsAppMetricsSupported provides a mock function with given fields: chartRefId
func (_m *ChartService) CheckIsAppMetricsSupported(chartRefId int) (bool, error) {
	ret := _m.Called(chartRefId)

	var r0 bool
	if rf, ok := ret.Get(0).(func(int) bool); ok {
		r0 = rf(chartRefId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(chartRefId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Create provides a mock function with given fields: templateRequest, ctx
func (_m *ChartService) Create(templateRequest chart.TemplateRequest, ctx context.Context) (*chart.TemplateRequest, error) {
	ret := _m.Called(templateRequest, ctx)

	var r0 *chart.TemplateRequest
	if rf, ok := ret.Get(0).(func(chart.TemplateRequest, context.Context) *chart.TemplateRequest); ok {
		r0 = rf(templateRequest, ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.TemplateRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(chart.TemplateRequest, context.Context) error); ok {
		r1 = rf(templateRequest, ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateChartFromEnvOverride provides a mock function with given fields: templateRequest, ctx
func (_m *ChartService) CreateChartFromEnvOverride(templateRequest chart.TemplateRequest, ctx context.Context) (*chart.TemplateRequest, error) {
	ret := _m.Called(templateRequest, ctx)

	var r0 *chart.TemplateRequest
	if rf, ok := ret.Get(0).(func(chart.TemplateRequest, context.Context) *chart.TemplateRequest); ok {
		r0 = rf(templateRequest, ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.TemplateRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(chart.TemplateRequest, context.Context) error); ok {
		r1 = rf(templateRequest, ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeploymentTemplateValidate provides a mock function with given fields: ctx, templatejson, chartRefId
func (_m *ChartService) DeploymentTemplateValidate(ctx context.Context, templatejson interface{}, chartRefId int) (bool, error) {
	ret := _m.Called(ctx, templatejson, chartRefId)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, interface{}, int) bool); ok {
		r0 = rf(ctx, templatejson, chartRefId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, interface{}, int) error); ok {
		r1 = rf(ctx, templatejson, chartRefId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExtractChartIfMissing provides a mock function with given fields: chartData, refChartDir, location
func (_m *ChartService) ExtractChartIfMissing(chartData []byte, refChartDir string, location string) (*chart.ChartDataInfo, error) {
	ret := _m.Called(chartData, refChartDir, location)

	var r0 *chart.ChartDataInfo
	if rf, ok := ret.Get(0).(func([]byte, string, string) *chart.ChartDataInfo); ok {
		r0 = rf(chartData, refChartDir, location)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.ChartDataInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func([]byte, string, string) error); ok {
		r1 = rf(chartData, refChartDir, location)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchChartInfoByFlag provides a mock function with given fields: userUploaded
func (_m *ChartService) FetchChartInfoByFlag(userUploaded bool) ([]*chart.ChartDto, error) {
	ret := _m.Called(userUploaded)

	var r0 []*chart.ChartDto
	if rf, ok := ret.Get(0).(func(bool) []*chart.ChartDto); ok {
		r0 = rf(userUploaded)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*chart.ChartDto)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(bool) error); ok {
		r1 = rf(userUploaded)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLatestChartForAppByAppId provides a mock function with given fields: appId
func (_m *ChartService) FindLatestChartForAppByAppId(appId int) (*chart.TemplateRequest, error) {
	ret := _m.Called(appId)

	var r0 *chart.TemplateRequest
	if rf, ok := ret.Get(0).(func(int) *chart.TemplateRequest); ok {
		r0 = rf(appId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.TemplateRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(appId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindPreviousChartByAppId provides a mock function with given fields: appId
func (_m *ChartService) FindPreviousChartByAppId(appId int) (*chart.TemplateRequest, error) {
	ret := _m.Called(appId)

	var r0 *chart.TemplateRequest
	if rf, ok := ret.Get(0).(func(int) *chart.TemplateRequest); ok {
		r0 = rf(appId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.TemplateRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(appId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FlaggerCanaryEnabled provides a mock function with given fields: values
func (_m *ChartService) FlaggerCanaryEnabled(values json.RawMessage) (bool, error) {
	ret := _m.Called(values)

	var r0 bool
	if rf, ok := ret.Get(0).(func(json.RawMessage) bool); ok {
		r0 = rf(values)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(json.RawMessage) error); ok {
		r1 = rf(values)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAppOverrideForDefaultTemplate provides a mock function with given fields: chartRefId
func (_m *ChartService) GetAppOverrideForDefaultTemplate(chartRefId int) (map[string]interface{}, error) {
	ret := _m.Called(chartRefId)

	var r0 map[string]interface{}
	if rf, ok := ret.Get(0).(func(int) map[string]interface{}); ok {
		r0 = rf(chartRefId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(chartRefId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByAppIdAndChartRefId provides a mock function with given fields: appId, chartRefId
func (_m *ChartService) GetByAppIdAndChartRefId(appId int, chartRefId int) (*chart.TemplateRequest, error) {
	ret := _m.Called(appId, chartRefId)

	var r0 *chart.TemplateRequest
	if rf, ok := ret.Get(0).(func(int, int) *chart.TemplateRequest); ok {
		r0 = rf(appId, chartRefId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.TemplateRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(appId, chartRefId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetLocationFromChartNameAndVersion provides a mock function with given fields: chartName, chartVersion
func (_m *ChartService) GetLocationFromChartNameAndVersion(chartName string, chartVersion string) string {
	ret := _m.Called(chartName, chartVersion)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(chartName, chartVersion)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetSchemaAndReadmeForTemplateByChartRefId provides a mock function with given fields: chartRefId
func (_m *ChartService) GetSchemaAndReadmeForTemplateByChartRefId(chartRefId int) ([]byte, []byte, error) {
	ret := _m.Called(chartRefId)

	var r0 []byte
	if rf, ok := ret.Get(0).(func(int) []byte); ok {
		r0 = rf(chartRefId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	var r1 []byte
	if rf, ok := ret.Get(1).(func(int) []byte); ok {
		r1 = rf(chartRefId)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]byte)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int) error); ok {
		r2 = rf(chartRefId)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// IsReadyToTrigger provides a mock function with given fields: appId, envId, pipelineId
func (_m *ChartService) IsReadyToTrigger(appId int, envId int, pipelineId int) (chart.IsReady, error) {
	ret := _m.Called(appId, envId, pipelineId)

	var r0 chart.IsReady
	if rf, ok := ret.Get(0).(func(int, int, int) chart.IsReady); ok {
		r0 = rf(appId, envId, pipelineId)
	} else {
		r0 = ret.Get(0).(chart.IsReady)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int, int) error); ok {
		r1 = rf(appId, envId, pipelineId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// JsonSchemaExtractFromFile provides a mock function with given fields: chartRefId
func (_m *ChartService) JsonSchemaExtractFromFile(chartRefId int) (map[string]interface{}, string, error) {
	ret := _m.Called(chartRefId)

	var r0 map[string]interface{}
	if rf, ok := ret.Get(0).(func(int) map[string]interface{}); ok {
		r0 = rf(chartRefId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	var r1 string
	if rf, ok := ret.Get(1).(func(int) string); ok {
		r1 = rf(chartRefId)
	} else {
		r1 = ret.Get(1).(string)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int) error); ok {
		r2 = rf(chartRefId)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// PatchEnvOverrides provides a mock function with given fields: values, oldChartType, newChartType
func (_m *ChartService) PatchEnvOverrides(values json.RawMessage, oldChartType string, newChartType string) (json.RawMessage, error) {
	ret := _m.Called(values, oldChartType, newChartType)

	var r0 json.RawMessage
	if rf, ok := ret.Get(0).(func(json.RawMessage, string, string) json.RawMessage); ok {
		r0 = rf(values, oldChartType, newChartType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(json.RawMessage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(json.RawMessage, string, string) error); ok {
		r1 = rf(values, oldChartType, newChartType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReadChartMetaDataForLocation provides a mock function with given fields: chartDir, fileName
func (_m *ChartService) ReadChartMetaDataForLocation(chartDir string, fileName string) (*chart.ChartYamlStruct, error) {
	ret := _m.Called(chartDir, fileName)

	var r0 *chart.ChartYamlStruct
	if rf, ok := ret.Get(0).(func(string, string) *chart.ChartYamlStruct); ok {
		r0 = rf(chartDir, fileName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.ChartYamlStruct)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(chartDir, fileName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateAppOverride provides a mock function with given fields: ctx, templateRequest
func (_m *ChartService) UpdateAppOverride(ctx context.Context, templateRequest *chart.TemplateRequest) (*chart.TemplateRequest, error) {
	ret := _m.Called(ctx, templateRequest)

	var r0 *chart.TemplateRequest
	if rf, ok := ret.Get(0).(func(context.Context, *chart.TemplateRequest) *chart.TemplateRequest); ok {
		r0 = rf(ctx, templateRequest)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*chart.TemplateRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *chart.TemplateRequest) error); ok {
		r1 = rf(ctx, templateRequest)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpgradeForApp provides a mock function with given fields: appId, chartRefId, newAppOverride, userId, ctx
func (_m *ChartService) UpgradeForApp(appId int, chartRefId int, newAppOverride map[string]interface{}, userId int32, ctx context.Context) (bool, error) {
	ret := _m.Called(appId, chartRefId, newAppOverride, userId, ctx)

	var r0 bool
	if rf, ok := ret.Get(0).(func(int, int, map[string]interface{}, int32, context.Context) bool); ok {
		r0 = rf(appId, chartRefId, newAppOverride, userId, ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int, int, map[string]interface{}, int32, context.Context) error); ok {
		r1 = rf(appId, chartRefId, newAppOverride, userId, ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ValidateUploadedFileFormat provides a mock function with given fields: fileName
func (_m *ChartService) ValidateUploadedFileFormat(fileName string) error {
	ret := _m.Called(fileName)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(fileName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewChartService interface {
	mock.TestingT
	Cleanup(func())
}

// NewChartService creates a new instance of ChartService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewChartService(t mockConstructorTestingTNewChartService) *ChartService {
	mock := &ChartService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
