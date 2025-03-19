package tasks

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

var (
	// runOnce is a duration ensuring the task is going to be run only once.
	runOnce time.Duration = 0
)

// New returns a set of background tasks of an application.
func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application app.App,
	ad adapter.Adapter,
	auth api.Authenticator,
) []*task.Task {
	return task.Combine[[]*task.Task](
		[]*task.Task{
			task.New(runCredentialsBCTask(auth), runOnce),
		},
		app.TaskApp(application, appLifecycle),
		adapter.TaskAdapter(ad, cfgSrv.GetPollingInterval()),
		thing.TaskCarCharger(ad, cfgSrv.GetPollingInterval(), task.WhenAppIsConnected(appLifecycle)),
	)
}

func runCredentialsBCTask(authenticator api.Authenticator) func() {
	return func() {
		if err := authenticator.EnsureBackwardsCompatibility(); err != nil {
			log.WithError(err).Error("credentials expiration BC task: can't ensure backwards compatibility")
		}
	}
}
