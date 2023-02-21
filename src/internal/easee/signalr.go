package easee

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/futurehomeno/cliffhanger/root"
	libsignalr "github.com/philippseith/signalr"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
)

const (
	signalRURI = "/hubs/chargers"
)

// SignalRManager is the interface for the Easee signalR manager.
// It manages the signalR connection and the chargers that are connected to it.
type SignalRManager interface {
	root.Service

	// Register registers a charger to be managed.
	Register(chargerID string, cache ObservationCache, callbacks map[ObservationID]func())
	// Unregister unregisters a charger from being managed.
	Unregister(chargerID string)
}

type signalRManager struct {
	mu       sync.RWMutex
	client   signalr.Client
	receiver *SignalRReceiver
	chargers map[string]chargerItem
	done     chan struct{}
}

func NewSignalRManager(client signalr.Client, receiver *SignalRReceiver) SignalRManager {
	return &signalRManager{
		client:   client,
		receiver: receiver,
		chargers: make(map[string]chargerItem),
		done:     make(chan struct{}),
	}
}

func (m *signalRManager) Start() error {
	go m.run()
	go m.handleObservations()

	return nil
}

func (m *signalRManager) Stop() error {
	close(m.done)
	m.client.Close()

	return nil
}

func (m *signalRManager) Register(chargerID string, cache ObservationCache, callbacks map[ObservationID]func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.chargers[chargerID]; ok {
		return
	}

	m.chargers[chargerID] = chargerItem{
		cache:     cache,
		callbacks: callbacks,
	}

	if m.client.Connected() {
		if err := m.client.SubscribeCharger(chargerID); err != nil {
			log.WithError(err).Error("failed to subscribe charger: ", chargerID)

			return
		}

		cache.setConnected(true)
	}
}

func (m *signalRManager) Unregister(chargerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.chargers[chargerID]; !ok {
		return
	}

	delete(m.chargers, chargerID)

	if err := m.client.UnsubscribeCharger(chargerID); err != nil {
		log.WithError(err).Error("failed to unsubscribe charger: ", chargerID)
	}
}

func (m *signalRManager) run() {
	ch := m.client.ObserveState()

	m.client.Start()

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

			for chargerID, charger := range m.chargers {
				charger.cache.setConnected(true)

				if err := m.client.SubscribeCharger(chargerID); err != nil {
					log.WithError(err).Error("failed to subscribe charger: ", chargerID)
					charger.cache.setConnected(false)
				}
			}

			m.mu.RUnlock()
		}
	}
}

