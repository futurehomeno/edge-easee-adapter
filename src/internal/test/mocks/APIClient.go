// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	api "github.com/futurehomeno/edge-easee-adapter/internal/model"

	"github.com/stretchr/testify/mock"
)

// APIClient is an autogenerated mock type for the APIClient type
type APIClient struct {
	mock.Mock
}

func (_m *APIClient) ChargerDetails(chargerID string) (api.ChargerDetails, error) {
	ret := _m.Called(chargerID)

	var r0 api.ChargerDetails
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (api.ChargerDetails, error)); ok {
		return rf(chargerID)
	}
	if rf, ok := ret.Get(0).(func(string) api.ChargerDetails); ok {
		r0 = rf(chargerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.ChargerDetails)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(chargerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChargerConfig provides a mock function with given fields: chargerID
func (_m *APIClient) ChargerConfig(chargerID string) (*api.ChargerConfig, error) {
	ret := _m.Called(chargerID)

	var r0 *api.ChargerConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*api.ChargerConfig, error)); ok {
		return rf(chargerID)
	}
	if rf, ok := ret.Get(0).(func(string) *api.ChargerConfig); ok {
		r0 = rf(chargerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ChargerConfig)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(chargerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChargerSiteInfo provides a mock function with given fields: chargerID
func (_m *APIClient) ChargerSiteInfo(chargerID string) (*api.ChargerSiteInfo, error) {
	ret := _m.Called(chargerID)

	var r0 *api.ChargerSiteInfo
	var r1 error
	if rf, ok := ret.Get(0).(func(string) (*api.ChargerSiteInfo, error)); ok {
		return rf(chargerID)
	}
	if rf, ok := ret.Get(0).(func(string) *api.ChargerSiteInfo); ok {
		r0 = rf(chargerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ChargerSiteInfo)
		}
	}

	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(chargerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Chargers provides a mock function with given fields:
func (_m *APIClient) Chargers() ([]api.Charger, error) {
	ret := _m.Called()

	var r0 []api.Charger
	var r1 error
	if rf, ok := ret.Get(0).(func() ([]api.Charger, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() []api.Charger); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]api.Charger)
		}
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Ping provides a mock function with given fields:
func (_m *APIClient) Ping() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SetCableAlwaysLocked provides a mock function with given fields: chargerID, locked
func (_m *APIClient) SetCableAlwaysLocked(chargerID string, locked bool) error {
	ret := _m.Called(chargerID, locked)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, bool) error); ok {
		r0 = rf(chargerID, locked)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StartCharging provides a mock function with given fields: chargerID
func (_m *APIClient) StartCharging(chargerID string) error {
	ret := _m.Called(chargerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(chargerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StopCharging provides a mock function with given fields: chargerID
func (_m *APIClient) StopCharging(chargerID string) error {
	ret := _m.Called(chargerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(chargerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDynamicCurrent provides a mock function with given fields: chargerID, current
func (_m *APIClient) UpdateDynamicCurrent(chargerID string, current float64) error {
	ret := _m.Called(chargerID, current)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, float64) error); ok {
		r0 = rf(chargerID, current)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateMaxCurrent provides a mock function with given fields: chargerID, current
func (_m *APIClient) UpdateMaxCurrent(chargerID string, current float64) error {
	ret := _m.Called(chargerID, current)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, float64) error); ok {
		r0 = rf(chargerID, current)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAPIClient creates a new instance of APIClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAPIClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *APIClient {
	mock := &APIClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
