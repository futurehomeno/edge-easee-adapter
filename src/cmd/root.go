package cmd

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/root"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Execute is an entry point to the edge application.
func Execute(packageName, version string) error {
	cfg := getConfigService().Model()

	if cfg.LogFormat == "text" {
		cfg.LogFormat = "budzik"

		if err := getConfigService().Save(); err != nil {
			log.Warnf("Save config err: %v", err)
		}
	}

	if err := bootstrap.InitializeLogger(cfg.LogFile, cfg.LogLevel, cfg.LogFormat); err != nil {
		return err
	}

	log.Infof("\t--- Start Easee v%s ---", version)
	defer log.Infof("\t+++ Stop Easee v%s +++", version)

	rootApp, err := Build(cfg, packageName, version)
	if err != nil {
		return fmt.Errorf("build app err: %w", err)
	}

	err = rootApp.Run()
	if err != nil {
		return fmt.Errorf("start app err: %w", err)
	}

	return nil
}

func Build(cfg *config.Config, packageName, version string) (root.App, error) {
	return root.NewEdgeAppBuilder().
		WithMQTT(getMQTT(cfg)).
		WithServiceDiscovery(fimptype.EaseeRn, discovery.ResourceTypeAd, packageName, "1", version).
		WithLifecycle(getLifecycle()).
		WithTopicSubscription(
			cliffRouter.TopicPatternAdapter(fimptype.EaseeRn),
			cliffRouter.TopicPatternDevices(fimptype.EaseeRn),
		).
		WithRouting(newRouting(cfg)...).
		WithTask(newTasks(cfg)...).
		WithServices(getSignalRManager(cfg), getEventListener(cfg), getSessionStorage(cfg)).
		Build()
}
