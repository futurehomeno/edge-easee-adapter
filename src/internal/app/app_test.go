package app_test

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffApp "github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/selection"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	mockapi "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/api"
	mockeddb "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/db"
	mockedmanifest "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/manifest"
	mocksignalr "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/signalr"
)

//nolint:godox
// TODO: Move as much test cases as possible to component tests to avoid excessive mocking.

func TestApplication_GetManifest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockLoader func(l *mockedmanifest.Loader)
		want       *manifest.Manifest
		wantErr    bool
	}{
		{
			name: "manifest is loaded successfully",
			mockLoader: func(l *mockedmanifest.Loader) {
				l.On("Load").Return(test.LoadManifest(t), nil)
			},
			want: test.LoadManifest(t),
		},
		{
			name: "manifest loading fails",
			mockLoader: func(l *mockedmanifest.Loader) {
				l.On("Load").Return(nil, errors.New("failed to load manifest"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loaderMock := mockedmanifest.NewLoader(t)
			if tt.mockLoader != nil {
				tt.mockLoader(loaderMock)
			}

			a := app.New(nil, nil, lifecycle.New(nil), loaderMock, nil, nil, nil, nil, nil)

			got, err := a.GetManifest()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplication_Configure_RejectsWrongModelType(t *testing.T) {
	t.Parallel()

	a := app.New(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	assert.Error(t, a.Configure("anything"))
}

func TestApplication_Uninstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		credentials         config.Credentials
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		configAssertions    func(c *config.Config)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			configAssertions: func(c *config.Config) {
				assert.Equal(t, &config.Config{}, c)
			},
		},
		{
			name: "adapter error on destroying all things",
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("DestroyAllThings").Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New(nil)
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(mockedadapter.Adapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			defer adapterMock.AssertExpectations(t)

			storage := fakes.NewConfigStorage(t, tt.cfg, config.Factory)
			cfgService := config.NewService(storage)

			credentials := newCredentialsStore(t, tt.credentials)
			sessionStorage := mockeddb.NewChargingSessionStorage(t)
			sessionStorage.On("Reset").Return(nil)
			application := app.New(adapterMock, cfgService, lc, nil, nil, nil, nil, credentials, sessionStorage)

			err := application.Uninstall()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)

			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}

			if tt.configAssertions != nil {
				tt.configAssertions(cfgService.Model())
			}
		})
	}
}

// The error hook lives on the global logger, so building a second application must not
// attach a second one - every warning would then land in the diag report twice.
func TestApplication_New_AttachesTheErrorHookOnce(t *testing.T) { //nolint:paralleltest
	newDiagApp(t)
	attached := len(log.StandardLogger().Hooks[log.WarnLevel])

	application := newDiagApp(t)
	assert.Len(t, log.StandardLogger().Hooks[log.WarnLevel], attached, "a second application must reuse the hook")

	const marker = "warning-recorded-once-marker"

	log.Warn(marker)

	report, err := application.ErrorsReport()
	require.NoError(t, err)

	assert.Equal(t, 1, strings.Count(strings.Join(report, "\n"), marker))
}

func newDiagApp(t *testing.T) app.ApplicationWithToken {
	t.Helper()

	return app.New(mockedadapter.NewAdapter(t), config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory)),
		lifecycle.New(nil), nil, nil, nil, nil, newCredentialsStore(t, config.Credentials{}), nil)
}

// Charging sessions are keyed by charger ID alone, so a reinstall against a different Easee
// account with a colliding ID would serve the previous account's sessions.
func TestApplication_Uninstall_ResetsSessionStorage(t *testing.T) {
	t.Parallel()

	adapterMock := mockedadapter.NewAdapter(t)
	adapterMock.On("DestroyAllThings").Return(nil)

	sessionStorage := mockeddb.NewChargingSessionStorage(t)
	sessionStorage.On("Reset").Return(errors.New("disk gone"))

	lc := lifecycle.New(nil)
	lc.MarkRunning()

	application := app.New(adapterMock, config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory)),
		lc, nil, nil, nil, nil, newCredentialsStore(t, config.Credentials{}), sessionStorage)

	assert.ErrorContains(t, application.Uninstall(), "reset session storage")
	assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth(),
		"a failed reset must not stop the app being marked not configured")
	sessionStorage.AssertCalled(t, "Reset")
}

