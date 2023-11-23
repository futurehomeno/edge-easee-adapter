package easee

import (
	"sync"

	"github.com/futurehomeno/cliffhanger/root"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

// SignalRManager is the interface for the Easee signalR manager.
// It manages the signalR connection and the chargers that are connected to it.
type SignalRManager interface {
	root.Service

	// Connected check if SignalR client is connected.
	Connected() bool
	// Register registers a charger to be managed.
	Register(chargerID string, handler ObservationsHandler) error
	// Unregister unregisters a charger from being managed.
	Unregister(chargerID string) error
}

type signalRManager struct {
	mu      sync.RWMutex
	running bool
	done    chan struct{}

	client   signalr.Client
	chargers map[string]ObservationsHandler
}

func NewSignalRManager(client signalr.Client) SignalRManager {
	return &signalRManager{
		client:   client,
		chargers: make(map[string]ObservationsHandler),
	}
}

func (m *signalRManager) Connected() bool {
	return m.client.Connected()
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

func (m *signalRManager) Register(chargerID string, handler ObservationsHandler) error {
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
		if err := m.client.SubscribeCharger(chargerID); err != nil {
			return err
		}
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
			if state == signalr.Disconnected {
				continue
			}

			m.mu.RLock()

			for chargerID := range m.chargers {
				if err := m.client.SubscribeCharger(chargerID); err != nil {
					log.WithError(err).Error("failed to subscribe charger: ", chargerID)

					continue
				}
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

func (m *signalRManager) handleObservation(observation signalr.Observation) error {
	if !observation.ID.Supported() {
		return nil
	}

	log.Debugf("received observation: %+v", observation)

	m.mu.RLock()
	chargerHandler, ok := m.chargers[observation.ChargerID]
	m.mu.RUnlock()

	if !ok {
		log.Warn("received observation for an unknown charger: ", observation.ChargerID)

		return nil
	}

	return chargerHandler.HandleObservation(observation)
}
