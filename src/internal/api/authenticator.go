package api

import (
	"errors"
	"time"

	"github.com/futurehomeno/cliffhanger/auth"
	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

const (
	notificationEaseeStatusOffline = "easee_status_offline"

	logoutAddress = "pt:j1/mt:cmd/rt:ad/rn:easee/ad:1"

	// cliffhanger returns a plain error while the backoff suppresses a refresh attempt.
	// Comparing the message is the only way to keep the caller's log downgrade.
	backoffSuspendedMessage = "token refresh suspended by backoff"
)

// Notifier is a service responsible for sending push notifications.
type Notifier interface {
	Event(event *notification.Event) error
}

// CredentialsStore persists the Easee tokens.
type CredentialsStore interface {
	Credentials() config.Credentials
	SetCredentials(config.Credentials) error
	RefreshCredentials(config.Credentials, string) error
	ClearCredentials() error
}

// Authenticator is the interface for the Easee authenticator.
type Authenticator interface {
	// Login logs in to the Easee API and persists credentials in the credentials store.
	Login(userName, password string) error
	// AccessToken is responsible for providing a valid access token for the Easee API.
	// It will automatically refresh the token if it's expired.
	// Returns an error if the application is not logged in.
	AccessToken() (string, error)
	// Logout used to remove the stored credentials.
	Logout() error
}

type authenticator struct {
	authenticator *auth.Authenticator
	http          HTTPClient
	creds         CredentialsStore
	backoff       backoff.Stateful
}

// NewAuthenticator creates a new instance of the Authenticator.
func NewAuthenticator(
	http HTTPClient,
	creds CredentialsStore,
	cfgSvc *config.Service,
	notify Notifier,
	mqtt *fimpgo.MqttTransport,
	serviceName fimptype.ServiceNameT,
) Authenticator {
	a := &authenticator{
		http:    http,
		creds:   creds,
		backoff: cfgSvc.AuthenticatorBackoffStateful(),
	}

	adapter := &credentialsAdapter{store: creds}

	a.authenticator = auth.NewAuthenticator(
		adapter,
		&tokenExchanger{http: http, snapshot: adapter},
		auth.AuthenticatorConfig{
			Backoff: a.backoff,
			// Easee has historically returned transient 401s on a still-valid refresh token,
			// so a rejection streak has to outlive the grace before concluding auth loss.
			UnauthorizedGrace: cfgSvc.AuthenticatorMaxUnauthorized(),
			OnAuthLoss:        authLossHandler(notify, mqtt, serviceName),
		},
	)

	return a
}

func (a *authenticator) Login(userName, password string) error {
	creds, err := a.http.Login(userName, password)
	if err != nil {
		return err
	}

	// The store keeps the new credentials in memory when the write fails, so the session is
	// usable until the next restart. Failing the login would mark the app not configured and
	// skip the charger setup over a disk error the next successful save repairs.
	if err = a.creds.SetCredentials(credentialsFromResponse(creds)); err != nil {
		log.Warnf("[auth] Store credentials err: %v", err)
	}

	a.backoff.Reset()

	return nil
}

func (a *authenticator) AccessToken() (string, error) {
	token, err := a.authenticator.AccessToken()
	if err == nil {
		return token, nil
	}

	// Both mean "not now, still trying": report them as backoff so the caller logs them at
	// debug instead of warning on every request for the whole grace window.
	if err.Error() == backoffSuspendedMessage || errors.Is(err, auth.ErrRefreshDeferred) {
		return "", ErrRefreshBackoff
	}

	return token, err
}

func (a *authenticator) Logout() error {
	log.Info("[auth] Clear credentials on explicit logout")

	return a.creds.ClearCredentials()
}

// authLossHandler notifies the user and asks the app to log out. It runs under the
// authenticator lock, so it must only publish - the credentials are already cleared.
func authLossHandler(notify Notifier, mqtt *fimpgo.MqttTransport, serviceName fimptype.ServiceNameT) func(string) {
	return func(reason string) {
		log.Infof("[auth] Trigger app logout: %s", reason)

		if err := notify.Event(&notification.Event{EventName: notificationEaseeStatusOffline}); err != nil {
			log.Errorf("[auth] Send push notification err: %v", err)
		}

		message := fimpgo.NewNullMessage("cmd.auth.logout", serviceName, nil, nil, nil)

		if err := mqtt.PublishToTopic(logoutAddress, message); err != nil {
			log.Errorf("[auth] Publish logout message to addr=%s err: %v", logoutAddress, err)
		}
	}
}

// credentialsAdapter translates between the persisted Easee credentials and the framework model.
type credentialsAdapter struct {
	store CredentialsStore
	// refreshing is the session handed to the framework for the exchange in progress. The
	// framework reads the credentials and writes them back under a single lock, so one copy
	// is enough to tell a refresh that still owns the session from one that does not.
	refreshing config.Credentials
	// rotated is the last session the framework handed back. It keeps a rotation the store
	// refused available for pairing: the framework exchanges such a rotation rather than what
	// the store holds, and the two access tokens differ.
	rotated config.Credentials
}

// accessTokenFor returns the access token issued alongside refreshToken. Easee rejects a pair
// drawn from two sessions, so the token cannot simply be read off the store.
func (c *credentialsAdapter) accessTokenFor(refreshToken string) string {
	if c.rotated.RefreshToken == refreshToken {
		return c.rotated.AccessToken
	}

	return c.refreshing.AccessToken
}

func (c *credentialsAdapter) Credentials() auth.Credentials {
	creds := c.store.Credentials()
	c.refreshing = creds

	return auth.Credentials{
		AccessToken:      creds.AccessToken,
		RefreshToken:     creds.RefreshToken,
		ExpiresAt:        creds.AccessTokenExpiresAt,
		RefreshExpiresAt: creds.RefreshTokenExpiresAt,
	}
}

// SetCredentials derives both expiry times from the JWTs themselves: Easee rotates the refresh
// token on every exchange and its response carries no refresh expiry, so without this the local
// "refresh token expired" check would go blind after the first refresh.
// It writes through RefreshCredentials with the refresh token the exchange started from, so a
// refresh landing after a logout or a new login cannot bring back the session it belonged to.
func (c *credentialsAdapter) SetCredentials(creds auth.Credentials) error {
	rotated := config.Credentials{
		AccessToken:           creds.AccessToken,
		RefreshToken:          creds.RefreshToken,
		AccessTokenExpiresAt:  tokenExpiration(creds.AccessToken, creds.ExpiresAt),
		RefreshTokenExpiresAt: tokenExpiration(creds.RefreshToken, creds.RefreshExpiresAt),
	}

	c.rotated = rotated

	return c.store.RefreshCredentials(rotated, c.refreshing.RefreshToken)
}

func (c *credentialsAdapter) ClearCredentials() error {
	return c.store.ClearCredentials()
}

// tokenExpiration reads the expiry claim of a JWT, falling back to the provided value.
func tokenExpiration(token string, fallback time.Time) time.Time {
	expiration, err := auth.TokenExpirationDate(token)
	if err != nil {
		log.Debugf("[auth] Read token expiration err: %v", err)

		return fallback
	}

	return expiration
}

// tokenExchanger refreshes the access token. Easee requires the expired access token alongside
// the refresh token, and rejects the pair unless both belong to the same session, so it is looked
// up by the refresh token the framework chose rather than read off the store - a login landing in
// between, or a rotation the store refused, would otherwise pair two different sessions.
type tokenExchanger struct {
	http     HTTPClient
	snapshot *credentialsAdapter
}

func (e *tokenExchanger) ExchangeRefreshToken(refreshToken string) (*auth.OAuth2TokenResponse, error) {
	credentials, err := e.http.RefreshToken(e.snapshot.accessTokenFor(refreshToken), refreshToken)
	if err != nil {
		return nil, err
	}

	return &auth.OAuth2TokenResponse{
		AccessToken:  credentials.AccessToken,
		ExpiresIn:    credentials.ExpiresIn,
		RefreshToken: credentials.RefreshToken,
	}, nil
}

func credentialsFromResponse(credentials *model.Credentials) config.Credentials {
	return config.Credentials{
		AccessToken:           credentials.AccessToken,
		RefreshToken:          credentials.RefreshToken,
		AccessTokenExpiresAt:  tokenExpiration(credentials.AccessToken, time.Now().Add(time.Duration(credentials.ExpiresIn)*time.Second)),
		RefreshTokenExpiresAt: tokenExpiration(credentials.RefreshToken, time.Time{}),
	}
}