// Credentials live outside the configuration, so no failure on the way may leave the Easee
// tokens on a hub the user has just uninstalled the app from.
func TestApplication_Uninstall_ClearsCredentialsDespiteFailures(t *testing.T) {
	t.Parallel()

	adapterMock := mockedadapter.NewAdapter(t)
	adapterMock.On("DestroyAllThings").Return(errors.New("oops"))

	credentials := newCredentialsStore(t, config.Credentials{AccessToken: "access", RefreshToken: "refresh"})

	lc := lifecycle.New(nil)
	lc.MarkRunning()

	sessionStorage := mockeddb.NewChargingSessionStorage(t)
	sessionStorage.On("Reset").Return(nil)

	application := app.New(adapterMock, config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory)),
		lc, nil, nil, nil, nil, credentials, sessionStorage)

	assert.ErrorContains(t, application.Uninstall(), "oops")
	assert.True(t, credentials.Credentials().Empty(), "the tokens must be gone even when a step failed")

	// The configuration and the tokens are gone, so the app must not stay marked running.
	assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
	assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
	assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
	assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
}

func TestApplication_Login(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC))
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name                string
		loginData           *cliffApp.LoginCredentials
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		mockClient          func(c *mockapi.Client)
		mockAuthenticator   func(a *mockapi.Authenticator)
		mockSignalRClient   func(c *mocksignalr.Client)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
	}{
		{
			name: "if login was successful, credentials and lifecycle should be set up",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Chargers").Return([]model.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("ChargerDetails", "123").Return(model.ChargerDetails{Product: "xd"}, nil)
				c.On("ChargerDetails", "456").Return(model.ChargerDetails{Product: "edi"}, nil)
				c.On("Ping").Return(nil)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("EnsureThings", adapter.ThingSeeds{
					&adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID: "123",
							Product:   "xd",
						},
					},
					&adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID: "456",
							Product:   "edi",
						},
					},
				}).Return(nil)
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Start")
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "if Easee API returned an error, login procedure should be skipped with no side effects on config",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.
					On("Login", "test-user", "test-password").
					Return(errors.New("oops"))
			},
			wantErr: true,
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
		},
		{
			name: "successful login, but ping failed for some reason",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Chargers").Return([]model.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("ChargerDetails", "123").Return(model.ChargerDetails{Product: "xd"}, nil)
				c.On("ChargerDetails", "456").Return(model.ChargerDetails{Product: "edi"}, nil)
				c.On("Ping").Return(errors.New("oops"))
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("EnsureThings", adapter.ThingSeeds{
					&adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID: "123",
							Product:   "xd",
						},
					},
					&adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID: "456",
							Product:   "edi",
						},
					},
				}).Return(nil)
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Start")
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "failed to register all things",
			loginData: &cliffApp.LoginCredentials{
				Username: "test-user",
				Password: "test-password",
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mockapi.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Chargers").Return([]model.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("ChargerDetails", "123").Return(model.ChargerDetails{Product: "xd"}, nil)
				c.On("ChargerDetails", "456").Return(model.ChargerDetails{Product: "edi"}, nil)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("EnsureThings", adapter.ThingSeeds{
					&adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID: "123",
							Product:   "xd",
						},
					},
					&adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID: "456",
							Product:   "edi",
						},
					},
				}).Return(errors.New("oops"))
			},
			// The seeds survive a partial sync, so the client is started for whatever did get
			// created - the login still reports the failure.
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Start").Maybe()
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			lc := lifecycle.New(nil)
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := mockedadapter.NewAdapter(t)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			clientMock := mockapi.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			authMock := mockapi.NewAuthenticator(t)
			if tt.mockAuthenticator != nil {
				tt.mockAuthenticator(authMock)
			}

			signalRClientMock := mocksignalr.NewClient(t)
			if tt.mockSignalRClient != nil {
				tt.mockSignalRClient(signalRClientMock)
			}

			// Read by the selection adoption on login.
			adapterMock.On("Things").Return([]adapter.Thing{}).Maybe()

			cfgService := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
			application := app.New(adapterMock, cfgService, lc, nil, clientMock, authMock, signalRClientMock, newCredentialsStore(t, config.Credentials{}), nil)

			err := application.Login(tt.loginData)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}
		})
	}
}

