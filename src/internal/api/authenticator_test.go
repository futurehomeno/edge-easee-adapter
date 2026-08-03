package api_test

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/httpclient"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockapi "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/api"
	mockedstorage "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/storage"
)

func TestLogin(t *testing.T) {
	t.Parallel()

	accessToken := jwtWithExpiry(time.Now().Add(time.Hour))
	refreshToken := jwtWithExpiry(time.Now().Add(24 * time.Hour))

	testCases := []struct {
		name          string
		loginError    error
		errorContains string
	}{
		{
			name:          "should return error when login has failed",
			loginError:    errors.New("expected response code to be 200"),
			errorContains: "expected response code to be 200",
		},
		{
			name: "should save tokens to the storage",
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			credentials := newCredentialsStore(t, config.Credentials{})

			httpClient := mockapi.NewHTTPClient(t)
			httpClient.On("Login", "user", "pwd").Return(&model.Credentials{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
			}, v.loginError)

			authenticator := newAuthenticator(t, httpClient, credentials, fakes.NewNotifier(t), 0)

			err := authenticator.Login("user", "pwd")

			if v.errorContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), v.errorContains)
				assert.True(t, credentials.Credentials().Empty())

				return
			}

			require.NoError(t, err)
			assert.Equal(t, accessToken, credentials.Credentials().AccessToken)
			assert.Equal(t, refreshToken, credentials.Credentials().RefreshToken)
			// Expiry times are derived from the tokens themselves, not from the response.
			assert.False(t, credentials.Credentials().RefreshTokenExpiresAt.IsZero())
		})
	}
}

func TestAccessToken(t *testing.T) {
	t.Parallel()

	freshAccessToken := jwtWithExpiry(time.Now().Add(time.Hour))
	freshRefreshToken := jwtWithExpiry(time.Now().Add(24 * time.Hour))

	testCases := []struct {
		name          string
		credentials   config.Credentials
		refreshError  error
		errorContains string
		expectedToken string
	}{
		{
			name:          "should return error when credentials are empty",
			errorContains: "not logged in",
		},
		{
			name: "should return access token when it isn't expired",
			credentials: config.Credentials{
				AccessToken:           "valid access token",
				RefreshToken:          freshRefreshToken,
				AccessTokenExpiresAt:  time.Now().Add(time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(2 * time.Hour),
			},
			expectedToken: "valid access token",
		},
		{
			name: "should return error when token refresh fails",
			credentials: config.Credentials{
				AccessToken:           "expired access token",
				RefreshToken:          freshRefreshToken,
				AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(time.Hour),
			},
			refreshError:  errors.New("failed to perform token refresh api call"),
			errorContains: "failed to perform token refresh api call",
		},
		{
			name: "should save refreshed token when all validations passed",
			credentials: config.Credentials{
				AccessToken:           "old access token",
				RefreshToken:          freshRefreshToken,
				AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
				RefreshTokenExpiresAt: time.Now().Add(time.Hour),
			},
			expectedToken: freshAccessToken,
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			credentials := newCredentialsStore(t, v.credentials)
			notifier := fakes.NewNotifier(t)

			httpClient := mockapi.NewHTTPClient(t)
			if !v.credentials.Empty() && v.credentials.AccessTokenExpired() {
				httpClient.On("RefreshToken", v.credentials.AccessToken, v.credentials.RefreshToken).
					Return(&model.Credentials{AccessToken: freshAccessToken, RefreshToken: freshRefreshToken}, v.refreshError)
			}

			authenticator := newAuthenticator(t, httpClient, credentials, notifier, 0)

			token, err := authenticator.AccessToken()

			if v.errorContains != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), v.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, v.expectedToken, token)
				assert.Equal(t, v.expectedToken, credentials.Credentials().AccessToken)
			}

			assert.True(t, notifier.NoEventsReceived())
		})
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()

	saveError := errors.New("error")

	testCases := []struct {
		name      string
		saveError error
	}{
		{
			name:      "should return error if reset fails",
			saveError: saveError,
		},
		{
			name: "credentials should be empty",
		},
	}

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()

			credentials := config.Credentials{AccessToken: "token", RefreshToken: "refresh token"}

			storage := mockedstorage.NewStorage[*config.Credentials](t)
			storage.On("Model").Return(&credentials).Maybe()
			storage.On("Reset").Return(v.saveError)

			authenticator := newAuthenticator(t, nil, config.NewCredentialsStoreWithStorage(storage), nil, 0)

			assert.Equal(t, v.saveError, authenticator.Logout())
		})
	}
}

