package tasks

import (
	"time"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/task"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/jwt"
)

// New returns a set of background tasks of an application.
func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application app.App,
	ad adapter.Adapter,
) []*task.Task {
	return task.Combine[[]*task.Task](
		[]*task.Task{
			task.New(handleCredentials(cfgSrv), 0),
		},
		app.TaskApp(application, appLifecycle),
		adapter.TaskAdapter(ad, cfgSrv.GetPollingInterval()),
		thing.TaskCarCharger(ad, cfgSrv.GetPollingInterval(), task.WhenAppIsConnected(appLifecycle)),
	)
}

func handleCredentials(cfgSrv *config.Service) func() {
	return func() {
		creds := cfgSrv.GetCredentials()
		if creds.ExpiresAt.IsZero() {
			return
		}

		accessTokenExpiresAt, err := jwt.ExpirationDate(creds.AccessToken)
		if err != nil {
			log.WithError(err).Error("auth token expiration BC task: can't get access token expires at")

			return
		}

		refreshTokenExpiresAt, err := jwt.ExpirationDate(creds.RefreshToken)
		if err != nil {
			log.WithError(err).Error("auth token expiration BC task: can't get refresh token expires at")

			return
		}

		newCreds := config.Credentials{
			AccessToken:           creds.AccessToken,
			RefreshToken:          creds.RefreshToken,
			ExpiresAt:             time.Time{},
			AccessTokenExpiresAt:  accessTokenExpiresAt,
			RefreshTokenExpiresAt: refreshTokenExpiresAt,
		}

		err = cfgSrv.SetCredentials(newCreds)
		if err != nil {
			log.WithError(err).Error("can't update credentials")
		}
	}
}
