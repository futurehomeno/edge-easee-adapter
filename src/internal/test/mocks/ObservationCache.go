// Code generated by mockery. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// ObservationCache is an autogenerated mock type for the ObservationCache type
type ObservationCache struct {
	mock.Mock
}

// CableLocked provides a mock function with given fields:
func (_m *ObservationCache) CableLocked() (bool, error) {
	ret := _m.Called()

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func() (bool, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ChargerState provides a mock function with given fields:
func (_m *ObservationCache) ChargerState() (string, error) {
	ret := _m.Called()

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func() (string, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LifetimeEnergy provides a mock function with given fields:
func (_m *ObservationCache) LifetimeEnergy() (float64, error) {
	ret := _m.Called()

	var r0 float64
	var r1 error
	if rf, ok := ret.Get(0).(func() (float64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() float64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(float64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SessionEnergy provides a mock function with given fields:
func (_m *ObservationCache) SessionEnergy() (float64, error) {
	ret := _m.Called()

	var r0 float64
	var r1 error
	if rf, ok := ret.Get(0).(func() (float64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() float64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(float64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TotalPower provides a mock function with given fields:
func (_m *ObservationCache) TotalPower() (float64, error) {
	ret := _m.Called()

	var r0 float64
	var r1 error
	if rf, ok := ret.Get(0).(func() (float64, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() float64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(float64)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// setCableLocked provides a mock function with given fields: locked
func (_m *ObservationCache) setCableLocked(locked bool) {
	_m.Called(locked)
}

// setChargerState provides a mock function with given fields: state
func (_m *ObservationCache) setChargerState(state string) {
	_m.Called(state)
}

// setLifetimeEnergy provides a mock function with given fields: energy
func (_m *ObservationCache) setLifetimeEnergy(energy float64) {
	_m.Called(energy)
}

// setSessionEnergy provides a mock function with given fields: energy
func (_m *ObservationCache) setSessionEnergy(energy float64) {
	_m.Called(energy)
}

// setTotalPower provides a mock function with given fields: power
func (_m *ObservationCache) setTotalPower(power float64) {
	_m.Called(power)
}

type mockConstructorTestingTNewObservationCache interface {
	mock.TestingT
	Cleanup(func())
}

// NewObservationCache creates a new instance of ObservationCache. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewObservationCache(t mockConstructorTestingTNewObservationCache) *ObservationCache {
	mock := &ObservationCache{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
