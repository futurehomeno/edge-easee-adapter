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

	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
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
		want             *easee.Credentials
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

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := easee.NewHTTPClient(httpClient, s.URL)

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
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		accessToken      string
		current          float64
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name:        "successful call to Easee API",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			current:     40,
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodPost,
					requestPath:   "/api/chargers/XX12345/settings",
					requestHeaders: map[string]string{
						"Authorization": "Bearer test.access.token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"dynamicChargerCurrent":40}`,
					responseCode: http.StatusAccepted,
				},
			}...),
		},
		{
			name:        "response code != 200",
			chargerID:   test.ChargerID,
			accessToken: test.AccessToken,
			current:     40,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodPost,
				requestPath:   "/api/chargers/XX12345/settings",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
					"Content-Type":  "application/*+json",
				},
				requestBody:  `{"dynamicChargerCurrent":40}`,
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        test.ChargerID,
			accessToken:      test.AccessToken,
			current:          40,
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

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := easee.NewHTTPClient(httpClient, s.URL)

			err := c.StartCharging(tt.accessToken, tt.chargerID, tt.current)
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
					requestPath:   "/api/chargers/XX12345/settings",
					requestHeaders: map[string]string{
						"Authorization": "Bearer test.access.token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"dynamicChargerCurrent":0}`,
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
				requestPath:   "/api/chargers/XX12345/settings",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
					"Content-Type":  "application/*+json",
				},
				requestBody:  `{"dynamicChargerCurrent":0}`,
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

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := easee.NewHTTPClient(httpClient, s.URL)

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
		want             *easee.ChargerConfig
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
				responseBody: marshal(t, test.ExampleChargerConfig(t)),
			}),
			want: test.ExampleChargerConfig(t),
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

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := easee.NewHTTPClient(httpClient, s.URL)

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

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := easee.NewHTTPClient(httpClient, s.URL)

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
		want             []easee.Charger
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
			want: []easee.Charger{
				{
					ID:            test.ChargerID,
					Name:          test.ChargerID,
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

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := easee.NewHTTPClient(httpClient, s.URL)

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

func TestClient_SetCableLock(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0o0, 12, 0o0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name             string
		chargerID        string
		locked           bool
		accessToken      string
		serverHandler    http.Handler
		forceServerError bool
		wantErr          bool
	}{
		{
			name:        "successful cable lock",
			chargerID:   test.ChargerID,
			locked:      true,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/chargers/XX12345/commands/lock_state",
					requestHeaders: map[string]string{
						"Authorization": "Bearer test.access.token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"state":true}`,
					responseCode: http.StatusAccepted,
				},
			}...),
		},
		{
			name:        "successful cable unlock",
			chargerID:   test.ChargerID,
			locked:      false,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, []call{
				{
					requestMethod: http.MethodGet,
					requestPath:   "/api/chargers/XX12345/commands/lock_state",
					requestHeaders: map[string]string{
						"Authorization": "Bearer test.access.token",
						"Content-Type":  "application/*+json",
					},
					requestBody:  `{"state":false}`,
					responseCode: http.StatusAccepted,
				},
			}...),
		},
		{
			name:        "response code != 202",
			chargerID:   test.ChargerID,
			locked:      true,
			accessToken: test.AccessToken,
			serverHandler: newTestHandler(t, call{
				requestMethod: http.MethodGet,
				requestPath:   "/api/chargers/XX12345/commands/lock_state",
				requestHeaders: map[string]string{
					"Authorization": "Bearer test.access.token",
					"Content-Type":  "application/*+json",
				},
				requestBody:  `{"state":true}`,
				responseCode: http.StatusInternalServerError,
			}),
			wantErr: true,
		},
		{
			name:             "http client error",
			chargerID:        test.ChargerID,
			locked:           true,
			accessToken:      test.AccessToken,
			forceServerError: true,
			wantErr:          true,
		},
		{
			name:      "error - access token empty",
			chargerID: test.ChargerID,
			locked:    true,
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

			httpClient := &http.Client{Timeout: 3 * time.Second}
			c := easee.NewHTTPClient(httpClient, s.URL)

			err := c.SetCableLock(tt.accessToken, tt.chargerID, tt.locked)
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

func marshal(t *testing.T, v interface{}) string {
	t.Helper()

	b, err := json.Marshal(v)
	assert.NoError(t, err)

	return string(b)
}
