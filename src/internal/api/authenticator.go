package api

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/michalkurzeja/go-clock"
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
}

type authenticator struct {
	lock                sync.Mutex
	cfg                 *config.Service
	http                HTTPClient
	notificationManager Notifier
	mqtt                *fimpgo.MqttTransport
	serviceName         fimptype.ServiceNameT
	backoff             backoff.Stateful
	maxUnauthorizedDur  time.Duration
	// unauthorizedSince records the first 401 from the refresh-token endpoint since the last
	// successful refresh. While non-zero, refresh attempts are gated by `backoff` to avoid
	// hammering the API. Once `time.Since(unauthorizedSince) > maxUnauthorizedDur` the
	// authenticator gives up and triggers an app logout.
	unauthorizedSince time.Time

	bcEnsured bool
}

// NewAuthenticator creates a new instance of the Authenticator.
func NewAuthenticator(http HTTPClient, cfgSvc *config.Service, notify Notifier, mqtt *fimpgo.MqttTransport, serviceName fimptype.ServiceNameT) Authenticator {
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
		maxUnauthorizedDur:  backoffCfg.MaxUnauthorizedDuration,
	}

	return a
}

func (a *authenticator) Login(userName, password string) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	creds, err := a.http.Login(userName, password)
	if err != nil {
		return err
	}

	a.unauthorizedSince = time.Time{}

	_, err = a.storeCredentials(creds)
	if err != nil {
		log.Error("[auth] Store credentials err: " + err.Error())
	}

	return nil
}

func (a *authenticator) AccessToken() (string, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if !a.bcEnsured {
		if err := a.ensureBackwardsCompatibility(); err != nil {
			return "", fmt.Errorf("failed to ensure backwards compatibility: %w", err)
		}

		a.bcEnsured = true
	}

	credentials := a.cfg.GetCredentials()
	if credentials.Empty() {
		return "", ErrNotLoggedIn
	}

	if !credentials.AccessTokenExpired() {
		return credentials.AccessToken, nil
	}

	if credentials.RefreshTokenExpired() {
		if err := a.triggerAppLogout(fmt.Sprintf("refresh token expired locally at %s", credentials.RefreshTokenExpiresAt.Format(time.RFC3339))); err != nil {
			log.Errorf("[auth] TriggerAppLogout err: %v", err)
			// Ensure credentials are cleared even if notification failed
			if clearErr := a.cfg.ClearCredentials(); clearErr != nil {
				log.Errorf("[auth] ClearCredentials fallback err: %v", clearErr)
			}
		}
		return "", errors.New("re-login required")
	}

	newCredentials, err := a.updateCredentials(credentials, 2, 30*time.Second)
	if err != nil {
		return "", fmt.Errorf("[auth] update credentials err: %w", err)
	}

	return newCredentials.AccessToken, nil
}

func (a *authenticator) Logout() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	log.Info("[auth] Logout: clearing credentials (explicit logout request)")

	return a.cfg.ClearCredentials()
}

// triggerAppLogout publishes the offline notification, sends cmd.auth.logout, and clears
// stored credentials. The reason is logged at Info level so any logout path leaves a
// single, searchable trace of WHY the user got logged out.
func (a *authenticator) triggerAppLogout(reason string) error {
	log.Infof("[auth] Triggering app logout: %s", reason)

	err := a.notificationManager.Event(&notification.Event{EventName: notificationEaseeStatusOffline})
	if err != nil {
		return fmt.Errorf("send push notification err: %w", err)
	}

	if err = a.sendAppLogoutMessage(); err != nil {
		return fmt.Errorf("send app logout message err: %w", err)
	}

	if err = a.cfg.ClearCredentials(); err != nil {
		return fmt.Errorf("clear credentials err: %w", err)
	}

	return nil // Success - no error
}

// TODO: Migrate it to use cliffhanger's event manager.
func (a *authenticator) sendAppLogoutMessage() error {
	message := fimpgo.NewNullMessage("cmd.auth.logout", a.serviceName, nil, nil, nil)

	if err := a.mqtt.PublishToTopic(logoutAddress, message); err != nil {
		return fmt.Errorf("publish a message to mqtt addr=%s msg=%v err: %w", logoutAddress, message, err)
	}

	return nil
}

func (a *authenticator) storeCredentials(credentials *model.Credentials) (config.Credentials, error) {
	a.backoff.Reset()
	ret := config.Credentials{}

	accessTokenExpDate, err := jwt.ExpirationDate(credentials.AccessToken)
	if err != nil {
		return ret, fmt.Errorf("extract expiration date from access token err: %w", err)
	}

	refreshTokenExpDate, err := jwt.ExpirationDate(credentials.RefreshToken)
	if err != nil {
		return ret, fmt.Errorf("extract expiration date from refresh token err: %w", err)
	}

	ret = config.Credentials{
		AccessToken:           credentials.AccessToken,
		RefreshToken:          credentials.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpDate,
		RefreshTokenExpiresAt: refreshTokenExpDate,
	}

	err = a.cfg.SetCredentials(ret)
	if err != nil {
		// Storage failure only - credentials are valid, just not persisted
		log.Warnf("[auth] Store credentials err: %v", err)
	}

	return ret, nil
}

