package api_test

import (
	"net/http"
	"testing"
	"time"

	mockedstorage "github.com/futurehomeno/cliffhanger/test/mocks/storage"
	"github.com/futurehomeno/fimpgo"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/routing"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/mocks"
)

// TODO: refactor it as e2e tests.

const (
	accessToken  = "eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoxMjMsInJvbGUiOiJhZG1pbiIsImV4cCI6MTcwODI4MDAwMH0." //nolint:gosec
	refreshToken = "eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoxMjMsInJvbGUiOiJhZG1pbiIsImV4cCI6MTcwODI4MDAwMH0." //nolint:gosec
)

func TestLogin(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		username      string
		password      string
		accessToken   string
		refreshToken  string
		saveError     error
		loginError    error
		errorContains string
	}{
		{
			name:          "should return error when login has failed",
			loginError:    errors.New("expected response code to be 200"),
			errorContains: "expected response code to be 200",
		},
		{
			name:          "should return error when storage failed to save",
			username:      "user",
			password:      "pwd",
			accessToken:   accessToken,
			refreshToken:  refreshToken,
			saveError:     errors.New("failed to save to the storage"),
			errorContains: "failed to save to the storage",
		},
		{
			name:         "should save tokens to the storage",
			username:     "user",
			password:     "pwd",
			accessToken:  accessToken,
			refreshToken: refreshToken,
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Config{}
			storage := mockedstorage.Storage[*config.Config]{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			notificationManager := fakes.NewNotifier(t)

			httpClient := mocks.NewHTTPClient(t)

			httpClient.On("Login", v.username, v.password).Return(&model.Credentials{
				AccessToken:  v.accessToken,
				RefreshToken: v.refreshToken,
			}, v.loginError)

			auth := api.NewAuthenticator(httpClient, cfgSrv, notificationManager, nil, "test")
			require.NoError(t, auth.EnsureBackwardsCompatibility())

			err := auth.Login(v.username, v.password)

			if v.errorContains != "" {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), v.errorContains)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, v.accessToken, cfg.AccessToken)
				assert.Equal(t, v.refreshToken, cfg.RefreshToken)
			}
		})
	}
}

