package signalr

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/root"
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
	root.Service

	// SubscribeCharger subscribes to receive observations for a particular charger (based on it's ID).
	// TODO: consider maybe exposing Invoke and moving all the subscription logic to the manager.
	SubscribeCharger(id string) error
	// UnsubscribeCharger unsubscribes from receiving charger observations.
	UnsubscribeCharger(id string) error
	// Connected returns true if the SignalR client is connected.
	Connected() bool
}

type client struct {
	mu        sync.Mutex
	running   bool
	done      chan struct{}
	connState State

	chargersMu sync.RWMutex
	chargers   map[string]bool // true means charger is subscribed

	c            signalr.Client
	cfg          *config.Service
	eventManager event.Manager
	serverStopFn context.CancelFunc
	connFactory  *connectionFactory
	receiver     *receiver
}

// NewClient creates a new SignalR client.
func NewClient(cfg *config.Service, eventManager event.Manager, authTokenProvider func() (string, error)) Client {
	return &client{
		chargers:     make(map[string]bool),
		cfg:          cfg,
		eventManager: eventManager,
		receiver:     newReceiver(),
		connFactory:  newConnectionFactory(cfg, authTokenProvider),
	}
}

func (c *client) SubscribeCharger(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return fmt.Errorf("client is not running")
	}

	c.chargersMu.Lock()
	defer c.chargersMu.Unlock()

	if _, ok := c.chargers[id]; ok {
		return nil
	}

	c.chargers[id] = false

	go func() {
		done := c.invoke("SubscribeWithCurrentState", id, true) // true stands for sending initial batch of data

		<-done

		c.chargersMu.Lock()
		defer c.chargersMu.Unlock()

		c.chargers[id] = true
	}()

	return nil
}

func (c *client) UnsubscribeCharger(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return fmt.Errorf("client is not running")
	}

	c.chargersMu.Lock()
	defer c.chargersMu.Unlock()

	delete(c.chargers, id)

	c.invoke("Unsubscribe", id)

	return nil
}

func (c *client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return false
	}

	return c.connState == Connected
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

func (c *client) Stop() error {
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

func (c *client) invoke(method string, args ...any) <-chan struct{} {
	done := make(chan struct{}, 1)

	go func() {
		ticker := time.NewTicker(c.cfg.GetSignalRInvokeTimeout())
		defer ticker.Stop()

		boff := c.exponentialBackoff()

		for {
			results := c.c.Invoke(method, args...)

			select {
			case r := <-results:
				if r.Error == nil {
					done <- struct{}{}

					return
				}

				interval := boff.NextBackOff()

				log.WithField("method", method).Warnf("signalR invoke error, retrying in %s...", interval)
				time.Sleep(interval)

				continue
			case <-ticker.C:
				interval := boff.NextBackOff()

				log.WithField("method", method).Warnf("signalR invoke timeout, retrying in %s...", interval)
				time.Sleep(interval)

				continue
			case <-c.done:
				done <- struct{}{}

				return
			}
		}
	}()

	return done
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

			c.eventManager.Publish(event.New("signalr-client-state", state))
		}
	}
}

func (c *client) exponentialBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 20 * time.Second
	b.RandomizationFactor = 0.2
	b.Multiplier = 1.5
	b.MaxInterval = 5 * time.Minute
	b.MaxElapsedTime = 0
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
