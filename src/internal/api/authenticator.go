package api

import (
	"fmt"
	"strings"
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
	AccessToken(useReason string) (string, error)
	// Logout used to remove credentials from the config
	Logout() error
}

type authenticator struct {
	mu                  sync.Mutex
	cfg                 *config.Service
	http                HTTPClient
	notificationManager Notifier
	mqtt                *fimpgo.MqttTransport
	serviceName         string
	backoff             backoff.Stateful

	bcEnsured bool
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

	return a
}

func (a *authenticator) Login(userName, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	creds, err := a.http.Login(userName, password)
	if err != nil {
		return err
	}

	_, err = a.storeCredentials(creds)
	if err != nil {
		log.Errorf("[auth] Credentials store err: %v", err.Error())
	}

	log.Debugf("[auth] User=%s logged in", userName)
	return nil
}

func (a *authenticator) AccessToken(useReason string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.bcEnsured {
		if err := a.ensureBackwardsCompatibility(); err != nil {
			return "", fmt.Errorf("failed to ensure backwards compatibility: %w", err)
		}

		a.bcEnsured = true
	}

	credentials := a.cfg.GetCredentials()
	if credentials.Empty() {
		return "", errors.New("credentials are empty: login first")
	}

	if !credentials.AccessTokenExpired() {
		return credentials.AccessToken, nil
	}

	if credentials.RefreshTokenExpired() {
		errStr := fmt.Sprintf("[auth] Refresh token expired at=%s", credentials.RefreshTokenExpiresAt.Format(time.RFC3339))
		return "", errors.Wrap(a.triggerAppLogout(), errStr)
	}

	newCredentials, err := a.updateCredentials(credentials, useReason, 3, 2*time.Second)
	if err != nil {
		return "", fmt.Errorf("[auth] update credentials err: %w", err)
	}

	return newCredentials.AccessToken, nil
}

func (a *authenticator) Logout() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg.ClearCredentials()
}

func (a *authenticator) triggerAppLogout() error {
	err := a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOffline})
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}

	if err = a.sendAppLogoutMessage(); err != nil {
		return fmt.Errorf("failed to send app logout message: %w", err)
	}

	if err = a.cfg.ClearCredentials(); err != nil {
		return fmt.Errorf("failed to clear credentials: %w", err)
	}

	return errors.New("re-login required")
}

// TODO: Migrate it to use cliffhanger's event manager.
func (a *authenticator) sendAppLogoutMessage() error {
	message := fimpgo.NewNullMessage("cmd.auth.logout", a.serviceName, nil, nil, nil)

	if err := a.mqtt.PublishToTopic(logoutAddress, message); err != nil {
		return fmt.Errorf("failed to publish a message to mqtt: address: %s, message: %v, err: %w", logoutAddress, message, err)
	}

	return nil
}

func (a *authenticator) storeCredentials(credentials *model.Credentials) (config.Credentials, error) {
	a.backoff.Reset()
	ret := config.Credentials{}

	accessTokenExpDate, err := jwt.ExpirationDate(credentials.AccessToken)
	if err != nil {
		return ret, fmt.Errorf("failed to extract expiration date from access token: %w", err)
	}

	refreshTokenExpDate, err := jwt.ExpirationDate(credentials.RefreshToken)
	if err != nil {
		return ret, fmt.Errorf("failed to extract expiration date from refresh token: %w", err)
	}

	ret = config.Credentials{
		AccessToken:           credentials.AccessToken,
		RefreshToken:          credentials.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpDate,
		RefreshTokenExpiresAt: refreshTokenExpDate,
	}

	err = a.cfg.SetCredentials(ret)
	if err != nil {
		return ret, fmt.Errorf("failed to save credentials in storage: %w", err)
	}

	return ret, nil
}

func (a *authenticator) updateCredentials(credentials config.Credentials, reason string, retries int, timeout time.Duration) (*config.Credentials, error) {
	if a.backoff.Should() {
		return nil, errors.New("too many requests: backoff")
	}

	for range retries {
		log.Infof("[auth] Refresh AccessToken reason=%s RefreshToken expires_at=%s (%s)",
			reason, credentials.RefreshTokenExpiresAt.Format(time.RFC3339), -time.Since(credentials.RefreshTokenExpiresAt))
		newCred, err := a.http.RefreshToken(credentials.AccessToken, credentials.RefreshToken)
		if err == nil {
			ret, err := a.storeCredentials(newCred)

			if err != nil {
				log.Error("[auth] Store credentials err: " + err.Error())
			} else {
				log.WithField("expires_at", ret.AccessTokenExpiresAt.Format(time.RFC3339)).Infof("[auth] New AccessToken dur=%s", -time.Since(ret.AccessTokenExpiresAt))
			}

			return &ret, nil
		}

		switch {
		case strings.Contains(err.Error(), "unauthorized"):
			if err := a.triggerAppLogout(); err != nil {
				log.Error("[auth] TriggerAppLogout err: " + err.Error())
			}

			a.backoff.Fail()
			return nil, errors.New("refreshToken expired")

		case strings.Contains(err.Error(), "timeout"):
			log.Warn("[auth] AccessToken refresh timeout")
			time.Sleep(timeout)

		default:
			a.backoff.Fail()
			return nil, err
		}
	}

	a.backoff.Fail()
	return nil, errors.New("failed to refresh AccessToken")
}

func (a *authenticator) ensureBackwardsCompatibility() error {
	log.Debug("[auth] ensuring backwards compatibility...")

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
		Info("[auth] ensuring backwards compatibility: updating token expiration times")

	return a.cfg.SetCredentials(config.Credentials{
		AccessToken:           creds.AccessToken,
		RefreshToken:          creds.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
	})
}
