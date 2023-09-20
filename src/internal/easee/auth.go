package easee

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

type connectivityStatus int

const (
	statusWorkingProperly    connectivityStatus = iota // Auth working properly
	statusWaitingToReconnect                           // Auth is interrupted, waiting before retry
	statusReconnecting                                 // Auth reconnecting after an interruption
	statusConnectionFailed                             // Auth reconnect attempt if failed, indicating a broken connection
)

const (
	notificationEaseeStatusOffline = "easee_status_offline"
	notificationEaseeStatusOnline  = "easee_status_online"
)

// Authenticator is the interface for the Easee authenticator.
type Authenticator interface {
	// Login logs in to the Easee API and persists credentials in config service.
	Login(userName, password string) error
	// AccessToken is responsible for providing a valid access token for the Easee API.
	// It will automatically refresh the token if it's expired.
	// Returns an error if the application is not logged in.
	AccessToken() (string, error)
	// Logout used to remove credentials from the config
	Logout() error
}

type authenticator struct {
	backoffCfg
	mu                  sync.Mutex
	cfgSvc              *config.Service
	http                HTTPClient
	notificationManager notification.Notification
}

type backoffCfg struct {
	status        connectivityStatus
	lengthSeconds int
	attempts      int
	maxAttempts   int
}

func NewAuthenticator(http HTTPClient, cfgSvc *config.Service, notify notification.Notification) Authenticator {
	return &authenticator{
		backoffCfg: backoffCfg{
			lengthSeconds: cfgSvc.GetBackoffCfg().LengthSeconds,
			maxAttempts:   cfgSvc.GetBackoffCfg().Attempts,
		},
		cfgSvc:              cfgSvc,
		http:                http,
		notificationManager: notify,
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

	err = a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOnline})
	a.validateNotificationPush(err, notificationEaseeStatusOnline)
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
		return "", fmt.Errorf("connection interrupted, waiting for user to re-login. state: %d", a.status)
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

func (a *authenticator) Logout() error {
	return a.cfgSvc.ClearCredentials()
}

// handleFailedRefreshToken sets a correct status when refresh operation has failed.
// relies on mutex protection within a caller.
func (a *authenticator) handleFailedRefreshToken(err error) error {
	// we aren't able to refresh anymore. Requires user re-login
	var httpError HTTPError
	if ok := errors.As(err, &httpError); ok && httpError.Status == http.StatusUnauthorized {
		notifError := a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOffline})
		a.validateNotificationPush(notifError, notificationEaseeStatusOffline)

		a.status = statusConnectionFailed
		if saveErr := a.cfgSvc.ClearCredentials(); saveErr != nil {
			return fmt.Errorf("unauthorized, re-login required; failed to clear credentials. %w , save error: %w", err, saveErr)
		}

		return fmt.Errorf("received unauthorized error, re-login is required. %w", err)
	}

	switch a.status { //nolint
	case statusReconnecting:
		if a.attempts == a.maxAttempts {
			notifError := a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOffline})
			a.validateNotificationPush(notifError, notificationEaseeStatusOffline)
			a.status = statusConnectionFailed

			return fmt.Errorf("failed delayed attempt to refresh token. Re-login required. %w", err)
		}

		go a.hookResetToReconnecting()
		a.status = statusWaitingToReconnect

		return fmt.Errorf("failed delayed attempt to refresh token. Trying again. %w", err)
	case statusWorkingProperly:
		a.status = statusWaitingToReconnect
		go a.hookResetToReconnecting()

		return fmt.Errorf("failed to refresh token. Suspending for %d seconds. %w", a.lengthSeconds, err)
	default:
		return fmt.Errorf("invalid auth status when refreshing token: %d. Error: %w", a.status, err)
	}
}

// hookResetToReconnecting resets status to statusReconnecting after a delay.
// must be called in a separate goroutine.
func (a *authenticator) hookResetToReconnecting() {
	time.Sleep(time.Second*time.Duration(a.lengthSeconds) + time.Millisecond*time.Duration(rand.Intn(500)+500)) //nolint:gosec
	a.mu.Lock()

	defer a.mu.Unlock()

	a.attempts++
	a.status = statusReconnecting
}

func (a *authenticator) validateNotificationPush(err error, notificationName string) {
	if err != nil {
		log.WithError(err).Errorf("Failed to send push notification: %s", notificationName)
	}
}
