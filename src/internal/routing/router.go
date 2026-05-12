package routing

import (
	cliffAdapter "github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/parameters"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
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
			debug.RouteCmdLogGetLevel(fimptype.EaseeService),
			debug.RouteCmdLogSetLevel(fimptype.EaseeService),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "polling_interval", cfgSrv.GetPollingInterval),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "polling_interval", cfgSrv.SetPollingInterval),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "current_wait_duration", cfgSrv.GetCurrentWaitDuration),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "current_wait_duration", cfgSrv.SetCurrentWaitDuration),
			cliffConfig.RouteCmdConfigGetString(fimptype.EaseeService, "easee_base_url", cfgSrv.GetEaseeBaseURL),
			cliffConfig.RouteCmdConfigSetString(fimptype.EaseeService, "easee_base_url", cfgSrv.SetEaseeBaseURL),
			cliffConfig.RouteCmdConfigGetFloat(fimptype.EaseeService, "slow_charging_current_in_amperes", cfgSrv.GetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigSetFloat(fimptype.EaseeService, "slow_charging_current_in_amperes", cfgSrv.SetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "http_timeout", cfgSrv.GetHTTPTimeout),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "http_timeout", cfgSrv.SetHTTPTimeout),
			cliffConfig.RouteCmdConfigGetString(fimptype.EaseeService, "signalr_base_url", cfgSrv.GetSignalRBaseURL),
			cliffConfig.RouteCmdConfigSetString(fimptype.EaseeService, "signalr_base_url", cfgSrv.SetSignalRBaseURL),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_conn_creation_timeout", cfgSrv.GetSignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_conn_creation_timeout", cfgSrv.SetSignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_keep_alive_interval", cfgSrv.GetSignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_keep_alive_interval", cfgSrv.SetSignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_timeout_interval", cfgSrv.GetSignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_timeout_interval", cfgSrv.SetSignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_initial_backoff", cfgSrv.GetSignalRInitialBackoff),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_initial_backoff", cfgSrv.SetSignalRInitialBackoff),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_repeated_backoff", cfgSrv.GetSignalRRepeatedBackoff),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_repeated_backoff", cfgSrv.SetSignalRRepeatedBackoff),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_final_backoff", cfgSrv.GetSignalRFinalBackoff),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_final_backoff", cfgSrv.SetSignalRFinalBackoff),
			cliffConfig.RouteCmdConfigGetInt(fimptype.EaseeService, "signalr_initial_failure_count", cfgSrv.GetSignalRInitialFailureCount),
			cliffConfig.RouteCmdConfigSetInt(fimptype.EaseeService, "signalr_initial_failure_count", cfgSrv.SetSignalRInitialFailureCount),
			cliffConfig.RouteCmdConfigGetInt(fimptype.EaseeService, "signalr_repeated_failure_count", cfgSrv.GetSignalRRepeatedFailureCount),
			cliffConfig.RouteCmdConfigSetInt(fimptype.EaseeService, "signalr_repeated_failure_count", cfgSrv.SetSignalRRepeatedFailureCount),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_invoke_timeout", cfgSrv.GetSignalRInvokeTimeout),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_invoke_timeout", cfgSrv.SetSignalRInvokeTimeout),
		},
		app.RouteApp(fimptype.EaseeService, appLifecycle, cfgSrv, config.Factory, nil, application, nil),
		cliffAdapter.RouteAdapter(adapter),
		thing.RouteCarCharger(adapter),
		parameters.RouteService(adapter),
	)
}
