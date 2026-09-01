package cmd

import (
	"fmt"
	"net/http"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/bootstrap"
	cliffCfg "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/database"
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/notification"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	cliffStorage "github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/routing"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	"github.com/futurehomeno/edge-easee-adapter/internal/tasks"
)

// services is a container for services that are common dependencies.
var services = &serviceContainer{}

// serviceContainer is a type representing a dependency injection container to be used during bootstrap of the application.
type serviceContainer struct {
	configService    *config.Service
	credentialsStore *config.CredentialsStore
	defaultStore     *cliffCfg.DefaultStore
	lifecycle        *lifecycle.Lifecycle
	mqtt             *fimpgo.MqttTransport

	application     app.ApplicationWithToken
	manifestLoader  manifest.Loader
	eventManager    event.Manager
	adapter         adapter.Adapter
	thingFactory    adapter.ThingFactory
	adapterState    adapter.State
	httpClient      *http.Client
	easeeHTTPClient api.HTTPClient
	easeeAPIClient  api.Client
	authenticator   api.Authenticator
	signalRClient   signalr.Client
	signalRManager  signalr.Manager
	eventListener   event.Listener
	sessionStorage  db.ChargingSessionStorage
}

func resetContainer() {
	services = &serviceContainer{}
}

func getConfigService() *config.Service {
	if services.configService == nil {
		workDir := bootstrap.GetConfigurationDirectory()
		cfg := config.New(workDir)
		services.configService = config.NewService(cliffStorage.New(cfg, workDir, "config.json"))

		err := services.configService.Load()
		if err != nil {
			log.Fatalf("[config] Load config. err: %v", err)
		}

		if err := migrateConfig(services.configService, getCredentialsStore()); err != nil {
			log.Fatalf("[config] Migrate config. err: %v", err)
		}
	}

	return services.configService
}

func getCredentialsStore() *config.CredentialsStore {
	if services.credentialsStore == nil {
		services.credentialsStore = config.NewCredentialsStore(bootstrap.GetConfigurationDirectory())

		if err := services.credentialsStore.Load(); err != nil {
			log.Fatalf("[config] Load credentials. err: %v", err)
		}
	}

	return services.credentialsStore
}

// migrateConfig returns the migration error instead of only logging it: the v5->v6 step moves
// tokens into the credentials store, and the app decides it is configured by reading that store
// alone, so continuing past a failed write brings an authenticated install up as logged out.
// The version bump does not land on failure, so the next start retries.
func migrateConfig(cfgSvc *config.Service, credentials *config.CredentialsStore) error {
	cfg := cfgSvc.Model()

	resetLogDefaults := func() error {
		cfg.LogLevel = "info"
		cfg.LogFormat = "budzik"

		return nil
	}

	err := cfgSvc.Migrate(
		cliffCfg.Migration{From: 0, To: 2, Do: resetLogDefaults},
		cliffCfg.Migration{From: 1, To: 2, Do: resetLogDefaults},
		cliffCfg.Migration{From: 2, To: 3, Do: cfg.MigrateAuthBackoff},
		cliffCfg.Migration{From: 3, To: 4, Do: cfg.MigrateOfferedCurrentWaitTime},
		cliffCfg.Migration{From: 4, To: 5, Do: cfg.MigrateSignalRFinalBackoff},
		cliffCfg.Migration{From: 5, To: 6, Do: func() error { return config.MigrateCredentials(cfg, credentials) }},
	)
	if err != nil {
		return fmt.Errorf("migrate config: %w", err)
	}

	return nil
}

func getDefaultStore() *cliffCfg.DefaultStore {
	if services.defaultStore == nil {
		services.defaultStore = getConfigService().DefaultStore()
	}

	return services.defaultStore
}

func getLifecycle() *lifecycle.Lifecycle {
	if services.lifecycle == nil {
		services.lifecycle = lifecycle.New(getDefaultStore())
	}

	return services.lifecycle
}

func getEventListener(cfg *config.Config) event.Listener {
	if services.eventListener == nil {
		services.eventListener = event.NewListener(
			getEventManager(cfg),
			parameters.NewInclusionReportSentEventHandler(getAdapter(cfg)),
		)
	}

	return services.eventListener
}

func getSessionStorage(cfg *config.Config) db.ChargingSessionStorage {
	if services.sessionStorage == nil {
		dataBase, err := database.NewDatabase(cfg.WorkDir)
		if err != nil {
			log.Errorf("[db] Create database. err: %v", err)

			return nil
		}

		services.sessionStorage = db.NewSessionStorage(dataBase)
	}

	return services.sessionStorage
}