func TestApplication_Logout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		setLifecycle        func(lc *lifecycle.Lifecycle)
		authLogoutError     error
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		mockSignalRClient   func(c *mocksignalr.Client)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Close").Return(nil)
			},
		},
		{
			name: "adapter error on destroying all things",
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			authLogoutError: errors.New("error"),
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthError, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			mockSignalRClient: func(c *mocksignalr.Client) {
				c.On("Close").Return(nil)
			},

			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New(nil)
			tt.setLifecycle(lc)

			clientMock := new(mockapi.Client)
			clientMock.On("Ping").Return(errors.New("oops"))

			authMock := &mockapi.Authenticator{}
			authMock.On("Logout").Return(tt.authLogoutError)

			signalRClientMock := mocksignalr.NewClient(t)
			if tt.mockSignalRClient != nil {
				tt.mockSignalRClient(signalRClientMock)
			}

			cfgService := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
			application := app.New(nil, cfgService, lc, nil, clientMock, authMock, signalRClientMock, newCredentialsStore(t, config.Credentials{}), nil)
			err := application.Logout()

			assert.Equal(t, tt.wantErr, err != nil, "failed error expectation")
			tt.lifecycleAssertions(lc)
		})
	}
}

func TestApplication_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		credentials         config.Credentials
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		mockClient          func(c *mockapi.Client)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
	}{
		{
			name: "successful thing initialization",
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Ping").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "empty credentials - unconfigure lifecycle",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
		},
		{
			name: "error on thing initialization",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(errors.New("oops"))
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthNotConfigured, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			wantErr: true,
		},
		{
			name: "successful thing initialization, but ping failed",
			cfg:  &config.Config{},
			credentials: config.Credentials{
				AccessToken:          "access-token",
				RefreshToken:         "refresh-token",
				AccessTokenExpiresAt: time.Date(2022, time.September, 10, 8, 0, 12, 0, time.UTC),
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppHealth(lifecycle.AppHealthNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *mockapi.Client) {
				c.On("Ping").Return(errors.New("oops"))
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppHealthRunning, lc.AppHealth())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateDisconnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			lc := lifecycle.New(nil)
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := mockedadapter.NewAdapter(t)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			// Read by the boot-time selection adoption.
			adapterMock.On("Things").Return([]adapter.Thing{}).Maybe()

			clientMock := new(mockapi.Client)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			// Initialize re-seeds when credentials exist but no things do, which these cases
			// do not exercise; a failure there is logged and does not affect the outcome.
			clientMock.On("Chargers").Return(nil, errors.New("not under test")).Maybe()

			defer func() {
				adapterMock.AssertExpectations(t)
				clientMock.AssertExpectations(t)
			}()

			storage := fakes.NewConfigStorage(t, tt.cfg, config.Factory)
			cfgService := config.NewService(storage)

			credentials := newCredentialsStore(t, tt.credentials)
			application := app.New(adapterMock, cfgService, lc, nil, clientMock, nil, nil, credentials, nil)

			err := application.Initialize()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)

			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}
		})
	}
}

func newCredentialsStore(t *testing.T, credentials config.Credentials) *config.CredentialsStore {
	t.Helper()

	return config.NewCredentialsStoreWithStorage(
		fakes.NewConfigStorage(t, &credentials, func() *config.Credentials { return &config.Credentials{} }),
	)
}

// The selection is persisted as a plain list where empty means "every charger", while
// adapter.SyncThings reads nil as "every device" and empty as "no devices". These tests pin
// that translation, the policies guarding it, and the boot-time adoption that gives installs
// upgraded from a version without a selection an explicit one.

