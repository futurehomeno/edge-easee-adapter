package cmd

import (
	"github.com/futurehomeno/cliffhanger/notification"
	"net/http"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/bootstrap"
	cliffCfg "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/manifest"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/storage"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
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
	configStorage storage.Storage
	lifecycle     *lifecycle.Lifecycle
	mqtt          *fimpgo.MqttTransport

	application     app.Application
	manifestLoader  manifest.Loader
	adapter         adapter.Adapter
	thingFactory    adapter.ThingFactory
	adapterState    adapter.State
	httpClient      *http.Client
	easeeHTTPClient easee.HTTPClient
	easeeAPIClient  easee.APIClient
	authenticator   easee.Authenticator
	signalRClient   signalr.Client
	signalRManager  easee.SignalRManager
}

func resetContainer() {
	services = &serviceContainer{}
}

// getConfigService initiates a configuration service and loads the config.
func getConfigService() *config.Service {
	if services.configService == nil {
		services.configService = config.NewService(getConfigStorage())

		err := services.configService.Load()
		if err != nil {
			log.WithError(err).Fatal("failed to load configuration")
		}
	}

	return services.configService
}

// getConfigStorage creates or returns an existing config storage.
func getConfigStorage() storage.Storage {
	if services.configStorage == nil {
		workDir := bootstrap.GetConfigurationDirectory()
		cfg := config.New(workDir)

		services.configStorage = cliffCfg.NewStorage(cfg, workDir)
	}

	return services.configStorage
}

// getLifecycle creates or returns existing lifecycle service.
func getLifecycle() *lifecycle.Lifecycle {
	if services.lifecycle == nil {
		services.lifecycle = lifecycle.New()
	}

	return services.lifecycle
}

// getMQTT creates or returns existing MQTT broker service.
func getMQTT() *fimpgo.MqttTransport {
	if services.mqtt == nil {
		cfg := getConfigService().Model().(*config.Config) //nolint:forcetypeassert
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

	return services.mqtt
}

// getApplication creates or returns existing application.
func getApplication() app.Application {
	if services.application == nil {
		services.application = app.New(
			getAdapter(),
			getConfigService(),
			getLifecycle(),
			getManifestLoader(),
			getEaseeAPIClient(),
			getAuthenticator(),
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
func getAdapter() adapter.Adapter {
	if services.adapter == nil {
		services.adapter = adapter.NewAdapter(
			getMQTT(),
			getThingFactory(),
			getAdapterState(),
			easee.ServiceName,
			"1",
		)
	}

	return services.adapter
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
func getThingFactory() adapter.ThingFactory {
	if services.thingFactory == nil {
		services.thingFactory = easee.NewThingFactory(getEaseeAPIClient(), getConfigService(), getSignalRManager(), getSignalRClient())
	}

	return services.thingFactory
}

// getEaseeHTTPClient creates or returns existing Easee HTTP client.
func getEaseeHTTPClient() easee.HTTPClient {
	if services.easeeHTTPClient == nil {
		services.easeeHTTPClient = easee.NewHTTPClient(
			getHTTPClient(),
			getConfigService().GetEaseeBaseURL(),
		)
	}

	return services.easeeHTTPClient
}

// getEaseeAPIClient creates or returns existing Easee HTTP client.
func getEaseeAPIClient() easee.APIClient {
	if services.easeeAPIClient == nil {
		services.easeeAPIClient = easee.NewAPIClient(
			getEaseeHTTPClient(),
			getAuthenticator(),
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

func getAuthenticator() easee.Authenticator {
	if services.authenticator == nil {
		services.authenticator = easee.NewAuthenticator(
			getEaseeHTTPClient(),
			getConfigService(),
			notification.NewNotification(getMQTT()),
		)
	}

	return services.authenticator
}

func getSignalRClient() signalr.Client {
	if services.signalRClient == nil {
		services.signalRClient = signalr.NewClient(getConfigService(), getAuthenticator().AccessToken)
	}

	return services.signalRClient
}

func getSignalRManager() easee.SignalRManager {
	if services.signalRManager == nil {
		services.signalRManager = easee.NewSignalRManager(getSignalRClient())
	}

	return services.signalRManager
}

// newRouting creates new set of routing.
func newRouting() []*cliffRouter.Routing {
	return routing.New(
		getConfigService(),
		getLifecycle(),
		getApplication(),
		getAdapter(),
	)
}

// newTasks creates new set of tasks.
func newTasks() []*task.Task {
	return tasks.New(
		getConfigService(),
		getLifecycle(),
		getApplication(),
		getAdapter(),
	)
}
