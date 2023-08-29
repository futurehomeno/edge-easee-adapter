package easee

import (
	"sync"

	"github.com/futurehomeno/cliffhanger/root"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

// SignalRManager is the interface for the Easee signalR manager.
// It manages the signalR connection and the chargers that are connected to it.
type SignalRManager interface {
	root.Service

	// Register registers a charger to be managed.
	Register(chargerID string, handler ObservationHandler) error
	// Unregister unregisters a charger from being managed.
	Unregister(chargerID string) error
}

type signalRManager struct {
	mu      sync.RWMutex
	running bool
	done    chan struct{}

	client   signalr.Client
	chargers map[string]ObservationHandler
}

func NewSignalRManager(client signalr.Client) SignalRManager {
	return &signalRManager{
		client:   client,
		chargers: make(map[string]ObservationHandler),
	}
}

func (m *signalRManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	m.done = make(chan struct{})

	go m.run()
	go m.handleObservations()

	m.running = true

	return nil
}

func (m *signalRManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	close(m.done)

	m.running = false

	return nil
}

func (m *signalRManager) Register(chargerID string, handler ObservationHandler) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.chargers[chargerID]; ok {
		return nil
	}

	if len(m.chargers) == 0 {
		if err := m.client.Start(); err != nil {
			return err
		}
	}

	m.chargers[chargerID] = handler

	if m.client.Connected() {
		if ch, err := m.client.SubscribeCharger(chargerID); err != nil {
			cache.setConnected(false)

			return err
		}
		cache.setConnected(true)
	}

	return nil
}

func (m *signalRManager) Unregister(chargerID string) error {
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

func (m *signalRManager) run() {
	ch := m.client.StateC()

	for {
		select {
		case <-m.done:
			return
		case state := <-ch:
			m.mu.RLock()

			if state == signalr.Disconnected {
				for _, item := range m.chargers {
					item.cache.setConnected(false)
				}

				m.mu.RUnlock()

				continue
			}

		chargerLoop:
			for chargerID, charger := range m.chargers {
				if err := m.client.SubscribeCharger(chargerID); err != nil {
					log.WithError(err).Error("failed to subscribe charger: ", chargerID)
					charger.cache.setConnected(false)

					continue chargerLoop
				}
				charger.cache.setConnected(true)
			}

			m.mu.RUnlock()
		}
	}
}

func (m *signalRManager) handleObservations() {
	obsCh := m.client.ObservationC()

	for {
		select {
		case <-m.done:
			return
		case observation := <-obsCh:
			if err := m.handleObservation(observation); err != nil {
				log.
					WithError(err).
					WithField("chargerID", observation.ChargerID).
					WithField("observationID", observation.ID).
					WithField("value", observation.Value).
					Error("failed to handle observation")
			}
		}
	}
}

//nolint:funlen
func (m *signalRManager) handleObservation(observation signalr.Observation) error { //nolint:cyclop
	if !observation.ID.Supported() {
		return nil
	}

	log.Debugf("received observation: %+v", observation)

	m.mu.RLock()
	chargerData, ok := m.chargers[observation.ChargerID]
	m.mu.RUnlock()

	if !ok {
		log.Warn("received observation for an unknown charger: ", observation.ChargerID)

		return nil
	}

	switch observation.ID {
	case signalr.ChargerOPState:
		val, err := observation.IntValue()
		if err != nil {
			return err
		}

		state := ChargerState(val).String()
		chargerData.cache.setChargerState(state)

		m.runCallback(chargerData, observation.ID)
	case signalr.SessionEnergy:
		val, err := observation.Float64Value()
		if err != nil {
			return err
		}

		chargerData.cache.setSessionEnergy(val)

		m.runCallback(chargerData, observation.ID)
	case signalr.CableLocked:
		val, err := observation.BoolValue()
		if err != nil {
			return err
		}

		chargerData.cache.setCableLocked(val)

		m.runCallback(chargerData, observation.ID)
	case signalr.TotalPower:
		val, err := observation.Float64Value()
		if err != nil {
			return err
		}

		chargerData.cache.setTotalPower(val * 1000)

		m.runCallback(chargerData, observation.ID)
	case signalr.LifetimeEnergy:
		val, err := observation.Float64Value()
		if err != nil {
			return err
		}

		chargerData.cache.setLifetimeEnergy(val)

		m.runCallback(chargerData, observation.ID)
	}

	return nil
}

func (m *signalRManager) runCallback(data chargerItem, id signalr.ObservationID) {
	cb, ok := data.callbacks[id]
	if !ok {
		return
	}

	if cb == nil {
		return
	}

	cb()
}

type chargerItem struct {
	cache     ObservationCache
	callbacks map[signalr.ObservationID]func()
}

var ( //nolint:gofumpt
	errNotConnected = errors.New("signalR connection is inactive, cannot determine actual state")
)
