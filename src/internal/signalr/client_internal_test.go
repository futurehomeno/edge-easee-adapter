package signalr

import (
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	mockedstorage "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/storage"
)

// TestClient_StartDuringCloseIsReArmed pins the recovery for the window Close() must open
// when it drops c.mu before wg.Wait(). A Start() landing there cannot start the goroutine
// itself, so it is remembered and applied on the way out - without that, a login racing the
// auth-loss teardown left every charger unsubscribed until the process restarted.
func TestClient_StartDuringCloseIsReArmed(t *testing.T) {
	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(&config.Config{}).Maybe()
	storage.On("Save").Return(nil).Maybe()

	// A token provider that always fails keeps the re-armed goroutine cheap: it retries
	// behind the backoff instead of reaching the network.
	c := &client{
		cfg:           config.NewService(storage),
		tokenProvider: func() (string, error) { return "", assert.AnError },
		backoff:       backoff.NewStateful(time.Millisecond, time.Millisecond, time.Millisecond, 1, 1),
	}

	t.Cleanup(func() { assert.NoError(t, c.Close()) })

	// Stand in for a live connection goroutine: Close blocks in wg.Wait() until this is
	// released, holding the window open for the Start below.
	c.running = true

	c.wg.Add(1)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() { defer wg.Done(); assert.NoError(t, c.Close()) }()

	require.Eventually(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()

		return c.closing
	}, time.Second, time.Millisecond, "Close never opened its drain window")

	c.Start()

	c.mu.Lock()
	requested := c.startRequested
	running := c.running
	c.mu.Unlock()

	assert.True(t, requested, "a Start during Close must be remembered")
	assert.False(t, running, "a Start during Close must not start the goroutine itself")

	c.wg.Done()
	wg.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	assert.True(t, c.running, "Close must re-arm the client for the Start it deferred")
	assert.False(t, c.startRequested, "the deferred Start must be consumed, not left pending")
}

// TestClient_ConcurrentCloseCancelsDeferredStart pins the ordering between a deferred Start
// and a later Close. A close landing while another is draining reports success, so it must
// also clear the pending start - otherwise the drain re-arms the client on its way out and
// leaves a logout racing a login with the connection still up.
func TestClient_ConcurrentCloseCancelsDeferredStart(t *testing.T) {
	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(&config.Config{}).Maybe()
	storage.On("Save").Return(nil).Maybe()

	c := &client{
		cfg:           config.NewService(storage),
		tokenProvider: func() (string, error) { return "", assert.AnError },
		backoff:       backoff.NewStateful(time.Millisecond, time.Millisecond, time.Millisecond, 1, 1),
	}

	c.running = true

	c.wg.Add(1)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() { defer wg.Done(); assert.NoError(t, c.Close()) }()

	require.Eventually(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()

		return c.closing
	}, time.Second, time.Millisecond, "Close never opened its drain window")

	// A login racing the teardown, then the logout that must win over it.
	c.Start()
	require.NoError(t, c.Close(), "a close during the drain still reports success")

	c.wg.Done()
	wg.Wait()

	c.mu.Lock()
	defer c.mu.Unlock()

	assert.False(t, c.running, "a Close that returned nil must not leave the client running")
	assert.False(t, c.startRequested, "the later Close must cancel the deferred Start")
}
