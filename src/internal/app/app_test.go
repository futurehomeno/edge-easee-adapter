package app_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	cliffApp "github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedmanifest "github.com/futurehomeno/cliffhanger/test/mocks/manifest"
	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/mocks"
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

			a := app.New(nil, nil, nil, loaderMock, nil, nil)

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

func TestApplication_Configure_NOOP(t *testing.T) {
	t.Parallel()

	a := app.New(nil, nil, nil, nil, nil, nil)
	err := a.Configure("anything")

	assert.NoError(t, err)
}

func TestApplication_Uninstall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		configAssertions    func(c *config.Config)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC), //nolint:gofumpt
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
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
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC), //nolint:gofumpt
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
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

			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(mockedadapter.Adapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			defer adapterMock.AssertExpectations(t)

			storage := fakes.NewConfigStorage(tt.cfg, config.Factory)
			cfgService := config.NewService(storage)

			application := app.New(adapterMock, cfgService, lc, nil, nil, nil)

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
				tt.configAssertions(cfgService.Model().(*config.Config)) //nolint:forcetypeassert
			}
		})
	}
}

func TestApplication_Login(t *testing.T) { //nolint:paralleltest
	clock.Mock(time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC)) //nolint:gofumpt
	t.Cleanup(func() {
		clock.Restore()
	})

	tests := []struct {
		name                string
		loginData           *cliffApp.LoginCredentials
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		mockClient          func(c *mocks.APIClient)
		mockAuthenticator   func(a *mocks.Authenticator)
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
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mocks.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Chargers").Return([]easee.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("Ping").Return(nil)
				c.On("ChargerConfig", "123").Return(test.ExampleChargerConfig(t), nil).Once()
				c.On("ChargerConfig", "456").Return(test.ExampleChargerConfig(t), nil).Once()
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.
					On("CreateThing", &adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID:  "123",
							MaxCurrent: 32,
						},
					}).
					Return(nil)

				a.
					On("CreateThing", &adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID:  "456",
							MaxCurrent: 32,
						},
					}).
					Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
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
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAuthenticator: func(a *mocks.Authenticator) {
				a.
					On("Login", "test-user", "test-password").
					Return(errors.New("oops"))
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Ping").Return(nil)
			},
			wantErr: true,
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
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
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mocks.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Chargers").Return([]easee.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("Ping").Return(errors.New("oops"))
				c.On("ChargerConfig", "123").Return(test.ExampleChargerConfig(t), nil).Once()
				c.On("ChargerConfig", "456").Return(test.ExampleChargerConfig(t), nil).Once()
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.
					On("CreateThing", &adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID:  "123",
							MaxCurrent: 32,
						},
					}).
					Return(nil)

				a.
					On("CreateThing", &adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID:  "456",
							MaxCurrent: 32,
						},
					}).
					Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
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
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAuthenticator: func(a *mocks.Authenticator) {
				a.On("Login", "test-user", "test-password").Return(nil)
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Chargers").Return([]easee.Charger{
					{ID: "123"},
					{ID: "456"},
				}, nil)
				c.On("Ping").Return(nil)
				c.On("ChargerConfig", "123").Return(test.ExampleChargerConfig(t), nil).Once()
				c.On("ChargerConfig", "456").Return(test.ExampleChargerConfig(t), nil).Once()
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.
					On("CreateThing", &adapter.ThingSeed{
						ID: "123",
						Info: easee.Info{
							ChargerID:  "123",
							MaxCurrent: 32,
						},
					}).
					Return(nil)

				a.
					On("CreateThing", &adapter.ThingSeed{
						ID: "456",
						Info: easee.Info{
							ChargerID:  "456",
							MaxCurrent: 32,
						},
					}).
					Return(errors.New("oops"))
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := mockedadapter.NewAdapter(t)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			clientMock := mocks.NewAPIClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			authMock := mocks.NewAuthenticator(t)
			if tt.mockAuthenticator != nil {
				tt.mockAuthenticator(authMock)
			}

			application := app.New(adapterMock, nil, lc, nil, clientMock, authMock)

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
		cfg                 *config.Config
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
		configAssertions    func(c *config.Config)
	}{
		{
			name: "successful config, lifecycle and adapter reset",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC), //nolint:gofumpt
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
				lc.SetConfigState(lifecycle.ConfigStateConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("DestroyAllThings").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
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
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC), //nolint:gofumpt
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateRunning, nil)
				lc.SetAuthState(lifecycle.AuthStateAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateConnected)
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

			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := new(mockedadapter.Adapter)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			defer adapterMock.AssertExpectations(t)

			storage := fakes.NewConfigStorage(tt.cfg, config.Factory)
			cfgService := config.NewService(storage)

			application := app.New(adapterMock, cfgService, lc, nil, nil, nil)

			err := application.Logout()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			if tt.lifecycleAssertions != nil {
				tt.lifecycleAssertions(lc)
			}

			if tt.configAssertions != nil {
				tt.configAssertions(cfgService.Model().(*config.Config)) //nolint:forcetypeassert
			}
		})
	}
}

