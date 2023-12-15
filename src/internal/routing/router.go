package routing

import (
	cliffAdapter "github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

const (
	// ServiceName represents Easee service name.
	ServiceName = "easee"
	// ResourceName represents Easee source name.
	ResourceName = "easee"
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
			cliffConfig.RouteCmdLogGetLevel(ServiceName, cfgSrv.GetLogLevel),
			cliffConfig.RouteCmdLogSetLevel(ServiceName, cfgSrv.SetLogLevel),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "polling_interval", cfgSrv.GetPollingInterval),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "polling_interval", cfgSrv.SetPollingInterval),
			cliffConfig.RouteCmdConfigGetString(ServiceName, "easee_base_url", cfgSrv.GetEaseeBaseURL),
			cliffConfig.RouteCmdConfigSetString(ServiceName, "easee_base_url", cfgSrv.SetEaseeBaseURL),
			cliffConfig.RouteCmdConfigGetFloat(ServiceName, "slow_charging_current_in_amperes", cfgSrv.GetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigSetFloat(ServiceName, "slow_charging_current_in_amperes", cfgSrv.SetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "http_timeout", cfgSrv.GetHTTPTimeout),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "http_timeout", cfgSrv.SetHTTPTimeout),
			cliffConfig.RouteCmdConfigGetString(ServiceName, "signalr_base_url", cfgSrv.GetSignalRBaseURL),
			cliffConfig.RouteCmdConfigSetString(ServiceName, "signalr_base_url", cfgSrv.SetSignalRBaseURL),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "signalr_conn_creation_timeout", cfgSrv.GetSignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "signalr_conn_creation_timeout", cfgSrv.SetSignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "signalr_keep_alive_interval", cfgSrv.GetSignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "signalr_keep_alive_interval", cfgSrv.SetSignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "signalr_timeout_interval", cfgSrv.GetSignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "signalr_timeout_interval", cfgSrv.SetSignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "signalr_initial_backoff", cfgSrv.GetSignalRInitialBackoff),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "signalr_initial_backoff", cfgSrv.SetSignalRInitialBackoff),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "signalr_repeated_backoff", cfgSrv.GetSignalRRepeatedBackoff),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "signalr_repeated_backoff", cfgSrv.SetSignalRRepeatedBackoff),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "signalr_final_backoff", cfgSrv.GetSignalRFinalBackoff),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "signalr_final_backoff", cfgSrv.SetSignalRFinalBackoff),
			cliffConfig.RouteCmdConfigGetInt(ServiceName, "signalr_initial_failure_count", cfgSrv.GetSignalRInitialFailureCount),
			cliffConfig.RouteCmdConfigSetInt(ServiceName, "signalr_initial_failure_count", cfgSrv.SetSignalRInitialFailureCount),
			cliffConfig.RouteCmdConfigGetInt(ServiceName, "signalr_repeated_failure_count", cfgSrv.GetSignalRRepeatedFailureCount),
			cliffConfig.RouteCmdConfigSetInt(ServiceName, "signalr_repeated_failure_count", cfgSrv.SetSignalRRepeatedFailureCount),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "signalr_invoke_timeout", cfgSrv.GetSignalRInvokeTimeout),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "signalr_invoke_timeout", cfgSrv.SetSignalRInvokeTimeout),
			cliffConfig.RouteCmdConfigGetDuration(ServiceName, "backoff_length", cfgSrv.GetBackoffLength),
			cliffConfig.RouteCmdConfigSetDuration(ServiceName, "backoff_length", cfgSrv.SetBackoffLength),
			cliffConfig.RouteCmdConfigGetInt(ServiceName, "backoff_max_attempts", cfgSrv.GetBackoffMaxAttempts),
			cliffConfig.RouteCmdConfigSetInt(ServiceName, "backoff_max_attempts", cfgSrv.SetBackoffMaxAttempts),
		},
		app.RouteApp(ServiceName, appLifecycle, cfgSrv, config.Factory, nil, application),
		cliffAdapter.RouteAdapter(adapter),
		thing.RouteCarCharger(adapter),
	)
}
