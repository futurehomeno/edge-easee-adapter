package signalr_test

import (
	"net"
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

	client := signalr.NewClient(cfg, func() (string, error) { return "test-token", nil })
	client.Start()

	t.Cleanup(func() { require.NoError(t, client.Close()) })

	assert.Equal(t, model.ClientStateConnected, waitForState(t, client.StateC()))

	require.NoError(t, client.Close())
	assert.Equal(t, model.ClientStateDisconnected, waitForState(t, client.StateC()))
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