func (a *authenticator) updateCredentials(credentials config.Credentials, retries int, timeout time.Duration) (*config.Credentials, error) { //nolint:cyclop
	if a.backoff.Should() {
		return nil, ErrRefreshBackoff
	}

	hours := -time.Since(credentials.RefreshTokenExpiresAt).Hours()
	minutes := -time.Since(credentials.RefreshTokenExpiresAt).Minutes() - 60*hours

	dbgStr := fmt.Sprintf("[auth] Refresh AT - RT expires_at=%s (%.1fh %.1fmin)",
		credentials.RefreshTokenExpiresAt.Format(time.RFC3339), hours, minutes)

	if hours < 22 {
		log.Info(dbgStr)
	} else {
		log.Trace(dbgStr)
	}

	for i := range retries {
		newCred, err := a.http.RefreshToken(credentials.AccessToken, credentials.RefreshToken)
		if err == nil {
			ret, err := a.storeCredentials(newCred)

			if err != nil {
				log.Error("[auth] Store credentials err: " + err.Error())
			} else if hours < 22 {
				log.Infof("[auth] New AT expires_at=%s (%.1fmin)", ret.AccessTokenExpiresAt.Format(time.RFC3339), -time.Since(ret.AccessTokenExpiresAt).Minutes())
			}

			a.unauthorizedSince = time.Time{}

			return &ret, nil
		}

		switch {
		case errors.Is(err, ErrUnauthorized):
			return a.handleUnauthorized(err)

		case errors.Is(err, ErrTimeout), errors.Is(err, ErrServer), errors.Is(err, ErrTransport):
			if i == retries-1 {
				// Last attempt, don't sleep before returning failure
				break
			}

			randomDelay, e := rand.Int(rand.Reader, big.NewInt(10))
			if e != nil {
				randomDelay = big.NewInt(0)
			}
			retryAfter := time.Duration(retries)*timeout + time.Second*time.Duration(randomDelay.Int64())
			log.Warnf("[auth] AT refresh err=%v retry in %ds", err, int(retryAfter.Seconds()))
			time.Sleep(retryAfter)

		default:
			a.backoff.Fail()
			return nil, err
		}
	}

	a.backoff.Fail()
	return nil, errors.New("failed to refresh AccessToken")
}

// handleUnauthorized records a 401 from /refresh_token. A single 401 (and most short-lived bursts)
// MUST NOT clear credentials - Easee has historically returned transient 401s on a still-valid RT.
// We rely on the stateful backoff to space retries, and only trigger the app logout once we have
// been unauthorized for `maxUnauthorizedDur` (default 6h) consecutively.
func (a *authenticator) handleUnauthorized(reqErr error) (*config.Credentials, error) {
	a.backoff.Fail()

	if a.unauthorizedSince.IsZero() {
		a.unauthorizedSince = clock.Now()
	}

	elapsed := clock.Now().Sub(a.unauthorizedSince)

	if elapsed < a.maxUnauthorizedDur {
		log.Warnf("[auth] Refresh token returned 401 - keeping credentials, retry gated by backoff (unauthorized for %s, threshold %s)",
			elapsed.Round(time.Second), a.maxUnauthorizedDur)

		return nil, fmt.Errorf("refresh token rejected (transient 401, retrying): %w", reqErr)
	}

	reason := fmt.Sprintf("refresh token rejected by API for %s (>= %s threshold)", elapsed.Round(time.Second), a.maxUnauthorizedDur)
	log.Errorf("[auth] %s", reason)

	if err := a.triggerAppLogout(reason); err != nil {
		log.Error("[auth] TriggerAppLogout err: " + err.Error())
		// Ensure credentials are cleared even if notification failed
		if clearErr := a.cfg.ClearCredentials(); clearErr != nil {
			log.Error("[auth] ClearCredentials fallback err: " + clearErr.Error())
		}
	}

	a.unauthorizedSince = time.Time{}

	return nil, errors.New("refreshToken expired")
}

func (a *authenticator) ensureBackwardsCompatibility() error {
	log.Debug("[auth] Ensure backwards compatibility")

	creds := a.cfg.GetCredentials()

	if creds.Empty() || !creds.RefreshTokenExpiresAt.IsZero() {
		return nil
	}

	// We're refreshing the field to make sure we have a correct time set there.
	accessTokenExpiresAt, err := jwt.ExpirationDate(creds.AccessToken)
	if err != nil {
		return fmt.Errorf("access token expiration time err: %w", err)
	}

	refreshTokenExpiresAt, err := jwt.ExpirationDate(creds.RefreshToken)
	if err != nil {
		return fmt.Errorf("refresh token expiration time err: %w", err)
	}

	log.WithField("access_token_expires_at", accessTokenExpiresAt.Format(time.RFC3339)).
		WithField("refresh_token_expires_at", refreshTokenExpiresAt.Format(time.RFC3339)).
		Info("[auth] Update token expiration times")

	return a.cfg.SetCredentials(config.Credentials{
		AccessToken:           creds.AccessToken,
		RefreshToken:          creds.RefreshToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt,
		RefreshTokenExpiresAt: refreshTokenExpiresAt,
	})
}
