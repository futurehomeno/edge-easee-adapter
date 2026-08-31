package signalr

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/root"
	"github.com/futurehomeno/cliffhanger/telemetry"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

type DisconnectionReason string

const (
	ChargerNotRegistered DisconnectionReason = "charger is not registered in a manager"
	ChargerNotSubscribed DisconnectionReason = "charger is not subscribed for SignalR observations"
	ChargerOffline       DisconnectionReason = "charger is offline"
)

// Manager owns the single signalR connection and the chargers subscribed on it.
type Manager interface {
	root.Service

	// Connected reports whether the charger is reachable, and why not when it is not.
	Connected(chargerID string) (bool, DisconnectionReason)
	Register(chargerID string, handler Handler)
	Unregister(chargerID string) error
}

type manager struct {
	mu sync.RWMutex

	running bool
	done    chan struct{}
	cfg     *config.Service

	subscriptions chan string

	client   Client
	tel      telemetry.Telemetry
	chargers map[string]*charger
}

func NewManager(cfg *config.Service, client Client, tel telemetry.Telemetry) Manager {
	return &manager{
		cfg:           cfg,
		client:        client,
		tel:           tel,
		chargers:      make(map[string]*charger),
		subscriptions: make(chan string, 32),
	}
}

func (m *manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.done = make(chan struct{})

	go m.run()

	m.running = true

	return nil
}

func (m *manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()

		return nil
	}

	m.running = false
	done := m.done
	m.mu.Unlock()

	err := m.client.Close()

	close(done)

	return err
}

func (m *manager) Register(chargerID string, handler Handler) {
	m.mu.Lock()

	if _, ok := m.chargers[chargerID]; ok {
		m.mu.Unlock()
		log.Warnf("Charger '%s' is already registered", chargerID)

		return
	}

	m.chargers[chargerID] = &charger{
		handler:      handler,
		isSubscribed: false,
		backoff:      m.cfg.SignalRBackoffStateful(),
	}

	m.client.Start()

	connected := m.client.Connected()
	m.mu.Unlock()

	// The charger is already in m.chargers, so a connect still in flight will pick it up
	// via the handleClientState sweep once it lands. Enqueuing here too would just add a
	// guaranteed "client is not running" failure while the handshake is still running.
	if connected {
		m.enqueueSubscription(chargerID)
	}
}

// enqueueSubscription hands a charger ID to the run loop for (re)subscription.
// The send never happens under m.mu and always has a cancellation escape via done.
func (m *manager) enqueueSubscription(chargerID string) {
	m.mu.RLock()
	done := m.done
	m.mu.RUnlock()

	select {
	case m.subscriptions <- chargerID:
	case <-done:
	}
}

