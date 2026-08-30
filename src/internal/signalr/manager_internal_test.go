package signalr

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	mockedstorage "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/storage"
)

const chargerID = "XX12345"

// A reconnect starts a new subscription-failure streak, so its first failure has to warn
// again instead of being downgraded to debug by the flag the previous connection left set.
func TestReconnectWarnsOnTheFirstSubscribeFailureAgain(t *testing.T) {
	m, hook := newTestManager(t)

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
	m, hook := newTestManager(t)

	require.NoError(t, m.handleSubscription(chargerID))

	m.handleClientState(model.ClientStateDisconnected)

	require.NoError(t, m.handleSubscription(chargerID))

	m.handleClientState(model.ClientStateConnected)
	hook.Reset()

	require.NoError(t, m.handleSubscription(chargerID))

	assert.Equal(t, 1, subscribeWarnings(hook), "a retry firing during the outage must not silence the reconnect")
}

// handleSubscription used to hold the manager write lock across SubscribeCharger, which
// blocks for up to SignalRInvokeTimeout - stalling Connected(), Register() and the observation
// dispatch loop, which runs on the very goroutine making the invoke.
func TestHandleSubscription_DoesNotHoldTheLockDuringInvoke(t *testing.T) {
	client := &blockingSubscribeClient{
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}

	m := newTestManagerWithClient(t, client)

	subscribed := make(chan struct{})

	go func() {
		defer close(subscribed)

		_ = m.handleSubscription(chargerID)
	}()

	<-client.entered

	queried := make(chan struct{})

	go func() {
		defer close(queried)

		m.Connected(chargerID)
	}()

	select {
	case <-queried:
	case <-time.After(5 * time.Second):
		t.Fatal("Connected blocked while a subscribe invoke was in flight")
	}

	close(client.release)
	<-subscribed
}

// Every reconnect enqueues a subscription for every charger, so a charger that keeps failing
// used to accumulate one more self-perpetuating retry chain - and one more blocking invoke
// per cycle - on each reconnect.
func TestHandleSubscription_ArmsAtMostOneRetryPerCharger(t *testing.T) {
	m, _ := newTestManager(t)

	// An open done channel parks the armed retry on its backoff timer, so the goroutine
	// count reflects how many chains were armed rather than how many have already exited.
	m.done = make(chan struct{})
	t.Cleanup(func() { close(m.done) })

	require.NoError(t, m.handleSubscription(chargerID))

	before := runtime.NumGoroutine()

	for range 5 {
		require.NoError(t, m.handleSubscription(chargerID))
	}

	assert.LessOrEqual(t, runtime.NumGoroutine()-before, 1, "further failures must not arm a second retry chain")
}

// The flag has to clear on every exit path of addChargerSubscription. A charger left armed
// can never schedule another retry, which strands it worse than the duplicate chains the
// flag exists to prevent.
func TestArmedRetryIsDisarmedSoItCanArmAgain(t *testing.T) {
	m, _ := newTestManager(t) // done is already closed, so the retry exits at once

	require.NoError(t, m.handleSubscription(chargerID))

	charger := m.chargers[chargerID]

	assert.Eventually(t, func() bool {
		m.mu.RLock()
		defer m.mu.RUnlock()

		return !charger.retryArmed
	}, time.Second, 10*time.Millisecond, "a retry that returns must disarm the charger")
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

func newTestManager(t *testing.T) (*manager, *logtest.Hook) {
	t.Helper()

	hook := logtest.NewLocal(log.StandardLogger())
	t.Cleanup(hook.Reset)

	return newTestManagerWithClient(t, &failingSubscribeClient{}), hook
}

func newTestManagerWithClient(t *testing.T, client Client) *manager {
	t.Helper()

	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(&config.Config{}).Maybe()
	storage.On("Save").Return(nil).Maybe()

	m, ok := NewManager(config.NewService(storage), client, nil).(*manager)
	require.True(t, ok)

	// The retries handleSubscription arms are not under test; an already closed done
	// channel makes them return instead of outliving the test on the backoff timer.
	m.done = make(chan struct{})
	close(m.done)

	m.chargers[chargerID] = &charger{backoff: m.cfg.SignalRBackoffStateful()}

	return m
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

// blockingSubscribeClient parks in SubscribeCharger until released, standing in for an
// invoke running up to SignalRInvokeTimeout.
type blockingSubscribeClient struct {
	failingSubscribeClient

	enterOnce sync.Once
	entered   chan struct{}
	release   chan struct{}
}

func (c *blockingSubscribeClient) SubscribeCharger(string) error {
	c.enterOnce.Do(func() { close(c.entered) })
	<-c.release

	return errors.New("not logged in")
}
