package easee_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/michalkurzeja/go-clock"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
)

func TestClient_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		username         string
		password         string
		serverHandler    http.Handler
		forceServerError bool
		want             *easee.Credentials
		wantErr          bool
	}{
		{
			name:     "successful call to Easee API",
			username: "test",
			password: "example",
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/token",
				requestBody:   `{"userName":"test","password":"example"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusOK,
				responseBody: `{"accessToken":"access-token","expiresIn":86400,"accessClaims":["User"],"tokenType":"Bearer","refreshToken":"refresh-token"}`,
			}),
			want: &easee.Credentials{
				AccessToken: "access-token",
				ExpiresIn:   86400,
				AccessClaims: []string{
					"User",
				},
				TokenType:    "Bearer",
				RefreshToken: "refresh-token",
			},
		},
		{
			name:     "response code != 200",
			username: "test",
			password: "example",
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/token",
				requestBody:   `{"userName":"test","password":"example"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:     "failed to unmarshal the response",
			username: "test",
			password: "example",
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/token",
				requestBody:   `{"userName":"test","password":"example"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusOK,
				responseBody: `{"field1":1,"field2":2}`,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			username:         "test",
			password:         "example",
			forceServerError: true,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			cfgService := makeConfigService(t, exampleConfig(t))
			httpClient := &http.Client{Timeout: 3 * time.Second}

			c := easee.NewClient(httpClient, cfgService, s.URL)

			got, err := c.Login(tt.username, tt.password)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_StartCharging(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		current          float64
		cfg              *config.Config
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name:      "successful call to Easee API",
			chargerID: "123456",
			current:   40,
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/123456/settings",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"dynamicChargerCurrent":40}`,
					responseCode: http.StatusAccepted,
					responseBody: exampleCommandBody(t),
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: failedCheckerBody(t),
				},
				{
					// previous check failed, retry...
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: successfulCheckerBody(t),
				},
			}...),
		},
		{
			name:      "response code != 200",
			chargerID: "123456",
			current:   40,
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/chargers/123456/settings",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
					"Content-Type":  "application/*+json",
				},
				requestBody:  `{"dynamicChargerCurrent":40}`,
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        "123456",
			current:          40,
			cfg:              exampleConfig(t),
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "return error if credentials are empty",
			chargerID: "123456",
			cfg:       &config.Config{},
			wantErr:   true,
		},
		{
			name:      "expired access token - refreshing it under the hood",
			chargerID: "123456",
			current:   40,
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/accounts/refresh_token",
					requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
					requestHeaders: map[string]string{
						"Content-Type": "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: `{"accessToken":"new-access-token","expiresIn":86400,"accessClaims":["User"],"tokenType":"Bearer","refreshToken":"refresh-token"}`,
				},
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/123456/settings",
					requestHeaders: map[string]string{
						"Authorization": "Bearer new-access-token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"dynamicChargerCurrent":40}`,
					responseCode: http.StatusAccepted,
					responseBody: exampleCommandBody(t),
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer new-access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: successfulCheckerBody(t),
				},
			}...),
		},
		{
			name:      "refreshing expired access token failed",
			chargerID: "123456",
			current:   40,
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/refresh_token",
				requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			cfgService := makeConfigService(t, tt.cfg)

			c := easee.NewClient(httpClient, cfgService, s.URL)

			err := c.StartCharging(tt.chargerID, tt.current)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_StopCharging(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		cfg              *config.Config
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name:      "successful call to Easee API",
			chargerID: "123456",
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/123456/settings",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"dynamicChargerCurrent":0}`,
					responseCode: http.StatusAccepted,
					responseBody: exampleCommandBody(t),
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: failedCheckerBody(t),
				},
				{
					// previous check failed, retry...
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: successfulCheckerBody(t),
				},
			}...),
		},
		{
			name:      "response code != 200",
			chargerID: "123456",
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/chargers/123456/settings",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
					"Content-Type":  "application/*+json",
				},
				requestBody:  `{"dynamicChargerCurrent":0}`,
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        "123456",
			cfg:              exampleConfig(t),
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "return error if credentials are empty",
			chargerID: "123456",
			cfg:       &config.Config{},
			wantErr:   true,
		},
		{
			name:      "expired access token - refreshing it under the hood",
			chargerID: "123456",
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/accounts/refresh_token",
					requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
					requestHeaders: map[string]string{
						"Content-Type": "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: `{"accessToken":"new-access-token","expiresIn":86400,"accessClaims":["User"],"tokenType":"Bearer","refreshToken":"refresh-token"}`,
				},
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/123456/settings",
					requestHeaders: map[string]string{
						"Authorization": "Bearer new-access-token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"dynamicChargerCurrent":0}`,
					responseCode: http.StatusAccepted,
					responseBody: exampleCommandBody(t),
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer new-access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: successfulCheckerBody(t),
				},
			}...),
		},
		{
			name:      "refreshing expired access token failed",
			chargerID: "123456",
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/refresh_token",
				requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			cfgService := makeConfigService(t, tt.cfg)

			c := easee.NewClient(httpClient, cfgService, s.URL)

			err := c.StopCharging(tt.chargerID)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_ChargerState(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		cfg              *config.Config
		serverHandler    http.Handler
		forceServerError bool
		want             *easee.ChargerState
		wantErr          bool
	}{
		{
			name:      "successful call to Easee API",
			chargerID: "123456",
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/123456/state",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusOK,
				responseBody: marshal(t, exampleChargerState(t)),
			}),
			want: exampleChargerState(t),
		},
		{
			name:      "response code != 200",
			chargerID: "123456",
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/123456/state",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        "123456",
			cfg:              exampleConfig(t),
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "return error if credentials are empty",
			chargerID: "123456",
			cfg:       &config.Config{},
			wantErr:   true,
		},
		{
			name:      "expired access token - refreshing it under the hood",
			chargerID: "123456",
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/accounts/refresh_token",
					requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
					requestHeaders: map[string]string{
						"Content-Type": "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: `{"accessToken":"new-access-token","expiresIn":86400,"accessClaims":["User"],"tokenType":"Bearer","refreshToken":"refresh-token"}`,
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/chargers/123456/state",
					requestHeaders: map[string]string{
						"Authorization": "Bearer new-access-token",
					},
					responseCode: http.StatusOK,
					responseBody: marshal(t, exampleChargerState(t)),
				},
			}...),
			want: exampleChargerState(t),
		},
		{
			name:      "refreshing expired access token failed",
			chargerID: "123456",
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/refresh_token",
				requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			cfgService := makeConfigService(t, tt.cfg)

			c := easee.NewClient(httpClient, cfgService, s.URL)

			got, err := c.ChargerState(tt.chargerID)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_ChargerConfig(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		cfg              *config.Config
		serverHandler    http.Handler
		forceServerError bool
		want             *easee.ChargerConfig
		wantErr          bool
	}{
		{
			name:      "successful call to Easee API",
			chargerID: "123456",
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/123456/config",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusOK,
				responseBody: marshal(t, test.ExampleChargerConfig(t)),
			}),
			want: test.ExampleChargerConfig(t),
		},
		{
			name:      "response code != 200",
			chargerID: "123456",
			cfg:       exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/123456/config",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        "123456",
			cfg:              exampleConfig(t),
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "return error if credentials are empty",
			chargerID: "123456",
			cfg:       &config.Config{},
			wantErr:   true,
		},
		{
			name:      "expired access token - refreshing it under the hood",
			chargerID: "123456",
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/accounts/refresh_token",
					requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
					requestHeaders: map[string]string{
						"Content-Type": "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: `{"accessToken":"new-access-token","expiresIn":86400,"accessClaims":["User"],"tokenType":"Bearer","refreshToken":"refresh-token"}`,
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/chargers/123456/config",
					requestHeaders: map[string]string{
						"Authorization": "Bearer new-access-token",
					},
					responseCode: http.StatusOK,
					responseBody: marshal(t, test.ExampleChargerConfig(t)),
				},
			}...),
			want: test.ExampleChargerConfig(t),
		},
		{
			name:      "refreshing expired access token failed",
			chargerID: "123456",
			cfg:       exampleConfigWithExpiredCredentials(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/refresh_token",
				requestBody:   `{"accessToken":"access-token","refreshToken":"refresh-token"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			cfgService := makeConfigService(t, tt.cfg)

			c := easee.NewClient(httpClient, cfgService, s.URL)

			got, err := c.ChargerConfig(tt.chargerID)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Ping(t *testing.T) {
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		cfg              *config.Config
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name: "successful call to Easee API",
			cfg:  exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/health",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusOK,
			}),
		},
		{
			name: "response code != 200",
			cfg:  exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/health",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			cfg:              exampleConfig(t),
			forceServerError: true,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			cfgService := makeConfigService(t, tt.cfg)

			c := easee.NewClient(httpClient, cfgService, s.URL)

			err := c.Ping()
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_Chargers(t *testing.T) {
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		cfg              *config.Config
		serverHandler    http.Handler
		forceServerError bool
		want             []easee.Charger
		wantErr          bool
	}{
		{
			name: "successful call to Easee API",
			cfg:  exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusOK,
				responseBody: `[{"id":"EHFM4754","name":"EHFM4754","color":4,"createdOn":"2021-09-22T12:01:43.299176","updatedOn":"2022-01-13T12:33:03.232669","backPlate":null,"levelOfAccess":1,"productCode":1}]`,
			}),
			want: []easee.Charger{
				{
					ID:            "EHFM4754",
					Name:          "EHFM4754",
					Color:         4,
					CreatedOn:     "2021-09-22T12:01:43.299176",
					UpdatedOn:     "2022-01-13T12:33:03.232669",
					BackPlate:     easee.BackPlate{},
					LevelOfAccess: 1,
					ProductCode:   1,
				},
			},
		},
		{
			name: "response code != 200",
			cfg:  exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			cfg:              exampleConfig(t),
			forceServerError: true,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			cfgService := makeConfigService(t, tt.cfg)

			c := easee.NewClient(httpClient, cfgService, s.URL)

			got, err := c.Chargers()
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_SetCableLock(t *testing.T) {
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		locked           bool
		cfg              *config.Config
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name:   "successful cable lock",
			locked: true,
			cfg:    exampleConfig(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/chargers/123456/commands/lock_state",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"state":true}`,
					responseCode: http.StatusAccepted,
					responseBody: exampleCommandBody(t),
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: failedCheckerBody(t),
				},
				{
					// previous check failed, retry...
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: successfulCheckerBody(t),
				},
			}...),
		},
		{
			name:   "successful cable unlock",
			locked: false,
			cfg:    exampleConfig(t),
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/chargers/123456/commands/lock_state",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"state":false}`,
					responseCode: http.StatusAccepted,
					responseBody: exampleCommandBody(t),
				},
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: failedCheckerBody(t),
				},
				{
					// previous check failed, retry...
					requestMethod: http.MethodGet,
					requestPath:   "/api/commands/123456/48/637886435126844439",
					requestHeaders: map[string]string{
						"Authorization": "Bearer access-token",
						"Content-Type":  "application/*+json",
					},
					responseCode: http.StatusOK,
					responseBody: successfulCheckerBody(t),
				},
			}...),
		},
		{
			name:   "response code != 202",
			locked: true,
			cfg:    exampleConfig(t),
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/123456/commands/lock_state",
				requestHeaders: map[string]string{
					"Authorization": "Bearer access-token",
					"Content-Type":  "application/*+json",
				},
				requestBody:  `{"state":true}`,
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			cfg:              exampleConfig(t),
			forceServerError: true,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.serverHandler)
			t.Cleanup(func() {
				s.Close()
			})

			if tt.forceServerError {
				s.Close()
			}

			httpClient := &http.Client{Timeout: 3 * time.Second}
			cfgService := makeConfigService(t, tt.cfg)

			c := easee.NewClient(httpClient, cfgService, s.URL)

			err := c.SetCableLock(testChargerID, tt.locked)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

