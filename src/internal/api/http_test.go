package api_test

//nolint:godox
// TODO: refactor there test to use our internal HTTP testing package.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockedstorage "github.com/futurehomeno/cliffhanger/test/mocks/storage"
	"github.com/michalkurzeja/go-clock"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
)

func TestClient_Login(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		username         string
		password         string
		serverHandler    http.Handler
		forceServerError bool
		want             *api.Credentials
		wantErr          bool
	}{
		{
			name:     "successful call to Easee API",
			username: "test",
			password: "example",
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/accounts/login",
				requestBody:   `{"userName":"test","password":"example"}`,
				requestHeaders: map[string]string{
					"Content-Type": "application/*+json",
				},
				responseCode: http.StatusOK,
				responseBody: `{"accessToken":"access-token","expiresIn":86400,"accessClaims":["User"],"tokenType":"Bearer","refreshToken":"refresh-token"}`,
			}),
			want: &api.Credentials{
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
				requestPath:   "/api/accounts/login",
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
				requestPath:   "/api/accounts/login",
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

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := api.NewHTTPClient(cfgSrv, httpClient, s.URL)

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

func TestClient_RefreshToken(t *testing.T) { //nolint:paralleltest
	testCases := []struct {
		name          string
		baseURLAdj    string
		responseData  string
		statusCode    int
		errorContains string
		expectedCreds api.Credentials
	}{
		{
			name:          "should fail due to invalid url",
			baseURLAdj:    "invalid",
			errorContains: "failed to create token refresh request",
		},
		{
			name:          "should fail due to 401 error",
			statusCode:    http.StatusUnauthorized,
			errorContains: "but got 401",
		},
		{
			name:          "should fail when invalid body",
			responseData:  "string",
			statusCode:    http.StatusOK,
			errorContains: "could not read token",
		},
		{
			name:          "should form valid credentials",
			responseData:  `{"accessToken":"access","refreshToken":"refresh"}`,
			statusCode:    http.StatusOK,
			expectedCreds: api.Credentials{RefreshToken: "refresh", AccessToken: "access"},
		},
	}

	for _, v := range testCases { //nolint:paralleltest
		t.Run(v.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(v.statusCode)
				_, _ = w.Write([]byte(v.responseData))
			}

			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			client := api.NewHTTPClient(cfgSrv, server.Client(), server.URL+v.baseURLAdj)
			creds, err := client.RefreshToken("", "")

			if v.errorContains != "" {
				assert.Contains(t, err.Error(), v.errorContains)
				assert.Nil(t, creds)
			} else {
				assert.Equal(t, v.expectedCreds, *creds)
			}
		})
	}
}