func TestApplication_Configure_Selection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		chargers     []model.Charger
		chargersErr  error
		owned        []string
		ensureErr    error
		selected     selection.Selection
		wantErr      string
		wantSeeded   []string
		wantSelected selection.Selection
	}{
		{
			name:         "valid subset is seeded and persisted",
			chargers:     []model.Charger{{ID: "123"}, {ID: "456"}},
			selected:     []string{"456"},
			wantSeeded:   []string{"456"},
			wantSelected: []string{"456"},
		},
		{
			name:         "an absent selection includes every charger and is materialised",
			chargers:     []model.Charger{{ID: "123"}, {ID: "456"}},
			wantSeeded:   []string{"123", "456"},
			wantSelected: []string{"123", "456"},
		},
		{
			// Distinct from the case above: the user unticked every charger, which is
			// obeyed rather than read as an install that was never configured.
			name:         "an explicit empty selection includes no charger",
			chargers:     []model.Charger{{ID: "123"}, {ID: "456"}},
			selected:     selection.Selection{},
			wantSelected: selection.Selection{},
		},
		{
			name:         "an absent selection keeps the chargers already seeded",
			chargers:     []model.Charger{{ID: "123"}, {ID: "456"}},
			owned:        []string{"123"},
			wantSeeded:   []string{"123"},
			wantSelected: []string{"123"},
		},
		{
			name:         "an absent selection drops owned chargers Easee no longer lists",
			chargers:     []model.Charger{{ID: "123"}},
			owned:        []string{"123", "gone"},
			wantSeeded:   []string{"123"},
			wantSelected: []string{"123"},
		},
		{
			name:     "unknown device id is rejected before anything is mutated",
			chargers: []model.Charger{{ID: "123"}},
			selected: []string{"unknown"},
			wantErr:  "unknown device IDs",
		},
		{
			name:        "a failed fetch mutates nothing",
			chargersErr: errors.New("oops"),
			selected:    []string{"123"},
			wantErr:     "fetch available chargers",
		},
		{
			name:       "a failed seed does not persist the selection",
			chargers:   []model.Charger{{ID: "123"}},
			selected:   []string{"123"},
			ensureErr:  errors.New("oops"),
			wantSeeded: []string{"123"},
			wantErr:    "sync things",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newSelectionHarness(t, tt.chargers, tt.chargersErr, tt.owned, tt.ensureErr)

			err := h.app.Configure(&config.Config{
				PublicConfig: config.PublicConfig{SelectedDevices: tt.selected},
			})

			assert.Equal(t, tt.wantSeeded, h.seeded())

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, h.cfg.SelectedDevices(), "a failed configure must not persist a selection")

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantSelected, h.cfg.SelectedDevices())
		})
	}
}

func TestApplication_Login_Selection(t *testing.T) {
	t.Parallel()

	eleven := make([]model.Charger, 0, 11)
	for i := range 11 {
		eleven = append(eleven, model.Charger{ID: strconv.Itoa(i)})
	}

	tests := []struct {
		name         string
		chargers     []model.Charger
		persisted    selection.Selection
		wantSeeded   []string
		wantSelected selection.Selection
	}{
		{
			// Materialised rather than left empty, so a later cmd.thing.delete can
			// express the exclusion instead of being undone by the next login.
			name:         "no selection seeds every charger and is materialised",
			chargers:     []model.Charger{{ID: "123"}, {ID: "456"}},
			wantSeeded:   []string{"123", "456"},
			wantSelected: []string{"123", "456"},
		},
		{
			name:         "a persisted selection is honoured",
			chargers:     []model.Charger{{ID: "123"}, {ID: "456"}},
			persisted:    []string{"123"},
			wantSeeded:   []string{"123"},
			wantSelected: []string{"123"},
		},
		{
			name:         "an unconfigured install auto-selects up to the cap",
			chargers:     eleven,
			wantSeeded:   []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
			wantSelected: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newSelectionHarness(t, tt.chargers, nil, nil, nil)

			if tt.persisted != nil {
				require.NoError(t, h.cfg.SetSelectedDevices(tt.persisted))
			}

			require.NoError(t, h.login())

			assert.Equal(t, tt.wantSeeded, h.seeded())
			assert.Equal(t, tt.wantSelected, h.cfg.SelectedDevices())
		})
	}
}