type call struct {
	requestMethod  string
	requestPath    string
	requestHeaders map[string]string
	requestBody    string

	responseCode int
	responseBody string
}

type testHandler struct {
	testingT       *testing.T
	calls          []call
	currentCallIdx int
}

func newTestHandler(t *testing.T, calls ...call) http.Handler {
	t.Helper()

	return &testHandler{
		testingT: t,
		calls:    calls,
	}
}

func (t *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	call := t.calls[t.currentCallIdx]
	t.currentCallIdx++

	if r.Method != call.requestMethod {
		t.testingT.Fatalf("request method mismatch: want: %s, got: %s", call.requestMethod, r.Method)
	}

	if r.URL.Path != call.requestPath {
		t.testingT.Fatalf("request path mismatch: want: %s, got: %s", call.requestPath, r.URL.Path)
	}

	if len(call.requestHeaders) != 0 {
		for k, v := range call.requestHeaders {
			got := r.Header.Get(k)

			if v != got {
				t.testingT.Fatalf("expected request header not found: header name: %s", k)
			}
		}
	}

	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	assert.NoError(t.testingT, err)

	if bodyString := string(b); bodyString != call.requestBody {
		t.testingT.Fatalf("incorrect request body: want: %s, got: %s", call.requestBody, bodyString)
	}

	w.WriteHeader(call.responseCode)
	_, err = w.Write([]byte(call.responseBody))
	assert.NoError(t.testingT, err)
}

