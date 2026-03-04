package cmd

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	"github.com/futurehomeno/cliffhanger/root"
	cliffRouter "github.com/futurehomeno/cliffhanger/router"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/routing"
)

type BudzikFormatter struct {
	TimestampFormat string
	LevelDesc       []string
}

func (f *BudzikFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)

	level := "D"

	if int(entry.Level) >= 0 && int(entry.Level) < len(f.LevelDesc) {
		level = f.LevelDesc[int(entry.Level)]
	}

	ret := fmt.Appendf(nil, "%s %s %s", timestamp, level, entry.Message)
	for k, v := range entry.Data {
		ret = fmt.Appendf(ret, " %s=%v", k, v)
	}

	ret = fmt.Appendf(ret, "\n")

	return ret, nil
}

func NewBudzikFormatter() *BudzikFormatter {
	lvlDesc := []string{"PANIC", "FATAL", "E", "W", "I", "D", "T", "?"}
	return &BudzikFormatter{TimestampFormat: "01-02 15:04:05", LevelDesc: lvlDesc}
}

// Execute is an entry point to the edge application.
func Execute(version string) error {
	cfg := getConfigService().Model()

	if err := bootstrap.InitializeLogger(cfg.LogFile, cfg.LogLevel, cfg.LogFormat); err != nil {
		return err
	}

	log.Infof("\t--- Start Easee v%s ---", version)
	defer log.Infof("\t+++ Stop Easee v%s +++", version)

	rootApp, err := Build(cfg)
	if err != nil {
		return fmt.Errorf("build app err: %w", err)
	}

	err = rootApp.Run()
	if err != nil {
		return fmt.Errorf("start app err: %w", err)
	}

	return nil
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
		WithServices(getSignalRManager(cfg), getEventListener(cfg), getSessionStorage(cfg)).
		Build()
}
