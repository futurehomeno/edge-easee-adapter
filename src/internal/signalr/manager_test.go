package signalr_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockedsignalr "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/signalr"
)

const testChargerID = "EH000001"

// TestManager_ResubscribesOnReconnect covers the logout/login cycle that left chargers
// permanently without observations: the client is closed and restarted, and the manager
// must subscribe again on the new connection even though it still holds the charger from
// the old one.
func TestManager_ResubscribesOnReconnect(t *testing.T) {
	t.Parallel()

	states := make(chan model.ClientState, 1)
	subscribed := make(chan string, 4)

	client := mockedsignalr.NewClient(t)
	client.On("StateC").Return((<-chan model.ClientState)(states))
	client.On("ObservationC").Return((<-chan model.Observation)(make(chan model.Observation)))
	client.On("Connected").Return(false).Maybe()
	client.On("Start").Maybe()
	client.On("Close").Return(nil).Maybe()
	client.On("SubscribeCharger", testChargerID).
		Run(func(args mock.Arguments) { subscribed <- args.String(0) }).
		Return(nil)

	manager := signalr.NewManager(config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory)), client, nil)

	require.NoError(t, manager.Start())
	t.Cleanup(func() { require.NoError(t, manager.Stop()) })

	manager.Register(testChargerID, onlineHandler{})
	assert.Equal(t, testChargerID, waitForSubscription(t, subscribed))

	states <- model.ClientStateConnected
	assert.Equal(t, testChargerID, waitForSubscription(t, subscribed))
}

func waitForSubscription(t *testing.T, subscribed <-chan string) string {
	t.Helper()

	select {
	case id := <-subscribed:
		return id
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a charger subscription")

		return ""
	}
}

type onlineHandler struct{}

func (onlineHandler) IsOnline() bool { return true }

func (onlineHandler) HandleObservation(model.Observation) error { return nil }
