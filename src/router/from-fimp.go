package router

import (
	"fmt"
	"path/filepath"

	//"github.com/thingsplex/easee/easee"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/edgeapp"
	"github.com/thingsplex/easee-ad/easee"

	log "github.com/sirupsen/logrus"
	"github.com/thingsplex/easee-ad/model"
)

// FromFimpRouter structure for fimp router
type FromFimpRouter struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	instanceID   string
	appLifecycle *edgeapp.Lifecycle
	configs      *model.Configs
	easee        *easee.Easee
}

// NewFromFimpRouter router to handle fimp messages
func NewFromFimpRouter(mqt *fimpgo.MqttTransport, easee *easee.Easee, appLifecycle *edgeapp.Lifecycle, configs *model.Configs) *FromFimpRouter {
	fc := FromFimpRouter{
		inboundMsgCh: make(fimpgo.MessageCh, 5),
		mqt:          mqt,
		easee:        easee,
		appLifecycle: appLifecycle,
		configs:      configs,
	}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)
	return &fc
}

// Start fimp router
func (fc *FromFimpRouter) Start() {

	// ------ Adapter topics ---------------------------------------------
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:dev/rn:%s/ad:1/#", model.ServiceName))
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:ad/rn:%s/ad:1", model.ServiceName))

	go func(msgChan fimpgo.MessageCh) {
		for {
			select {
			case newMsg := <-msgChan:
				fc.routeFimpMessage(newMsg)
			}
		}
	}(fc.inboundMsgCh)
}

