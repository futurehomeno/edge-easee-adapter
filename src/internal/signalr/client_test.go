package signalr_test

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
)

// TestClient_PublishesDisconnectOnClose guards the notice the manager needs to drop its
// subscriptions: Close() cancels the client context, and that path used to update the
// internal state without ever telling anyone.
func TestClient_PublishesDisconnectOnClose(t *testing.T) {
	address := freeAddress(t)

	server := test.NewSignalRServer(t, address)
	server.Start()

	t.Cleanup(server.Close)

	cfg := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
	require.NoError(t, cfg.SetSignalRBaseURL("http://"+address))

	client := signalr.NewClient(cfg, func() (string, error) { return "test-token", nil }, nil)
	client.Start()

	t.Cleanup(func() { require.NoError(t, client.Close()) })

	assert.Equal(t, model.ClientStateConnected, waitForState(t, client.StateC()))

	require.NoError(t, client.Close())
	assert.Equal(t, model.ClientStateDisconnected, waitForState(t, client.StateC()))
}

// TestClient_CloseIsNotDefeatedByConcurrentStart hammers the window Close() opens when it
// drops c.mu before wg.Wait() - which it must, because the connection goroutine takes the
// lock on its way out. A Start() landing in that window used to either panic with "WaitGroup
// misuse" or hand Wait a goroutine holding a fresh, uncancelled context, hanging Close()
// forever. Run with -race.
func TestClient_CloseIsNotDefeatedByConcurrentStart(t *testing.T) {
	cfg := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
	require.NoError(t, cfg.SetSignalRBaseURL("http://"+freeAddress(t)))

	// A failing token provider keeps handleConnection cheap: it retries behind the backoff,
	// which Close() cuts short by cancelling the context.
	client := signalr.NewClient(cfg, func() (string, error) { return "", assert.AnError }, nil)

	t.Cleanup(func() { require.NoError(t, client.Close()) })

	done := make(chan struct{})

	go func() {
		defer close(done)

		var wg sync.WaitGroup

		for range 16 {
			wg.Add(2)

			go func() { defer wg.Done(); client.Start() }()
			go func() { defer wg.Done(); assert.NoError(t, client.Close()) }()
		}

		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("Start/Close deadlocked")
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "localhost:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}

func waitForState(t *testing.T, states <-chan model.ClientState) model.ClientState {
	t.Helper()

	select {
	case state := <-states:
		return state
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for a client state")

		return model.ClientStateDisconnected
	}
}