// A charger missing from the list must not be re-seeded away on the first attempt: the sync
// destroys every thing absent from the seeds, so a truncated response would silently drop it.
func TestApplication_Login_MissingSelectedRetriesThenProceeds(t *testing.T) {
	t.Parallel()

	h := newSelectionHarness(t, []model.Charger{{ID: "123"}}, nil, nil, nil)
	require.NoError(t, h.cfg.SetSelectedDevices([]string{"123", "gone"}))

	for range 3 {
		err := h.login()
		require.ErrorContains(t, err, "gone")
		assert.Empty(t, h.seeded(), "nothing may be seeded while the charger might still come back")
	}

	// A different missing set gets a fresh budget rather than the exhausted one.
	require.NoError(t, h.cfg.SetSelectedDevices([]string{"123", "other"}))

	for range 3 {
		require.ErrorContains(t, h.login(), "other")
	}

	require.NoError(t, h.login())
	assert.Equal(t, []string{"123"}, h.seeded())

	// The vanished charger is dropped from the selection, so the next login does not
	// spend the budget on it all over again.
	assert.Equal(t, selection.Selection{"123"}, h.cfg.SelectedDevices())
	require.NoError(t, h.login())
	assert.Equal(t, []string{"123"}, h.seeded())
}

// Cleaning out the last entry leaves an empty selection, which means "no chargers" - the
// narrowed selection must not widen back to "every charger" on the login after that.
func TestApplication_Login_MissingSelectedDropsLastEntry(t *testing.T) {
	t.Parallel()

	h := newSelectionHarness(t, []model.Charger{{ID: "123"}}, nil, nil, nil)
	require.NoError(t, h.cfg.SetSelectedDevices(selection.Selection{"gone"}))

	for range 3 {
		require.ErrorContains(t, h.login(), "gone")
	}

	require.NoError(t, h.login())
	assert.Empty(t, h.seeded(), "the only selected charger is gone, so nothing may be seeded")
	assert.Equal(t, selection.Selection{}, h.cfg.SelectedDevices())

	require.NoError(t, h.login())
	assert.Empty(t, h.seeded(), "an empty selection means no chargers, not every charger")
	assert.Equal(t, selection.Selection{}, h.cfg.SelectedDevices())
}

func TestApplication_Initialize_AdoptSeededSelection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		credentials  config.Credentials
		owned        []string
		persisted    selection.Selection
		wantSelected selection.Selection
	}{
		{
			name:         "an upgraded install adopts the chargers it already seeded",
			credentials:  config.Credentials{AccessToken: "token", RefreshToken: "refresh"},
			owned:        []string{"123", "456"},
			wantSelected: []string{"123", "456"},
		},
		{
			name:         "an existing selection is left alone",
			credentials:  config.Credentials{AccessToken: "token", RefreshToken: "refresh"},
			owned:        []string{"123", "456"},
			persisted:    []string{"123"},
			wantSelected: []string{"123"},
		},
		{
			name:        "a fresh install adopts nothing",
			credentials: config.Credentials{AccessToken: "token", RefreshToken: "refresh"},
		},
		{
			name:  "a logged out install adopts nothing",
			owned: []string{"123"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newSelectionHarness(t, nil, nil, tt.owned, nil)
			h.credentials = tt.credentials

			if tt.persisted != nil {
				require.NoError(t, h.cfg.SetSelectedDevices(tt.persisted))
			}

			h.adapter.On("InitializeThings").Return(nil)
			h.client.On("Ping").Return(nil).Maybe()

			require.NoError(t, h.newApp().Initialize())
			assert.Equal(t, tt.wantSelected, h.cfg.SelectedDevices())
		})
	}
}

