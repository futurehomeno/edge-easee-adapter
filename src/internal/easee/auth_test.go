package easee

import (
	"fmt"
	mockedstorage "github.com/futurehomeno/cliffhanger/test/mocks/storage"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLogin(t *testing.T) {
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

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				data := fmt.Sprintf(`{"accessToken":"%s","refreshToken":"%s"}`,
					v.accessToken,
					v.refreshToken,
				)
				w.WriteHeader(v.loginStatus)
				w.Write([]byte(data))
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			cfg := config.Config{}
			storage := mockedstorage.Storage{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			client := NewHTTPClient(httpClient, server.URL)
			auth := authenticator{http: client, cfgSvc: config.NewService(&storage)}

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
	testCases := []struct {
		name          string
		credentials   config.Credentials
		authStatus    connectivityStatus
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

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				data := fmt.Sprintf(`{"accessToken":"%s","refreshToken":"%s"}`,
					v.accessToken,
					v.refreshToken,
				)
				w.WriteHeader(v.refreshStatus)
				w.Write([]byte(data))
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			// mock cfgSvc
			cfg := config.Config{Credentials: v.credentials}
			storage := mockedstorage.Storage{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(v.saveError)

			// mock httpClient
			httpClient := &http.Client{Timeout: 3 * time.Second}
			client := NewHTTPClient(httpClient, server.URL)
			auth := authenticator{http: client, cfgSvc: config.NewService(&storage), backoffCfg: backoffCfg{
				status:        v.authStatus,
				lengthSeconds: 0,
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

func TestHandleFailedRefreshToken(t *testing.T) {
	testCases := []struct {
		name           string
		errIn          error
		auth           authenticator
		expectedStatus connectivityStatus
		errorContains  string
	}{
		{
			name:           "should return error when input error is http error with 401 status code",
			errIn:          HttpError{Status: http.StatusUnauthorized},
			expectedStatus: statusConnectionFailed,
			errorContains:  "unauthorized error",
		},
		{
			name:  "should set status to failed when reached maximum attempts",
			errIn: errors.New("error"),
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:      statusReconnecting,
					maxAttempts: 2,
					attempts:    2,
				},
			},
			expectedStatus: statusConnectionFailed,
			errorContains:  "failed delayed attempt",
		},
		{
			name:  "should make another attempt to reconnect when haven't reached maximum",
			errIn: errors.New("error"),
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:      statusReconnecting,
					maxAttempts: 2,
					attempts:    1,
				},
			},
			expectedStatus: statusWaitingToReconnect,
			errorContains:  "failed delayed attempt",
		},
		{
			name:  "should fire first backoff when encountered problem for the first time",
			errIn: errors.New("error"),
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:      statusWorkingProperly,
					maxAttempts: 2,
					attempts:    0,
				},
			},
			expectedStatus: statusWaitingToReconnect,
			errorContains:  "failed to refresh token",
		},
		{
			name:  "should return error when invalid status is encountered",
			errIn: errors.New("error"),
			auth: authenticator{
				backoffCfg: backoffCfg{
					status:      statusConnectionFailed,
					maxAttempts: 2,
					attempts:    0,
				},
			},
			expectedStatus: statusConnectionFailed,
			errorContains:  "invalid auth status",
		},
	}

	for _, val := range testCases {
		// copy val to avoid capturing
		v := val
		t.Run(v.name, func(t *testing.T) {
			// mock cfgSvc
			cfg := config.Config{Credentials: config.Credentials{}}
			storage := mockedstorage.Storage{}
			storage.On("Model").Return(&cfg)
			storage.On("Save").Return(nil)

			v.auth.lengthSeconds = 1
			v.auth.cfgSvc = config.NewService(&storage)
			err := v.auth.handleFailedRefreshToken(v.errIn)
			assert.Equal(t, v.expectedStatus, v.auth.status)
			assert.Contains(t, err.Error(), v.errorContains)
		})
	}
}

func TestHookResetToReconnecting(t *testing.T) {
	auth := authenticator{backoffCfg: backoffCfg{lengthSeconds: 0}}
	auth.hookResetToReconnecting()
	assert.Equal(t, statusReconnecting, auth.status, "connectivity status is incorrect")
	assert.Equal(t, 1, auth.attempts)
}