func marshal(t *testing.T, v interface{}) string {
	t.Helper()

	b, err := json.Marshal(v)
	assert.NoError(t, err)

	return string(b)
}

func exampleConfig(t *testing.T) *config.Config {
	t.Helper()

	return &config.Config{
		Credentials: config.Credentials{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			ExpiresAt:    time.Date(2022, time.October, 24, 8, 00, 12, 00, time.UTC),
		},
		CommandCheckInterval: "5ms",
		CommandCheckTimeout:  "100ms",
		CommandCheckSleep:    "0ms",
	}
}

func exampleConfigWithExpiredCredentials(t *testing.T) *config.Config {
	t.Helper()

	cfg := exampleConfig(t)
	cfg.Credentials.ExpiresAt = time.Date(2022, time.April, 24, 8, 00, 12, 00, time.UTC)

	return cfg
}

func makeConfigService(t *testing.T, cfg *config.Config) *config.Service {
	t.Helper()

	storage := fakes.NewConfigStorage(cfg, config.Factory)

	return config.NewService(storage)
}

func exampleChargerState(t *testing.T) *easee.ChargerState {
	t.Helper()

	return &easee.ChargerState{
		ChargerOpMode:  easee.ChargerMode(3),
		TotalPower:     2,
		LifetimeEnergy: 1234,
		SessionEnergy:  234,
		Voltage:        200,
	}
}

func exampleCommandBody(t *testing.T) string {
	t.Helper()

	return `[{"device":"123456","commandId":48,"ticks":637886435126844439}]`
}

func successfulCheckerBody(t *testing.T) string {
	t.Helper()

	return `{"ticks": 637886435126844439, "id": 48, "timestamp": "2022-05-20T12:00:49.177859Z", "deliveredAt": "2022-05-20T12:00:49.96Z", "wasAccepted": true,"resultCode": 2, "comment": null}`
}

func failedCheckerBody(t *testing.T) string {
	t.Helper()

	return `{"ticks": 637886435126844439, "id": 48, "timestamp": "2022-05-20T11:38:32.6844439Z", "deliveredAt": null, "wasAccepted": false,"resultCode": 1, "comment": null}`
}
