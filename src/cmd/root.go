package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/discovery"
	"github.com/futurehomeno/cliffhanger/root"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

func Execute(packageName, version string) error {
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
	if err := debug.InitializeLogger(getDefaultStore()); err != nil {
		log.Errorf("InitializeLogger err: %v", err)
	}

	log.Infof("--- Start %s v%s ---", packageName, version)

	path, err := filepath.Abs(bootstrap.GetWorkingDirectory())
	if err != nil {
		log.Errorf("get working directory err: %v", err)
	}

	log.Infof("Working dir=%s", path)

	cfgPath, err := filepath.Abs(bootstrap.GetConfigurationDirectory())
	if err != nil {
		log.Errorf("gt config directory err: %v", err)
	}

	log.Infof("Config dir=%s", cfgPath)

	return root.NewEdgeAppBuilder().
		WithMQTT(getMQTT(cfg)).
		WithServiceDiscovery(fimptype.EaseeRn, discovery.ResourceTypeAd, packageName, "1", version).
		WithLifecycle(getLifecycle()).
		WithTopicSubscription(
			cmdTopic(fimptype.ResourceTypeAdapter),
			cmdTopic(fimptype.ResourceTypeDevice),
		).
		WithRouting(newRouting(cfg)...).
		WithTask(newTasks(cfg)...).
		WithServices(getSignalRManager(cfg), getEventListener(cfg), getSessionStorage(cfg)).
		Build()
}

func cmdTopic(resourceType fimptype.ResourceTypeT) string {
	return (&cliffRouter.TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		MessageType:     fimptype.MsgTypeCmd,
		ResourceType:    resourceType,
		ResourceName:    fimptype.EaseeRn,
		ResourceAddress: "1",
	}).String()
}