func (fc *FromFimpRouter) routeFimpMessage(newMsg *fimpgo.Message) {
	//addr := strings.Replace(newMsg.Addr.ServiceAddress, "_0", "", 1)
	switch newMsg.Payload.Service {
	case "chargepoint":
		switch newMsg.Payload.Type {
		case "cmd.state.get_report":
			log.Debug("cmd.state.get_report")
			chargerID := newMsg.Addr.ServiceAddress
			err := fc.easee.GetChargerState(chargerID)
			if err != nil {
				log.Error(err)
				break
			}
			err = fc.SendChargerState(chargerID, newMsg)
			if err != nil {
				log.Error(err)
				break
			}

		case "cmd.mode.set":
			log.Debug("cmd.mode.set")
			chargerID := newMsg.Addr.ServiceAddress
			val, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Incorrect value format")
				return
			}
			switch val {
			case "start":
				err := fc.easee.StartCharging(chargerID)
				if err != nil {
					log.Error("Error starting charging", err)
				}
				fc.SendChangerModeEvent(chargerID, "start", newMsg)
			case "stop":
				err := fc.easee.StopCharing(chargerID)
				if err != nil {
					log.Error("Error stopping charging", err)
				}
				fc.SendChangerModeEvent(chargerID, "stop", newMsg)
			case "pause":
				err := fc.easee.PauseCharging(chargerID)
				if err != nil {
					log.Error("pause charging failed - ", err)
				}
				fc.SendChangerModeEvent(chargerID, "pause", newMsg)
			case "resume":
				err := fc.easee.ResumeCharging(chargerID)
				if err != nil {
					log.Error("resume charging failed - ", err)
				}
				fc.SendChangerModeEvent(chargerID, "resume", newMsg)
			}
		}
	case "meter_elec":
		switch newMsg.Payload.Type {
		case "cmd.meter.get_report":
			log.Debug("cmd.meter.get_report")
			chargerID := newMsg.Addr.ServiceAddress
			unit, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Incorrect value format")
				return
			}
			err = fc.easee.GetChargerState(chargerID)
			if err != nil {
				log.Error(err)
				return
			}
			err = fc.SendMeterReport(chargerID, unit, newMsg)
			if err != nil {
				log.Error(err)
			}
		}

	case "out_lvl_switch":
		//addr = strings.Replace(addr, "l", "", 1)
		switch newMsg.Payload.Type {
		case "cmd.binary.set":
			log.Debug("out_lvl_switch - cmd.binary.set")
			fc.SendInclusionReports()
			// TODO: This is example . Add your logic here or remove
		case "cmd.lvl.set":
			// TODO: This is an example . Add your logic here or remove
		}
	case "out_bin_switch":
		// TODO: delete this test
		log.Debug("Sending switch")
		err := fc.easee.GetProducts()
		if err != nil {
			log.Error(err)
		}
		err = fc.easee.GetConfigForAllProducts()
		if err != nil {
			log.Error(err)
		}
		fc.easee.GetStateForAllProducts()
		if err != nil {
			log.Error(err)
		}
		fc.easee.SaveProductsToFile()

		// time.Sleep(10 * time.Second)
		// fc.easee.ClearProducts()

		// TODO: This is an example . Add your logic here or remove
	case model.ServiceName:
		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
		switch newMsg.Payload.Type {
		case "cmd.auth.login":
			if fc.appLifecycle.AuthState() == edgeapp.AuthStateNotAuthenticated {
				log.Info("cmd.auth.login - App not authenticated")
				login := easee.Login{}
				err := newMsg.Payload.GetObjectValue(&login)
				if err != nil {
					log.Error("Incorrect login message ")
					return
				}
				fc.appLifecycle.SetAuthState(edgeapp.AuthStateInProgress)

				if login.Username != "" && login.Password != "" {
					err := fc.easee.Login(login)
					if err != nil {
						log.Debug(err)
					}
					fc.configs.AccessToken = fc.easee.GetAccessToken()
					fc.configs.RefreshToken = fc.easee.GetRefreshToken()
					fc.configs.SetExpiresAt(fc.easee.GetExpiresIn())
					fc.configs.SaveToFile()
					fc.appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
					fc.appLifecycle.SetConfigState(edgeapp.ConfigStateInProgress)

					err = fc.easee.GetProducts()
					if err != nil {
						log.Error(err)
					}
					err = fc.easee.GetConfigForAllProducts()
					if err != nil {
						log.Error(err)
					}
					fc.easee.GetStateForAllProducts()
					if err != nil {
						log.Error(err)
					}
					fc.easee.SaveProductsToFile()
					fc.SendInclusionReports()
					fc.appLifecycle.SetConfigState(edgeapp.ConfigStateConfigured)
					fc.appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)
				} else {
					fc.appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
					fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
				}
				//loginMsg := fimpgo.NewMessage("evt.auth.login_report", model.ServiceName, fimpgo.VTypeString, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)

				msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
				msg.Source = model.ServiceName
				if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
					// if response topic is not set , sending back to default application event topic
					fc.mqt.Publish(adr, msg)
				}
			} else if fc.appLifecycle.AuthState() == edgeapp.AuthStateInProgress {
				log.Info("cmd.auth.login - auth state in progress ")
			} else if fc.appLifecycle.AuthState() == edgeapp.AuthStateAuthenticated {
				log.Info("cmd.auth.login - app is already authenticated")
			} else {
				log.Info("cmd.auth.login - missing auth state")
			}

		case "cmd.app.get_manifest":
			mode, err := newMsg.Payload.GetStringValue()
			if err != nil {
				log.Error("Incorrect request format ")
				return
			}
			manifest := edgeapp.NewManifest()
			err = manifest.LoadFromFile(filepath.Join(fc.configs.GetDefaultDir(), "app-manifest.json"))
			if err != nil {
				log.Error("Failed to load manifest file .Error :", err.Error())
				return
			}
			if mode == "manifest_state" {
				manifest.AppState = *fc.appLifecycle.GetAllStates()
				manifest.ConfigState = fc.configs
			}
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, manifest, nil, nil, newMsg.Payload)
			msg.Source = model.ServiceName
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.get_state":
			msg := fimpgo.NewMessage("evt.app.manifest_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
			msg.Source = model.ServiceName
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.get_extended_report":

			msg := fimpgo.NewMessage("evt.config.extended_report", model.ServiceName, fimpgo.VTypeObject, fc.configs, nil, nil, newMsg.Payload)
			msg.Source = model.ServiceName
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.extended_set":
			conf := model.Configs{}
			err := newMsg.Payload.GetObjectValue(&conf)
			if err != nil {
				// TODO: This is an example . Add your logic here or remove
				log.Error("Can't parse configuration object")
				return
			}
			fc.configs.Param1 = conf.Param1
			fc.configs.Param2 = conf.Param2
			fc.configs.SaveToFile()
			log.Debugf("App reconfigured . New parameters : %v", fc.configs)
			// TODO: This is an example . Add your logic here or remove
			configReport := edgeapp.ConfigReport{
				OpStatus: "ok",
				AppState: *fc.appLifecycle.GetAllStates(),
			}
			msg := fimpgo.NewMessage("evt.app.config_report", model.ServiceName, fimpgo.VTypeObject, configReport, nil, nil, newMsg.Payload)
			msg.Source = model.ServiceName
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.log.set_level":
			// Configure log level
			level, err := newMsg.Payload.GetStringValue()
			if err != nil {
				return
			}
			logLevel, err := log.ParseLevel(level)
			if err == nil {
				log.SetLevel(logLevel)
				fc.configs.LogLevel = level
				fc.configs.SaveToFile()
			}
			log.Info("Log level updated to = ", logLevel)

		case "cmd.system.reconnect":
			// This is optional operation.
			//fc.appLifecycle.PublishEvent(edgeapp.EventConfigured, "from-fimp-router", nil)
			//val := map[string]string{"status":status,"error":errStr}
			val := edgeapp.ButtonActionResponse{
				Operation:       "cmd.system.reconnect",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			msg.Source = model.ServiceName
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.app.factory_reset":
			val := edgeapp.ButtonActionResponse{
				Operation:       "cmd.app.factory_reset",
				OperationStatus: "ok",
				Next:            "config",
				ErrorCode:       "",
				ErrorText:       "",
			}
			fc.appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
			fc.appLifecycle.SetAppState(edgeapp.AppStateNotConfigured, nil)
			fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
			msg := fimpgo.NewMessage("evt.app.config_action_report", model.ServiceName, fimpgo.VTypeObject, val, nil, nil, newMsg.Payload)
			msg.Source = model.ServiceName
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.network.get_all_nodes":
			// TODO: This is an example . Add your logic here or remove
		case "cmd.thing.get_inclusion_report":
			chargerID, err := newMsg.Payload.GetStringValue()
			if err != nil {
				// handle err
				log.Error(fmt.Errorf("Can't get strValue, error: %s", err))
			}
			err = fc.SendInclusionReport(chargerID, newMsg.Payload)
			if err != nil {
				log.Error(fmt.Errorf("Did not manage to send inclusion report: %s", err))
			}
		case "cmd.thing.delete":
			// remove device from network
			val, err := newMsg.Payload.GetStrMapValue()
			if err != nil {
				log.Error("Wrong msg format")
				return
			}
			chargerID, ok := val["address"]
			if ok {
				log.Info(chargerID)
				err = fc.easee.RemoveProduct(chargerID)
				if err != nil {
					log.Debug(err)
					break
				}
				err = fc.SendExclusionReport(chargerID, newMsg.Payload)
				if err != nil {
					log.Debug(err)
				}
			} else {
				log.Error("Incorrect address")

			}
		}

	}

}
