package signalr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/backoff"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

const (
	signalRURI = "/hubs/chargers"
)

// Client is the interface for the SignalR client.
type Client interface {
	// Start starts the SignalR client.
	Start()
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

	connection    signalr.Client
	cfg           *config.Service
	tokenProvider func() (string, error)
	receiver      *receiver
	backoff       *backoff.Exponential

	states       chan ClientState
	observations chan Observation

	connState ClientState
}

// NewClient creates a new SignalR client.
func NewClient(cfg *config.Service, tokenProvider func() (string, error)) Client {
	observations := make(chan Observation, 100)

	backoff := backoff.NewExponential(cfg.GetSignalRInitialBackoff(),
		cfg.GetSignalRRepeatedBackoff(),
		cfg.GetSignalRFinalBackoff(),
		cfg.GetSignalRInitialFailureCount(),
		cfg.GetSignalRRepeatedFailureCount())

	return &client{
		cfg:           cfg,
		tokenProvider: tokenProvider,
		receiver:      newReceiver(observations),
		backoff:       backoff,
		states:        make(chan ClientState, 10),
		observations:  observations,
	}
}

func (c *client) SubscribeCharger(id string) error {
	return c.invoke("SubscribeWithCurrentState", id, true) // true stands for sending initial batch of data
}

func (c *client) UnsubscribeCharger(id string) error {
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

func (c *client) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	go c.handleConnection(ctx)

	c.running = true
}

func (c *client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		return nil
	}

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	c.backoff.Reset()
	c.running = false

	return nil
}

func (c *client) invoke(method string, args ...any) error {
	c.mu.Lock()
	if !c.running || c.connection == nil {
		c.mu.Unlock()

		return errors.New("client is not running")
	}

	c.mu.Unlock()

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

func (c *client) handleConnection(ctx context.Context) {
	for {
		if client, err := c.getClient(ctx); err != nil {
			log.WithError(err).Warn("Unable to start signalr client")
		} else {
			c.connection = client
			c.connection.Start()

			c.notifyState(ctx)
		}

		c.connection = nil

		select {
		case <-ctx.Done():
			return
		case <-time.After(c.backoff.Next()):
		}
	}
}

func (c *client) notifyState(ctx context.Context) {
	ch := make(chan signalr.ClientState, 1)

	cancel := c.connection.ObserveStateChanged(ch)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			c.updateState(ClientStateDisconnected)

			return

		case clientState := <-ch:
			state := ClientStateDisconnected
			if clientState == signalr.ClientConnected {
				state = ClientStateConnected

				c.backoff.Reset()
			}

			if c.updateState(state) {
				c.states <- state
			}

			if clientState == signalr.ClientClosed {
				return
			}
		}
	}
}

func (c *client) updateState(state ClientState) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connState != state {
		c.connState = state
		log.Info("signalR client state: ", state)

		return true
	}

	return false
}

func (c *client) getClient(ctx context.Context) (signalr.Client, error) {
	connection, err := c.getConnection()
	if err != nil {
		return nil, err
	}

	return signalr.NewClient(
		ctx,
		signalr.KeepAliveInterval(c.cfg.GetSignalRKeepAliveInterval()),
		signalr.TimeoutInterval(c.cfg.GetSignalRTimeoutInterval()),
		signalr.WithConnection(connection),
		signalr.WithReceiver(c.receiver),
		signalr.Logger(newLogger(), false),
	)
}

func (c *client) getConnection() (signalr.Connection, error) {
	token, err := c.tokenProvider()
	if err != nil {
		// Currently we have a bug, when authorization gets broken the signalR library may start
		// calling this method in a forever loop (with -1 timeout) trying to create a connection,
		// when error returned - it is being logged.
		// Implementing a proper start up -> shutdown should be done, but require a bit more thought.
		// This is a hacky solution to avoid spam of logs.
		time.Sleep(time.Minute)

		return nil, fmt.Errorf("unable to get access token (signalR): %w", err)
	}

	headers := func() http.Header {
		h := make(http.Header)
		h.Add("Authorization", "Bearer "+token)

		return h
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.GetSignalRConnCreationTimeout())
	defer cancel()

	url := c.cfg.GetSignalRBaseURL() + signalRURI

	conn, err := signalr.NewHTTPConnection(ctx, url, signalr.WithHTTPHeaders(headers))
	if err != nil {
		// See the comment above for another sleep.
		time.Sleep(30 * time.Second)

		return nil, fmt.Errorf("unable to instantiate signalR connection: %w", err)
	}

	return conn, nil
}
