package cmd

import (
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
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/fimpgo"
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
	configService *config.Service
	lifecycle     *lifecycle.Lifecycle
	mqtt          *fimpgo.MqttTransport

	application     app.Application
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

// getConfigService initiates a configuration service and loads the config.
func getConfigService() *config.Service {
	if services.configService == nil {
		workDir := bootstrap.GetConfigurationDirectory()
		cfg := config.New(workDir)
		services.configService = config.NewService(cliffCfg.NewStorage(cfg, workDir))

		err := services.configService.Load()
		if err != nil {
			log.WithError(err).Fatal("failed to load configuration")
		}
	}

	return services.configService
}

// getLifecycle creates or returns existing lifecycle service.
func getLifecycle() *lifecycle.Lifecycle {
	if services.lifecycle == nil {
		services.lifecycle = lifecycle.New()
	}

	return services.lifecycle
}

// getEventListener creates or returns existing event listener service.
func getEventListener(cfg *config.Config) event.Listener {
	if services.eventListener == nil {
		services.eventListener = event.NewListener(
			getEventManager(cfg),
			parameters.NewInclusionReportSentEventHandler(getAdapter(cfg)),
		)
	}

	return services.eventListener
}

// getEventListener creates or returns existing event listener service.
func getSessionStorage(cfg *config.Config) db.ChargingSessionStorage {
	if services.sessionStorage == nil {
		dataBase, err := database.NewDatabase(cfg.WorkDir)
		if err != nil {
			log.WithError(err).Error("can't create db")

			return nil
		}

		services.sessionStorage = db.NewSessionStorage(dataBase)
	}

	return services.sessionStorage
}

// getMQTT creates or returns existing MQTT broker service.
func getMQTT(cfg *config.Config) *fimpgo.MqttTransport {
	if services.mqtt == nil {
		services.mqtt = fimpgo.NewMqttTransport(
			cfg.MQTTServerURI,
			cfg.MQTTClientIDPrefix,
			cfg.MQTTUsername,
			cfg.MQTTPassword,
			true,
			1,
			1,
		)
	}

	services.mqtt.SetDefaultSource(routing.ResourceName)

	return services.mqtt
}

// getApplication creates or returns existing application.
func getApplication(cfg *config.Config) app.Application {
	if services.application == nil {
		services.application = app.New(
			getAdapter(cfg),
			getConfigService(),
			getLifecycle(),
			getManifestLoader(),
			getEaseeAPIClient(cfg),
			getAuthenticator(cfg),
			getSignalRClient(cfg),
		)
	}

	return services.application
}

// getManifestLoader creates or returns existing application manifestLoader.
func getManifestLoader() manifest.Loader {
	if services.manifestLoader == nil {
		services.manifestLoader = manifest.NewLoader(getConfigService().GetWorkDir())
	}

	return services.manifestLoader
}

// getAdapter creates or returns existing adapter service.
func getAdapter(cfg *config.Config) adapter.Adapter {
	if services.adapter == nil {
		services.adapter = adapter.NewAdapter(
			getMQTT(cfg),
			getEventManager(cfg),
			getThingFactory(cfg),
			getAdapterState(),
			routing.ServiceName,
			"1",
		)
	}

	return services.adapter
}

// getEventManager creates or returns existing event manager service.
func getEventManager(_ *config.Config) event.Manager {
	if services.eventManager == nil {
		services.eventManager = event.NewManager()
	}

	return services.eventManager
}

// getAdapterState creates or returns existing adapter state service.
func getAdapterState() adapter.State {
	if services.adapterState == nil {
		var err error

		services.adapterState, err = adapter.NewState(getConfigService().GetWorkDir())
		if err != nil {
			log.WithError(err).Fatal("failed to initialize adapter state")
		}
	}

	return services.adapterState
}

// getThingFactory creates or returns existing thing factory service.
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

// getEaseeHTTPClient creates or returns existing Easee HTTP client.
func getEaseeHTTPClient() api.HTTPClient {
	if services.easeeHTTPClient == nil {
		services.easeeHTTPClient = api.NewHTTPClient(
			getConfigService(),
			getHTTPClient(),
			getConfigService().GetEaseeBaseURL(),
		)
	}

	return services.easeeHTTPClient
}

// getEaseeAPIClient creates or returns existing Easee HTTP client.
func getEaseeAPIClient(cfg *config.Config) api.Client {
	if services.easeeAPIClient == nil {
		services.easeeAPIClient = api.NewAPIClient(
			getEaseeHTTPClient(),
			getAuthenticator(cfg),
		)
	}

	return services.easeeAPIClient
}

// getHTTPClient creates or returns existing HTTP client with predefined timeout.
func getHTTPClient() *http.Client {
	if services.httpClient == nil {
		services.httpClient = &http.Client{
			Timeout: getConfigService().GetHTTPTimeout(),
		}
	}

	return services.httpClient
}

func getAuthenticator(cfg *config.Config) api.Authenticator {
	if services.authenticator == nil {
		services.authenticator = api.NewAuthenticator(
			getEaseeHTTPClient(),
			getConfigService(),
			notification.NewNotification(getMQTT(cfg)),
			getMQTT(cfg),
			routing.ServiceName,
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

// newRouting creates new set of routing.
func newRouting(cfg *config.Config) []*cliffRouter.Routing {
	return routing.New(
		getConfigService(),
		getLifecycle(),
		getApplication(cfg),
		getAdapter(cfg),
	)
}

// newTasks creates new set of tasks.
func newTasks(cfg *config.Config) []*task.Task {
	return tasks.New(
		getConfigService(),
		getLifecycle(),
		getApplication(cfg),
		getAdapter(cfg),
		getAuthenticator(cfg),
	)
}
