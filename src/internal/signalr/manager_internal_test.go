package signalr

import (
	"errors"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	mockedstorage "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/storage"
)

// A reconnect starts a new subscription-failure streak, so its first failure has to warn
// again instead of being downgraded to debug by the flag the previous connection left set.
func TestReconnectWarnsOnTheFirstSubscribeFailureAgain(t *testing.T) {
	const chargerID = "XX12345"

	m, hook := newTestManager(t, chargerID)

	require.NoError(t, m.handleSubscription(chargerID))
	require.NoError(t, m.handleSubscription(chargerID))

	m.handleClientState(model.ClientStateDisconnected)
	m.handleClientState(model.ClientStateConnected)

	require.NoError(t, m.handleSubscription(chargerID))

	assert.Equal(t, 2, subscribeWarnings(hook), "the first subscribe failure of each connection must warn")
}

// Nothing cancels the retries armed before a disconnect, so one can still fire during the
// outage. That failure must not consume the warning the next connection is entitled to.
func TestStaleRetryDuringOutageDoesNotStealTheReconnectWarning(t *testing.T) {
	const chargerID = "XX12345"

	m, hook := newTestManager(t, chargerID)

	require.NoError(t, m.handleSubscription(chargerID))

	m.handleClientState(model.ClientStateDisconnected)

	require.NoError(t, m.handleSubscription(chargerID))

	m.handleClientState(model.ClientStateConnected)
	hook.Reset()

	require.NoError(t, m.handleSubscription(chargerID))

	assert.Equal(t, 1, subscribeWarnings(hook), "a retry firing during the outage must not silence the reconnect")
}

func subscribeWarnings(hook *logtest.Hook) int {
	warnings := 0

	for _, entry := range hook.AllEntries() {
		if entry.Level == log.WarnLevel && strings.Contains(entry.Message, "Failed to subscribe") {
			warnings++
		}
	}

	return warnings
}

func newTestManager(t *testing.T, chargerID string) (*manager, *logtest.Hook) {
	t.Helper()

	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(&config.Config{}).Maybe()
	storage.On("Save").Return(nil).Maybe()

	m, ok := NewManager(config.NewService(storage), &failingSubscribeClient{}, nil).(*manager)
	require.True(t, ok)

	// The retries handleSubscription arms are not under test; an already closed done
	// channel makes them return instead of outliving the test on the backoff timer.
	m.done = make(chan struct{})
	close(m.done)

	m.chargers[chargerID] = &charger{backoff: m.cfg.SignalRBackoffStateful()}

	hook := logtest.NewLocal(log.StandardLogger())
	t.Cleanup(hook.Reset)

	return m, hook
}

// failingSubscribeClient is a stub rather than a generated mock: the mock package imports
// this one, so a test in package signalr cannot use it.
type failingSubscribeClient struct{}

func (c *failingSubscribeClient) Start()                                 {}
func (c *failingSubscribeClient) Close() error                           { return nil }
func (c *failingSubscribeClient) SubscribeCharger(string) error          { return errors.New("not logged in") }
func (c *failingSubscribeClient) UnsubscribeCharger(string) error        { return nil }
func (c *failingSubscribeClient) Connected() bool                        { return false }
func (c *failingSubscribeClient) StateC() <-chan model.ClientState       { return nil }
func (c *failingSubscribeClient) ObservationC() <-chan model.Observation { return nil }
