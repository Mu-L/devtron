// Code generated by mockery v2.42.0. DO NOT EDIT.

package mocks

import (
	read "github.com/devtron-labs/devtron/pkg/build/artifacts/imageTagging/read"
	mock "github.com/stretchr/testify/mock"
)

// ImageTaggingReadService is an autogenerated mock type for the ImageTaggingReadService type
type ImageTaggingReadService struct {
	mock.Mock
}

// GetImageTaggingServiceConfig provides a mock function with given fields:
func (_m *ImageTaggingReadService) GetImageTaggingServiceConfig() *read.ImageTaggingServiceConfig {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for GetImageTaggingServiceConfig")
	}

	var r0 *read.ImageTaggingServiceConfig
	if rf, ok := ret.Get(0).(func() *read.ImageTaggingServiceConfig); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*read.ImageTaggingServiceConfig)
		}
	}

	return r0
}

// GetTagNamesByArtifactId provides a mock function with given fields: artifactId
func (_m *ImageTaggingReadService) GetTagNamesByArtifactId(artifactId int) ([]string, error) {
	ret := _m.Called(artifactId)

	if len(ret) == 0 {
		panic("no return value specified for GetTagNamesByArtifactId")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(int) ([]string, error)); ok {
		return rf(artifactId)
	}
	if rf, ok := ret.Get(0).(func(int) []string); ok {
		r0 = rf(artifactId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(artifactId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUniqueTagsByAppId provides a mock function with given fields: appId
func (_m *ImageTaggingReadService) GetUniqueTagsByAppId(appId int) ([]string, error) {
	ret := _m.Called(appId)

	if len(ret) == 0 {
		panic("no return value specified for GetUniqueTagsByAppId")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(int) ([]string, error)); ok {
		return rf(appId)
	}
	if rf, ok := ret.Get(0).(func(int) []string); ok {
		r0 = rf(appId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(appId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewImageTaggingReadService creates a new instance of ImageTaggingReadService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewImageTaggingReadService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ImageTaggingReadService {
	mock := &ImageTaggingReadService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
