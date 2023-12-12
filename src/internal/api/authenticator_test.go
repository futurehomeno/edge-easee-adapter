package api //nolint:testpackage

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/notification"
	mockedstorage "github.com/futurehomeno/cliffhanger/test/mocks/storage"
	"github.com/futurehomeno/fimpgo"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		username      string
		password      string
		loginStatus   int
		accessToken   string
		refreshToken  string
		saveError     error
		errorContains string
	}{
		{
			name:          "should return error when login has failed",
			loginStatus:   http.StatusBadRequest,
			errorContains: "expected response code to be 200",
		},
		{
			name:          "should return error when storage failed to save",
			username:      "user",
			password:      "pwd",
			loginStatus:   http.StatusOK,
			accessToken:   "accessToken",
			refreshToken:  "refreshing",
			saveError:     errors.New("failed to save to the storage"),
			errorContains: "failed to save to the storage",
		},
		{
			name:         "should save tokens to the storage",
			username:     "user",
			password:     "pwd",
			loginStatus:  http.StatusOK,
			accessToken:  "accessToken",
			refreshToken: "refreshing",
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			handler := func(w http.ResponseWriter, r *http.Request) {
				data := fmt.Sprintf(`{"accessToken":"%s","refreshToken":"%s"}`,
					v.accessToken,
					v.refreshToken,
				)
				w.WriteHeader(v.loginStatus)
				_, _ = w.Write([]byte(data))
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			cfg := config.Config{}
			storage := mockedstorage.Storage[*config.Config]{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			notificationManager := &NotificationMock{}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			client := NewHTTPClient(cfgSrv, httpClient, server.URL)
			auth := authenticator{http: client, cfgSvc: config.NewService(&storage), notificationManager: notificationManager}

			err := auth.Login(v.username, v.password)
			if err != nil {
				assert.Contains(t, err.Error(), v.errorContains)
			} else {
				assert.Equal(t, v.accessToken, cfg.AccessToken)
				assert.Equal(t, v.refreshToken, cfg.RefreshToken)
				assert.Equal(t, statusWorkingProperly, auth.status)
			}
		})
	}
}

