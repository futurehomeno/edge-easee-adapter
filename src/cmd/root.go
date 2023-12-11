package cmd

import (
	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/root"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/routing"
)

// Execute is an entry point to the edge application.
func Execute() {
	cfg := getConfigService().Model()

	bootstrap.InitializeLogger(cfg.LogFile, cfg.LogLevel, cfg.LogFormat)

	edgeApp, err := Build(cfg)
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

func Build(cfg *config.Config) (root.App, error) {
	return root.NewEdgeAppBuilder().
		WithMQTT(getMQTT(cfg)).
		WithServiceDiscovery(routing.GetDiscoveryResource()).
		WithLifecycle(getLifecycle()).
		WithTopicSubscription(
			cliffRouter.TopicPatternAdapter(routing.ServiceName),
			cliffRouter.TopicPatternDevices(routing.ServiceName),
		).
		WithRouting(newRouting(cfg)...).
		WithTask(newTasks(cfg)...).
		WithServices(getSignalRManager(cfg)).
		Build()
}
