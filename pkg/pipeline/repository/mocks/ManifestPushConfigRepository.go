// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	repository "github.com/devtron-labs/devtron/pkg/pipeline/repository"
	mock "github.com/stretchr/testify/mock"
)

// ManifestPushConfigRepository is an autogenerated mock type for the ManifestPushConfigRepository type
type ManifestPushConfigRepository struct {
	mock.Mock
}

// GetManifestPushConfigByAppIdAndEnvId provides a mock function with given fields: appId, envId
func (_m *ManifestPushConfigRepository) GetManifestPushConfigByAppIdAndEnvId(appId int, envId int) (*repository.ManifestPushConfig, error) {
	ret := _m.Called(appId, envId)

	var r0 *repository.ManifestPushConfig
	if rf, ok := ret.Get(0).(func(int, int) *repository.ManifestPushConfig); ok {
		r0 = rf(appId, envId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*repository.ManifestPushConfig)
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

// SaveConfig provides a mock function with given fields: manifestPushConfig
func (_m *ManifestPushConfigRepository) SaveConfig(manifestPushConfig *repository.ManifestPushConfig) (*repository.ManifestPushConfig, error) {
	ret := _m.Called(manifestPushConfig)

	var r0 *repository.ManifestPushConfig
	if rf, ok := ret.Get(0).(func(*repository.ManifestPushConfig) *repository.ManifestPushConfig); ok {
		r0 = rf(manifestPushConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*repository.ManifestPushConfig)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*repository.ManifestPushConfig) error); ok {
		r1 = rf(manifestPushConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateConfig provides a mock function with given fields: manifestPushConfig
func (_m *ManifestPushConfigRepository) UpdateConfig(manifestPushConfig *repository.ManifestPushConfig) error {
	ret := _m.Called(manifestPushConfig)

	var r0 error
	if rf, ok := ret.Get(0).(func(*repository.ManifestPushConfig) error); ok {
		r0 = rf(manifestPushConfig)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewManifestPushConfigRepository interface {
	mock.TestingT
	Cleanup(func())
}

// NewManifestPushConfigRepository creates a new instance of ManifestPushConfigRepository. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewManifestPushConfigRepository(t mockConstructorTestingTNewManifestPushConfigRepository) *ManifestPushConfigRepository {
	mock := &ManifestPushConfigRepository{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
