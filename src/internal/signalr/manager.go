package signalr

import (
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/root"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Manager is the interface for the Easee signalR manager.
// It manages the signalR connection and the chargers that are connected to it.
type Manager interface {
	root.Service

	// Connected check if SignalR client is connected.
	Connected(chargerID string) bool
	// Register registers a charger to be managed.
	Register(chargerID string, handler Handler)
	// Unregister unregisters a charger from being managed.
	Unregister(chargerID string) error
}

type manager struct {
	mu              sync.RWMutex
	clientStartLock sync.Mutex

	running bool
	done    chan struct{}
	cfg     *config.Service

	subscriptions  chan string
	clientStarting bool

	client   Client
	chargers map[string]*charger
}

func NewManager(cfg *config.Service, client Client) Manager {
	return &manager{
		cfg:      cfg,
		client:   client,
		chargers: make(map[string]*charger),
	}
}

func (m *manager) Connected(chargerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if charger, ok := m.chargers[chargerID]; ok {
		return charger.isSubscribed && charger.handler.IsOnline()
	}

	return false
}

func (m *manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	if m.done != nil {
		close(m.done)
	}

	m.done = make(chan struct{})

	go m.run()

	m.running = true

	return nil
}

func (m *manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.done != nil {
		close(m.done)
	}

	m.running = false

	return nil
}

func (m *manager) Register(chargerID string, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.chargers[chargerID]; ok {
		log.Warnf("Charger '%s' is already registered", chargerID)

		return
	}

	backoff := backoff.NewStateful(m.cfg.GetSignalRInitialBackoff(),
		m.cfg.GetSignalRRepeatedBackoff(),
		m.cfg.GetSignalRFinalBackoff(),
		m.cfg.GetSignalRInitialFailureCount(),
		m.cfg.GetSignalRRepeatedFailureCount())

	m.chargers[chargerID] = &charger{
		handler:      handler,
		isSubscribed: false,
		backoff:      backoff,
	}

	m.ensureClientStarted()

	if m.subscriptions != nil {
		m.subscriptions <- chargerID
	}
}

func (m *manager) Unregister(chargerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.chargers[chargerID]; !ok {
		return nil
	}

	if err := m.client.UnsubscribeCharger(chargerID); err != nil {
		return err
	}

	delete(m.chargers, chargerID)

	if len(m.chargers) == 0 {
		if err := m.client.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) run() {
	states := m.client.StateC()
	observations := m.client.ObservationC()

	for {
		select {
		case <-m.done:
			return

		case chargerID, ok := <-m.subscriptions:
			if !ok {
				continue
			}

			m.handleSubscription(chargerID)

		case state := <-states:
			m.handleClientState(state)

		case observation := <-observations:
			m.handleObservation(observation)
		}
	}
}

func (m *manager) handleSubscription(chargerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	charger, ok := m.chargers[chargerID]
	if !ok {
		return
	}

	if err := m.client.SubscribeCharger(chargerID); err != nil {
		log.Warnf("Failed to subscribe charger '%s'", chargerID)

		if m.subscriptions == nil {
			return
		}

		go m.addChargerSubscription(chargerID, charger)

		return
	}

	charger.backoff.Reset()
	charger.isSubscribed = true

	log.Debugf("signalR: subscribed charger '%s'", chargerID)
}

func (m *manager) addChargerSubscription(chargerID string, charger *charger) {
	timer := time.NewTimer(charger.backoff.Next())
	defer timer.Stop()

	select {
	case <-m.done:
	case <-timer.C:
		m.mu.Lock()
		defer m.mu.Unlock()

		if m.subscriptions != nil {
			m.subscriptions <- chargerID
		}
	}
}

func (m *manager) handleClientState(state ClientState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch state {
	case ClientStateConnected:
		log.Debug("signalR: client connected")

		m.subscriptions = make(chan string, 1+len(m.chargers))

		for chargerID := range m.chargers {
			select {
			case <-m.done:
			case m.subscriptions <- chargerID:
			}
		}

	case ClientStateDisconnected:
		log.Debug("signalR: client disconnected")

		for _, charger := range m.chargers {
			charger.backoff.Reset()
			charger.isSubscribed = false
		}

		if m.subscriptions != nil {
			close(m.subscriptions)
		}

		m.subscriptions = nil

	default:
		log.Warnf("Unknown client state %v", state)
	}
}

func (m *manager) handleObservation(observation Observation) {
	if !observation.ID.Supported() {
		return
	}

	log.Debugf("received observation: %+v", observation)

	m.mu.RLock()
	chargerHandler, ok := m.chargers[observation.ChargerID]
	m.mu.RUnlock()

	if !ok {
		return
	}

	if err := chargerHandler.handler.HandleObservation(observation); err != nil {
		log.
			WithError(err).
			WithField("chargerID", observation.ChargerID).
			WithField("observationID", observation.ID).
			WithField("value", observation.Value).
			Error("failed to handle observation")
	}
}

func (m *manager) ensureClientStarted() {
	if m.client.Connected() {
		return
	}

	m.clientStartLock.Lock()
	if m.clientStarting {
		m.clientStartLock.Unlock()

		return
	}

	log.Trace("signalR: Starting client")

	m.clientStarting = true
	m.clientStartLock.Unlock()

	if len(m.chargers) != 0 {
		m.client.Start()
	}

	m.clientStartLock.Lock()
	defer m.clientStartLock.Unlock()

	m.clientStarting = false
}

type charger struct {
	handler      Handler
	isSubscribed bool
	backoff      backoff.Stateful
}
