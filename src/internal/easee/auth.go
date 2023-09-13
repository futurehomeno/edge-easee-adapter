package easee

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

type connectivityStatus int

const (
	statusWorkingProperly    connectivityStatus = iota // Auth working properly
	statusWaitingToReconnect                           // Auth is interrupted, waiting before retry
	statusReconnecting                                 // Auth reconnecting after an interruption
	statusConnectionFailed                             // Auth reconnect attempt if failed, indicating a broken connection
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
	backoffCfg
	mu     sync.Mutex
	cfgSvc *config.Service
	http   HTTPClient
}

type backoffCfg struct {
	status        connectivityStatus
	lengthSeconds int
	attempts      int
	maxAttempts   int
}

func NewAuthenticator(http HTTPClient, cfgSvc *config.Service) Authenticator {
	return &authenticator{
		backoffCfg: backoffCfg{
			lengthSeconds: cfgSvc.GetBackoffCfg().LengthSeconds,
			maxAttempts:   cfgSvc.GetBackoffCfg().Attempts,
		},
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

	a.status = statusWorkingProperly
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

	if a.status == statusWaitingToReconnect || a.status == statusConnectionFailed {
		return "", errors.New("connection interrupted, waiting for user to re-login")
	}

	newCredentials, err := a.http.RefreshToken(credentials.AccessToken, credentials.RefreshToken)
	if err != nil {
		return "", a.handleFailedRefreshToken(err)
	}

	// reset backoff stats
	a.status = statusWorkingProperly
	a.attempts = 0

	err = a.cfgSvc.SetCredentials(newCredentials.AccessToken, newCredentials.RefreshToken, newCredentials.ExpiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to save credentials in storage: %w", err)
	}

	return newCredentials.AccessToken, nil
}

// handleFailedRefreshToken sets a correct status when refresh operation has failed.
// relies on mutex protection within a caller
func (a *authenticator) handleFailedRefreshToken(err error) error {
	if httpErr, ok := err.(HttpError); ok && httpErr.Status == 401 {
		// TODO notify user
		a.status = statusConnectionFailed
		a.cfgSvc.ClearCredentials()
		return fmt.Errorf("received unauthorized error, re-login is required. %w", err)
	}

	switch a.status {
	case statusReconnecting:
		if a.attempts == a.maxAttempts {
			// TODO notify user
			a.status = statusConnectionFailed
			return fmt.Errorf("failed delayed attempt to refresh token. Re-login required. %w", err)
		}

		go a.hookResetToReconnecting()
		a.status = statusWaitingToReconnect
		return fmt.Errorf("failed delayed attempt to refresh token. Trying again. %w", err)
	case statusWorkingProperly:
		a.status = statusWaitingToReconnect
		go a.hookResetToReconnecting()
		return fmt.Errorf("failed to refresh token. Suspending for 5 minutes. %w", err)
	default:
		return fmt.Errorf("invalid auth status when refreshing token: %d. Error: %w", a.status, err)
	}
}

func (a *authenticator) hookResetToReconnecting() {
	time.Sleep(time.Second * time.Duration(a.lengthSeconds))
	a.mu.Lock()
	defer a.mu.Unlock()
	a.attempts += 1
	a.status = statusReconnecting
}
