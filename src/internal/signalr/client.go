package signalr

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/philippseith/signalr"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// State represents the state of the SignalR client.
type State int

const (
	Connected State = iota
	Disconnected
)

// Client is the interface for the SignalR client.
type Client interface {
	// Start starts the SignalR client.
	Start()
	// Close stops the SignalR client.
	Close()

	// SubscribeCharger subscribes to receive observations for a particular charger (based on it's ID).
	SubscribeCharger(id string) error
	// UnsubscribeCharger unsubscribes from receiving charger observations.
	UnsubscribeCharger(id string) error
	// Connected returns true if the SignalR client is connected.
	Connected() bool
	// ObserveState returns a channel that will receive state updates.
	ObserveState() <-chan State
}

type client struct {
	mu   sync.Mutex
	done chan struct{}

	c              signalr.Client
	cfg            *config.Service
	serverStopFn   context.CancelFunc
	stateObservers []chan State
}

// NewClient creates a new SignalR client.
func NewClient(cfg *config.Service, receiver any, connFactory func() (signalr.Connection, error)) (Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	c, err := signalr.NewClient(
		ctx,
		signalr.KeepAliveInterval(cfg.GetSignalRKeepAliveInterval()),
		signalr.TimeoutInterval(cfg.GetSignalRTimeoutInterval()),
		signalr.WithConnector(connFactory),
		signalr.WithReceiver(receiver),
		signalr.Logger(NewLogger(), true),
	)
	if err != nil {
		cancel()

		return nil, err
	}

	return &client{
		c:            c,
		cfg:          cfg,
		serverStopFn: cancel,
		done:         make(chan struct{}),
	}, nil
}

func (c *client) SubscribeCharger(id string) error {
	return c.invoke("SubscribeWithCurrentState", id, true) // true stands for sending initial batch of data
}

func (c *client) UnsubscribeCharger(id string) error {
	return c.invoke("Unsubscribe", id)
}

func (c *client) Connected() bool {
	return c.c.State() == signalr.ClientConnected
}

func (c *client) ObserveState() <-chan State {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan State, 1)
	c.stateObservers = append(c.stateObservers, ch)

	return ch
}

func (c *client) Start() {
	c.c.Start()

	go c.notifyStateObservers()
}

func (c *client) Close() {
	c.serverStopFn()
	close(c.done)
}

func (c *client) invoke(method string, args ...any) error {
	timer := time.NewTimer(c.cfg.GetSignalRInvokeTimeout())
	defer timer.Stop()

	results := c.c.Invoke(method, args...)

	select {
	case result := <-results:
		return result.Error
	case <-timer.C:
		return fmt.Errorf("timeout")
	}
}

func (c *client) notifyStateObservers() {
	ch := make(chan signalr.ClientState, 1)
	c.c.ObserveStateChanged(ch)

	for {
		select {
		case <-c.done:
			return
		case newState := <-ch:
			state := Disconnected
			if newState == signalr.ClientConnected {
				state = Connected
			}

			c.mu.Lock()

			for _, observer := range c.stateObservers {
				observer <- state
			}

			c.mu.Unlock()
		}
	}
}