func TestAccessToken(t *testing.T) {
	t.Parallel()

	mqttAddr := test.SetupMQTTContainer(t)

	testCases := []struct {
		name              string
		credentialsCfg    config.Credentials
		refreshTokenError error
		accessToken       string
		refreshToken      string
		saveError         error
		errorContains     string
		expectedToken     string
		userNotified      bool
	}{
		{
			name:          "should return error when credentials are empty",
			errorContains: "credentials are empty",
		},
		{
			name: "should return access token when it isn't expired",
			credentialsCfg: config.Credentials{
				AccessTokenExpiresAt:  time.Now().Add(time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(2 * time.Hour),
				AccessToken:           "valid access token",
			},
			accessToken:   "valid access token",
			expectedToken: "valid access token",
		},
		{
			name: "should log out when token refresh operation does not return http 200",
			credentialsCfg: config.Credentials{
				AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(time.Hour),
			},

			refreshTokenError: api.HTTPError{
				Message:    "failed to perform token refresh api call",
				StatusCode: http.StatusBadRequest,
			},
			errorContains: "failed to perform token refresh api call",
		},
		{
			name: "should return error when failed to set credentials",
			credentialsCfg: config.Credentials{
				AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(time.Hour),
			},

			saveError:     errors.New("failed to save"),
			errorContains: "failed to save",
		},
		{
			name: "should save refreshed token when all validations passed",
			credentialsCfg: config.Credentials{
				AccessToken:           "old_access_token",
				AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(time.Hour),
			},
			expectedToken: accessToken,
			accessToken:   accessToken,
			refreshToken:  refreshToken,
		},
	}

	for _, val := range testCases {
		v := val
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.Config{
				Credentials: v.credentialsCfg,
			}
			storage := mockedstorage.Storage[*config.Config]{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			cfgSrv := config.NewConfigServiceWithStorage(&storage)
			notificationManager := fakes.NewNotifier(t)

			mqtt := fimpgo.NewMqttTransport(mqttAddr, "", "", "", true, 1, 1)
			require.NoError(t, mqtt.Start())

			t.Cleanup(mqtt.Stop)

			httpClient := mocks.NewHTTPClient(t)

			if !clock.Now().After(v.credentialsCfg.RefreshTokenExpiresAt) && clock.Now().After(v.credentialsCfg.AccessTokenExpiresAt) {
				httpClient.On("RefreshToken", cfg.AccessToken, cfg.RefreshToken).Return(&model.Credentials{
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
				}, v.refreshTokenError)
			}

			auth := api.NewAuthenticator(httpClient, cfgSrv, notificationManager, mqtt, "test")
			require.NoError(t, auth.EnsureBackwardsCompatibility())

			token, err := auth.AccessToken()

			if v.errorContains != "" {
				assert.NotNil(t, err)
				assert.Contains(t, err.Error(), v.errorContains)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, v.expectedToken, token)
				assert.Equal(t, v.accessToken, cfg.AccessToken)
				assert.Equal(t, v.refreshToken, cfg.RefreshToken)
			}

			if v.userNotified {
				assert.Equal(t, notificationManager.ReceivedEventsCount(), 1)
				assert.True(t, notificationManager.IsEventReceived("easee_status_offline"))
			} else {
				assert.True(t, notificationManager.NoEventsReceived())
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

			cfg := config.Config{
				Credentials: config.Credentials{
					AccessToken:           "token",
					RefreshToken:          "refresh token",
					AccessTokenExpiresAt:  time.Now().Add(time.Hour),
					RefreshTokenExpiresAt: time.Now().Add(time.Hour),
				},
			}

			storage := mockedstorage.Storage[*config.Config]{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			auth := api.NewAuthenticator(nil, config.NewService(&storage), nil, nil, "test")
			require.NoError(t, auth.EnsureBackwardsCompatibility())

			err := auth.Logout()

			assert.Equal(t, v.saveError, err, "should return the same error from the Save()")
			assert.Equal(t, config.Credentials{}, cfg.Credentials)
		})
	}
}

//nolint:paralleltest
func TestHandleFailedRefreshToken(t *testing.T) {
	cfg := config.Config{
		Credentials: config.Credentials{
			AccessToken:           accessToken,
			RefreshToken:          refreshToken,
			RefreshTokenExpiresAt: time.Now().Add(time.Hour),
			AccessTokenExpiresAt:  time.Now(),
		},
	}

	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(&cfg)
	storage.On("Save").Return(nil)

	configService := config.NewService(storage)
	err := configService.SetAuthenticatorBackoffCfg(config.BackoffCfg{
		InitialBackoff:       time.Second,
		RepeatedBackoff:      time.Second,
		FinalBackoff:         time.Second,
		InitialFailureCount:  1,
		RepeatedFailureCount: 1,
	})
	require.NoError(t, err)

	notificationManager := fakes.NewNotifier(t)

	client := mocks.NewHTTPClient(t)
	client.On("RefreshToken", accessToken, refreshToken).
		Return(
			nil,
			api.HTTPError{
				Message:    "failed to perform token refresh api call",
				StatusCode: http.StatusNotFound,
			},
		)

	mqtt := fimpgo.NewMqttTransport(
		cfg.MQTTServerURI,
		cfg.MQTTClientIDPrefix,
		cfg.MQTTUsername,
		cfg.MQTTPassword,
		true,
		1,
		1,
	)

	auth := api.NewAuthenticator(client, configService, notificationManager, mqtt, routing.ServiceName)
	require.NoError(t, auth.EnsureBackwardsCompatibility())

	_, err = auth.AccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to perform token refresh api call")

	for i := 0; i < 10; i++ {
		_, err = auth.AccessToken()
		assert.Contains(t, err.Error(), "too many requests: backoff is in use")
	}

	time.Sleep(1 * time.Second)

	_, err = auth.AccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to perform token refresh api call")

	_, err = auth.AccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many requests: backoff is in use")
}
