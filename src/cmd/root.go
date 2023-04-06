package cmd

import (
	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/root"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/routing"
)

// Execute is an entry point to the edge application.
func Execute() {
	cfg := getConfigService().Model().(*config.Config) //nolint:forcetypeassert

	bootstrap.InitializeLogger(cfg.LogFile, cfg.LogLevel, cfg.LogFormat)

	edgeApp, err := buildEdgeApp()
	if err != nil {
		log.WithError(err).Fatalf("failed to build the edge application")
	}

	err = edgeApp.Start()
	if err != nil {
		log.WithError(err).Fatalf("failed to start the edge application")
	}

	bootstrap.WaitForShutdown()

	err = edgeApp.Stop()
	if err != nil {
		log.WithError(err).Fatalf("failed to stop the edge application")
	}
}

func buildEdgeApp() (root.App, error) {
	return root.NewEdgeAppBuilder().
		WithMQTT(getMQTT()).
		WithServiceDiscovery(routing.GetDiscoveryResource()).
		WithLifecycle(getLifecycle()).
		WithTopicSubscription(
			cliffRouter.TopicPatternAdapter(easee.ServiceName),
			cliffRouter.TopicPatternDevices(easee.ServiceName),
		).
		WithRouting(newRouting()...).
		WithTask(newTasks()...).
		WithServices(getSignalRManager()).
		Build()
}