func getMQTT(cfg *config.Config) *fimpgo.MqttTransport {
	if services.mqtt == nil {
		errHandler := func(err error) {
			log.Errorf("[mqtt] Unrecoverable error. err: %v", err)
			panic(err)
		}

		services.mqtt = fimpgo.NewMqttTransport(
			cfg.MQTTServerURI,
			cfg.MQTTClientIDPrefix,
			cfg.MQTTUsername,
			cfg.MQTTPassword,
			true,
			1,
			1,
			errHandler,
		)
	}

	services.mqtt.SetDefaultSource(fimptype.EaseeRn)

	return services.mqtt
}

func getApplication(cfg *config.Config) app.ApplicationWithToken {
	if services.application == nil {
		services.application = app.New(
			getAdapter(cfg),
			getConfigService(),
			getLifecycle(),
			getManifestLoader(),
			getEaseeAPIClient(cfg),
			getAuthenticator(cfg),
			getSignalRClient(cfg),
			getCredentialsStore(),
		)
	}

	return services.application
}

func getManifestLoader() manifest.Loader {
	if services.manifestLoader == nil {
		services.manifestLoader = manifest.NewLoader(getConfigService().Model().WorkDir)
	}

	return services.manifestLoader
}

func getAdapter(cfg *config.Config) adapter.Adapter {
	if services.adapter == nil {
		services.adapter = adapter.NewAdapter(
			getMQTT(cfg),
			getEventManager(cfg),
			getThingFactory(cfg),
			getAdapterState(),
			fimptype.EaseeRn,
			"1",
		)
	}

	return services.adapter
}

func getEventManager(_ *config.Config) event.Manager {
	if services.eventManager == nil {
		services.eventManager = event.NewManager()
	}

	return services.eventManager
}

func getAdapterState() adapter.State {
	if services.adapterState == nil {
		var err error

		services.adapterState, err = adapter.NewState(getConfigService().Model().WorkDir)
		if err != nil {
			log.Fatalf("[adapter] Initialize adapter state. err: %v", err)
		}
	}

	return services.adapterState
}

func getThingFactory(cfg *config.Config) adapter.ThingFactory {
	if services.thingFactory == nil {
		services.thingFactory = easee.NewThingFactory(
			getEaseeAPIClient(cfg),
			getConfigService(),
			getSignalRManager(cfg),
			getSessionStorage(cfg),
		)
	}

	return services.thingFactory
}

func getEaseeHTTPClient() api.HTTPClient {
	if services.easeeHTTPClient == nil {
		services.easeeHTTPClient = api.NewHTTPClient(
			getConfigService(),
			getHTTPClient(),
			getConfigService().EaseeBaseURL(),
		)
	}

	return services.easeeHTTPClient
}

func getEaseeAPIClient(cfg *config.Config) api.Client {
	if services.easeeAPIClient == nil {
		services.easeeAPIClient = api.NewAPIClient(
			getEaseeHTTPClient(),
			getAuthenticator(cfg),
		)
	}

	return services.easeeAPIClient
}

func getHTTPClient() *http.Client {
	if services.httpClient == nil {
		services.httpClient = &http.Client{
			Timeout: getConfigService().HTTPTimeout(),
		}
	}

	return services.httpClient
}

func getAuthenticator(cfg *config.Config) api.Authenticator {
	if services.authenticator == nil {
		services.authenticator = api.NewAuthenticator(
			getEaseeHTTPClient(),
			getCredentialsStore(),
			getConfigService(),
			notification.NewNotification(getMQTT(cfg)),
			getMQTT(cfg),
			fimptype.EaseeService,
		)
	}

	return services.authenticator
}

func getSignalRClient(cfg *config.Config) signalr.Client {
	if services.signalRClient == nil {
		services.signalRClient = signalr.NewClient(getConfigService(), getAuthenticator(cfg).AccessToken)
	}

	return services.signalRClient
}

func getSignalRManager(cfg *config.Config) signalr.Manager {
	if services.signalRManager == nil {
		services.signalRManager = signalr.NewManager(getConfigService(), getSignalRClient(cfg))
	}

	return services.signalRManager
}

func newRouting(cfg *config.Config) []*cliffRouter.Routing {
	return routing.New(
		getConfigService(),
		getLifecycle(),
		getApplication(cfg),
		getAdapter(cfg),
	)
}

func newTasks(cfg *config.Config) []*task.Task {
	return tasks.New(
		getConfigService(),
		getLifecycle(),
		getApplication(cfg),
		getAdapter(cfg),
	)
}
