// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	protect "github.com/devtron-labs/devtron/enterprise/pkg/protect"
	mock "github.com/stretchr/testify/mock"
)

// ResourceProtectionRepository is an autogenerated mock type for the ResourceProtectionRepository type
type ResourceProtectionRepository struct {
	mock.Mock
}

// ConfigureResourceProtection provides a mock function with given fields: appId, envId, state, userId
func (_m *ResourceProtectionRepository) ConfigureResourceProtection(appId int, envId int, state protect.ProtectionState, userId int32) error {
	ret := _m.Called(appId, envId, state, userId)

	var r0 error
	if rf, ok := ret.Get(0).(func(int, int, protect.ProtectionState, int32) error); ok {
		r0 = rf(appId, envId, state, userId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetResourceProtectMetadata provides a mock function with given fields: appId
func (_m *ResourceProtectionRepository) GetResourceProtectMetadata(appId int) ([]*protect.ResourceProtectionDto, error) {
	ret := _m.Called(appId)

	var r0 []*protect.ResourceProtectionDto
	if rf, ok := ret.Get(0).(func(int) []*protect.ResourceProtectionDto); ok {
		r0 = rf(appId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*protect.ResourceProtectionDto)
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

type mockConstructorTestingTNewResourceProtectionRepository interface {
	mock.TestingT
	Cleanup(func())
}

// NewResourceProtectionRepository creates a new instance of ResourceProtectionRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewResourceProtectionRepository(t mockConstructorTestingTNewResourceProtectionRepository) *ResourceProtectionRepository {
	mock := &ResourceProtectionRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
