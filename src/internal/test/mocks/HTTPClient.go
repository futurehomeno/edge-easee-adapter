// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	api "github.com/futurehomeno/edge-easee-adapter/internal/api"
	mock "github.com/stretchr/testify/mock"
)

// HTTPClient is an autogenerated mock type for the HTTPClient type
type HTTPClient struct {
	mock.Mock
}

// ChargerConfig provides a mock function with given fields: accessToken, chargerID
func (_m *HTTPClient) ChargerConfig(accessToken string, chargerID string) (*api.ChargerConfig, error) {
	ret := _m.Called(accessToken, chargerID)

<<<<<<< HEAD
	var r0 *api.ChargerConfig
=======
	if len(ret) == 0 {
		panic("no return value specified for ChargerConfig")
	}

	var r0 *model.ChargerConfig
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*api.ChargerConfig, error)); ok {
		return rf(accessToken, chargerID)
	}
	if rf, ok := ret.Get(0).(func(string, string) *api.ChargerConfig); ok {
		r0 = rf(accessToken, chargerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ChargerConfig)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accessToken, chargerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChargerDetails provides a mock function with given fields: accessToken, chargerID
func (_m *HTTPClient) ChargerDetails(accessToken string, chargerID string) (api.ChargerDetails, error) {
	ret := _m.Called(accessToken, chargerID)

<<<<<<< HEAD
	var r0 api.ChargerDetails
=======
	if len(ret) == 0 {
		panic("no return value specified for ChargerDetails")
	}

	var r0 model.ChargerDetails
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (api.ChargerDetails, error)); ok {
		return rf(accessToken, chargerID)
	}
	if rf, ok := ret.Get(0).(func(string, string) api.ChargerDetails); ok {
		r0 = rf(accessToken, chargerID)
	} else {
		r0 = ret.Get(0).(api.ChargerDetails)
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accessToken, chargerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

<<<<<<< HEAD
// ChargerSessions provides a mock function with given fields: accessToken, chargerID
func (_m *HTTPClient) ChargerSessions(accessToken string, chargerID string) (api.ChargeSessions, error) {
	ret := _m.Called(accessToken, chargerID)

	var r0 api.ChargeSessions
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (api.ChargeSessions, error)); ok {
		return rf(accessToken, chargerID)
	}
	if rf, ok := ret.Get(0).(func(string, string) api.ChargeSessions); ok {
		r0 = rf(accessToken, chargerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(api.ChargeSessions)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accessToken, chargerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

=======
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
// ChargerSiteInfo provides a mock function with given fields: accessToken, chargerID
func (_m *HTTPClient) ChargerSiteInfo(accessToken string, chargerID string) (*api.ChargerSiteInfo, error) {
	ret := _m.Called(accessToken, chargerID)

<<<<<<< HEAD
	var r0 *api.ChargerSiteInfo
=======
	if len(ret) == 0 {
		panic("no return value specified for ChargerSiteInfo")
	}

	var r0 *model.ChargerSiteInfo
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*api.ChargerSiteInfo, error)); ok {
		return rf(accessToken, chargerID)
	}
	if rf, ok := ret.Get(0).(func(string, string) *api.ChargerSiteInfo); ok {
		r0 = rf(accessToken, chargerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.ChargerSiteInfo)
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
func (_m *HTTPClient) Chargers(accessToken string) ([]api.Charger, error) {
	ret := _m.Called(accessToken)

<<<<<<< HEAD
	var r0 []api.Charger
=======
	if len(ret) == 0 {
		panic("no return value specified for Chargers")
	}

	var r0 []model.Charger
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
	var r1 error
	if rf, ok := ret.Get(0).(func(string) ([]api.Charger, error)); ok {
		return rf(accessToken)
	}
	if rf, ok := ret.Get(0).(func(string) []api.Charger); ok {
		r0 = rf(accessToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]api.Charger)
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
func (_m *HTTPClient) Login(userName string, password string) (*api.Credentials, error) {
	ret := _m.Called(userName, password)

<<<<<<< HEAD
	var r0 *api.Credentials
=======
	if len(ret) == 0 {
		panic("no return value specified for Login")
	}

	var r0 *model.Credentials
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*api.Credentials, error)); ok {
		return rf(userName, password)
	}
	if rf, ok := ret.Get(0).(func(string, string) *api.Credentials); ok {
		r0 = rf(userName, password)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.Credentials)
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

	if len(ret) == 0 {
		panic("no return value specified for Ping")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(accessToken)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RefreshToken provides a mock function with given fields: accessToken, refreshToken
func (_m *HTTPClient) RefreshToken(accessToken string, refreshToken string) (*api.Credentials, error) {
	ret := _m.Called(accessToken, refreshToken)

<<<<<<< HEAD
	var r0 *api.Credentials
=======
	if len(ret) == 0 {
		panic("no return value specified for RefreshToken")
	}

	var r0 *model.Credentials
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
	var r1 error
	if rf, ok := ret.Get(0).(func(string, string) (*api.Credentials, error)); ok {
		return rf(accessToken, refreshToken)
	}
	if rf, ok := ret.Get(0).(func(string, string) *api.Credentials); ok {
		r0 = rf(accessToken, refreshToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*api.Credentials)
		}
	}

	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(accessToken, refreshToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

<<<<<<< HEAD
=======
// SetCableAlwaysLocked provides a mock function with given fields: accessToken, chargerID, locked
func (_m *HTTPClient) SetCableAlwaysLocked(accessToken string, chargerID string, locked bool) error {
	ret := _m.Called(accessToken, chargerID, locked)

	if len(ret) == 0 {
		panic("no return value specified for SetCableAlwaysLocked")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, bool) error); ok {
		r0 = rf(accessToken, chargerID, locked)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
// StopCharging provides a mock function with given fields: accessToken, chargerID
func (_m *HTTPClient) StopCharging(accessToken string, chargerID string) error {
	ret := _m.Called(accessToken, chargerID)

	if len(ret) == 0 {
		panic("no return value specified for StopCharging")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string) error); ok {
		r0 = rf(accessToken, chargerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDynamicCurrent provides a mock function with given fields: accessToken, chargerID, current
func (_m *HTTPClient) UpdateDynamicCurrent(accessToken string, chargerID string, current float64) error {
	ret := _m.Called(accessToken, chargerID, current)

	if len(ret) == 0 {
		panic("no return value specified for UpdateDynamicCurrent")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, float64) error); ok {
		r0 = rf(accessToken, chargerID, current)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateMaxCurrent provides a mock function with given fields: accessToken, chargerID, current
func (_m *HTTPClient) UpdateMaxCurrent(accessToken string, chargerID string, current float64) error {
	ret := _m.Called(accessToken, chargerID, current)

	if len(ret) == 0 {
		panic("no return value specified for UpdateMaxCurrent")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, float64) error); ok {
		r0 = rf(accessToken, chargerID, current)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewHTTPClient creates a new instance of HTTPClient. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewHTTPClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *HTTPClient {
	mock := &HTTPClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
