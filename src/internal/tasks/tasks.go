package tasks

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"
	log "github.com/sirupsen/logrus"

	easeeapp "github.com/futurehomeno/edge-easee-adapter/internal/app"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// New returns a set of background tasks of an application.
func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application easeeapp.ApplicationWithToken,
	ad adapter.Adapter,
) []*task.Task {
	interval := cfgSrv.TokenRefreshInterval()

	if interval > 1000*time.Millisecond {
		maxValue := big.NewInt(5000)
		refreshTokenIntervalRandMs, err := rand.Int(rand.Reader, maxValue)
		if err != nil {
			refreshTokenIntervalRandMs = big.NewInt(0)
		}
		interval += time.Duration(refreshTokenIntervalRandMs.Int64()) * time.Millisecond
	}

	log.Infof("Refresh token interval=%s", interval)

	return task.Combine(
		[]*task.Task{task.New(refreshTokenFn(application, appLifecycle), interval)},
		app.TaskApp(application, appLifecycle),
		adapter.TaskAdapter(ad, cfgSrv.PollingInterval()),
		thing.TaskCarCharger(ad, cfgSrv.PollingInterval(), task.WhenAppIsConnected(appLifecycle)),
	)
}

func refreshTokenFn(application easeeapp.ApplicationWithToken, appLifecycle *lifecycle.Lifecycle) func() {
	return func() {
		if appLifecycle.AuthState() != lifecycle.AuthStateNotAuthenticated {
			application.RefreshToken()
		}
	}
}
