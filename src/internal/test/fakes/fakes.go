package fakes

import (
	"sync"
	"testing"

	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/cliffhanger/storage"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
)

type fakeConfigStorage[T any] struct {
	mu           sync.RWMutex
	model        T
	modelFactory func() T
}

// NewConfigStorage returns a fake implementation of storage.Storage.
// Not suitable for production use.
func NewConfigStorage[T any](t *testing.T, model T, modelFactory func() T) storage.Storage[T] {
	t.Helper()

	return &fakeConfigStorage[T]{
		model:        model,
		modelFactory: modelFactory,
	}
}

func (f *fakeConfigStorage[T]) Load() error {
	return nil
}

func (f *fakeConfigStorage[T]) Save() error {
	return nil
}

func (f *fakeConfigStorage[T]) Reset() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.model = f.modelFactory()

	return nil
}

func (f *fakeConfigStorage[T]) Model() T {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.model
}

var _ api.Notifier = (*FakeNotifier)(nil)

type FakeNotifier struct {
	events []*notification.Event
}

// NewNotifier returns a fake implementation of api.Notifier.
// Not suitable for production use.
func NewNotifier(t *testing.T) *FakeNotifier {
	t.Helper()

	return &FakeNotifier{}
}

func (f *FakeNotifier) Event(event *notification.Event) error {
	f.events = append(f.events, event)

	return nil
}

func (f *FakeNotifier) ReceivedEventsCount() int {
	return len(f.events)
}

func (f *FakeNotifier) IsEventReceived(eventName string) bool {
	for _, e := range f.events {
		if e.EventName == eventName {
			return true
		}
	}

	return false
}

func (f *FakeNotifier) NoEventsReceived() bool {
	return f.ReceivedEventsCount() == 0
}
