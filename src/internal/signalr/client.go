package signalr

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

const (
	signalRURI = "/hubs/chargers"
)

// Client is the interface for the SignalR client.
type Client interface {
	Start()
	Close() error

	SubscribeCharger(id string) error
	UnsubscribeCharger(id string) error
	Connected() bool
	StateC() <-chan model.ClientState
	ObservationC() <-chan model.Observation
}

type client struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	running bool
	closing bool
	cancel  context.CancelFunc

	connection    signalr.Client
	cfg           *config.Service
	tokenProvider func() (string, error)
	receiver      *receiver
	backoff       backoff.Stateful
	tel           telemetry.Telemetry

	states       chan model.ClientState
	observations chan model.Observation

	connState model.ClientState
}

func NewClient(cfg *config.Service, tokenProvider func() (string, error), tel telemetry.Telemetry) Client {
	observations := make(chan model.Observation, 100)

	return &client{
		cfg:           cfg,
		tokenProvider: tokenProvider,
		tel:           tel,
		receiver:      newReceiver(observations),
		backoff:       cfg.SignalRBackoffStateful(),
		states:        make(chan model.ClientState, 10),
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

	return c.connState == model.ClientStateConnected
}

func (c *client) StateC() <-chan model.ClientState {
	return c.states
}

func (c *client) ObservationC() <-chan model.Observation {
	return c.observations
}

func (c *client) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// handleConnection takes c.mu on its way out, so Close() cannot hold the lock across
	// wg.Wait(). Starting in that window either panics ("Add called concurrently with Wait")
	// or hands Wait a goroutine holding a fresh, uncancelled context. Register re-arms the
	// client once the close completes.
	if c.running || c.closing {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		c.handleConnection(ctx)
	}()

	c.running = true
}

func (c *client) Close() error {
	c.mu.Lock()

	if !c.running || c.closing {
		c.mu.Unlock()

		return nil
	}

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	c.backoff.Reset()
	c.running = false
	c.closing = true
	c.mu.Unlock()

	c.wg.Wait()

	c.mu.Lock()
	c.closing = false
	c.mu.Unlock()

	return nil
}

func (c *client) invoke(method string, args ...any) error {
	c.mu.Lock()
	conn := c.connection
	running := c.running
	c.mu.Unlock()

	if !running || conn == nil {
		return errors.New("client is not running")
	}

	timer := time.NewTimer(c.cfg.SignalRInvokeTimeout())
	defer timer.Stop()

	results := conn.Invoke(method, args...)

	select {
	case result := <-results:
		return result.Error
	case <-timer.C:
		return errors.New("timeout")
	}
}

func (c *client) handleConnection(ctx context.Context) {
	defer telemetry.RecoverAndEmit(c.tel, "handleConnection", true)

	for {
		if conn, err := c.getClient(ctx); err != nil {
			log.Warnf("Unable to start signalr client err: %v", err)
		} else {
			c.setConnection(conn)
			conn.Start()

			c.notifyState(ctx, conn)

			// Each signalr.NewClient derives a cancellable child of ctx and releases it only
			// in Stop(). Dropping the superseded connection without stopping it leaked one
			// per reconnect, for the lifetime of the process.
			conn.Stop()
		}

		c.setConnection(nil)

		select {
		case <-ctx.Done():
			return
		case <-time.After(c.backoff.Next()):
		}
	}
}

func (c *client) setConnection(conn signalr.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.connection = conn
}

func (c *client) notifyState(ctx context.Context, conn signalr.Client) {
	ch := make(chan signalr.ClientState, 1)

	cancel := conn.ObserveStateChanged(ch)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			// Close() cancels the context, so this is the only notice of a shutdown the
			// manager gets; swallowing it leaves its per-charger subscriptions marked live
			// against a connection that is gone. The send cannot block on a done context,
			// hence the buffered non-blocking form.
			if c.updateState(model.ClientStateDisconnected) {
				select {
				case c.states <- model.ClientStateDisconnected:
				default:
				}
			}

			return

		case clientState := <-ch:
			state := model.ClientStateDisconnected
			if clientState == signalr.ClientConnected {
				state = model.ClientStateConnected

				c.backoff.Reset()
			}

			if c.updateState(state) {
				select {
				case c.states <- state:
				case <-ctx.Done():
				}
			}

			if clientState == signalr.ClientClosed {
				return
			}
		}
	}
}

func (c *client) updateState(state model.ClientState) bool {
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
	connection, err := c.getConnection(ctx)
	if err != nil {
		return nil, err
	}

	return signalr.NewClient(
		ctx,
		signalr.KeepAliveInterval(c.cfg.SignalRKeepAliveInterval()),
		signalr.TimeoutInterval(c.cfg.SignalRTimeoutInterval()),
		signalr.WithConnection(connection),
		signalr.WithReceiver(c.receiver),
		signalr.Logger(newLogger(), false),
	)
}

func sleepCtx(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (c *client) getConnection(ctx context.Context) (signalr.Connection, error) {
	token, err := c.tokenProvider()
	if err != nil {
		// The signalR library retries connection creation in a tight forever loop (-1 timeout)
		// once authorization breaks, logging every failure. Sleeping here throttles that spam.
		sleepCtx(ctx, time.Minute)

		return nil, fmt.Errorf("unable to get access token (signalR): %w", err)
	}

	headers := func() http.Header {
		h := make(http.Header)
		h.Add("Authorization", "Bearer "+token)

		return h
	}

	connCtx, cancel := context.WithTimeout(ctx, c.cfg.SignalRConnCreationTimeout())
	defer cancel()

	url := c.cfg.SignalRBaseURL() + signalRURI

	conn, err := signalr.NewHTTPConnection(connCtx, url, signalr.WithHTTPHeaders(headers))
	if err != nil {
		// See the comment above for another sleep.
		sleepCtx(ctx, 30*time.Second)

		return nil, fmt.Errorf("unable to instantiate signalR connection: %w", err)
	}

	return conn, nil
}
