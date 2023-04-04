// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	easee "github.com/futurehomeno/edge-easee-adapter/internal/easee"
	mock "github.com/stretchr/testify/mock"
)

// HTTPClient is an autogenerated mock type for the HTTPClient type
type HTTPClient struct {
	mock.Mock
}

// ChargerConfig provides a mock function with given fields: accessToken, chargerID
func (_m *HTTPClient) ChargerConfig(accessToken string, chargerID string) (*easee.ChargerConfig, error) {
	ret := _m.Called(accessToken, chargerID)

	var r0 *easee.ChargerConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*easee.ChargerConfig, error)); ok {
		return rf(accessToken, chargerID)
	}
	if rf, ok := ret.Get(0).(func(string, string) *easee.ChargerConfig); ok {
		r0 = rf(accessToken, chargerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*easee.ChargerConfig)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accessToken, chargerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Chargers provides a mock function with given fields: accessToken
func (_m *HTTPClient) Chargers(accessToken string) ([]easee.Charger, error) {
	ret := _m.Called(accessToken)

	var r0 []easee.Charger
	var r1 error
	if rf, ok := ret.Get(0).(func(string) ([]easee.Charger, error)); ok {
		return rf(accessToken)
	}
	if rf, ok := ret.Get(0).(func(string) []easee.Charger); ok {
		r0 = rf(accessToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]easee.Charger)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(accessToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Login provides a mock function with given fields: userName, password
func (_m *HTTPClient) Login(userName string, password string) (*easee.Credentials, error) {
	ret := _m.Called(userName, password)

	var r0 *easee.Credentials
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*easee.Credentials, error)); ok {
		return rf(userName, password)
	}
	if rf, ok := ret.Get(0).(func(string, string) *easee.Credentials); ok {
		r0 = rf(userName, password)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*easee.Credentials)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(userName, password)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Ping provides a mock function with given fields: accessToken
func (_m *HTTPClient) Ping(accessToken string) error {
	ret := _m.Called(accessToken)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(accessToken)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RefreshToken provides a mock function with given fields: accessToken, refreshToken
func (_m *HTTPClient) RefreshToken(accessToken string, refreshToken string) (*easee.Credentials, error) {
	ret := _m.Called(accessToken, refreshToken)

	var r0 *easee.Credentials
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*easee.Credentials, error)); ok {
		return rf(accessToken, refreshToken)
	}
	if rf, ok := ret.Get(0).(func(string, string) *easee.Credentials); ok {
		r0 = rf(accessToken, refreshToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*easee.Credentials)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accessToken, refreshToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetCableLock provides a mock function with given fields: accessToken, chargerID, locked
func (_m *HTTPClient) SetCableLock(accessToken string, chargerID string, locked bool) error {
	ret := _m.Called(accessToken, chargerID, locked)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, bool) error); ok {
		r0 = rf(accessToken, chargerID, locked)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartCharging provides a mock function with given fields: accessToken, chargerID, current
func (_m *HTTPClient) StartCharging(accessToken string, chargerID string, current float64) error {
	ret := _m.Called(accessToken, chargerID, current)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, float64) error); ok {
		r0 = rf(accessToken, chargerID, current)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StopCharging provides a mock function with given fields: accessToken, chargerID
func (_m *HTTPClient) StopCharging(accessToken string, chargerID string) error {
	ret := _m.Called(accessToken, chargerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(accessToken, chargerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewHTTPClient interface {
	mock.TestingT
	Cleanup(func())
}

// NewHTTPClient creates a new instance of HTTPClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewHTTPClient(t mockConstructorTestingTNewHTTPClient) *HTTPClient {
	mock := &HTTPClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}