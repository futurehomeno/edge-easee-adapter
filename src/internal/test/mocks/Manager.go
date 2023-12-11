// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	signalr "github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	mock "github.com/stretchr/testify/mock"
)

// Manager is an autogenerated mock type for the Manager type
type Manager struct {
	mock.Mock
}

// Connected provides a mock function with given fields: chargerID
func (_m *Manager) Connected(chargerID string) bool {
	ret := _m.Called(chargerID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(chargerID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Register provides a mock function with given fields: chargerID, handler
func (_m *Manager) Register(chargerID string, handler signalr.Handler) {
	_m.Called(chargerID, handler)
}

// Start provides a mock function with given fields:
func (_m *Manager) Start() error {
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
func (_m *Manager) Stop() error {
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
func (_m *Manager) Unregister(chargerID string) error {
	ret := _m.Called(chargerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(chargerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewManager creates a new instance of Manager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *Manager {
	mock := &Manager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
