package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/fimpgo"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/jwt"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

const (
	notificationEaseeStatusOffline = "easee_status_offline"

	logoutAddress = "pt:j1/mt:cmd/rt:ad/rn:easee/ad:1"
)

// Notifier is a service responsible for sending push notifications.
type Notifier interface {
	Event(event *notification.Event) error
}

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
	// EnsureBackwardsCompatibility is used to ensure that the application is compatible with the current version of credentials layout.
	// In other words, it populates RefreshTokenExpiresAt if possible.
	EnsureBackwardsCompatibility() error
}

type authenticator struct {
	mu                  sync.Mutex
	cfg                 *config.Service
	http                HTTPClient
	notificationManager Notifier
	mqtt                *fimpgo.MqttTransport
	serviceName         string
	backoff             backoff.Stateful
}

// NewAuthenticator creates a new instance of the Authenticator.
func NewAuthenticator(http HTTPClient, cfgSvc *config.Service, notify Notifier, mqtt *fimpgo.MqttTransport, serviceName string) Authenticator {
	backoffCfg := cfgSvc.GetAuthenticatorBackoffCfg()

	statefulBackoff := backoff.NewStateful(
		backoffCfg.InitialBackoff,
		backoffCfg.RepeatedBackoff,
		backoffCfg.FinalBackoff,
		backoffCfg.InitialFailureCount,
		backoffCfg.RepeatedFailureCount,
	)

	a := &authenticator{
		cfg:                 cfgSvc,
		http:                http,
		notificationManager: notify,
		mqtt:                mqtt,
		serviceName:         serviceName,
		backoff:             statefulBackoff,
	}

	// Lock the mutex to ensure that the authenticator is not used before it's fully initialized.
	// A call to EnsureBackwardsCompatibility releases the lock.
	a.mu.Lock()

	return a
}

func (a *authenticator) EnsureBackwardsCompatibility() error {
	log.Debug("authenticator: ensuring backwards compatibility...")

	// Check the constructor for details.
	if a.mu.TryLock() {
		log.Warnf("authenticator: ensuring backwards compatibility: a lock was in an unlocked state")
	}

	defer a.mu.Unlock()

	creds := a.cfg.GetCredentials()

	if creds.Empty() || !creds.RefreshTokenExpiresAt.IsZero() {
		return nil
	}

	// We're refreshing the field to make sure we have a correct time set there.
	accessTokenExpiresAt, err := jwt.ExpirationDate(creds.AccessToken)
	if err != nil {
		return fmt.Errorf("cant't get access token expiration time: %w", err)
	}

	refreshTokenExpiresAt, err := jwt.ExpirationDate(creds.RefreshToken)
	if err != nil {
		return fmt.Errorf("cant't get refresh token expiration time: %w", err)
	}

	log.WithField("access_token_expires_at", accessTokenExpiresAt.Format(time.RFC3339)).
		WithField("refresh_token_expires_at", refreshTokenExpiresAt.Format(time.RFC3339)).
		Info("authenticator: ensuring backwards compatibility: updating token expiration times")

	return a.cfg.SetCredentials(config.Credentials{
		AccessToken:           creds.AccessToken,
		RefreshToken:          creds.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
	})
}

func (a *authenticator) Login(userName, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	creds, err := a.http.Login(userName, password)
	if err != nil {
		return err
	}

	err = a.updateCredentials(creds)
	if err != nil {
		return err
	}

	a.backoff.Reset()

	return nil
}

func (a *authenticator) AccessToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	credentials := a.cfg.GetCredentials()
	if credentials.Empty() {
		return "", errors.New("credentials are empty: login first")
	}

	if !credentials.AccessTokenExpired() {
		return credentials.AccessToken, nil
	}

	if credentials.RefreshTokenExpired() {
		return "", errors.Wrap(a.triggerAppLogout(credentials), "refresh token expired: re-login required")
	}

	log.WithField("expired_at", credentials.AccessTokenExpiresAt.Format(time.RFC3339)).
		Debug("authenticator: access token expired, refreshing...")

	if a.backoff.Should() {
		return "", errors.New("too many requests: backoff is in use")
	}

	newCredentials, err := a.http.RefreshToken(credentials.AccessToken, credentials.RefreshToken)
	if err != nil {
		return "", a.handleRefreshFailure(err, credentials)
	}

	a.backoff.Reset()

	err = a.updateCredentials(newCredentials)
	if err != nil {
		return "", err
	}

	return newCredentials.AccessToken, nil
}

func (a *authenticator) Logout() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.cfg.ClearCredentials()
}

func (a *authenticator) handleRefreshFailure(err error, credentials config.Credentials) error {
	a.backoff.Fail()

	var httpError HTTPError
	if ok := errors.As(err, &httpError); ok && httpError.StatusCode == http.StatusUnauthorized {
		if err := a.triggerAppLogout(credentials); err != nil {
			return fmt.Errorf("failed to trigger app logout on expired refresh token: %w", err)
		}

		return fmt.Errorf("received unauthorized error: re-login is required: %w", err)
	}

	return fmt.Errorf("failed to refresh the auth token: try again later: %w", err)
}

func (a *authenticator) triggerAppLogout(credentials config.Credentials) error {
	log.WithField("expired_at", credentials.RefreshTokenExpiresAt.Format(time.RFC3339)).
		Info("authenticator: refresh token expired, triggering app logout")

	err := a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOffline})
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	if err = a.sendAppLogoutMessage(); err != nil {
		return fmt.Errorf("failed to send app logout message: %w", err)
	}

	return nil
}

// TODO: Migrate it to use cliffhanger's event manager.
func (a *authenticator) sendAppLogoutMessage() error {
	message := fimpgo.NewNullMessage("cmd.auth.logout", a.serviceName, nil, nil, nil)

	if err := a.mqtt.PublishToTopic(logoutAddress, message); err != nil {
		return fmt.Errorf("failed to publish a message to mqtt: address: %s, message: %v, err: %w", logoutAddress, message, err)
	}

	return nil
}

func (a *authenticator) updateCredentials(credentials *model.Credentials) error {
	accessTokenExpDate, err := jwt.ExpirationDate(credentials.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to extract expiration date from access token: %w", err)
	}

	refreshTokenExpDate, err := jwt.ExpirationDate(credentials.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to extract expiration date from access token: %w", err)
	}

	newCreds := config.Credentials{
		AccessToken:           credentials.AccessToken,
		RefreshToken:          credentials.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpDate,
		RefreshTokenExpiresAt: refreshTokenExpDate,
	}

	err = a.cfg.SetCredentials(newCreds)
	if err != nil {
		return fmt.Errorf("failed to save credentials in storage: %w", err)
	}

	return nil
}
