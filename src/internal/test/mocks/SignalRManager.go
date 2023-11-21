// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	easee "github.com/futurehomeno/edge-easee-adapter/internal/easee"
	mock "github.com/stretchr/testify/mock"

	signalr "github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

// SignalRManager is an autogenerated mock type for the SignalRManager type
type SignalRManager struct {
	mock.Mock
}

// Register provides a mock function with given fields: chargerID, cache, callbacks
func (_m *SignalRManager) Register(chargerID string, cache easee.ObservationCache, callbacks map[signalr.ObservationID]func()) error {
	ret := _m.Called(chargerID, cache, callbacks)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, easee.ObservationCache, map[signalr.ObservationID]func()) error); ok {
		r0 = rf(chargerID, cache, callbacks)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Start provides a mock function with given fields:
func (_m *SignalRManager) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Stop provides a mock function with given fields:
func (_m *SignalRManager) Stop() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Unregister provides a mock function with given fields: chargerID
func (_m *SignalRManager) Unregister(chargerID string) error {
	ret := _m.Called(chargerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(chargerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewSignalRManager creates a new instance of SignalRManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewSignalRManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *SignalRManager {
	mock := &SignalRManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
