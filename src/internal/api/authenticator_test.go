package api_test

import (
	"net/http"
	"testing"
	"time"

	mockedstorage "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/storage"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockapi "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/api"
)

//nolint:godox
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
			name:          "should not return error when storage failed to save",
			username:      "user",
			password:      "pwd",
			accessToken:   accessToken,
			refreshToken:  refreshToken,
			saveError:     nil,
			errorContains: "",
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

			cfgSrv := config.NewService(&storage)

			notificationManager := fakes.NewNotifier(t)

			httpClient := mockapi.NewHTTPClient(t)

			httpClient.On("Login", v.username, v.password).Return(&model.Credentials{
				AccessToken:  v.accessToken,
				RefreshToken: v.refreshToken,
			}, v.loginError)

			auth := api.NewAuthenticator(httpClient, cfgSrv, notificationManager, nil, "test")

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
			name: "should not return error when failed to set credentials",
			credentialsCfg: config.Credentials{
				AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(time.Hour),
			},
			expectedToken: accessToken,
			accessToken:   accessToken,
			refreshToken:  refreshToken,
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

			cfgSrv := config.NewService(&storage)
			notificationManager := fakes.NewNotifier(t)

			mqtt := fimpgo.NewMqttTransport(mqttAddr, "", "", "", true, 1, 1, nil)
			require.NoError(t, mqtt.Start(5*time.Second))

			t.Cleanup(mqtt.Stop)

			httpClient := mockapi.NewHTTPClient(t)

			if !clock.Now().After(v.credentialsCfg.RefreshTokenExpiresAt) && clock.Now().After(v.credentialsCfg.AccessTokenExpiresAt) {
				httpClient.On("RefreshToken", cfg.AccessToken, cfg.RefreshToken).Return(&model.Credentials{
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
				}, v.refreshTokenError)
			}

			auth := api.NewAuthenticator(httpClient, cfgSrv, notificationManager, mqtt, "test")

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

			err := auth.Logout()

			assert.Equal(t, v.saveError, err, "should return the same error from the Save()")
			assert.Equal(t, config.Credentials{}, cfg.Credentials)
		})
	}
}

// TestUnauthorizedDoesNotImmediatelyLogout asserts that a single (or short burst of) 401
// from /refresh_token does NOT clear local credentials. The authenticator must keep
// retrying (gated by backoff) until MaxUnauthorizedDuration has elapsed; only then is the
// app logout triggered. Regression coverage for the bug where one 401 logged the user out.
//
//nolint:paralleltest
func TestUnauthorizedDoesNotImmediatelyLogout(t *testing.T) {
	now := time.Date(2026, time.May, 7, 0, 0, 0, 0, time.UTC)
	clock.Mock(now)
	t.Cleanup(clock.Restore)

	cfg := config.Config{
		Credentials: config.Credentials{
			AccessToken:           accessToken,
			RefreshToken:          refreshToken,
			AccessTokenExpiresAt:  now.Add(-time.Minute),
			RefreshTokenExpiresAt: now.Add(7 * 24 * time.Hour),
		},
	}

	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(&cfg)
	storage.On("Save").Return(nil)

	configService := config.NewService(storage)
	require.NoError(t, configService.SetAuthenticatorBackoff(
		time.Nanosecond, time.Nanosecond, time.Nanosecond,
		1_000, 1_000,
		2*time.Hour,
	))

	notificationManager := fakes.NewNotifier(t)

	client := mockapi.NewHTTPClient(t)
	client.On("RefreshToken", accessToken, refreshToken).
		Return(nil, api.HTTPError{
			Message:    "InvalidRefreshToken",
			StatusCode: http.StatusUnauthorized,
		})

	mqtt := fimpgo.NewMqttTransport(cfg.MQTTServerURI, cfg.MQTTClientIDPrefix, cfg.MQTTUsername, cfg.MQTTPassword, true, 1, 1, nil)

	auth := api.NewAuthenticator(client, configService, notificationManager, mqtt, fimptype.EaseeService)

	// 1. First 401: credentials must remain, no notification, no logout.
	_, err := auth.AccessToken()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transient 401")
	assert.NotEqual(t, config.Credentials{}, cfg.Credentials, "credentials must NOT be cleared on a single 401")
	assert.True(t, notificationManager.NoEventsReceived(), "no offline notification should fire below the threshold")

	// 2. Repeated 401s within the threshold also keep credentials intact.
	clock.Mock(now.Add(30 * time.Minute))

	_, err = auth.AccessToken()
	require.Error(t, err)
	assert.NotEqual(t, config.Credentials{}, cfg.Credentials, "credentials must remain after repeated 401s within threshold")
	assert.True(t, notificationManager.NoEventsReceived())

	// 3. Cross MaxUnauthorizedDuration: app logout fires and credentials are cleared.
	clock.Mock(now.Add(3 * time.Hour))

	_, err = auth.AccessToken()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "refreshToken expired")
	assert.Equal(t, config.Credentials{}, cfg.Credentials, "credentials must be cleared once threshold exceeded")
	assert.True(t, notificationManager.IsEventReceived("easee_status_offline"))
}

