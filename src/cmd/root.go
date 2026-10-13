package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/root"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/utils"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

func Execute(packageName, version string) error {
	defer utils.PrintStackOnRecover("Execute", true)

	rootApp, err := Build(getConfigService().Model(), packageName, version)
	if err != nil {
		return fmt.Errorf("build app err: %w", err)
	}

	defer log.Infof("+++ Stop %s v%s +++", packageName, version)

	err = rootApp.Run()
	if err != nil {
		return fmt.Errorf("start app err: %w", err)
	}

	return nil
}

func Build(cfg *config.Config, packageName, version string) (root.App, error) {
	services.version = version

	if err := debug.InitializeLogger(getDefaultStore()); err != nil {
		log.Errorf("Initialize logger err: %v", err)
	}

	log.Infof("--- Start %s v%s ---", packageName, version)

	path, err := filepath.Abs(bootstrap.GetWorkingDirectory())
	if err != nil {
		log.Errorf("Working directory err: %v", err)
	}

	log.Infof("Working dir=%s", path)

	cfgPath, err := filepath.Abs(bootstrap.GetConfigurationDirectory())
	if err != nil {
		log.Errorf("Config directory err: %v", err)
	}

	log.Infof("Config dir=%s", cfgPath)

	return root.NewEdgeAppBuilder().
		WithMQTT(getMQTT(cfg)).
		WithServiceDiscovery(fimptype.EaseeRn, discovery.ResourceTypeAd, packageName, "1", version).
		WithLifecycle(getLifecycle()).
		WithTelemetry(getTelemetry(cfg)).
		WithTopicSubscription(
			cliffRouter.TopicPatternAdapter(fimptype.EaseeRn, fimptype.MsgTypeCmd),
			cliffRouter.TopicPatternDevice(fimptype.EaseeRn, fimptype.MsgTypeCmd),
		).
		WithRouterOptions(cliffRouter.WithStatsCallback(cliffRouter.DefaultLogStats("cmd.auth."))).
		WithRouting(newRouting(cfg)...).
		WithTask(newTasks(cfg)...).
		WithServices(getSessionStorage(cfg), getSignalRManager(cfg), getEventListener(cfg)).
		Build()
}