// TestUnauthorizedDoesNotImmediatelyLogout asserts that a single (or short burst of) 401
// from /refresh_token does NOT clear local credentials. The authenticator must keep
// retrying (gated by backoff) until the unauthorized grace has elapsed; only then is the
// app logout triggered. Regression coverage for the bug where one 401 logged the user out.
func TestUnauthorizedDoesNotImmediatelyLogout(t *testing.T) {
	t.Parallel()

	const grace = 300 * time.Millisecond

	credentials := newCredentialsStore(t, config.Credentials{
		AccessToken:           "expired access token",
		RefreshToken:          jwtWithExpiry(time.Now().Add(7 * 24 * time.Hour)),
		AccessTokenExpiresAt:  time.Now().Add(-time.Minute),
		RefreshTokenExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})

	notifier := fakes.NewNotifier(t)

	httpClient := mockapi.NewHTTPClient(t)
	httpClient.On("RefreshToken", "expired access token", credentials.Credentials().RefreshToken).
		Return(nil, fmt.Errorf("InvalidRefreshToken: %w", httpclient.ErrUnauthorized))

	authenticator := newAuthenticator(t, httpClient, credentials, notifier, grace)

	_, err := authenticator.AccessToken()
	require.Error(t, err)
	assert.False(t, credentials.Credentials().Empty(), "credentials must NOT be cleared on a single 401")
	assert.True(t, notifier.NoEventsReceived(), "no offline notification should fire below the threshold")

	time.Sleep(grace)

	_, err = authenticator.AccessToken()
	require.Error(t, err)
	assert.True(t, credentials.Credentials().Empty(), "credentials must be cleared once the grace elapsed")
	assert.True(t, notifier.IsEventReceived("easee_status_offline"))
}

// TestExpiredRefreshTokenLogsOut asserts a refresh token past its local expiry logs the app
// out without even calling the API.
func TestExpiredRefreshTokenLogsOut(t *testing.T) {
	t.Parallel()

	credentials := newCredentialsStore(t, config.Credentials{
		AccessToken:           "expired access token",
		RefreshToken:          "expired refresh token",
		AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(-time.Minute),
	})

	notifier := fakes.NewNotifier(t)

	authenticator := newAuthenticator(t, mockapi.NewHTTPClient(t), credentials, notifier, time.Hour)

	_, err := authenticator.AccessToken()
	require.Error(t, err)
	assert.True(t, credentials.Credentials().Empty())
	assert.True(t, notifier.IsEventReceived("easee_status_offline"))
}

// TestRefreshBackoff asserts that repeated failures suspend further refresh attempts and that
// the suspension is reported as ErrRefreshBackoff, which callers downgrade to a debug log.
func TestRefreshBackoff(t *testing.T) {
	t.Parallel()

	credentials := newCredentialsStore(t, config.Credentials{
		AccessToken:           "expired access token",
		RefreshToken:          jwtWithExpiry(time.Now().Add(24 * time.Hour)),
		AccessTokenExpiresAt:  time.Now().Add(-time.Hour),
		RefreshTokenExpiresAt: time.Now().Add(24 * time.Hour),
	})

	httpClient := mockapi.NewHTTPClient(t)
	httpClient.On("RefreshToken", "expired access token", credentials.Credentials().RefreshToken).
		Return(nil, errors.New("failed to perform token refresh api call"))

	authenticator := newAuthenticator(t, httpClient, credentials, fakes.NewNotifier(t), 0)

	_, err := authenticator.AccessToken()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to perform token refresh api call")

	_, err = authenticator.AccessToken()
	require.ErrorIs(t, err, api.ErrRefreshBackoff)
}

func newAuthenticator(
	t *testing.T,
	httpClient api.HTTPClient,
	credentials *config.CredentialsStore,
	notifier api.Notifier,
	grace time.Duration,
) api.Authenticator {
	t.Helper()

	cfg := &config.Config{}
	storage := mockedstorage.NewStorage[*config.Config](t)
	storage.On("Model").Return(cfg).Maybe()
	storage.On("Save").Return(nil).Maybe()

	cfgSrv := config.NewService(storage)
	require.NoError(t, cfgSrv.SetAuthenticatorBackoff(time.Hour, time.Hour, time.Hour, 1, 1, grace))

	mqtt := fimpgo.NewMqttTransport("", "", "", "", true, 1, 1, nil)

	return api.NewAuthenticator(httpClient, credentials, cfgSrv, notifier, mqtt, fimptype.EaseeService)
}

func newCredentialsStore(t *testing.T, credentials config.Credentials) *config.CredentialsStore {
	t.Helper()

	return config.NewCredentialsStoreWithStorage(
		fakes.NewConfigStorage(t, &credentials, func() *config.Credentials { return &config.Credentials{} }),
	)
}

// jwtWithExpiry builds an unsigned JWT carrying only an expiry claim.
func jwtWithExpiry(expiresAt time.Time) string {
	encode := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

	return encode(`{"alg":"none"}`) + "." + encode(fmt.Sprintf(`{"exp":%d}`, expiresAt.Unix())) + "."
}
