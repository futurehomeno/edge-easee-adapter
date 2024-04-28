package test

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	libsignalr "github.com/philippseith/signalr"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

var (
	// DefaultSignalRAddr is the default address for the test signalR server
	DefaultSignalRAddr = "localhost:9999"
)

// SignalRServer is a test signalR server.
type SignalRServer struct {
	t *testing.T

	signalr libsignalr.Server
	http    *http.Server
	router  *http.ServeMux
	hub     *signalRHub

	running            bool
	mockedObservations []observationBatch
}

// NewSignalRServer creates a new test signalR server.
func NewSignalRServer(t *testing.T, address string) *SignalRServer {
	t.Helper()

	hub := newSignalRHub(t)
	router := http.NewServeMux()

	srv, err := libsignalr.NewServer(context.Background(), libsignalr.UseHub(hub))
	require.NoError(t, err)

	srv.MapHTTP(libsignalr.WithHTTPServeMux(router), "/hubs/chargers")

	return &SignalRServer{
		t:       t,
		router:  router,
		hub:     hub,
		signalr: srv,
		http:    &http.Server{Addr: address, Handler: router}, //nolint:gosec
	}
}

func (s *SignalRServer) Start() {
	if s.running {
		return
	}

	log.Infof("signalR test server: starting on addr %s", s.http.Addr)

	go s.scheduleObservations()
	go s.runHTTPServer() //nolint:staticcheck

	s.running = true
}

func (s *SignalRServer) Close() {
	if !s.running {
		return
	}

	log.Infof("signalR test server: stopping")

	err := s.http.Shutdown(context.Background())
	require.NoError(s.t, err)

	s.running = false
}

func (s *SignalRServer) MockObservations(delay time.Duration, o []model.Observation) {
	s.mockedObservations = append(s.mockedObservations, observationBatch{
		delay:        delay,
		observations: o,
	})
}

func (s *SignalRServer) scheduleObservations() {
	for _, batch := range s.mockedObservations {
		time.Sleep(batch.delay)

		s.hub.propagate(batch.observations)
	}
}

func (s *SignalRServer) runHTTPServer() {
	if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		s.t.Fatal("signalR test server: http server error", err) //nolint:staticcheck
	}
}

type signalRHub struct {
	libsignalr.Hub

	t  *testing.T
	mu sync.Mutex

	numSubscriptions int
	observations     []model.Observation
}

func newSignalRHub(t *testing.T) *signalRHub {
	t.Helper()

	return &signalRHub{t: t}
}

func (h *signalRHub) SubscribeWithCurrentState(chargerID string, sendInitialObservations bool) {
	log.Infof("signalR test server: SubscribeWithCurrentState called: chargerID %s, sendInitialObservations %t", chargerID, sendInitialObservations)

	h.mu.Lock()
	defer h.mu.Unlock()

	h.numSubscriptions++

	for _, o := range h.observations {
		h.Clients().Caller().Send("productUpdate", o)
	}
}

// OnConnected is called when the hub is connected
func (h *signalRHub) OnConnected(connID string) {
	log.Infof("signalR test server: new client connected: connID %s", connID)
}

// OnDisconnected is called when the hub is disconnected
func (h *signalRHub) OnDisconnected(connID string) {
	log.Infof("signalR test server: client disconnected: connID %s", connID)
}

func (h *signalRHub) propagate(observations []model.Observation) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.observations = observations

	if h.numSubscriptions == 0 {
		return
	}

	for _, o := range observations {
		h.Clients().All().Send("productUpdate", o)
	}
}

type observationBatch struct {
	delay        time.Duration
	observations []model.Observation
}
