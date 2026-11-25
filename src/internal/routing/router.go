package routing

import (
	"fmt"
	"strings"

	cliffAdapter "github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/bootstrap"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/selection"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
)

// Controller is the part of a charger's controller the routing drives.
type Controller interface {
	chargepoint.AdjustableOfferedCurrentController
	Bind(request *fimpgo.Message)
	Unbind(request *fimpgo.Message)
}

func New(
	cfgSrv *config.Service,
	appLifecycle *lifecycle.Lifecycle,
	application app.App,
	adapter cliffAdapter.Adapter,
	tel telemetry.Telemetry,
	controllerByAddress func(string) Controller,
) []*router.Routing {
	// Shared by the app and adapter routes so cmd.thing.delete cannot interleave with
	// cmd.config.extended_set rewriting the selection it reads.
	locker := router.NewMessageHandlerLocker()

	devices := selection.NewStore(
		func() selection.Selection { return cfgSrv.SelectedDevices() },
		func(next selection.Selection) error { return cfgSrv.SetSelectedDevices(next) },
	)

	return router.Combine(
		[]*router.Routing{router.NewRouting(bindCommand(adapter, controllerByAddress, true), router.ForService(chargepoint.Chargepoint))},
		bootstrap.DefaultRoute(fimptype.EaseeService, func() any { return cfgSrv.PublicConfig() }, tel),
		[]*router.Routing{
			routeOfferedPause(adapter, controllerByAddress),
			cliffConfig.RouteCmdConfigGetInt(fimptype.EaseeService, "initial_charging_current", cfgSrv.InitialChargingCurrent),
			cliffConfig.RouteCmdConfigSetInt(fimptype.EaseeService, "initial_charging_current", cfgSrv.SetInitialChargingCurrent),
			cliffConfig.RouteCmdConfigGetInt(fimptype.EaseeService, "min_charging_current", cfgSrv.MinChargingCurrent),
			cliffConfig.RouteCmdConfigSetInt(fimptype.EaseeService, "min_charging_current", cfgSrv.SetMinChargingCurrent),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "polling_interval", cfgSrv.PollingInterval),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "polling_interval", cfgSrv.SetPollingInterval),
			cliffConfig.RouteCmdConfigGetDuration(fimptype.EaseeService, "current_wait_duration", cfgSrv.CurrentWaitDuration),
			cliffConfig.RouteCmdConfigSetDuration(fimptype.EaseeService, "current_wait_duration", cfgSrv.SetCurrentWaitDuration),
			cliffConfig.RouteCmdConfigGetString(fimptype.EaseeService, "easee_base_url", cfgSrv.EaseeBaseURL),
			cliffConfig.RouteCmdConfigSetString(fimptype.EaseeService, "easee_base_url", cfgSrv.SetEaseeBaseURL),
			cliffConfig.RouteCmdConfigGetFloat(fimptype.EaseeService, "slow_charging_current_in_amperes", cfgSrv.SlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigSetFloat(fimptype.EaseeService, "slow_charging_current_in_amperes", cfgSrv.SetSlowChargingCurrentInAmperes),
			cliffConfig.RouteCmdConfigGetBool(fimptype.EaseeService, "use_locked_3_phases", cfgSrv.UseLocked3Phases),
			cliffConfig.RouteCmdConfigSetBool(fimptype.EaseeService, "use_locked_3_phases", cfgSrv.SetUseLocked3Phases),
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
		[]*router.Routing{router.NewRouting(bindCommand(adapter, controllerByAddress, false), router.ForService(chargepoint.Chargepoint))},
	)
}

// bindCommand runs first and last on every chargepoint message, so a write the controller defers knows the command
// to answer if Easee refuses it later. Both match on the service only: the divert may rename the interface between.
// Report requests are not bound, so they never wait for a command of the same charger.
func bindCommand(ad cliffAdapter.Adapter, controllerByAddress func(string) Controller, bind bool) router.MessageHandlerFn {
	return func(message *fimpgo.Message) *fimpgo.Message {
		t := ad.ThingByTopic(message.Topic)
		if t == nil || strings.HasSuffix(message.Payload.Interface, ".get_report") {
			return nil
		}

		if ctrl := controllerByAddress(t.Address()); ctrl != nil && bind {
			ctrl.Bind(message)
		} else if ctrl != nil {
			ctrl.Unbind(message)
		}

		return nil
	}
}

// routeOfferedPause hands offered currents under 6 A to the controller, which pauses below min_charging_current and
// writes the rest. Cliffhanger checks an offered current only against 0 and sup_max_current, so these values would
// reach the controller through it as well (issue 56). On success the message interface is renamed so the chargepoint
// handler skips the message.
func routeOfferedPause(ad cliffAdapter.Adapter, controllerByAddress func(string) Controller) *router.Routing {
	return router.NewRouting(
		router.NewMessageHandler(router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			return nil, divertOfferedPause(ad, controllerByAddress, message)
		})),
		router.ForService(chargepoint.Chargepoint),
		router.ForType(chargepoint.CmdCurrentSessionSetCurrent),
	)
}

// The returned error is answered to the caller, as cliffhanger's handler answers its own.
func divertOfferedPause(ad cliffAdapter.Adapter, controllerByAddress func(string) Controller, message *fimpgo.Message) error {
	current, err := message.Payload.GetIntValue()
	if err != nil || current < 0 || current >= 6 {
		return nil //nolint:nilerr // A malformed value is cliffhanger's handler's to answer.
	}

	t := ad.ThingByTopic(message.Topic)
	if t == nil {
		return nil
	}

	ctrl := controllerByAddress(t.Address())
	if ctrl == nil {
		return nil
	}

	if err := ctrl.SetChargepointOfferedCurrent(current); err != nil {
		log.Errorf("[routing] Offered %dA on %s err: %v", current, message.Topic, err)

		return nil
	}

	message.Payload.Interface += ".paused"

	// The renamed message skips cliffhanger's handler and with it the confirmation report.
	if s, ok := t.ServiceByTopic(message.Topic).(chargepoint.Service); ok {
		if _, err := s.SendCurrentSessionReport(true); err != nil {
			return fmt.Errorf("offered %d A set, session report failed: %w", current, err)
		}
	}

	return nil
}