// The re-seed calls the cloud, so an expired refresh token makes it trigger an auth loss whose
// logout lands on its own goroutine. The lifecycle therefore has to be marked before the
// re-seed rather than after it: marking afterwards would overwrite that logout and leave the
// app claiming a session it no longer has, with nothing to re-fire it - cleared credentials
// report "not logged in" instead of another auth loss.
func TestApplication_Initialize_MarksLifecycleBeforeReSeeding(t *testing.T) {
	t.Parallel()

	adapterMock := mockedadapter.NewAdapter(t)
	adapterMock.On("InitializeThings").Return(nil)
	adapterMock.On("Things").Return([]adapter.Thing{})

	lc := lifecycle.New(nil)

	var authWhileReSeeding lifecycle.State

	clientMock := mockapi.NewClient(t)
	clientMock.On("Ping").Return(nil).Maybe()
	clientMock.On("Chargers").
		Run(func(mock.Arguments) { authWhileReSeeding = lc.AuthState() }).
		Return(nil, errors.New("not logged in"))

	signalRClient := mocksignalr.NewClient(t)
	signalRClient.On("Start").Maybe()

	application := app.New(
		adapterMock,
		config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory)),
		lc, nil, clientMock, mockapi.NewAuthenticator(t), signalRClient,
		newCredentialsStore(t, config.Credentials{AccessToken: "token", RefreshToken: "refresh"}),
		nil,
	)

	require.NoError(t, application.Initialize())

	assert.Equal(t, lifecycle.AuthStateAuthenticated, authWhileReSeeding,
		"the lifecycle must be marked before the re-seed, so an auth loss it triggers is not overwritten")
}

// EnsureThings seeds per charger and joins the failures, so a charger that failed to create
// leaves the others behind and a re-seed gated on zero things never fires. cliffhanger
// excludes it "until the next sync" - which for this adapter never comes, because
// configureChargers has no caller but Login and Check() is a no-op.
func TestApplication_Initialize_ReSeedsWhenSelectedChargerHasNoThing(t *testing.T) {
	t.Parallel()

	thing := mockedadapter.NewThing(t)

	adapterMock := mockedadapter.NewAdapter(t)
	adapterMock.On("InitializeThings").Return(nil)
	adapterMock.On("Things").Return([]adapter.Thing{thing})
	adapterMock.On("ThingByID", "123").Return(thing)
	adapterMock.On("ThingByID", "456").Return(nil)

	reSeeded := false

	clientMock := mockapi.NewClient(t)
	clientMock.On("Ping").Return(nil).Maybe()
	clientMock.On("Chargers").
		Run(func(mock.Arguments) { reSeeded = true }).
		Return(nil, errors.New("service unavailable"))

	cfg := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
	require.NoError(t, cfg.SetSelectedDevices(selection.Selection{"123", "456"}))

	application := app.New(
		adapterMock, cfg, lifecycle.New(nil), nil, clientMock, mockapi.NewAuthenticator(t),
		mocksignalr.NewClient(t),
		newCredentialsStore(t, config.Credentials{AccessToken: "token", RefreshToken: "refresh"}),
		nil,
	)

	require.NoError(t, application.Initialize())

	assert.True(t, reSeeded, "a selected charger without a thing must re-seed, not just an empty adapter")
}

type selectionHarness struct {
	t           *testing.T
	app         app.ApplicationWithToken
	cfg         *config.Service
	adapter     *mockedadapter.Adapter
	client      *mockapi.Client
	auth        *mockapi.Authenticator
	credentials config.Credentials

	lock sync.Mutex
	ids  []string
}

