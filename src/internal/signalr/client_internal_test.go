package signalr

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/philippseith/signalr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
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

// fakeLibClient mirrors philippseith/signalr v0.11.0: state changes reach only the
// observers registered at the moment they happen, because ObserveStateChanged never
// replays the current state. It models the fast handshake the customer hub hit - the run
// loop reaches ClientConnected before Start returns - so an observer registered after
// Start misses the connection for good.
type fakeLibClient struct {
	signalr.Client

	mu        sync.Mutex
	wg        sync.WaitGroup
	state     signalr.ClientState
	observers []chan signalr.ClientState
	connect   bool
	// sequence, when set, replaces the default Start states and is delivered strictly in order.
	sequence []signalr.ClientState

	// stopped, when set, is closed by Stop so a cancellation can tell whether it came first.
	stopped             chan struct{}
	stopOnce            sync.Once
	cancelledBeforeStop bool
}

func (f *fakeLibClient) Start() {
	if f.sequence != nil {
		// Delivered off Start and one at a time, so the order is fixed and the reader can run.
		go func() {
			for _, state := range f.sequence {
				f.deliver(state)
			}
		}()

		return
	}

	f.setState(signalr.ClientConnecting)

	if !f.connect {
		return
	}

	f.setState(signalr.ClientConnected)
}

func (f *fakeLibClient) Stop() {
	f.wg.Wait()
	f.setState(signalr.ClientClosed)

	if f.stopped != nil {
		f.stopOnce.Do(func() { close(f.stopped) })
	}
}

func (f *fakeLibClient) State() signalr.ClientState {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.state
}

func (f *fakeLibClient) ObserveStateChanged(ch chan signalr.ClientState) context.CancelFunc {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.observers = append(f.observers, ch)

	return func() {
		f.mu.Lock()
		defer f.mu.Unlock()

		if f.stopped != nil {
			select {
			case <-f.stopped:
			default:
				f.cancelledBeforeStop = true
			}
		}

		for i, observer := range f.observers {
			if observer == ch {
				f.observers = append(f.observers[:i], f.observers[i+1:]...)

				break
			}
		}
	}
}

func (f *fakeLibClient) deliver(state signalr.ClientState) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.state = state

	for _, ch := range f.observers {
		timer := time.NewTimer(5 * time.Second)

		select {
		case ch <- state:
		case <-timer.C:
		}

		timer.Stop()
	}
}

func (f *fakeLibClient) setState(state signalr.ClientState) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.state = state

	for _, ch := range f.observers {
		f.wg.Add(1)

		go func(ch chan signalr.ClientState) {
			defer f.wg.Done()

			f.mu.Lock()
			defer f.mu.Unlock()

			for _, current := range f.observers {
				if current == ch {
					// The library blocks here until the reader takes it or the client
					// context ends; the timeout stands in for that context.
					timer := time.NewTimer(5 * time.Second)
					defer timer.Stop()

					select {
					case ch <- state:
					case <-timer.C:
					}
				}
			}
		}(ch)
	}
}

func newTestClient(t *testing.T, timeout time.Duration, newConn func(ctx context.Context) (signalr.Client, error)) *client {
	t.Helper()

	cfg := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
	require.NoError(t, cfg.SetSignalRTimeoutInterval(timeout))
	require.NoError(t, cfg.SetSignalRInitialBackoff(time.Millisecond))
	require.NoError(t, cfg.SetSignalRRepeatedBackoff(time.Millisecond))
	require.NoError(t, cfg.SetSignalRFinalBackoff(time.Millisecond))

	c, ok := NewClient(cfg, func() (string, error) { return "test-token", nil }, nil).(*client)
	require.True(t, ok)

	c.newConn = newConn

	return c
}

// The observer has to be registered before Start, or a handshake that completes within it
// publishes ClientConnected to nobody and the adapter stays blind until the server hangs up.
func TestClient_PublishesConnectedOnFastHandshake(t *testing.T) {
	c := newTestClient(t, time.Minute, func(context.Context) (signalr.Client, error) {
		return &fakeLibClient{connect: true}, nil
	})

	c.Start()
	t.Cleanup(func() { require.NoError(t, c.Close()) })

	assert.Equal(t, model.ClientStateConnected, waitForClientState(t, c.StateC()))
}

// A connection that runs but never reports connected must be abandoned within the timeout
// interval instead of being kept forever.
func TestClient_RedialsWhenConnectedNeverArrives(t *testing.T) {
	dials := make(chan struct{}, 10)

	c := newTestClient(t, 50*time.Millisecond, func(context.Context) (signalr.Client, error) {
		select {
		case dials <- struct{}{}:
		default:
		}

		return &fakeLibClient{}, nil
	})

	c.Start()
	t.Cleanup(func() { require.NoError(t, c.Close()) })

	for i := range 2 {
		select {
		case <-dials:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for dial %d", i+1)
		}
	}
}

// The library's pending state sends hold its mutex until the client context ends, and the
// observer cancellation needs that mutex: cancelling before Stop can deadlock the loop.
func TestClient_StopsConnectionBeforeCancellingObserver(t *testing.T) {
	fake := &fakeLibClient{stopped: make(chan struct{})}

	c := newTestClient(t, 50*time.Millisecond, func(context.Context) (signalr.Client, error) {
		return fake, nil
	})

	c.Start()

	select {
	case <-fake.stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the connection to be stopped")
	}

	require.NoError(t, c.Close())

	fake.mu.Lock()
	defer fake.mu.Unlock()

	assert.False(t, fake.cancelledBeforeStop, "observer was cancelled before the connection was stopped")
}

// A Connecting delivered after Connected is stale, not a disconnect: publishing it would make
// the manager drop every subscription on a live connection with nothing left to restore them.
func TestClient_IgnoresStaleConnectingAfterConnected(t *testing.T) {
	c := newTestClient(t, time.Minute, func(context.Context) (signalr.Client, error) {
		return &fakeLibClient{sequence: []signalr.ClientState{signalr.ClientConnected, signalr.ClientConnecting}}, nil
	})

	c.Start()
	t.Cleanup(func() { require.NoError(t, c.Close()) })

	assert.Equal(t, model.ClientStateConnected, waitForClientState(t, c.StateC()))

	select {
	case state := <-c.StateC():
		t.Fatalf("unexpected state %s after a stale Connecting", state)
	case <-time.After(200 * time.Millisecond):
	}

	assert.True(t, c.Connected())
}

func waitForClientState(t *testing.T, states <-chan model.ClientState) model.ClientState {
	t.Helper()

	select {
	case state := <-states:
		return state
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a client state")

		return model.ClientStateDisconnected
	}
}
