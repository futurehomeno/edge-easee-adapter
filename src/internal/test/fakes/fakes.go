package fakes

import (
	"sync"

	"github.com/futurehomeno/cliffhanger/storage"
)

type fakeConfigStorage[T any] struct {
	mu           sync.RWMutex
	model        T
	modelFactory func() T
}

// NewConfigStorage returns a fake implementation for storage.Storage.
// Not suitable for production use.
func NewConfigStorage[T any](model T, modelFactory func() T) storage.Storage[T] {
	return &fakeConfigStorage[T]{model: model, modelFactory: modelFactory}
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
