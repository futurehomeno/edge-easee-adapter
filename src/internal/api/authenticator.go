package api

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/fimpgo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/jwt"
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
	mu                  sync.Mutex
	cfgSvc              *config.Service
	http                HTTPClient
	notificationManager notification.Notification
	mqtt                *fimpgo.MqttTransport
	serviceName         string
	backoff             backoff.Stateful
}

func NewAuthenticator(http HTTPClient, cfgSvc *config.Service, notify notification.Notification, mqtt *fimpgo.MqttTransport, serviceName string) Authenticator {
	backoff := backoff.NewStateful(
		cfgSvc.GetAPIInitialBackoff(),
		cfgSvc.GetAPIRepeatedBackoff(),
		cfgSvc.GetAPIFinalBackoff(),
		cfgSvc.GetAPIInitialFailureCount(),
		cfgSvc.GetAPIRepeatedFailureCount(),
	)

	return &authenticator{
		cfgSvc:              cfgSvc,
		http:                http,
		notificationManager: notify,
		mqtt:                mqtt,
		serviceName:         serviceName,
		backoff:             backoff,
	}
}

func (a *authenticator) Login(userName, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	creds, err := a.http.Login(userName, password)
	if err != nil {
		return err
	}

	accessTokenExpDate, err := jwt.ExpirationDate(creds.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to extract expiration date from access token: %w", err)
	}

	refreshTokenExpDate, err := jwt.ExpirationDate(creds.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to extract expiration date from access token: %w", err)
	}

	if err = a.cfgSvc.SetCredentials(config.Credentials{
		AccessToken:           creds.AccessToken,
		RefreshToken:          creds.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpDate,
		RefreshTokenExpiresAt: refreshTokenExpDate,
	}); err != nil {
		return fmt.Errorf("failed to save credentials in storage: %w", err)
	}

	a.backoff.Reset()

	return nil
}

func (a *authenticator) AccessToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	credentials := a.cfgSvc.GetCredentials()
	if credentials.Empty() {
		return "", errors.New("credentials are empty - login first")
	}

	if !credentials.AccessTokenExpired() {
		return credentials.AccessToken, nil
	}

	if credentials.RefreshTokenExpired() {
		return "", a.triggerAppLogout()
	}

	if a.backoff.Should() {
		return "", errors.New("too many requests, backoff is in use")
	}

	newCredentials, err := a.http.RefreshToken(credentials.AccessToken, credentials.RefreshToken)
	if err != nil {
		return "", a.handleFailedRefreshToken(err)
	}

	a.backoff.Reset()

	accessTokenExpDate, err := jwt.ExpirationDate(newCredentials.AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to extract expiration date from access token: %w", err)
	}

	refreshTokenExpDate, err := jwt.ExpirationDate(newCredentials.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to extract expiration date from access token: %w", err)
	}

	newCreds := config.Credentials{
		AccessToken:           newCredentials.AccessToken,
		RefreshToken:          newCredentials.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpDate,
		RefreshTokenExpiresAt: refreshTokenExpDate,
	}

	err = a.cfgSvc.SetCredentials(newCreds)
	if err != nil {
		return "", fmt.Errorf("failed to save credentials in storage: %w", err)
	}

	return newCredentials.AccessToken, nil
}

func (a *authenticator) Logout() error {
	return a.cfgSvc.ClearCredentials()
}

// handleFailedRefreshToken sets a correct status when refresh operation has failed.
func (a *authenticator) handleFailedRefreshToken(err error) error {
	a.backoff.Fail()

	var httpError HTTPError
	if ok := errors.As(err, &httpError); ok && httpError.Status == http.StatusUnauthorized {
		if err := a.handleConnectionFailed(err); err != nil {
			return errors.Wrap(err, "failed to handle failed connection on unauthorized")
		}

		return fmt.Errorf("received unauthorized error, re-login is required. %w", err)
	}

	return err
}

func (a *authenticator) handleConnectionFailed(err error) error {
	notifError := a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOffline})
	a.validateNotificationPush(notifError, notificationEaseeStatusOffline)

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
		a.serviceName,
		nil, nil, nil,
	)

	if err := a.mqtt.Publish(address, message); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to publish a message to mqtt. Address: %s, message: %v", logoutAddress, message))
	}

	return nil
}

func (a *authenticator) validateNotificationPush(err error, notificationName string) {
	if err != nil {
		log.WithError(err).Errorf("Failed to send push notification: %s", notificationName)
	}
}
