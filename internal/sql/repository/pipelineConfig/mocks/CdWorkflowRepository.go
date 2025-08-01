// Code generated by mockery v2.42.0. DO NOT EDIT.

package mocks

import (
	context "context"

	bean "github.com/devtron-labs/devtron/api/bean"

	mock "github.com/stretchr/testify/mock"

	pg "github.com/go-pg/pg"

	pipelineConfig "github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
)

// CdWorkflowRepository is an autogenerated mock type for the CdWorkflowRepository type
type CdWorkflowRepository struct {
	mock.Mock
}

// CheckWorkflowRunnerByReferenceId provides a mock function with given fields: referenceId
func (_m *CdWorkflowRepository) CheckWorkflowRunnerByReferenceId(referenceId string) (bool, error) {
	ret := _m.Called(referenceId)

	if len(ret) == 0 {
		panic("no return value specified for CheckWorkflowRunnerByReferenceId")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (bool, error)); ok {
		return rf(referenceId)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(referenceId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(referenceId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExistsByStatus provides a mock function with given fields: status
func (_m *CdWorkflowRepository) ExistsByStatus(status string) (bool, error) {
	ret := _m.Called(status)

	if len(ret) == 0 {
		panic("no return value specified for ExistsByStatus")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (bool, error)); ok {
		return rf(status)
	}
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(status)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(status)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchAllCdStagesLatestEntity provides a mock function with given fields: pipelineIds
func (_m *CdWorkflowRepository) FetchAllCdStagesLatestEntity(pipelineIds []int) ([]*pipelineConfig.CdWorkflowStatus, error) {
	ret := _m.Called(pipelineIds)

	if len(ret) == 0 {
		panic("no return value specified for FetchAllCdStagesLatestEntity")
	}

	var r0 []*pipelineConfig.CdWorkflowStatus
	var r1 error
	if rf, ok := ret.Get(0).(func([]int) ([]*pipelineConfig.CdWorkflowStatus, error)); ok {
		return rf(pipelineIds)
	}
	if rf, ok := ret.Get(0).(func([]int) []*pipelineConfig.CdWorkflowStatus); ok {
		r0 = rf(pipelineIds)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflowStatus)
		}
	}

	if rf, ok := ret.Get(1).(func([]int) error); ok {
		r1 = rf(pipelineIds)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchAllCdStagesLatestEntityStatus provides a mock function with given fields: wfrIds
func (_m *CdWorkflowRepository) FetchAllCdStagesLatestEntityStatus(wfrIds []int) ([]*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(wfrIds)

	if len(ret) == 0 {
		panic("no return value specified for FetchAllCdStagesLatestEntityStatus")
	}

	var r0 []*pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func([]int) ([]*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(wfrIds)
	}
	if rf, ok := ret.Get(0).(func([]int) []*pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(wfrIds)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func([]int) error); ok {
		r1 = rf(wfrIds)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchArtifactsByCdPipelineId provides a mock function with given fields: pipelineId, runnerType, offset, limit, searchString
func (_m *CdWorkflowRepository) FetchArtifactsByCdPipelineId(pipelineId int, runnerType bean.WorkflowType, offset int, limit int, searchString string) ([]pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(pipelineId, runnerType, offset, limit, searchString)

	if len(ret) == 0 {
		panic("no return value specified for FetchArtifactsByCdPipelineId")
	}

	var r0 []pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, bean.WorkflowType, int, int, string) ([]pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(pipelineId, runnerType, offset, limit, searchString)
	}
	if rf, ok := ret.Get(0).(func(int, bean.WorkflowType, int, int, string) []pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(pipelineId, runnerType, offset, limit, searchString)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, bean.WorkflowType, int, int, string) error); ok {
		r1 = rf(pipelineId, runnerType, offset, limit, searchString)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FetchEnvAllCdStagesLatestEntityStatus provides a mock function with given fields: wfrIds, envID
func (_m *CdWorkflowRepository) FetchEnvAllCdStagesLatestEntityStatus(wfrIds []int, envID int) ([]*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(wfrIds, envID)

	if len(ret) == 0 {
		panic("no return value specified for FetchEnvAllCdStagesLatestEntityStatus")
	}

	var r0 []*pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func([]int, int) ([]*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(wfrIds, envID)
	}
	if rf, ok := ret.Get(0).(func([]int, int) []*pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(wfrIds, envID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func([]int, int) error); ok {
		r1 = rf(wfrIds, envID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindAllTriggeredWorkflowCountInLast24Hour provides a mock function with given fields:
func (_m *CdWorkflowRepository) FindAllTriggeredWorkflowCountInLast24Hour() (int, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for FindAllTriggeredWorkflowCountInLast24Hour")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func() (int, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() int); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindArtifactByPipelineIdAndRunnerType provides a mock function with given fields: pipelineId, runnerType, limit, runnerStatuses
func (_m *CdWorkflowRepository) FindArtifactByPipelineIdAndRunnerType(pipelineId int, runnerType bean.WorkflowType, limit int, runnerStatuses []string) ([]pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(pipelineId, runnerType, limit, runnerStatuses)

	if len(ret) == 0 {
		panic("no return value specified for FindArtifactByPipelineIdAndRunnerType")
	}

	var r0 []pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, bean.WorkflowType, int, []string) ([]pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(pipelineId, runnerType, limit, runnerStatuses)
	}
	if rf, ok := ret.Get(0).(func(int, bean.WorkflowType, int, []string) []pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(pipelineId, runnerType, limit, runnerStatuses)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, bean.WorkflowType, int, []string) error); ok {
		r1 = rf(pipelineId, runnerType, limit, runnerStatuses)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindBasicWorkflowRunnerById provides a mock function with given fields: wfrId
func (_m *CdWorkflowRepository) FindBasicWorkflowRunnerById(wfrId int) (*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(wfrId)

	if len(ret) == 0 {
		panic("no return value specified for FindBasicWorkflowRunnerById")
	}

	var r0 *pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int) (*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(wfrId)
	}
	if rf, ok := ret.Get(0).(func(int) *pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(wfrId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(wfrId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindById provides a mock function with given fields: wfId
func (_m *CdWorkflowRepository) FindById(wfId int) (*pipelineConfig.CdWorkflow, error) {
	ret := _m.Called(wfId)

	if len(ret) == 0 {
		panic("no return value specified for FindById")
	}

	var r0 *pipelineConfig.CdWorkflow
	var r1 error
	if rf, ok := ret.Get(0).(func(int) (*pipelineConfig.CdWorkflow, error)); ok {
		return rf(wfId)
	}
	if rf, ok := ret.Get(0).(func(int) *pipelineConfig.CdWorkflow); ok {
		r0 = rf(wfId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pipelineConfig.CdWorkflow)
		}
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(wfId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindByWorkflowIdAndRunnerType provides a mock function with given fields: ctx, wfId, runnerType
func (_m *CdWorkflowRepository) FindByWorkflowIdAndRunnerType(ctx context.Context, wfId int, runnerType bean.WorkflowType) (pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(ctx, wfId, runnerType)

	if len(ret) == 0 {
		panic("no return value specified for FindByWorkflowIdAndRunnerType")
	}

	var r0 pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, int, bean.WorkflowType) (pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(ctx, wfId, runnerType)
	}
	if rf, ok := ret.Get(0).(func(context.Context, int, bean.WorkflowType) pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(ctx, wfId, runnerType)
	} else {
		r0 = ret.Get(0).(pipelineConfig.CdWorkflowRunner)
	}

	if rf, ok := ret.Get(1).(func(context.Context, int, bean.WorkflowType) error); ok {
		r1 = rf(ctx, wfId, runnerType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindCdWorkflowMetaByEnvironmentId provides a mock function with given fields: appId, environmentId, offset, size
func (_m *CdWorkflowRepository) FindCdWorkflowMetaByEnvironmentId(appId int, environmentId int, offset int, size int) ([]pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(appId, environmentId, offset, size)

	if len(ret) == 0 {
		panic("no return value specified for FindCdWorkflowMetaByEnvironmentId")
	}

	var r0 []pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int, int, int) ([]pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(appId, environmentId, offset, size)
	}
	if rf, ok := ret.Get(0).(func(int, int, int, int) []pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(appId, environmentId, offset, size)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, int, int, int) error); ok {
		r1 = rf(appId, environmentId, offset, size)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindCdWorkflowMetaByPipelineId provides a mock function with given fields: pipelineId, offset, size
func (_m *CdWorkflowRepository) FindCdWorkflowMetaByPipelineId(pipelineId int, offset int, size int) ([]pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(pipelineId, offset, size)

	if len(ret) == 0 {
		panic("no return value specified for FindCdWorkflowMetaByPipelineId")
	}

	var r0 []pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int, int) ([]pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(pipelineId, offset, size)
	}
	if rf, ok := ret.Get(0).(func(int, int, int) []pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(pipelineId, offset, size)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, int, int) error); ok {
		r1 = rf(pipelineId, offset, size)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLastPreOrPostTriggeredByEnvironmentId provides a mock function with given fields: appId, environmentId
func (_m *CdWorkflowRepository) FindLastPreOrPostTriggeredByEnvironmentId(appId int, environmentId int) (pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(appId, environmentId)

	if len(ret) == 0 {
		panic("no return value specified for FindLastPreOrPostTriggeredByEnvironmentId")
	}

	var r0 pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int) (pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(appId, environmentId)
	}
	if rf, ok := ret.Get(0).(func(int, int) pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(appId, environmentId)
	} else {
		r0 = ret.Get(0).(pipelineConfig.CdWorkflowRunner)
	}

	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(appId, environmentId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLastPreOrPostTriggeredByPipelineId provides a mock function with given fields: pipelineId
func (_m *CdWorkflowRepository) FindLastPreOrPostTriggeredByPipelineId(pipelineId int) (pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(pipelineId)

	if len(ret) == 0 {
		panic("no return value specified for FindLastPreOrPostTriggeredByPipelineId")
	}

	var r0 pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int) (pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(pipelineId)
	}
	if rf, ok := ret.Get(0).(func(int) pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(pipelineId)
	} else {
		r0 = ret.Get(0).(pipelineConfig.CdWorkflowRunner)
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(pipelineId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLastUnFailedProcessedRunner provides a mock function with given fields: appId, environmentId
func (_m *CdWorkflowRepository) FindLastUnFailedProcessedRunner(appId int, environmentId int) (*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(appId, environmentId)

	if len(ret) == 0 {
		panic("no return value specified for FindLastUnFailedProcessedRunner")
	}

	var r0 *pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int) (*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(appId, environmentId)
	}
	if rf, ok := ret.Get(0).(func(int, int) *pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(appId, environmentId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(appId, environmentId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLatestByPipelineIdAndRunnerType provides a mock function with given fields: pipelineId, runnerType
func (_m *CdWorkflowRepository) FindLatestByPipelineIdAndRunnerType(pipelineId int, runnerType bean.WorkflowType) (pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(pipelineId, runnerType)

	if len(ret) == 0 {
		panic("no return value specified for FindLatestByPipelineIdAndRunnerType")
	}

	var r0 pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, bean.WorkflowType) (pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(pipelineId, runnerType)
	}
	if rf, ok := ret.Get(0).(func(int, bean.WorkflowType) pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(pipelineId, runnerType)
	} else {
		r0 = ret.Get(0).(pipelineConfig.CdWorkflowRunner)
	}

	if rf, ok := ret.Get(1).(func(int, bean.WorkflowType) error); ok {
		r1 = rf(pipelineId, runnerType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLatestCdWorkflowByPipelineId provides a mock function with given fields: pipelineIds
func (_m *CdWorkflowRepository) FindLatestCdWorkflowByPipelineId(pipelineIds []int) (*pipelineConfig.CdWorkflow, error) {
	ret := _m.Called(pipelineIds)

	if len(ret) == 0 {
		panic("no return value specified for FindLatestCdWorkflowByPipelineId")
	}

	var r0 *pipelineConfig.CdWorkflow
	var r1 error
	if rf, ok := ret.Get(0).(func([]int) (*pipelineConfig.CdWorkflow, error)); ok {
		return rf(pipelineIds)
	}
	if rf, ok := ret.Get(0).(func([]int) *pipelineConfig.CdWorkflow); ok {
		r0 = rf(pipelineIds)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pipelineConfig.CdWorkflow)
		}
	}

	if rf, ok := ret.Get(1).(func([]int) error); ok {
		r1 = rf(pipelineIds)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLatestCdWorkflowByPipelineIdV2 provides a mock function with given fields: pipelineIds
func (_m *CdWorkflowRepository) FindLatestCdWorkflowByPipelineIdV2(pipelineIds []int) ([]*pipelineConfig.CdWorkflow, error) {
	ret := _m.Called(pipelineIds)

	if len(ret) == 0 {
		panic("no return value specified for FindLatestCdWorkflowByPipelineIdV2")
	}

	var r0 []*pipelineConfig.CdWorkflow
	var r1 error
	if rf, ok := ret.Get(0).(func([]int) ([]*pipelineConfig.CdWorkflow, error)); ok {
		return rf(pipelineIds)
	}
	if rf, ok := ret.Get(0).(func([]int) []*pipelineConfig.CdWorkflow); ok {
		r0 = rf(pipelineIds)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflow)
		}
	}

	if rf, ok := ret.Get(1).(func([]int) error); ok {
		r1 = rf(pipelineIds)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLatestCdWorkflowRunnerByEnvironmentIdAndRunnerType provides a mock function with given fields: appId, environmentId, runnerType
func (_m *CdWorkflowRepository) FindLatestCdWorkflowRunnerByEnvironmentIdAndRunnerType(appId int, environmentId int, runnerType bean.WorkflowType) (pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(appId, environmentId, runnerType)

	if len(ret) == 0 {
		panic("no return value specified for FindLatestCdWorkflowRunnerByEnvironmentIdAndRunnerType")
	}

	var r0 pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int, bean.WorkflowType) (pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(appId, environmentId, runnerType)
	}
	if rf, ok := ret.Get(0).(func(int, int, bean.WorkflowType) pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(appId, environmentId, runnerType)
	} else {
		r0 = ret.Get(0).(pipelineConfig.CdWorkflowRunner)
	}

	if rf, ok := ret.Get(1).(func(int, int, bean.WorkflowType) error); ok {
		r1 = rf(appId, environmentId, runnerType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLatestRunnerByPipelineIdsAndRunnerType provides a mock function with given fields: ctx, pipelineIds, runnerType
func (_m *CdWorkflowRepository) FindLatestRunnerByPipelineIdsAndRunnerType(ctx context.Context, pipelineIds []int, runnerType bean.WorkflowType) ([]pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(ctx, pipelineIds, runnerType)

	if len(ret) == 0 {
		panic("no return value specified for FindLatestRunnerByPipelineIdsAndRunnerType")
	}

	var r0 []pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []int, bean.WorkflowType) ([]pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(ctx, pipelineIds, runnerType)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []int, bean.WorkflowType) []pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(ctx, pipelineIds, runnerType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []int, bean.WorkflowType) error); ok {
		r1 = rf(ctx, pipelineIds, runnerType)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLatestWfrByAppIdAndEnvironmentId provides a mock function with given fields: appId, environmentId
func (_m *CdWorkflowRepository) FindLatestWfrByAppIdAndEnvironmentId(appId int, environmentId int) (*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(appId, environmentId)

	if len(ret) == 0 {
		panic("no return value specified for FindLatestWfrByAppIdAndEnvironmentId")
	}

	var r0 *pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int) (*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(appId, environmentId)
	}
	if rf, ok := ret.Get(0).(func(int, int) *pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(appId, environmentId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(appId, environmentId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindPreviousCdWfRunnerByStatus provides a mock function with given fields: pipelineId, currentWFRunnerId, status
func (_m *CdWorkflowRepository) FindPreviousCdWfRunnerByStatus(pipelineId int, currentWFRunnerId int, status []string) ([]*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(pipelineId, currentWFRunnerId, status)

	if len(ret) == 0 {
		panic("no return value specified for FindPreviousCdWfRunnerByStatus")
	}

	var r0 []*pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int, []string) ([]*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(pipelineId, currentWFRunnerId, status)
	}
	if rf, ok := ret.Get(0).(func(int, int, []string) []*pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(pipelineId, currentWFRunnerId, status)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, int, []string) error); ok {
		r1 = rf(pipelineId, currentWFRunnerId, status)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindRetriedWorkflowCountByReferenceId provides a mock function with given fields: wfrId
func (_m *CdWorkflowRepository) FindRetriedWorkflowCountByReferenceId(wfrId int) (int, error) {
	ret := _m.Called(wfrId)

	if len(ret) == 0 {
		panic("no return value specified for FindRetriedWorkflowCountByReferenceId")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(int) (int, error)); ok {
		return rf(wfrId)
	}
	if rf, ok := ret.Get(0).(func(int) int); ok {
		r0 = rf(wfrId)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(wfrId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindWorkflowRunnerByCdWorkflowId provides a mock function with given fields: wfIds
func (_m *CdWorkflowRepository) FindWorkflowRunnerByCdWorkflowId(wfIds []int) ([]*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(wfIds)

	if len(ret) == 0 {
		panic("no return value specified for FindWorkflowRunnerByCdWorkflowId")
	}

	var r0 []*pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func([]int) ([]*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(wfIds)
	}
	if rf, ok := ret.Get(0).(func([]int) []*pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(wfIds)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func([]int) error); ok {
		r1 = rf(wfIds)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindWorkflowRunnerById provides a mock function with given fields: wfrId
func (_m *CdWorkflowRepository) FindWorkflowRunnerById(wfrId int) (*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(wfrId)

	if len(ret) == 0 {
		panic("no return value specified for FindWorkflowRunnerById")
	}

	var r0 *pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int) (*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(wfrId)
	}
	if rf, ok := ret.Get(0).(func(int) *pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(wfrId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(wfrId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetConnection provides a mock function with given fields:
func (_m *CdWorkflowRepository) GetConnection() *pg.DB {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetConnection")
	}

	var r0 *pg.DB
	if rf, ok := ret.Get(0).(func() *pg.DB); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pg.DB)
		}
	}

	return r0
}

// GetLatestTriggersOfHelmPipelinesStuckInNonTerminalStatuses provides a mock function with given fields: getPipelineDeployedWithinHours
func (_m *CdWorkflowRepository) GetLatestTriggersOfHelmPipelinesStuckInNonTerminalStatuses(getPipelineDeployedWithinHours int) ([]*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(getPipelineDeployedWithinHours)

	if len(ret) == 0 {
		panic("no return value specified for GetLatestTriggersOfPipelinesStuckInNonTerminalStatuses")
	}

	var r0 []*pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int) ([]*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(getPipelineDeployedWithinHours)
	}
	if rf, ok := ret.Get(0).(func(int) []*pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(getPipelineDeployedWithinHours)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(getPipelineDeployedWithinHours)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPreviousQueuedRunners provides a mock function with given fields: cdWfrId, pipelineId
func (_m *CdWorkflowRepository) GetPreviousQueuedRunners(cdWfrId int, pipelineId int) ([]*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(cdWfrId, pipelineId)

	if len(ret) == 0 {
		panic("no return value specified for GetPreviousQueuedRunners")
	}

	var r0 []*pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int) ([]*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(cdWfrId, pipelineId)
	}
	if rf, ok := ret.Get(0).(func(int, int) []*pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(cdWfrId, pipelineId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(cdWfrId, pipelineId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsLatestCDWfr provides a mock function with given fields: pipelineId, wfrId
func (_m *CdWorkflowRepository) IsLatestCDWfr(pipelineId int, wfrId int) (bool, error) {
	ret := _m.Called(pipelineId, wfrId)

	if len(ret) == 0 {
		panic("no return value specified for IsLatestCDWfr")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int) (bool, error)); ok {
		return rf(pipelineId, wfrId)
	}
	if rf, ok := ret.Get(0).(func(int, int) bool); ok {
		r0 = rf(pipelineId, wfrId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(pipelineId, wfrId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsLatestWf provides a mock function with given fields: pipelineId, wfId
func (_m *CdWorkflowRepository) IsLatestWf(pipelineId int, wfId int) (bool, error) {
	ret := _m.Called(pipelineId, wfId)

	if len(ret) == 0 {
		panic("no return value specified for IsLatestWf")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(int, int) (bool, error)); ok {
		return rf(pipelineId, wfId)
	}
	if rf, ok := ret.Get(0).(func(int, int) bool); ok {
		r0 = rf(pipelineId, wfId)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(int, int) error); ok {
		r1 = rf(pipelineId, wfId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveWorkFlow provides a mock function with given fields: ctx, wf
func (_m *CdWorkflowRepository) SaveWorkFlow(ctx context.Context, wf *pipelineConfig.CdWorkflow) error {
	ret := _m.Called(ctx, wf)

	if len(ret) == 0 {
		panic("no return value specified for SaveWorkFlow")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *pipelineConfig.CdWorkflow) error); ok {
		r0 = rf(ctx, wf)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SaveWorkFlowRunner provides a mock function with given fields: wfr
func (_m *CdWorkflowRepository) SaveWorkFlowRunner(wfr *pipelineConfig.CdWorkflowRunner) (*pipelineConfig.CdWorkflowRunner, error) {
	ret := _m.Called(wfr)

	if len(ret) == 0 {
		panic("no return value specified for SaveWorkFlowRunner")
	}

	var r0 *pipelineConfig.CdWorkflowRunner
	var r1 error
	if rf, ok := ret.Get(0).(func(*pipelineConfig.CdWorkflowRunner) (*pipelineConfig.CdWorkflowRunner, error)); ok {
		return rf(wfr)
	}
	if rf, ok := ret.Get(0).(func(*pipelineConfig.CdWorkflowRunner) *pipelineConfig.CdWorkflowRunner); ok {
		r0 = rf(wfr)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pipelineConfig.CdWorkflowRunner)
		}
	}

	if rf, ok := ret.Get(1).(func(*pipelineConfig.CdWorkflowRunner) error); ok {
		r1 = rf(wfr)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveWorkFlows provides a mock function with given fields: wfs
func (_m *CdWorkflowRepository) SaveWorkFlows(wfs ...*pipelineConfig.CdWorkflow) error {
	_va := make([]interface{}, len(wfs))
	for _i := range wfs {
		_va[_i] = wfs[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for SaveWorkFlows")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(...*pipelineConfig.CdWorkflow) error); ok {
		r0 = rf(wfs...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateIsArtifactUploaded provides a mock function with given fields: wfrId, isArtifactUploaded
func (_m *CdWorkflowRepository) UpdateIsArtifactUploaded(wfrId int, isArtifactUploaded bool) error {
	ret := _m.Called(wfrId, isArtifactUploaded)

	if len(ret) == 0 {
		panic("no return value specified for UpdateIsArtifactUploaded")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(int, bool) error); ok {
		r0 = rf(wfrId, isArtifactUploaded)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateRunnerStatusToFailedForIds provides a mock function with given fields: errMsg, triggeredBy, cdWfrIds
func (_m *CdWorkflowRepository) UpdateRunnerStatusToFailedForIds(errMsg string, triggeredBy int32, cdWfrIds ...int) error {
	_va := make([]interface{}, len(cdWfrIds))
	for _i := range cdWfrIds {
		_va[_i] = cdWfrIds[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, errMsg, triggeredBy)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for UpdateRunnerStatusToFailedForIds")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, int32, ...int) error); ok {
		r0 = rf(errMsg, triggeredBy, cdWfrIds...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateWorkFlow provides a mock function with given fields: wf
func (_m *CdWorkflowRepository) UpdateWorkFlow(wf *pipelineConfig.CdWorkflow) error {
	ret := _m.Called(wf)

	if len(ret) == 0 {
		panic("no return value specified for UpdateWorkFlow")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*pipelineConfig.CdWorkflow) error); ok {
		r0 = rf(wf)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateWorkFlowRunner provides a mock function with given fields: wfr
func (_m *CdWorkflowRepository) UpdateWorkFlowRunner(wfr *pipelineConfig.CdWorkflowRunner) error {
	ret := _m.Called(wfr)

	if len(ret) == 0 {
		panic("no return value specified for UpdateWorkFlowRunner")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*pipelineConfig.CdWorkflowRunner) error); ok {
		r0 = rf(wfr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateWorkFlowRunners provides a mock function with given fields: wfr
func (_m *CdWorkflowRepository) UpdateWorkFlowRunners(wfr []*pipelineConfig.CdWorkflowRunner) error {
	ret := _m.Called(wfr)

	if len(ret) == 0 {
		panic("no return value specified for UpdateWorkFlowRunners")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]*pipelineConfig.CdWorkflowRunner) error); ok {
		r0 = rf(wfr)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateWorkFlowRunnersWithTxn provides a mock function with given fields: wfrs, tx
func (_m *CdWorkflowRepository) UpdateWorkFlowRunnersWithTxn(wfrs []*pipelineConfig.CdWorkflowRunner, tx *pg.Tx) error {
	ret := _m.Called(wfrs, tx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateWorkFlowRunnersWithTxn")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func([]*pipelineConfig.CdWorkflowRunner, *pg.Tx) error); ok {
		r0 = rf(wfrs, tx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewCdWorkflowRepository creates a new instance of CdWorkflowRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewCdWorkflowRepository(t interface {
	mock.TestingT
	Cleanup(func())
}) *CdWorkflowRepository {
	mock := &CdWorkflowRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