func newSelectionHarness(t *testing.T, chargers []model.Charger, chargersErr error, owned []string, ensureErr error) *selectionHarness {
	t.Helper()

	h := &selectionHarness{
		t:       t,
		cfg:     config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory)),
		adapter: mockedadapter.NewAdapter(t),
		client:  mockapi.NewClient(t),
		auth:    mockapi.NewAuthenticator(t),
	}

	things := make([]adapter.Thing, 0, len(owned))

	for _, id := range owned {
		thing := mockedadapter.NewThing(t)
		thing.On("InclusionReport").Return(&fimptype.ThingInclusionReport{DeviceId: id}).Maybe()
		things = append(things, thing)
	}

	h.adapter.On("Things").Return(things).Maybe()

	for i, id := range owned {
		h.adapter.On("ThingByID", id).Return(things[i]).Maybe()
	}

	h.adapter.On("ThingByID", mock.Anything).Return(nil).Maybe()

	h.client.On("Chargers").Return(chargers, chargersErr).Maybe()

	for _, charger := range chargers {
		h.client.On("ChargerDetails", charger.ID).Return(model.ChargerDetails{}, nil).Maybe()
	}

	// Only reached for a selected charger the fetch no longer lists.
	h.adapter.On("ExchangeID", mock.Anything).Return("", false).Maybe()
	h.adapter.On("ExchangeAddress", mock.Anything).Return("", false).Maybe()
	h.adapter.On("DestroyThingByAddress", mock.Anything).Return(nil).Maybe()

	h.adapter.On("EnsureThings", mock.Anything).Run(func(args mock.Arguments) {
		h.lock.Lock()
		defer h.lock.Unlock()

		h.ids = nil
		for _, seed := range args.Get(0).(adapter.ThingSeeds) { //nolint:forcetypeassert
			h.ids = append(h.ids, seed.ID)
		}
	}).Return(ensureErr).Maybe()

	h.app = h.newApp()

	return h
}

// newApp rebuilds the application, e.g. after the harness credentials changed.
func (h *selectionHarness) newApp() app.ApplicationWithToken {
	h.t.Helper()

	signalRClient := mocksignalr.NewClient(h.t)
	signalRClient.On("Start").Maybe()

	h.app = app.New(h.adapter, h.cfg, lifecycle.New(nil), nil, h.client, h.auth, signalRClient,
		newCredentialsStore(h.t, h.credentials), nil)

	return h.app
}

func (h *selectionHarness) login() error {
	h.t.Helper()

	h.auth.On("Login", "user", "password").Return(nil).Maybe()
	h.client.On("Ping").Return(nil).Maybe()

	h.lock.Lock()
	h.ids = nil
	h.lock.Unlock()

	return h.app.Login(&cliffApp.LoginCredentials{Username: "user", Password: "password"})
}

func (h *selectionHarness) seeded() []string {
	h.lock.Lock()
	defer h.lock.Unlock()

	return h.ids
}

// A nil selection means "every charger", so applyChargers must never hand one back when the
// sync did not run. A transient ChargerDetails failure used to erase an explicit selection and
// silently widen the next successful sync to the whole account.
func TestApplication_Login_ChargerDetailsFailureKeepsSelection(t *testing.T) {
	t.Parallel()

	selected := selection.Selection{"123", "456"}

	adapterMock := mockedadapter.NewAdapter(t)
	adapterMock.On("Things").Return([]adapter.Thing{}).Maybe()

	clientMock := mockapi.NewClient(t)
	clientMock.On("Chargers").Return([]model.Charger{{ID: "123"}, {ID: "456"}}, nil)
	clientMock.On("ChargerDetails", mock.Anything).Return(model.ChargerDetails{}, errors.New("internal server error"))
	clientMock.On("Ping").Return(nil).Maybe()

	authMock := mockapi.NewAuthenticator(t)
	authMock.On("Login", "user", "password").Return(nil)

	signalRClient := mocksignalr.NewClient(t)
	signalRClient.On("Start").Maybe()

	cfg := config.NewService(fakes.NewConfigStorage(t, &config.Config{}, config.Factory))
	require.NoError(t, cfg.SetSelectedDevices(selected))

	application := app.New(
		adapterMock, cfg, lifecycle.New(nil), nil, clientMock, authMock, signalRClient,
		newCredentialsStore(t, config.Credentials{AccessToken: "token", RefreshToken: "refresh"}),
		nil,
	)

	require.Error(t, application.Login(&cliffApp.LoginCredentials{Username: "user", Password: "password"}))
	assert.Equal(t, selected, cfg.SelectedDevices(), "a failed detail fetch must not widen the selection to every charger")
}

// An upgrade that happens while logged out never reaches the boot-time adoption, so the
// first login must adopt before the cap runs - otherwise chargers this hub already had,
// but which sit outside the first maxAutoSelected of the account, are destroyed.
func TestApplication_Login_AdoptsSeededSelectionWhenLoggedOut(t *testing.T) {
	t.Parallel()

	chargers := make([]model.Charger, 0, 12)
	for i := range 12 {
		chargers = append(chargers, model.Charger{ID: strconv.Itoa(i)})
	}

	owned := []string{"0", "11"}

	h := newSelectionHarness(t, chargers, nil, owned, nil)

	require.NoError(t, h.login())

	assert.Equal(t, owned, h.seeded(), "the chargers this hub already had must survive the cap")
	assert.Equal(t, selection.Selection(owned), h.cfg.SelectedDevices())
}

