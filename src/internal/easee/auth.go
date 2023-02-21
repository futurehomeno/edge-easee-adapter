package easee

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Authenticator is the interface for the Easee authenticator.
type Authenticator interface {
	// Login logs in to the Easee API and persists credentials in config service.
	Login(userName, password string) error
	// AccessToken is responsible for providing a valid access token for the Easee API.
	// It will automatically refresh the token if it's expired.
	// Returns an error if the application is not logged in.
	AccessToken() (string, error)
}

type authenticator struct {
	mu     sync.Mutex
	cfgSvc *config.Service
	http   HTTPClient
}

func NewAuthenticator(http HTTPClient, cfgSvc *config.Service) Authenticator {
	return &authenticator{
		cfgSvc: cfgSvc,
		http:   http,
	}
}

func (a *authenticator) Login(userName, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	creds, err := a.http.Login(userName, password)
	if err != nil {
		return err
	}

	if err = a.cfgSvc.SetCredentials(creds.AccessToken, creds.RefreshToken, creds.ExpiresIn); err != nil {
		return fmt.Errorf("failed to save credentials in storage: %w", err)
	}

	return nil
}

func (a *authenticator) AccessToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	credentials := a.cfgSvc.GetCredentials()
	if credentials.Empty() {
		return "", errors.New("credentials are empty - login first")
	}

	if !credentials.Expired() {
		return credentials.AccessToken, nil
	}

	newCredentials, err := a.http.RefreshToken(credentials.AccessToken, credentials.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to fetch new credentials: %w", err)
	}

	err = a.cfgSvc.SetCredentials(newCredentials.AccessToken, newCredentials.RefreshToken, newCredentials.ExpiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to save credentials in storage: %w", err)
	}

	return newCredentials.AccessToken, nil
}
