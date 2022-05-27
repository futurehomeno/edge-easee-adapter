package fakes

import (
	"sync"

	"github.com/futurehomeno/cliffhanger/storage"
)

type fakeConfigStorage struct {
	mu           sync.RWMutex
	model        interface{}
	modelFactory func() interface{}
}

// NewConfigStorage returns a fake implementation for storage.Storage.
// Not suitable for production use.
func NewConfigStorage(model interface{}, modelFactory func() interface{}) storage.Storage {
	return &fakeConfigStorage{model: model, modelFactory: modelFactory}
}

func (f *fakeConfigStorage) Load() error {
	return nil
}

func (f *fakeConfigStorage) Save() error {
	return nil
}

func (f *fakeConfigStorage) Reset() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.model = f.modelFactory()

	return nil
}

func (f *fakeConfigStorage) Model() interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	return f.model
}