func TestClient_UpdateMaxCurrent(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		accessToken      string
		serverHandler    http.Handler
		forceServerError bool
		current          float64
		wantErr          bool
	}{
		{
			name:        "successful call to Easee API",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/XX12345/settings",
					requestBody:   `{"maxChargerCurrent":10}`,
					requestHeaders: map[string]string{
						"Authorization": "Bearer test.access.token",
					},
					responseCode: http.StatusAccepted,
				},
			}...),
			current: 10,
		},
		{
			name:        "response code != 200",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/chargers/XX12345/settings",
				requestBody:   `{"maxChargerCurrent":10}`,
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			current: 10,
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        test.ChargerID,
			accessToken:      test.AccessToken,
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "return error if access token is empty",
			chargerID: test.ChargerID,
			wantErr:   true,
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

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := api.NewHTTPClient(cfgSrv, httpClient, s.URL)

			err := c.UpdateMaxCurrent(tt.accessToken, tt.chargerID, tt.current)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_UpdateDynamicCurrent(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		accessToken      string
		serverHandler    http.Handler
		forceServerError bool
		current          float64
		wantErr          bool
	}{
		{
			name:        "successful call to Easee API",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/XX12345/settings",
					requestBody:   `{"dynamicChargerCurrent":10}`,
					requestHeaders: map[string]string{
						"Authorization": "Bearer test.access.token",
					},
					responseCode: http.StatusAccepted,
				},
			}...),
			current: 10,
		},
		{
			name:        "response code != 200",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/chargers/XX12345/settings",
				requestBody:   `{"dynamicChargerCurrent":10}`,
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			current: 10,
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        test.ChargerID,
			accessToken:      test.AccessToken,
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "return error if access token is empty",
			chargerID: test.ChargerID,
			wantErr:   true,
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

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := api.NewHTTPClient(cfgSrv, httpClient, s.URL)

			err := c.UpdateDynamicCurrent(tt.accessToken, tt.chargerID, tt.current)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_StopCharging(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		accessToken      string
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name:        "successful call to Easee API",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/XX12345/commands/pause_charging",
					requestHeaders: map[string]string{
						"Authorization": "Bearer test.access.token",
					},
					responseCode: http.StatusAccepted,
				},
			}...),
		},
		{
			name:        "response code != 200",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/chargers/XX12345/commands/pause_charging",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        test.ChargerID,
			accessToken:      test.AccessToken,
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:        "return error if access token is empty",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			wantErr:     true,
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

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := api.NewHTTPClient(cfgSrv, httpClient, s.URL)

			err := c.StopCharging(tt.accessToken, tt.chargerID)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_ChargerConfig(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		accessToken      string
		serverHandler    http.Handler
		forceServerError bool
		want             *api.ChargerConfig
		wantErr          bool
	}{
		{
			name:        "successful call to Easee API",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/XX12345/config",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusOK,
				responseBody: `{"maxChargerCurrent":32, "detectedPowerGridType":1}`,
			}),
			want: &api.ChargerConfig{
				DetectedPowerGridType: 1,
			},
		},
		{
			name:        "response code != 200",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/XX12345/config",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        test.ChargerID,
			accessToken:      test.AccessToken,
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "return error if access token is empty",
			chargerID: test.ChargerID,
			wantErr:   true,
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

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := api.NewHTTPClient(cfgSrv, httpClient, s.URL)

			got, err := c.ChargerConfig(tt.accessToken, tt.chargerID)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_Ping(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		accessToken      string
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name:        "successful call to Easee API",
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/health",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusOK,
			}),
		},
		{
			name:        "response code != 200",
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/health",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			accessToken:      test.AccessToken,
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:    "error - access token empty",
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

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := api.NewHTTPClient(cfgSrv, httpClient, s.URL)

			err := c.Ping(tt.accessToken)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestClient_Chargers(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		accessToken      string
		serverHandler    http.Handler
		forceServerError bool
		want             []api.Charger
		wantErr          bool
	}{
		{
			name:        "successful call to Easee API",
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusOK,
				responseBody: `[{"id":"XX12345","name":"XX12345","color":4,"createdOn":"2021-09-22T12:01:43.299176","updatedOn":"2022-01-13T12:33:03.232669","backPlate":null,"levelOfAccess":1,"productCode":1}]`,
			}),
			want: []api.Charger{
				{
					ID:            test.ChargerID,
					Name:          test.ChargerID,
					Color:         4,
					CreatedOn:     "2021-09-22T12:01:43.299176",
					UpdatedOn:     "2022-01-13T12:33:03.232669",
					BackPlate:     api.BackPlate{},
					LevelOfAccess: 1,
					ProductCode:   1,
				},
			},
		},
		{
			name:        "response code != 200",
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
				},
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			accessToken:      test.AccessToken,
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:    "error - access token empty",
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

			storage := mockedstorage.Storage[*config.Config]{}

			cfgSrv := config.NewConfigServiceWithStorage(&storage)

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := api.NewHTTPClient(cfgSrv, httpClient, s.URL)

			got, err := c.Chargers(tt.accessToken)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
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

	if r.URL.EscapedPath() != call.requestPath {
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