func TestAccessToken(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		credentials   config.Credentials
		authStatus    connectionStatus
		refreshStatus int
		accessToken   string
		refreshToken  string
		saveError     error
		errorContains string
		expectedToken string
	}{
		{
			name:          "should return error when credentials are empty",
			errorContains: "credentials are empty",
		},
		{
			name: "should return access token when it isn't expired",
			credentials: config.Credentials{
				ExpiresAt:   time.Now().Add(time.Hour),
				AccessToken: "valid access token",
			},
			accessToken:   "valid access token",
			expectedToken: "valid access token",
		},
		{
			name: "should return error when status is statusWaitingToReconnect",
			credentials: config.Credentials{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			authStatus:    statusWaitingToReconnect,
			errorContains: "connection interrupted",
		},
		{
			name: "should return error when status is statusConnectionFailed",
			credentials: config.Credentials{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			authStatus:    statusConnectionFailed,
			errorContains: "connection interrupted",
		},
		{
			name: "should return error when not 200 code is returned from RefreshToken",
			credentials: config.Credentials{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			authStatus:    statusWorkingProperly,
			refreshStatus: http.StatusBadRequest,
			accessToken:   "access token",
			refreshToken:  "refresh token",
			errorContains: "failed to refresh token",
		},
		{
			name: "should return error when failed to set credentials",
			credentials: config.Credentials{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			authStatus:    statusWorkingProperly,
			refreshStatus: http.StatusOK,
			accessToken:   "access token",
			refreshToken:  "refresh token",
			saveError:     errors.New("failed to save"),
			errorContains: "failed to save",
		},
		{
			name: "should save refreshed token when all validations passed",
			credentials: config.Credentials{
				ExpiresAt: time.Now().Add(-time.Hour),
			},
			authStatus:    statusWorkingProperly,
			refreshStatus: http.StatusOK,
			accessToken:   "access token",
			refreshToken:  "refresh token",
			expectedToken: "access token",
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			handler := func(w http.ResponseWriter, r *http.Request) {
				data := fmt.Sprintf(`{"accessToken":"%s","refreshToken":"%s"}`,
					v.accessToken,
					v.refreshToken,
				)
				w.WriteHeader(v.refreshStatus)
				_, _ = w.Write([]byte(data))
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			// mock cfgSvc
			cfg := config.Config{Credentials: v.credentials, Backoff: config.BackoffCfg{Length: "0"}}
			storage := mockedstorage.Storage[*config.Config]{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)
			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			// mock httpClient
			httpClient := &http.Client{Timeout: 3 * time.Second}
			client := NewHTTPClient(cfgSrv, httpClient, server.URL)
			auth := authenticator{http: client, cfgSvc: config.NewService(&storage), backoffCfg: backoffCfg{
				status: v.authStatus,
			}}

			token, err := auth.AccessToken()
			if err != nil {
				assert.Contains(t, err.Error(), v.errorContains)
			} else {
				assert.Equal(t, v.expectedToken, token)
				assert.Equal(t, v.accessToken, cfg.AccessToken)
				assert.Equal(t, v.refreshToken, cfg.RefreshToken)
				assert.Equal(t, statusWorkingProperly, auth.status)
				assert.Equal(t, 0, auth.attempts)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		saveError error
	}{
		{
			name:      "should return error if save fails",
			saveError: errors.New("error"),
		},
		{
			name: "credentials should be empty",
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Config{Credentials: config.Credentials{AccessToken: "token"}}
			storage := mockedstorage.Storage[*config.Config]{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			auth := authenticator{cfgSvc: config.NewService(&storage)}
			err := auth.Logout()

			assert.Equal(t, v.saveError, err, "should return the same error from the Save()")
			assert.Equal(t, config.Credentials{}, cfg.Credentials)
		})
	}
}

func TestHandleFailedRefreshToken(t *testing.T) {
	t.Parallel()

	testCases := []*struct {
		name                      string
		errIn                     error
		saveError                 error
		backoffCfg                config.BackoffCfg
		auth                      authenticator
		expectedStatus            connectionStatus
		errorContains             string
		notificationManagerCalled int
	}{
		{
			name:                      "should return error when input error is http error with 401 status code",
			errIn:                     HTTPError{Status: http.StatusUnauthorized},
			expectedStatus:            statusConnectionFailed,
			errorContains:             "unauthorized error",
			notificationManagerCalled: 1,
		},
		{
			name:       "should set status to failed when reached maximum attempts",
			errIn:      errors.New("error"),
			backoffCfg: config.BackoffCfg{MaxAttempts: 2, Length: "0"},
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:   statusReconnecting,
					attempts: 2,
				},
			},
			expectedStatus:            statusConnectionFailed,
			errorContains:             "failed delayed attempt",
			notificationManagerCalled: 1,
		},
		{
			name:       "should make another attempt to reconnect when haven't reached maximum",
			errIn:      errors.New("error"),
			backoffCfg: config.BackoffCfg{MaxAttempts: 2, Length: "0"},
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:   statusReconnecting,
					attempts: 1,
				},
			},
			expectedStatus: statusWaitingToReconnect,
			errorContains:  "failed delayed attempt",
		},
		{
			name:       "should fire first backoff when encountered problem for the first time",
			errIn:      errors.New("error"),
			backoffCfg: config.BackoffCfg{MaxAttempts: 2, Length: "0"},
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:   statusWorkingProperly,
					attempts: 0,
				},
			},
			expectedStatus: statusWaitingToReconnect,
			errorContains:  "failed to refresh token",
		},
		{
			name:       "should return error when invalid status is encountered",
			errIn:      errors.New("error"),
			backoffCfg: config.BackoffCfg{MaxAttempts: 2, Length: "0"},
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:   statusConnectionFailed,
					attempts: 0,
				},
			},
			expectedStatus: statusConnectionFailed,
			errorContains:  "invalid auth status",
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			// mock cfgSvc
			cfg := config.Config{Credentials: config.Credentials{}, Backoff: v.backoffCfg}
			storage := mockedstorage.Storage[*config.Config]{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			notificationManager := &NotificationMock{}
			notificationManager.On("Event", &notification.Event{EventName: notificationEaseeStatusOffline}).Return(nil)

			cfgDir := path.Join("./../../testdata/testing/", "")
			mqttCfg := config.New(cfgDir)
			v.auth.mqtt = fimpgo.NewMqttTransport(
				mqttCfg.MQTTServerURI,
				mqttCfg.MQTTClientIDPrefix,
				mqttCfg.MQTTUsername,
				mqttCfg.MQTTPassword,
				true,
				1,
				1)

			v.auth.notificationManager = notificationManager
			v.auth.cfgSvc = config.NewService(&storage)
			err := v.auth.handleFailedRefreshToken(v.errIn)
			assert.Equal(t, v.expectedStatus, v.auth.status)
			assert.Contains(t, err.Error(), v.errorContains)
			notificationManager.AssertNumberOfCalls(t, "Event", v.notificationManagerCalled)
		})
	}
}

func TestHookResetToReconnecting(t *testing.T) {
	t.Parallel()

	cfg := config.Config{Backoff: config.BackoffCfg{Length: "0"}}
	storage := mockedstorage.Storage[*config.Config]{}
	storage.On("Model").Return(&cfg)

	auth := authenticator{cfgSvc: config.NewService(&storage)}
	auth.hookResetToReconnecting()
	assert.Equal(t, statusReconnecting, auth.status, "connection status is incorrect")
	assert.Equal(t, 1, auth.attempts)
}

type NotificationMock struct {
	mock.Mock
}

// enforce interface.
var _ notification.Notification = &NotificationMock{}

func (m *NotificationMock) Message(arg string) error {
	args := m.Called(arg)
	return args.Error(0) //nolint
}

func (m *NotificationMock) Event(event *notification.Event) error {
	args := m.Called(event)
	return args.Error(0) //nolint
}

func (m *NotificationMock) EventWithProps(event *notification.Event, props map[string]string) error {
	args := m.Called(event, props)
	return args.Error(0) //nolint
}
