package tasks

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"

	easeeapp "github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// New returns a set of background tasks of an application.
func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application easeeapp.AppliacationWithToken,
	ad adapter.Adapter,
) []*task.Task {
	return task.Combine(
		[]*task.Task{task.New(refreshTokenFn(application), cfgSrv.GetTokenRefreshInterval())},
		app.TaskApp(application, appLifecycle),
		adapter.TaskAdapter(ad, cfgSrv.GetPollingInterval()),
		thing.TaskCarCharger(ad, cfgSrv.GetPollingInterval(), task.WhenAppIsConnected(appLifecycle)),
	)
}

func refreshTokenFn(application easeeapp.AppliacationWithToken) func() {
	return func() {
		application.RefreshToken()
	}
}