func TestApplication_Initialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		cfg                 *config.Config
		setLifecycle        func(lc *lifecycle.Lifecycle)
		mockAdapter         func(a *mockedadapter.Adapter)
		mockClient          func(c *mocks.APIClient)
		wantErr             bool
		lifecycleAssertions func(lc *lifecycle.Lifecycle)
	}{
		{
			name: "successful thing initialization",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC), //nolint:gofumpt
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Ping").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateConfigured, lc.ConfigState())
			},
		},
		{
			name: "empty credentials - unconfigure lifecycle",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Ping").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
		},
		{
			name: "error on thing initialization",
			cfg:  &config.Config{},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(errors.New("oops"))
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Ping").Return(nil)
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateNotConfigured, lc.AppState())
				assert.Equal(t, lifecycle.AuthStateNotAuthenticated, lc.AuthState())
				assert.Equal(t, lifecycle.ConnStateConnected, lc.ConnectionState())
				assert.Equal(t, lifecycle.ConfigStateNotConfigured, lc.ConfigState())
			},
			wantErr: true,
		},
		{
			name: "successful thing initialization, but ping failed",
			cfg: &config.Config{
				Credentials: config.Credentials{
					AccessToken:  "access-token",
					RefreshToken: "refresh-token",
					ExpiresAt:    time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC), //nolint:gofumpt
				},
			},
			setLifecycle: func(lc *lifecycle.Lifecycle) {
				lc.SetAppState(lifecycle.AppStateNotConfigured, nil)
				lc.SetAuthState(lifecycle.AuthStateNotAuthenticated)
				lc.SetConnectionState(lifecycle.ConnStateDisconnected)
				lc.SetConfigState(lifecycle.ConfigStateNotConfigured)
			},
			mockAdapter: func(a *mockedadapter.Adapter) {
				a.On("InitializeThings").Return(nil)
			},
			mockClient: func(c *mocks.APIClient) {
				c.On("Ping").Return(errors.New("oops"))
			},
			lifecycleAssertions: func(lc *lifecycle.Lifecycle) {
				assert.Equal(t, lifecycle.AppStateRunning, lc.AppState())
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

			lc := lifecycle.New()
			if tt.setLifecycle != nil {
				tt.setLifecycle(lc)
			}

			adapterMock := mockedadapter.NewAdapter(t)
			if tt.mockAdapter != nil {
				tt.mockAdapter(adapterMock)
			}

			clientMock := new(mocks.APIClient)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			defer func() {
				adapterMock.AssertExpectations(t)
				clientMock.AssertExpectations(t)
			}()

			storage := fakes.NewConfigStorage(tt.cfg, config.Factory)
			cfgService := config.NewService(storage)

			application := app.New(adapterMock, cfgService, lc, nil, clientMock, nil)

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
