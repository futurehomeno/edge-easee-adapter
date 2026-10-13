package routing

import (
	cliffAdapter "github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/bootstrap"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/selection"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application app.App,
	adapter cliffAdapter.Adapter,
	tel telemetry.Telemetry,
) []*router.Routing {
	// Shared by the app and adapter routes so cmd.thing.delete cannot interleave with
	// cmd.config.extended_set rewriting the selection it reads.
	locker := router.NewMessageHandlerLocker()

	devices := selection.NewStore(
		func() selection.Selection { return cfgSrv.SelectedDevices() },
		func(next selection.Selection) error { return cfgSrv.SetSelectedDevices(next) },
	)

	return router.Combine(
		bootstrap.DefaultRoute(fimptype.EaseeService, func() any { return cfgSrv.PublicConfig() }, tel),
		[]*router.Routing{
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "polling_interval", cfgSrv.PollingInterval),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "polling_interval", cfgSrv.SetPollingInterval),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "current_wait_duration", cfgSrv.CurrentWaitDuration),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "current_wait_duration", cfgSrv.SetCurrentWaitDuration),
			cliffConfig.RouteCmdConfigGetString(fimptype.EaseeService, "easee_base_url", cfgSrv.EaseeBaseURL),
			cliffConfig.RouteCmdConfigSetString(fimptype.EaseeService, "easee_base_url", cfgSrv.SetEaseeBaseURL),
			cliffConfig.RouteCmdConfigGetFloat(fimptype.EaseeService, "slow_charging_current_in_amperes", cfgSrv.SlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigSetFloat(fimptype.EaseeService, "slow_charging_current_in_amperes", cfgSrv.SetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "http_timeout", cfgSrv.HTTPTimeout),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "http_timeout", cfgSrv.SetHTTPTimeout),
			cliffConfig.RouteCmdConfigGetString(fimptype.EaseeService, "signalr_base_url", cfgSrv.SignalRBaseURL),
			cliffConfig.RouteCmdConfigSetString(fimptype.EaseeService, "signalr_base_url", cfgSrv.SetSignalRBaseURL),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_conn_creation_timeout", cfgSrv.SignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_conn_creation_timeout", cfgSrv.SetSignalRConnCreationTimeout),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_keep_alive_interval", cfgSrv.SignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_keep_alive_interval", cfgSrv.SetSignalRKeepAliveInterval),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_timeout_interval", cfgSrv.SignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_timeout_interval", cfgSrv.SetSignalRTimeoutInterval),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_initial_backoff", cfgSrv.SignalRInitialBackoff),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_initial_backoff", cfgSrv.SetSignalRInitialBackoff),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_repeated_backoff", cfgSrv.SignalRRepeatedBackoff),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_repeated_backoff", cfgSrv.SetSignalRRepeatedBackoff),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_final_backoff", cfgSrv.SignalRFinalBackoff),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_final_backoff", cfgSrv.SetSignalRFinalBackoff),
			cliffConfig.RouteCmdConfigGetInt(fimptype.EaseeService, "signalr_initial_failure_count", cfgSrv.SignalRInitialFailureCount),
			cliffConfig.RouteCmdConfigSetInt(fimptype.EaseeService, "signalr_initial_failure_count", cfgSrv.SetSignalRInitialFailureCount),
			cliffConfig.RouteCmdConfigGetInt(fimptype.EaseeService, "signalr_repeated_failure_count", cfgSrv.SignalRRepeatedFailureCount),
			cliffConfig.RouteCmdConfigSetInt(fimptype.EaseeService, "signalr_repeated_failure_count", cfgSrv.SetSignalRRepeatedFailureCount),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "signalr_invoke_timeout", cfgSrv.SignalRInvokeTimeout),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "signalr_invoke_timeout", cfgSrv.SetSignalRInvokeTimeout),
		},
		app.RouteApp(fimptype.EaseeService, appLifecycle, cfgSrv, config.Factory, locker, application, nil),
		cliffAdapter.RouteAdapter(adapter, cliffAdapter.WithSelection(devices, locker)),
		thing.RouteCarCharger(adapter),
	)
}
