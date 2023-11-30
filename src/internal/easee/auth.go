package easee

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/fimpgo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

type connectionStatus int

const (
	statusWorkingProperly    connectionStatus = iota // Auth working properly
	statusWaitingToReconnect                         // Auth is interrupted, waiting before retry
	statusReconnecting                               // Auth reconnecting after an interruption
	statusConnectionFailed                           // Auth reconnect attempt if failed, indicating a broken connection
)

const (
	notificationEaseeStatusOffline = "easee_status_offline"

	logoutAddress = "pt:j1/mt:cmd/rt:ad/rn:easee/ad:1"
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
	mqtt                *fimpgo.MqttTransport
}

type backoffCfg struct {
	status   connectionStatus
	attempts int
}

func NewAuthenticator(http HTTPClient, cfgSvc *config.Service, notify notification.Notification, mqtt *fimpgo.MqttTransport) Authenticator {
	return &authenticator{
		cfgSvc:              cfgSvc,
		http:                http,
		notificationManager: notify,
		mqtt:                mqtt,
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
		if err := a.handleConnectionFailed(err); err != nil {
			return errors.Wrap(err, "failed to handle failed connection on unauthorized")
		}

		return fmt.Errorf("received unauthorized error, re-login is required. %w", err)
	}

	switch a.status { //nolint
	case statusReconnecting:
		if a.attempts == a.cfgSvc.GetBackoffMaxAttempts() {
			if err := a.handleConnectionFailed(err); err != nil {
				return errors.Wrap(err, "failed to handle failed connection when reconnecting")
			}

			return fmt.Errorf("failed delayed attempt to refresh token. Re-login required. %w", err)
		}

		go a.hookResetToReconnecting()
		a.status = statusWaitingToReconnect

		return fmt.Errorf("failed delayed attempt to refresh token. Trying again. %w", err)
	case statusWorkingProperly:
		a.status = statusWaitingToReconnect
		go a.hookResetToReconnecting()

		return fmt.Errorf("failed to refresh token. Suspending for %v seconds. %w", a.cfgSvc.GetBackoffLength(), err)
	default:
		return fmt.Errorf("invalid auth status when refreshing token: %d. Error: %w", a.status, err)
	}
}

func (a *authenticator) handleConnectionFailed(err error) error {
	notifError := a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOffline})
	a.validateNotificationPush(notifError, notificationEaseeStatusOffline)

	a.status = statusConnectionFailed
	if logoutErr := a.triggerAppLogout(); logoutErr != nil {
		return fmt.Errorf("unauthorized, re-login required; failed to clear credentials. %w , logout error: %w", err, logoutErr)
	}

	return nil
}

// triggerAppLogout sends a mqtt message with request to log out a user
// as we don't have a way of internal communication and invoking of the cliffhanger code
// sending an external message is the only way to achieve that without duplicating logic across different places.
func (a *authenticator) triggerAppLogout() error {
	address, err := fimpgo.NewAddressFromString(logoutAddress)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to create address from %s", logoutAddress))
	}

	message := fimpgo.NewNullMessage(
		"cmd.auth.logout",
		ServiceName,
		nil, nil, nil,
	)

	if err := a.mqtt.Publish(address, message); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to publish a message to mqtt. Address: %s, message: %v", logoutAddress, message))
	}

	return nil
}

// hookResetToReconnecting resets status to statusReconnecting after a delay.
// must be called in a separate goroutine.
func (a *authenticator) hookResetToReconnecting() {
	time.Sleep(a.cfgSvc.GetBackoffLength() + time.Millisecond*time.Duration(rand.Intn(500)+500)) //nolint:gosec
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
