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
			cliffConfig.RouteCmdLogGetLevel(easee.ServiceName, cfgSrv.GetLogLevel),
			cliffConfig.RouteCmdLogSetLevel(easee.ServiceName, cfgSrv.SetLogLevel),
			cliffConfig.RouteCmdConfigGetDuration(easee.ServiceName, "polling_interval", cfgSrv.GetPollingInterval),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "polling_interval", cfgSrv.SetPollingInterval),
			cliffConfig.RouteCmdConfigGetString(easee.ServiceName, "easee_base_url", cfgSrv.GetEaseeBaseURL),
			cliffConfig.RouteCmdConfigSetString(easee.ServiceName, "easee_base_url", cfgSrv.SetEaseeBaseURL),
			cliffConfig.RouteCmdConfigGetFloat(easee.ServiceName, "slow_charging_current_in_amperes", cfgSrv.GetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigSetFloat(easee.ServiceName, "slow_charging_current_in_amperes", cfgSrv.SetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigGetDuration(easee.ServiceName, "http_timeout", cfgSrv.GetHTTPTimeout),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "http_timeout", cfgSrv.SetHTTPTimeout),
			cliffConfig.RouteCmdConfigGetString(easee.ServiceName, "signalr_base_url", cfgSrv.GetSignalRBaseURL),
			cliffConfig.RouteCmdConfigSetString(easee.ServiceName, "signalr_base_url", cfgSrv.SetSignalRBaseURL),
			cliffConfig.RouteCmdConfigGetDuration(easee.ServiceName, "signalr_conn_creation_timeout", cfgSrv.GetSignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "signalr_conn_creation_timeout", cfgSrv.SetSignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigGetDuration(easee.ServiceName, "signalr_keep_alive_interval", cfgSrv.GetSignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "signalr_keep_alive_interval", cfgSrv.SetSignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigGetDuration(easee.ServiceName, "signalr_timeout_interval", cfgSrv.GetSignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "signalr_timeout_interval", cfgSrv.SetSignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigGetDuration(easee.ServiceName, "signalr_invoke_timeout", cfgSrv.GetSignalRInvokeTimeout),
			cliffConfig.RouteCmdConfigSetDuration(easee.ServiceName, "signalr_invoke_timeout", cfgSrv.SetSignalRInvokeTimeout),
			cliffConfig.RouteCmdConfigGetInt(easee.ServiceName, "signalr_invoke_retry_count", cfgSrv.GetSignalRInvokeRetryCount),
			cliffConfig.RouteCmdConfigSetInt(easee.ServiceName, "signalr_invoke_retry_count", cfgSrv.SetSignalRInvokeRetryCount),
		},
		app.RouteApp(easee.ServiceName, appLifecycle, cfgSrv, config.Factory, nil, application),
		cliffAdapter.RouteAdapter(adapter),
		thing.RouteCarCharger(adapter),
	)
}
