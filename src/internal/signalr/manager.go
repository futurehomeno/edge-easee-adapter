package signalr

import (
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

// DisconnectionReason is a reason for the disconnection of a charger.
type DisconnectionReason string

const (
	ChargerNotRegistered DisconnectionReason = "charger is not registered in a manager"
	ChargerNotSubscribed DisconnectionReason = "charger is not subscribed for SignalR observations"
	ChargerOffline       DisconnectionReason = "charger is offline"
)

// Manager is the interface for the Easee signalR manager.
// It manages the signalR connection and the chargers that are connected to it.
type Manager interface {
	root.Service

	// Connected check if SignalR client is connected.
	// If the connection is not active, it returns false and a reason for the disconnection.
	Connected(chargerID string) (bool, DisconnectionReason)
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

	delete(m.chargers, chargerID)

	if err := m.client.UnsubscribeCharger(chargerID); err != nil {
		return err
	}

	if len(m.chargers) == 0 {
		if err := m.client.Close(); err != nil {
			return err
		}
	}

	return nil
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
	defer telemetry.RecoverAndEmit(nil, "manager.run", true)

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
	defer m.mu.Unlock()

	charger, ok := m.chargers[chargerID]
	if !ok {
		return fmt.Errorf("unknown charger")
	}

	if err := m.client.SubscribeCharger(chargerID); err != nil {
		log.Warnf("Failed to subscribe charger '%s'", chargerID)

		if m.subscriptions == nil {
			return fmt.Errorf("subscriptions channel closed")
		}

		go m.addChargerSubscription(chargerID, charger)

		return nil
	}

	charger.backoff.Reset()
	charger.isSubscribed = true

	log.Debugf("signalR: subscribed charger '%s'", chargerID)
	return nil
}

func (m *manager) addChargerSubscription(chargerID string, charger *charger) {
	m.mu.Lock()
	timer := time.NewTimer(charger.backoff.Next())
	m.mu.Unlock()

	defer timer.Stop()

	select {
	case <-m.done:
	case <-timer.C:
		m.mu.Lock()
		ok := m.subscriptions != nil
		m.mu.Unlock()

		if ok {
			m.subscriptions <- chargerID
		}
	}
}

func (m *manager) handleClientState(state model.ClientState) {
	switch state {
	case model.ClientStateConnected:
		log.Info("signalR: client connected")

		m.mu.Lock()
		m.subscriptions = make(chan string, 1+len(m.chargers))
		chargersIDs := make([]string, 0, len(m.chargers))

		for chargerID := range m.chargers {
			chargersIDs = append(chargersIDs, chargerID)
		}

		m.mu.Unlock()

		for _, chargerID := range chargersIDs {
			select {
			case <-m.done:
			case m.subscriptions <- chargerID:
			}
		}

	case model.ClientStateDisconnected:
		log.Warn("signalR: client disconnected")

		m.mu.Lock()
		for _, charger := range m.chargers {
			charger.backoff.Reset()
			charger.isSubscribed = false
		}

		if m.subscriptions != nil {
			close(m.subscriptions)
			m.subscriptions = nil
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
		return fmt.Errorf("no handler")
	}

	if err := chargerHandler.handler.HandleObservation(observation); err != nil {
		return fmt.Errorf("obs='%s' err: %w", observation.ID.Str(), err)
	}

	return nil
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
