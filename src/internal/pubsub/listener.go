package pubsub

import "github.com/futurehomeno/cliffhanger/event"

// Domain is the domain of device manager events.
const Domain = "offered_current_manager"

// RefreshOfferedCurrentEvent is an event that is published when offered should be refreshed.
type RefreshOfferedCurrentEvent struct {
	Value int64
}

// NewOfferedCurrentRefreshEvent creates a new offered current refresh event.
func NewOfferedCurrentRefreshEvent(value int64) *event.Event {
	return &event.Event{
		Domain:  Domain,
		Payload: &RefreshOfferedCurrentEvent{Value: value},
	}
}

// NewOfferedCurrentListener creates a new device change listener.
func NewOfferedCurrentListener(eventManager event.Manager, value int64, processor event.Processor) event.Listener {
	return event.NewListener(
		processor,
		eventManager,
		"offered_current",
		10,
		WaitForOfferedCurrentRefresh(value),
	)
}

func WaitForOfferedCurrentRefresh(value int64) event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		event.WaitForPayload(&RefreshOfferedCurrentEvent{Value: value}),
	)
}
