package api //nolint:testpackage

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/notification"
	mockedstorage "github.com/futurehomeno/cliffhanger/test/mocks/storage"
	"github.com/futurehomeno/fimpgo"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

const (
	accessToken  = "eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoxMjMsInJvbGUiOiJhZG1pbiIsImV4cCI6MTcwODI4MDAwMH0."
	refreshToken = "eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoxMjMsInJvbGUiOiJhZG1pbiIsImV4cCI6MTcwODI4MDAwMH0."
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
			accessToken:   accessToken,
			refreshToken:  refreshToken,
			saveError:     errors.New("failed to save to the storage"),
			errorContains: "failed to save to the storage",
		},
		{
			name:         "should save tokens to the storage",
			username:     "user",
			password:     "pwd",
			loginStatus:  http.StatusOK,
			accessToken:  accessToken,
			refreshToken: refreshToken,
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			handler := func(w http.ResponseWriter, _ *http.Request) {
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
			auth := authenticator{
				http:                client,
				cfgSvc:              config.NewService(&storage),
				notificationManager: notificationManager,
				backoff:             backoff.NewStateful(10, 10, 10, 10, 10),
			}

			err := auth.Login(v.username, v.password)
			if err != nil {
				assert.Contains(t, err.Error(), v.errorContains)
			} else {
				assert.Equal(t, v.accessToken, cfg.AccessToken)
				assert.Equal(t, v.refreshToken, cfg.RefreshToken)
			}
		})
	}
}

func TestAccessToken(t *testing.T) {
	t.Parallel()

	now := time.Date(2025, 2, 17, 10, 0, 0, 0, time.UTC)

	clockMock := clock.Mock(now)
	defer clock.Restore()

	testCases := []struct {
		name          string
		credentials   config.Credentials
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
				AccessTokenExpiresAt: time.Now().Add(time.Hour),
				AccessToken:          "valid access token",
			},
			accessToken:   "valid access token",
			expectedToken: "valid access token",
		},
		{
			name: "should return error when status is statusWaitingToReconnect",
			credentials: config.Credentials{
				AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(-time.Hour),
			},
			errorContains: "connection interrupted",
		},
		{
			name: "should return error when status is statusConnectionFailed",
			credentials: config.Credentials{
				AccessTokenExpiresAt: time.Now().Add(-time.Hour),
			},
			errorContains: "connection interrupted",
		},
		{
			name: "should log out when not 200 code is returned from RefreshToken", ///todo repair
			credentials: config.Credentials{
				AccessTokenExpiresAt: time.Now().Add(-time.Hour),
			},
			refreshStatus: http.StatusBadRequest,
			accessToken:   "",
			refreshToken:  "",
		},
		{
			name: "should return error when failed to set credentials",
			credentials: config.Credentials{
				AccessTokenExpiresAt: time.Now().Add(-time.Hour),
			},
			refreshStatus: http.StatusOK,
			saveError:     errors.New("failed to save"),
			errorContains: "failed to save",
		},
		{
			name: "should save refreshed token when all validations passed",
			credentials: config.Credentials{
				AccessTokenExpiresAt: time.Now().Add(-time.Hour),
			},
			refreshStatus: http.StatusOK,
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			handler := func(w http.ResponseWriter, _ *http.Request) {
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
			auth := authenticator{
				http:   client,
				cfgSvc: config.NewService(&storage),
				mqtt: fimpgo.NewMqttTransport(
					cfg.MQTTServerURI,
					cfg.MQTTClientIDPrefix,
					cfg.MQTTUsername,
					cfg.MQTTPassword,
					true,
					1,
					1,
				),
				backoff: backoff.NewStateful(0, 0, 0, 0, 0),
			}

			token, err := auth.AccessToken()
			if err != nil {
				assert.Contains(t, err.Error(), v.errorContains)
			} else {
				assert.Equal(t, v.expectedToken, token)
				assert.Equal(t, v.accessToken, cfg.AccessToken)
				assert.Equal(t, v.refreshToken, cfg.RefreshToken)
			}

			clockMock.Add(5 * time.Minute)
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
	backoff := backoff.NewStateful(
		5*time.Second,
		1*time.Second,
		1*time.Second,
		1,
		1,
	)

	handler := func(w http.ResponseWriter, _ *http.Request) {
		data := fmt.Sprintf(`{"accessToken":"%s","refreshToken":"%s"}`,
			accessToken,
			refreshToken,
		)

		w.WriteHeader(http.StatusNotFound)

		_, _ = w.Write([]byte(data))
	}
	server := httptest.NewServer(http.HandlerFunc(handler))

	defer server.Close()

	cfg := config.Config{Credentials: config.Credentials{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: time.Now().Add(time.Hour),
		AccessTokenExpiresAt:  time.Now(),
	}, Backoff: config.BackoffCfg{Length: "0"}}

	storage := mockedstorage.Storage[*config.Config]{}
	storage.On("Model").Return(&cfg)
	storage.On("Save").Return("ok")

	cfgSrv := config.NewConfigServiceWithStorage(&storage)

	notificationManager := &NotificationMock{}

	httpClient := &http.Client{Timeout: 3 * time.Second}
	client := NewHTTPClient(cfgSrv, httpClient, server.URL)
	auth := authenticator{
		http:                client,
		cfgSvc:              config.NewService(&storage),
		notificationManager: notificationManager,
		backoff:             backoff,
		mqtt: fimpgo.NewMqttTransport(
			cfg.MQTTServerURI,
			cfg.MQTTClientIDPrefix,
			cfg.MQTTUsername,
			cfg.MQTTPassword,
			true,
			1,
			1,
		),
	}

	_, err := auth.AccessToken()
	assert.Contains(t, err.Error(), "failed to perform token refresh api call:")

	for i := 0; i < 10; i++ {
		_, err = auth.AccessToken()
		assert.Contains(t, err.Error(), "too many requests, backoff is in use")
	}

	time.Sleep(5 * time.Second)

	_, err = auth.AccessToken()
	assert.Contains(t, err.Error(), "failed to perform token refresh api call:")

	_, err = auth.AccessToken()
	assert.Contains(t, err.Error(), "too many requests, backoff is in use")
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
