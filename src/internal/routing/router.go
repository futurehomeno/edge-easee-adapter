package routing

import (
	cliffAdapter "github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
)

// New returns a new routing table.
func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application app.App,
	adapter cliffAdapter.Adapter,
) []*router.Routing {
	return router.Combine(
		[]*router.Routing{
			cliffConfig.RouteCmdLogSetLevel(easee.ServiceName, cfgSrv.SetLogLevel),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "polling_interval", cfgSrv.SetPollingInterval),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "easee_backoff", cfgSrv.SetEaseeBackoff),
			cliffConfig.RouteCmdConfigSetString(easee.ServiceName, "easee_base_url", cfgSrv.SetEaseeBaseURL),
			cliffConfig.RouteCmdConfigSetFloat(easee.ServiceName, "slow_charging_current_in_amperes", cfgSrv.SetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "http_timeout", cfgSrv.SetHTTPTimeout),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "observations_period", cfgSrv.SetObservationsPeriod),
		},
		app.RouteApp(easee.ServiceName, appLifecycle, cfgSrv, config.Factory, nil, application),
		cliffAdapter.RouteAdapter(adapter, nil),
		thing.RouteCarCharger(adapter),
	)
}
