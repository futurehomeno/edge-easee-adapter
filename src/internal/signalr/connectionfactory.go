package signalr

import (
	"context"
	"fmt"
	"net/http"

	"github.com/philippseith/signalr"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

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
