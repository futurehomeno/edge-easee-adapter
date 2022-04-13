package tasks

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// New returns a set of background tasks of an application.
func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application app.App,
	adapter adapter.Adapter,
) []*task.Task {
	return task.Combine(
		app.TaskApp(application, appLifecycle),
		thing.TaskCarCharger(adapter, cfgSrv.GetPollingInterval(), task.WhenAppIsConnected(appLifecycle)),
	)
}
