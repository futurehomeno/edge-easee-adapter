package signalr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// State represents the state of the SignalR client.
type State int

func (s State) String() string {
	if s == Disconnected {
		return "disconnected"
	}

	return "connected"
}

const (
	Disconnected State = iota
	Connected
)

// Client is the interface for the SignalR client.
type Client interface {
	// Start starts the SignalR client.
	Start() error
	// Close stops the SignalR client.
	Close() error

	// SubscribeCharger subscribes to receive observations for a particular charger (based on it's ID).
	SubscribeCharger(id string) error
	// UnsubscribeCharger unsubscribes from receiving charger observations.
	UnsubscribeCharger(id string) error
	// Connected returns true if the SignalR client is connected.
	Connected() bool
	// StateC returns a channel that will receive state updates.
	StateC() <-chan State
	// ObservationC returns a channel that will receive charger observations.
	ObservationC() <-chan Observation
}

type client struct {
	mu      sync.Mutex
	running bool
	done    chan struct{}

	c            signalr.Client
	cfg          *config.Service
	serverStopFn context.CancelFunc
	connFactory  *connectionFactory
	receiver     *receiver
	stateC       chan State
	connState    State
}

// NewClient creates a new SignalR client.
func NewClient(cfg *config.Service, authTokenProvider func() (string, error)) Client {
	return &client{
		cfg:         cfg,
		receiver:    newReceiver(),
		connFactory: newConnectionFactory(cfg, authTokenProvider),
		stateC:      make(chan State, 10),
	}
}

func (c *client) SubscribeCharger(id string) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()

		return fmt.Errorf("client is not running")
	}

	c.mu.Unlock()

	err := c.invoke("SubscribeWithCurrentState", id, true) // true stands for sending initial batch of data
	if err == nil {
		log.WithField("chargerID", id).Info("successfully subscribed charger for receiving signalR events")
	}

	return err
}

func (c *client) UnsubscribeCharger(id string) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()

		return fmt.Errorf("client is not running")
	}

	c.mu.Unlock()

	err := c.invoke("Unsubscribe", id)
	if err == nil {
		log.WithField("chargerID", id).Info("successfully unsubscribed charger from receiving signalR events")
	}

	return err
}

func (c *client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return false
	}

	return c.connState == Connected
}

func (c *client) StateC() <-chan State {
	return c.stateC
}

func (c *client) ObservationC() <-chan Observation {
	return c.receiver.observationC()
}

func (c *client) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	client, stopFn, err := c.clientFactory(c.connFactory, c.receiver)
	if err != nil {
		return err
	}

	c.c = client
	c.serverStopFn = stopFn

	c.done = make(chan struct{})

	c.c.Start()

	go c.notifyState()

	c.running = true

	return nil
}

func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	c.serverStopFn()
	close(c.done)

	c.running = false

	return nil
}

func (c *client) invoke(method string, args ...any) error {
	timer := time.NewTimer(c.cfg.GetSignalRInvokeTimeout())
	defer timer.Stop()

	bckoff := c.exponentialBackoff()

	for i := 0; i < c.cfg.GetSignalRInvokeRetryCount(); i++ {
		results := c.c.Invoke(method, args...)

		select {
		case result := <-results:
			return result.Error
		case <-timer.C:
			interval := bckoff.NextBackOff()

			log.WithField("method", method).Warnf("signalR invoke timeout, retrying in %s...", interval)
			time.Sleep(interval)

			continue
		}
	}

	return fmt.Errorf("timeout after %d retries", c.cfg.GetSignalRInvokeRetryCount())
}

func (c *client) notifyState() {
	ch := make(chan signalr.ClientState, 1)
	cancel := c.c.ObserveStateChanged(ch)

	for {
		select {
		case <-c.done:
			cancel()

			return
		case newState := <-ch:
			var state State
			if newState == signalr.ClientConnected {
				state = Connected
			}

			c.mu.Lock()

			if c.connState == state {
				c.mu.Unlock()

				continue
			}

			c.connState = state

			c.mu.Unlock()

			log.Info("signalR client state: ", state)

			c.stateC <- state
		}
	}
}

func (c *client) exponentialBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 20 * time.Second
	b.RandomizationFactor = 0.2
	b.Multiplier = 1.5
	b.MaxInterval = 5 * time.Minute
	b.MaxElapsedTime = 15 * time.Minute
	b.Reset()

	return b
}

func (c *client) clientFactory(connFactory *connectionFactory, rec *receiver) (signalr.Client, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())

	client, err := signalr.NewClient(
		ctx,
		signalr.KeepAliveInterval(c.cfg.GetSignalRKeepAliveInterval()),
		signalr.TimeoutInterval(c.cfg.GetSignalRTimeoutInterval()),
		signalr.WithConnector(connFactory.Create),
		signalr.WithReceiver(rec),
		signalr.Logger(newLogger(), false),
	)
	if err != nil {
		cancel()

		return nil, nil, err
	}

	return client, cancel, nil
}

type receiver struct {
	signalr.Receiver

	observations chan Observation
}

func newReceiver() *receiver {
	return &receiver{
		observations: make(chan Observation, 100),
	}
}

func (r *receiver) ProductUpdate(o Observation) {
	r.observations <- o
}

func (r *receiver) CommandResponse(resp any) {
	res, _ := json.MarshalIndent(resp, "", "\t")
	log.Info("command response: ", string(res))
}

func (r *receiver) observationC() <-chan Observation {
	return r.observations
}

const (
	signalRURI = "/hubs/chargers"
)

type connectionFactory struct {
	cfg           *config.Service
	tokenProvider func() (string, error)
}

func newConnectionFactory(cfg *config.Service, tokenProvider func() (string, error)) *connectionFactory {
	return &connectionFactory{
		cfg:           cfg,
		tokenProvider: tokenProvider,
	}
}

func (f *connectionFactory) Create() (signalr.Connection, error) {
	token, err := f.tokenProvider()
	if err != nil {
		return nil, fmt.Errorf("unable to get access token: %w", err)
	}

	headers := func() http.Header {
		h := make(http.Header)
		h.Add("Authorization", "Bearer "+token)

		return h
	}

	ctx, cancel := context.WithTimeout(context.Background(), f.cfg.GetSignalRConnCreationTimeout())
	defer cancel()

	conn, err := signalr.NewHTTPConnection(ctx, f.url(), signalr.WithHTTPHeaders(headers))
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate signalR connection: %w", err)
	}

	return conn, nil
}

func (f *connectionFactory) url() string {
	return f.cfg.GetSignalRBaseURL() + signalRURI
}
