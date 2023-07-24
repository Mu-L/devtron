// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	bean "github.com/devtron-labs/devtron/pkg/globalPolicy/bean"

	mock "github.com/stretchr/testify/mock"
)

// GlobalPolicyService is an autogenerated mock type for the GlobalPolicyService type
type GlobalPolicyService struct {
	mock.Mock
}

// CreateOrUpdateGlobalPolicy provides a mock function with given fields: policy
func (_m *GlobalPolicyService) CreateOrUpdateGlobalPolicy(policy *bean.GlobalPolicyDto) error {
	ret := _m.Called(policy)

	var r0 error
	if rf, ok := ret.Get(0).(func(*bean.GlobalPolicyDto) error); ok {
		r0 = rf(policy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteGlobalPolicy provides a mock function with given fields: policyId, userId
func (_m *GlobalPolicyService) DeleteGlobalPolicy(policyId int, userId int32) error {
	ret := _m.Called(policyId, userId)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, int32) error); ok {
		r0 = rf(policyId, userId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAllGlobalPolicies provides a mock function with given fields: policyOf, policyVersion
func (_m *GlobalPolicyService) GetAllGlobalPolicies(policyOf bean.GlobalPolicyType, policyVersion bean.GlobalPolicyVersion) ([]*bean.GlobalPolicyDto, error) {
	ret := _m.Called(policyOf, policyVersion)

	var r0 []*bean.GlobalPolicyDto
	if rf, ok := ret.Get(0).(func(bean.GlobalPolicyType, bean.GlobalPolicyVersion) []*bean.GlobalPolicyDto); ok {
		r0 = rf(policyOf, policyVersion)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*bean.GlobalPolicyDto)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(bean.GlobalPolicyType, bean.GlobalPolicyVersion) error); ok {
		r1 = rf(policyOf, policyVersion)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBlockageStateForACIPipelineTrigger provides a mock function with given fields: ciPipelineId, parentCiPipelineId, branchValues, toOnlyGetBlockedStatePolicies
func (_m *GlobalPolicyService) GetBlockageStateForACIPipelineTrigger(ciPipelineId int, parentCiPipelineId int, branchValues []string, toOnlyGetBlockedStatePolicies bool) (bool, bool, *bean.ConsequenceDto, error) {
	ret := _m.Called(ciPipelineId, parentCiPipelineId, branchValues, toOnlyGetBlockedStatePolicies)

	var r0 bool
	if rf, ok := ret.Get(0).(func(int, int, []string, bool) bool); ok {
		r0 = rf(ciPipelineId, parentCiPipelineId, branchValues, toOnlyGetBlockedStatePolicies)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(int, int, []string, bool) bool); ok {
		r1 = rf(ciPipelineId, parentCiPipelineId, branchValues, toOnlyGetBlockedStatePolicies)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 *bean.ConsequenceDto
	if rf, ok := ret.Get(2).(func(int, int, []string, bool) *bean.ConsequenceDto); ok {
		r2 = rf(ciPipelineId, parentCiPipelineId, branchValues, toOnlyGetBlockedStatePolicies)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).(*bean.ConsequenceDto)
		}
	}

	var r3 error
	if rf, ok := ret.Get(3).(func(int, int, []string, bool) error); ok {
		r3 = rf(ciPipelineId, parentCiPipelineId, branchValues, toOnlyGetBlockedStatePolicies)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// GetById provides a mock function with given fields: id
func (_m *GlobalPolicyService) GetById(id int) (*bean.GlobalPolicyDto, error) {
	ret := _m.Called(id)

	var r0 *bean.GlobalPolicyDto
	if rf, ok := ret.Get(0).(func(int) *bean.GlobalPolicyDto); ok {
		r0 = rf(id)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bean.GlobalPolicyDto)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(id)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMandatoryPluginsForACiPipeline provides a mock function with given fields: ciPipelineId, appId, branchValues, toOnlyGetBlockedStatePolicies
func (_m *GlobalPolicyService) GetMandatoryPluginsForACiPipeline(ciPipelineId int, appId int, branchValues []string, toOnlyGetBlockedStatePolicies bool) (*bean.MandatoryPluginDto, map[string]*bean.ConsequenceDto, error) {
	ret := _m.Called(ciPipelineId, appId, branchValues, toOnlyGetBlockedStatePolicies)

	var r0 *bean.MandatoryPluginDto
	if rf, ok := ret.Get(0).(func(int, int, []string, bool) *bean.MandatoryPluginDto); ok {
		r0 = rf(ciPipelineId, appId, branchValues, toOnlyGetBlockedStatePolicies)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bean.MandatoryPluginDto)
		}
	}

	var r1 map[string]*bean.ConsequenceDto
	if rf, ok := ret.Get(1).(func(int, int, []string, bool) map[string]*bean.ConsequenceDto); ok {
		r1 = rf(ciPipelineId, appId, branchValues, toOnlyGetBlockedStatePolicies)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(map[string]*bean.ConsequenceDto)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(int, int, []string, bool) error); ok {
		r2 = rf(ciPipelineId, appId, branchValues, toOnlyGetBlockedStatePolicies)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// GetOnlyBlockageStateForCiPipeline provides a mock function with given fields: ciPipelineId, branchValues
func (_m *GlobalPolicyService) GetOnlyBlockageStateForCiPipeline(ciPipelineId int, branchValues []string) (bool, bool, *bean.ConsequenceDto, error) {
	ret := _m.Called(ciPipelineId, branchValues)

	var r0 bool
	if rf, ok := ret.Get(0).(func(int, []string) bool); ok {
		r0 = rf(ciPipelineId, branchValues)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(int, []string) bool); ok {
		r1 = rf(ciPipelineId, branchValues)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 *bean.ConsequenceDto
	if rf, ok := ret.Get(2).(func(int, []string) *bean.ConsequenceDto); ok {
		r2 = rf(ciPipelineId, branchValues)
	} else {
		if ret.Get(2) != nil {
			r2 = ret.Get(2).(*bean.ConsequenceDto)
		}
	}

	var r3 error
	if rf, ok := ret.Get(3).(func(int, []string) error); ok {
		r3 = rf(ciPipelineId, branchValues)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// GetPolicyOffendingPipelinesWfTree provides a mock function with given fields: policyId
func (_m *GlobalPolicyService) GetPolicyOffendingPipelinesWfTree(policyId int) (*bean.PolicyOffendingPipelineWfTreeObject, error) {
	ret := _m.Called(policyId)

	var r0 *bean.PolicyOffendingPipelineWfTreeObject
	if rf, ok := ret.Get(0).(func(int) *bean.PolicyOffendingPipelineWfTreeObject); ok {
		r0 = rf(policyId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bean.PolicyOffendingPipelineWfTreeObject)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(policyId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewGlobalPolicyService interface {
	mock.TestingT
	Cleanup(func())
}

// NewGlobalPolicyService creates a new instance of GlobalPolicyService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewGlobalPolicyService(t mockConstructorTestingTNewGlobalPolicyService) *GlobalPolicyService {
	mock := &GlobalPolicyService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
