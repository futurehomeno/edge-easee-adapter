package cmd

import (
	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/edge"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/router"
)

func Execute() {
	cfg := getConfigService().Model().(*config.Config)

	bootstrap.InitializeLogger(cfg.LogFile, cfg.LogLevel, cfg.LogFormat)

	edgeApp, err := edge.NewBuilder().
		WithMQTT(getMQTT()).
		WithServiceDiscovery(router.GetDiscoveryResource()).
		WithLifecycle(getLifecycle()).
		WithTopicSubscription(
			cliffRouter.TopicPatternAdapter(easee.ServiceName),
			cliffRouter.TopicPatternDevices(easee.ServiceName),
		).
		WithRouting(newRouting()...).
		WithTask(newTasks()...).
		Build()
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