// The budget keys on the set of missing chargers, not on the order the selection happens
// to be stored in - re-ticking the same devices in another order must not hand back a
// fresh set of attempts.
func TestApplication_Login_MissingSelectedBudgetIsOrderIndependent(t *testing.T) {
	t.Parallel()

	h := newSelectionHarness(t, []model.Charger{{ID: "123"}}, nil, nil, nil)
	require.NoError(t, h.cfg.SetSelectedDevices([]string{"123", "gone", "other"}))

	for range 3 {
		require.ErrorContains(t, h.login(), "gone")
	}

	require.NoError(t, h.cfg.SetSelectedDevices([]string{"other", "123", "gone"}))

	require.NoError(t, h.login(), "the same missing set reordered must not reset the budget")
	assert.Equal(t, []string{"123"}, h.seeded())
}

// Adoption must not filter the owned chargers against the response it was handed: the
// result would be a subset of that response by construction, so a transiently short list
// would silently prune the selection where the retry budget can no longer see it.
func TestApplication_Login_PartialChargerListDoesNotPruneAdoptedSelection(t *testing.T) {
	t.Parallel()

	h := newSelectionHarness(t, []model.Charger{{ID: "A1"}}, nil, []string{"A1", "A2"}, nil)

	require.ErrorContains(t, h.login(), "A2", "a partial list must spend the retry budget, not prune")
	assert.Equal(t, selection.Selection{"A1", "A2"}, h.cfg.SelectedDevices(), "the absent charger keeps its place")
}

// Logging into a different Easee account must not adopt the previous account's thing IDs:
// they are absent from the new account's charger list, so the hub would end up selecting
// devices it can never see and seeding nothing.
func TestApplication_Login_AccountSwitchIgnoresStaleThings(t *testing.T) {
	t.Parallel()

	h := newSelectionHarness(t, []model.Charger{{ID: "B1"}, {ID: "B2"}}, nil, []string{"A1", "A2"}, nil)

	require.NoError(t, h.login(), "the stale selection must not burn the retry budget")

	assert.Equal(t, []string{"B1", "B2"}, h.seeded())
	assert.Equal(t, selection.Selection{"B1", "B2"}, h.cfg.SelectedDevices())
}

// The fallthrough after a skipped adoption is the auto-selection cap, not "everything the new
// account lists" - an installer account switch must not flood the hub with every charger.
func TestApplication_Login_AccountSwitchFallsThroughToTheCap(t *testing.T) {
	t.Parallel()

	chargers := make([]model.Charger, 0, 12)
	for i := range 12 {
		chargers = append(chargers, model.Charger{ID: strconv.Itoa(i)})
	}

	h := newSelectionHarness(t, chargers, nil, []string{"A1", "A2"}, nil)

	require.NoError(t, h.login())

	first10 := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	assert.Equal(t, first10, h.seeded(), "the cap, not the whole account listing, materialises the selection")
	assert.Equal(t, selection.Selection(first10), h.cfg.SelectedDevices())
}

// A fetch that succeeds but lists nothing leaves the selection empty, which is the same
// shape as "never configured" - without a guard the sync would read it as "seed no
// devices" and destroy every thing on the hub on a single bad response.
func TestApplication_Login_EmptyChargerListDoesNotDestroyThings(t *testing.T) {
	t.Parallel()

	h := newSelectionHarness(t, nil, nil, []string{"A1", "A2"}, nil)

	for range 3 {
		require.ErrorContains(t, h.login(), "A1")
		assert.Empty(t, h.seeded(), "nothing may be seeded while the chargers might still come back")
	}

	require.NoError(t, h.login(), "an account that really lists nothing must stop blocking login")
	assert.Empty(t, h.seeded())
}
