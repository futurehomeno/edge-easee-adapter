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
			task.New(handleCredentialsBCTask(cfgSrv), 0),
		},
		app.TaskApp(application, appLifecycle),
		adapter.TaskAdapter(ad, cfgSrv.GetPollingInterval()),
		thing.TaskCarCharger(ad, cfgSrv.GetPollingInterval(), task.WhenAppIsConnected(appLifecycle)),
	)
}

func handleCredentialsBCTask(cfgSrv *config.Service) func() {
	return func() {
		creds := cfgSrv.GetCredentials()

		if creds.Empty() || !creds.RefreshTokenExpiresAt.IsZero() {
			return
		}

		// We're refreshing the field to make sure we have a correct time set there.
		accessTokenExpiresAt, err := jwt.ExpirationDate(creds.AccessToken)
		if err != nil {
			log.WithError(err).Error("credentials expiration BC task: can't get access token expires at")

			return
		}

		refreshTokenExpiresAt, err := jwt.ExpirationDate(creds.RefreshToken)
		if err != nil {
			log.WithError(err).Error("credentials expiration BC task: can't get refresh token expires at")

			return
		}

		log.WithField("access_token_expires_at", accessTokenExpiresAt.Format(time.RFC3339)).
			WithField("refresh_token_expires_at", refreshTokenExpiresAt.Format(time.RFC3339)).
			Info("credentials expiration BC task: updating token expiration times")

		newCreds := config.Credentials{
			AccessToken:           creds.AccessToken,
			RefreshToken:          creds.RefreshToken,
			AccessTokenExpiresAt:  accessTokenExpiresAt,
			RefreshTokenExpiresAt: refreshTokenExpiresAt,
		}

		err = cfgSrv.SetCredentials(newCreds)
		if err != nil {
			log.WithError(err).Error("credentials expiration BC task: can't update credentials")
		}
	}
}
