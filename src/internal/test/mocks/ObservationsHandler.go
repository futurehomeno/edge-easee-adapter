// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	"github.com/stretchr/testify/mock"

	signalr "github.com/futurehomeno/edge-easee-adapter/internal/model"
)

// ObservationsHandler is an autogenerated mock type for the ObservationsHandler type
type ObservationsHandler struct {
	mock.Mock
}

// HandleObservation provides a mock function with given fields: observation
func (_m *ObservationsHandler) HandleObservation(observation signalr.Observation) error {
	ret := _m.Called(observation)

	var r0 error
	if rf, ok := ret.Get(0).(func(signalr.Observation) error); ok {
		r0 = rf(observation)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewObservationsHandler creates a new instance of ObservationsHandler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewObservationsHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *ObservationsHandler {
	mock := &ObservationsHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
