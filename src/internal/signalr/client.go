package signalr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
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
	StateC() <-chan ClientState
	// ObservationC returns a channel that will receive charger observations.
	ObservationC() <-chan Observation
}

type client struct {
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc

	connection   signalr.Client
	cfg          *config.Service
	serverStopFn context.CancelFunc
	connFactory  *connectionFactory
	receiver     *receiver

	states       chan ClientState
	observations chan Observation

	connState ClientState
}

// NewClient creates a new SignalR client.
func NewClient(cfg *config.Service, authTokenProvider func() (string, error)) Client {
	observations := make(chan Observation, 100)

	return &client{
		cfg:          cfg,
		receiver:     newReceiver(observations),
		connFactory:  newConnectionFactory(cfg, authTokenProvider),
		states:       make(chan ClientState, 10),
		observations: observations,
	}
}

func (c *client) SubscribeCharger(id string) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()

		return fmt.Errorf("client is not running")
	}

	c.mu.Unlock()

	return c.invoke("SubscribeWithCurrentState", id, true) // true stands for sending initial batch of data
}

func (c *client) UnsubscribeCharger(id string) error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()

		return fmt.Errorf("client is not running")
	}

	c.mu.Unlock()

	return c.invoke("Unsubscribe", id)
}

func (c *client) Connected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return false
	}

	return c.connState == ClientStateConnected
}

func (c *client) StateC() <-chan ClientState {
	return c.states
}

func (c *client) ObservationC() <-chan Observation {
	return c.observations
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

	c.connection = client
	c.serverStopFn = stopFn

	if c.cancel != nil {
		c.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	c.connection.Start()

	go c.notifyState(ctx)

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

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	c.running = false

	return nil
}

func (c *client) invoke(method string, args ...any) error {
	timer := time.NewTimer(c.cfg.GetSignalRInvokeTimeout())
	defer timer.Stop()

	results := c.connection.Invoke(method, args...)

	select {
	case result := <-results:
		return result.Error
	case <-timer.C:
		return fmt.Errorf("timeout")
	}
}

func (c *client) notifyState(ctx context.Context) {
	ch := make(chan signalr.ClientState, 1)
	cancel := c.connection.ObserveStateChanged(ch)

	for {
		select {
		case <-ctx.Done():
			cancel()

			return
		case newState := <-ch:
			state := ClientStateDisconnected
			if newState == signalr.ClientConnected {
				state = ClientStateConnected
			}

			c.mu.Lock()

			if c.connState == state {
				c.mu.Unlock()

				continue
			}

			c.connState = state

			c.mu.Unlock()

			log.Info("signalR client state: ", state)

			c.states <- state
		}
	}
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
