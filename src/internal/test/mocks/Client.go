// Code generated by mockery. DO NOT EDIT.

package mocks

import (
	signalr "github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// Close provides a mock function with no fields
func (_m *Client) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Connected provides a mock function with no fields
func (_m *Client) Connected() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Connected")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

<<<<<<< HEAD
// ObservationC provides a mock function with given fields:
func (_m *Client) ObservationC() <-chan signalr.Observation {
	ret := _m.Called()

	var r0 <-chan signalr.Observation
	if rf, ok := ret.Get(0).(func() <-chan signalr.Observation); ok {
=======
// ObservationC provides a mock function with no fields
func (_m *Client) ObservationC() <-chan model.Observation {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ObservationC")
	}

	var r0 <-chan model.Observation
	if rf, ok := ret.Get(0).(func() <-chan model.Observation); ok {
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan signalr.Observation)
		}
	}

	return r0
}

// Start provides a mock function with no fields
func (_m *Client) Start() {
	_m.Called()
}

<<<<<<< HEAD
// StateC provides a mock function with given fields:
func (_m *Client) StateC() <-chan signalr.ClientState {
	ret := _m.Called()

	var r0 <-chan signalr.ClientState
	if rf, ok := ret.Get(0).(func() <-chan signalr.ClientState); ok {
=======
// StateC provides a mock function with no fields
func (_m *Client) StateC() <-chan model.ClientState {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for StateC")
	}

	var r0 <-chan model.ClientState
	if rf, ok := ret.Get(0).(func() <-chan model.ClientState); ok {
>>>>>>> 0f5e7f3 (DEV-4696 Easee rate limit reached 429 error (#52))
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan signalr.ClientState)
		}
	}

	return r0
}

// SubscribeCharger provides a mock function with given fields: id
func (_m *Client) SubscribeCharger(id string) error {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for SubscribeCharger")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UnsubscribeCharger provides a mock function with given fields: id
func (_m *Client) UnsubscribeCharger(id string) error {
	ret := _m.Called(id)

	if len(ret) == 0 {
		panic("no return value specified for UnsubscribeCharger")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(id)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewClient creates a new instance of Client. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *Client {
	mock := &Client{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