func (m *manager) Unregister(chargerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.chargers[chargerID]; !ok {
		return nil
	}

	delete(m.chargers, chargerID)

	var errs error

	if err := m.client.UnsubscribeCharger(chargerID); err != nil {
		errs = errors.Join(errs, err)
	}

	if len(m.chargers) == 0 {
		if err := m.client.Close(); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (m *manager) Connected(chargerID string) (bool, DisconnectionReason) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	charger, ok := m.chargers[chargerID]
	if !ok {
		return false, ChargerNotRegistered
	}

	if !charger.isSubscribed {
		return false, ChargerNotSubscribed
	}

	if !charger.handler.IsOnline() {
		return false, ChargerOffline
	}

	return true, ""
}

func (m *manager) run() {
	defer telemetry.RecoverAndEmit(m.tel, "manager.run", true)

	m.mu.RLock()
	done := m.done
	m.mu.RUnlock()

	states := m.client.StateC()
	observations := m.client.ObservationC()

	for {
		select {
		case <-done:
			return

		case chargerID := <-m.subscriptions:
			if err := m.handleSubscription(chargerID); err != nil {
				log.Warnf("Handle subscription chargerID=%s err: %v", chargerID, err)
			}

		case state := <-states:
			m.handleClientState(state)

		case observation := <-observations:
			if err := m.handleObservation(observation); err != nil {
				log.Warnf("Handle observation chargerID=%s err: %v", observation.ChargerID, err)
			}
		}
	}
}

func (m *manager) handleSubscription(chargerID string) error {
	m.mu.Lock()

	charger, ok := m.chargers[chargerID]
	if !ok {
		m.mu.Unlock()

		return fmt.Errorf("unknown charger")
	}

	// A retry armed before a disconnect can fire after the reconnect sweep already
	// re-subscribed this charger; don't subscribe twice on the same connection.
	if charger.isSubscribed {
		m.mu.Unlock()

		return nil
	}

	m.mu.Unlock()

	// Invoked unlocked: SubscribeCharger blocks for up to SignalRInvokeTimeout, and this runs
	// on the run loop, the only drainer of the observation channel. Holding m.mu across it
	// stalled Connected(), Register() and every observation lookup for the whole invocation.
	err := m.client.SubscribeCharger(chargerID)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Unregister can drop this charger - and a later Register re-add a different struct under
	// the same ID - while the invoke ran unlocked, so identity is the test, not presence.
	if current, ok := m.chargers[chargerID]; !ok || current != charger {
		// A Subscribe that wins this race establishes a cloud subscription after
		// Unregister's own Unsubscribe already ran, orphaning it with no local charger
		// left to receive its observations until the next reconnect. Compensated whatever
		// the invoke reported: a local timeout does not cancel the server-side operation,
		// so an error is no proof the subscription was not established. Unsubscribing one
		// that never existed is a no-op, which is the cheaper way to be wrong.
		if unsubErr := m.client.UnsubscribeCharger(chargerID); unsubErr != nil {
			log.Warnf("signalR: cleanup after unregister race chargerID=%s err: %v", chargerID, unsubErr)
		}

		return nil
	}

	if err != nil {
		// A sustained outage (e.g. not logged in) retries forever; warn on the first
		// failure of a streak only, so it does not accumulate thousands of lines.
		if charger.subscribeFailed {
			log.Debugf("Failed to subscribe charger '%s'", chargerID)
		} else {
			log.Warnf("Failed to subscribe charger '%s': %v", chargerID, err)
			charger.subscribeFailed = true
		}

		// One retry chain per charger. Every reconnect enqueues a subscription for every
		// charger, so without this a charger that keeps failing accumulated one more
		// self-perpetuating chain - and one more blocking invoke per cycle - per reconnect.
		if !charger.retryArmed {
			charger.retryArmed = true

			go m.addChargerSubscription(chargerID, charger)
		}

		return nil
	}

	charger.backoff.Reset()
	charger.isSubscribed = true
	charger.subscribeFailed = false

	log.Debugf("signalR: subscribed charger '%s'", chargerID)
	return nil
}

func (m *manager) addChargerSubscription(chargerID string, charger *charger) {
	m.mu.Lock()
	timer := time.NewTimer(charger.backoff.Next())
	done := m.done
	m.mu.Unlock()

	defer timer.Stop()

	// Disarmed on every exit path - a charger left armed can never schedule another retry,
	// which strands it worse than the duplicate chains the flag exists to prevent - but never
	// via defer: enqueueSubscription returns as soon as the run loop takes the ID, and that
	// same loop may already have armed the next chain by then. A deferred disarm would clear
	// that fresh flag, so the failure after it would arm a duplicate.
	select {
	case <-done:
		m.disarmRetry(charger)
	case <-timer.C:
		m.disarmRetry(charger)
		m.enqueueSubscription(chargerID)
	}
}

func (m *manager) disarmRetry(charger *charger) {
	m.mu.Lock()
	defer m.mu.Unlock()

	charger.retryArmed = false
}

func (m *manager) handleClientState(state model.ClientState) {
	switch state {
	case model.ClientStateConnected:
		log.Info("signalR: client connected")

		m.mu.Lock()
		chargersIDs := make([]string, 0, len(m.chargers))

		for chargerID, charger := range m.chargers {
			// A subscription belongs to the connection that made it. Clearing the flag here
			// rather than trusting the disconnect notice makes this sweep the authority: a
			// missed notice would otherwise leave the charger permanently unsubscribed.
			charger.isSubscribed = false
			chargersIDs = append(chargersIDs, chargerID)
			// A new connection starts a new failure streak, so its first failure has to warn
			// again - otherwise the only visible sign of a charger that never comes back is a
			// debug line the hub does not log. Cleared here rather than on disconnect: a retry
			// armed before the disconnect would otherwise fail during the outage and take the
			// warning with it.
			charger.subscribeFailed = false
		}

		m.mu.Unlock()

		// Offload the re-subscription sends: this runs on the run() goroutine, which is
		// also the only drainer of m.subscriptions, so sending inline could self-block.
		go func() {
			for _, chargerID := range chargersIDs {
				m.enqueueSubscription(chargerID)
			}
		}()

	case model.ClientStateDisconnected:
		log.Warn("signalR: client disconnected")

		m.mu.Lock()
		for _, charger := range m.chargers {
			charger.backoff.Reset()
			charger.isSubscribed = false
		}

		m.mu.Unlock()

	default:
		log.Warnf("Unknown client state %v", state)
	}
}

func (m *manager) handleObservation(observation model.Observation) error {
	if !observation.ID.Supported() {
		return nil
	}

	m.mu.RLock()
	chargerHandler, ok := m.chargers[observation.ChargerID]
	m.mu.RUnlock()

	if !ok {
		return errors.New("no handler")
	}

	if err := chargerHandler.handler.HandleObservation(observation); err != nil {
		return fmt.Errorf("obs='%s' err: %w", observation.ID.Str(), err)
	}

	return nil
}

type charger struct {
	handler         Handler
	isSubscribed    bool
	subscribeFailed bool
	retryArmed      bool
	backoff         backoff.Stateful
}