// TestUnauthorizedThenSuccessResetsThreshold asserts that a successful refresh after some
// transient 401s clears the unauthorized timer, so the next 401 starts a fresh countdown
// rather than immediately exceeding the threshold.
//
//nolint:paralleltest
func TestUnauthorizedThenSuccessResetsThreshold(t *testing.T) {
	now := time.Date(2026, time.May, 7, 0, 0, 0, 0, time.UTC)
	clock.Mock(now)
	t.Cleanup(clock.Restore)

	cfg := config.Config{
		Credentials: config.Credentials{
			AccessToken:           accessToken,
			RefreshToken:          refreshToken,
			AccessTokenExpiresAt:  now.Add(-time.Minute),
			RefreshTokenExpiresAt: now.Add(7 * 24 * time.Hour),
		},
	}

	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(&cfg)
	storage.On("Save").Return(nil)

	configService := config.NewService(storage)
	require.NoError(t, configService.SetAuthenticatorBackoff(
		time.Nanosecond, time.Nanosecond, time.Nanosecond,
		1_000, 1_000,
		2*time.Hour,
	))

	notificationManager := fakes.NewNotifier(t)

	client := mockapi.NewHTTPClient(t)
	// Sequence: 401, then a successful refresh.
	client.On("RefreshToken", accessToken, refreshToken).
		Return(nil, api.HTTPError{Message: "InvalidRefreshToken", StatusCode: http.StatusUnauthorized}).
		Once()
	client.On("RefreshToken", accessToken, refreshToken).
		Return(&model.Credentials{AccessToken: accessToken, RefreshToken: refreshToken}, nil).
		Once()

	mqtt := fimpgo.NewMqttTransport(cfg.MQTTServerURI, cfg.MQTTClientIDPrefix, cfg.MQTTUsername, cfg.MQTTPassword, true, 1, 1, nil)

	auth := api.NewAuthenticator(client, configService, notificationManager, mqtt, fimptype.EaseeService)

	// First refresh: 401, credentials kept, transient err.
	_, err := auth.AccessToken()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transient 401")

	// Second refresh: succeeds — must reset unauthorizedSince so a future 401 starts fresh.
	clock.Mock(now.Add(time.Hour))
	cfg.AccessTokenExpiresAt = now.Add(-time.Minute) // re-expire to force another refresh

	_, err = auth.AccessToken()
	require.NoError(t, err)
	assert.True(t, notificationManager.NoEventsReceived(), "no logout should fire after a successful refresh")
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
	err := configService.SetAuthenticatorBackoff(
		time.Second, time.Second, time.Second,
		1, 1,
		0,
	)
	require.NoError(t, err)

	notificationManager := fakes.NewNotifier(t)

	client := mockapi.NewHTTPClient(t)
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
		nil,
	)

	auth := api.NewAuthenticator(client, configService, notificationManager, mqtt, fimptype.EaseeService)

	_, err = auth.AccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to perform token refresh api call")

	// allow 2 retries without backoff
	for range 2 {
		_, err = auth.AccessToken()
		assert.Error(t, err)
	}

	// block more requests with backoff
	for range 8 {
		_, err = auth.AccessToken()
		assert.Contains(t, err.Error(), "too many requests: backoff")
	}

	time.Sleep(1 * time.Second)

	_, err = auth.AccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to perform token refresh api call")

	_, err = auth.AccessToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many requests: backoff")
}