func (m *signalRManager) handleObservations() {
	for {
		select {
		case <-m.done:
			return
		case observation := <-m.receiver.observationsCh():
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
func (m *signalRManager) handleObservation(observation Observation) error { //nolint:cyclop
	if !observation.ID.Supported() {
		return nil
	}

	m.mu.RLock()
	chargerData, ok := m.chargers[observation.ChargerID]
	m.mu.RUnlock()

	if !ok {
		log.Warn("received observation for an unknown charger: ", observation.ChargerID)

		return nil
	}

	switch observation.ID {
	case ChargerOPState:
		val, err := observation.IntValue()
		if err != nil {
			return err
		}

		state := ChargerState(val).String()
		chargerData.cache.setChargerState(state)

		m.runCallback(chargerData, observation.ID)
	case SessionEnergy:
		val, err := observation.Float64Value()
		if err != nil {
			return err
		}

		chargerData.cache.setSessionEnergy(val)

		m.runCallback(chargerData, observation.ID)
	case CableLocked:
		val, err := observation.BoolValue()
		if err != nil {
			return err
		}

		chargerData.cache.setCableLocked(val)

		m.runCallback(chargerData, observation.ID)
	case TotalPower:
		val, err := observation.Float64Value()
		if err != nil {
			return err
		}

		chargerData.cache.setTotalPower(val * 1000)

		m.runCallback(chargerData, observation.ID)
	case LifetimeEnergy:
		val, err := observation.Float64Value()
		if err != nil {
			return err
		}

		chargerData.cache.setLifetimeEnergy(val)

		m.runCallback(chargerData, observation.ID)
	}

	return nil
}

func (m *signalRManager) runCallback(data chargerItem, id ObservationID) {
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
	callbacks map[ObservationID]func()
}

type SignalRReceiver struct {
	libsignalr.Receiver

	observations chan Observation
}

func NewSignalRReceiver() *SignalRReceiver {
	return &SignalRReceiver{
		observations: make(chan Observation, 100),
	}
}

func (r *SignalRReceiver) ProductUpdate(o Observation) {
	log.Infof("product update: data: %+v\n", o)

	r.observations <- o
}

func (r *SignalRReceiver) observationsCh() <-chan Observation {
	return r.observations
}

var ( //nolint:gofumpt
	errNotConnected = errors.New("signalR connection is inactive, cannot determine actual state")
)

// ObservationCache is a cache for charger observations.
type ObservationCache interface {
	// ChargerState returns the charger state.
	ChargerState() (string, error)
	// SessionEnergy returns the session energy.
	SessionEnergy() (float64, error)
	// CableLocked returns the cable locked state.
	CableLocked() (bool, error)
	// TotalPower returns the total power.
	TotalPower() (float64, error)
	// LifetimeEnergy returns the lifetime energy.
	LifetimeEnergy() (float64, error)

	setChargerState(state string)
	setSessionEnergy(energy float64)
	setCableLocked(locked bool)
	setTotalPower(power float64)
	setLifetimeEnergy(energy float64)

	isConnected() bool
	setConnected(connected bool)
}

type cache struct {
	mu sync.RWMutex

	connected bool

	chargerState   string
	cableLocked    bool
	sessionEnergy  float64
	totalPower     float64
	lifetimeEnergy float64
}

func NewObservationCache() ObservationCache {
	return &cache{}
}

func (c *cache) ChargerState() (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return "", errNotConnected
	}

	return c.chargerState, nil
}

func (c *cache) SessionEnergy() (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return 0, errNotConnected
	}

	return c.sessionEnergy, nil
}

func (c *cache) CableLocked() (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return false, errNotConnected
	}

	return c.cableLocked, nil
}

func (c *cache) TotalPower() (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return 0, errNotConnected
	}

	return c.totalPower, nil
}

func (c *cache) LifetimeEnergy() (float64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return 0, errNotConnected
	}

	return c.lifetimeEnergy, nil
}

func (c *cache) setSessionEnergy(energy float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sessionEnergy = energy
}

func (c *cache) setCableLocked(locked bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cableLocked = locked
}

func (c *cache) setTotalPower(power float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalPower = power
}

func (c *cache) setLifetimeEnergy(energy float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lifetimeEnergy = energy
}

func (c *cache) setChargerState(state string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.chargerState = state
}

func (c *cache) isConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.connected
}

func (c *cache) setConnected(connected bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connected = connected
}

type SignalRConnectionFactory struct {
	auth   Authenticator
	cfgSvc *config.Service
}

func NewSignalRConnectionFactory(auth Authenticator, cfgSvc *config.Service) *SignalRConnectionFactory {
	return &SignalRConnectionFactory{auth: auth, cfgSvc: cfgSvc}
}

func (f *SignalRConnectionFactory) Create() (libsignalr.Connection, error) {
	token, err := f.auth.AccessToken()
	if err != nil {
		return nil, fmt.Errorf("unable to get access token: %w", err)
	}

	headers := func() http.Header {
		h := make(http.Header)
		h.Add("Authorization", "Bearer "+token)

		return h
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.cfgSvc.GetSignalRConnCreationTimeout())
	defer cancel()

	conn, err := libsignalr.NewHTTPConnection(
		ctx,
		f.cfgSvc.GetEaseeBaseURL()+signalRURI,
		libsignalr.WithHTTPHeaders(headers),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate signalR connection: %w", err)
	}

	return conn, nil
}
