package router

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/edgeapp"

	log "github.com/sirupsen/logrus"

	easee2 "github.com/futurehomeno/edge-easee-adapter/_old/easee"
	"github.com/futurehomeno/edge-easee-adapter/_old/model"
)

// FromFimpRouter structure for fimp router
type FromFimpRouter struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	instanceID   string
	appLifecycle *edgeapp.Lifecycle
	configs      *model.Configs
	easee        *easee2.Easee
}

// NewFromFimpRouter router to handle fimp messages
func NewFromFimpRouter(mqt *fimpgo.MqttTransport, easee *easee2.Easee, appLifecycle *edgeapp.Lifecycle, configs *model.Configs) *FromFimpRouter {
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

		case "cmd.current_session.get_report":
			log.Debug("cmd.current_session.get_report")
			chargerID := newMsg.Addr.ServiceAddress
			for _, product := range fc.easee.Products {
				if chargerID == product.Charger.ID {
					log.Debug("found correct charger: ", chargerID)
					err := fc.SendSessionEnergyReport(chargerID, newMsg)
					if err != nil {
						log.Error(err)
					}
					break
				}
			}

		case "cmd.charge.start":
			log.Debug("cmd.charge.start")
			chargerID := newMsg.Addr.ServiceAddress

			err := fc.easee.ResumeCharging(chargerID)
			if err != nil {
				log.Error("Error starting charging", err)

				return
			}

			fc.SendChangerStateEvent(chargerID, "charging", newMsg)

		case "cmd.charge.stop":
			log.Debug("cmd.charge.stop")
			chargerID := newMsg.Addr.ServiceAddress

			err := fc.easee.PauseCharging(chargerID)
			if err != nil {
				log.Error("Error stopping charging", err)

				return
			}

			fc.SendChangerStateEvent(chargerID, "ready_to_charge", newMsg)

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
		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: fc.configs.InstanceAddress}
		switch newMsg.Payload.Type {
		case "cmd.auth.login":
			if fc.appLifecycle.AuthState() == edgeapp.AuthStateNotAuthenticated {
				log.Info("cmd.auth.login - App not authenticated")
				login := easee2.Login{}
				err := newMsg.Payload.GetObjectValue(&login)
				if err != nil {
					log.Error("Incorrect login message ")
					return
				}

				fc.appLifecycle.SetAuthState(edgeapp.AuthStateInProgress)

				if login.Username != "" && login.Password != "" {
					err := fc.easee.Login(login)

					authenticated := false
					configured := false
					connected := false

					if err != nil {
						log.Debug(err)
						authenticated = false
					} else {
						authenticated = true

						fc.configs.AccessToken = fc.easee.GetAccessToken()
						fc.configs.RefreshToken = fc.easee.GetRefreshToken()
						fc.configs.SetExpiresAt(fc.easee.GetExpiresIn())
						fc.configs.SaveToFile()
						// fc.appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated) // not necessarily. fc.easee.Login ^ does not actually return err if username or password is wrong.
						fc.appLifecycle.SetConfigState(edgeapp.ConfigStateInProgress)

						err = fc.easee.GetProducts()
						if err != nil {
							log.Error(err)
							configured = false
						} else {
							configured = true
							err = fc.easee.GetConfigForAllProducts()
							if err != nil {
								log.Error(err)
								configured = false
							} else {
								err = fc.easee.GetStateForAllProducts()
								if err != nil {
									log.Error(err)
									connected = false
								} else {
									connected = true
								}
							}
						}
					}

					if authenticated && configured && connected {
						log.Debug("authenticated && configured && connected")
						fc.easee.SaveProductsToFile()
						fc.SendInclusionReports()
						fc.SendStateForAllChargers()
						fc.SendWattReportForAllProducts()
						fc.appLifecycle.SetConfigState(edgeapp.ConfigStateConfigured)
						fc.appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
						fc.appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)
						fc.appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)
						msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
						msg.Source = model.ServiceName
						if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
							// if response topic is not set , sending back to default application event topic
							fc.mqt.Publish(adr, msg)
						}
						log.Debug("sent auth.status_report")

					} else {
						log.Debug("not authenticated")
						fc.appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
						fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
						fc.appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)
						fc.appLifecycle.SetAppState(edgeapp.AppStateNotConfigured, nil)

						loginval := map[string]interface{}{
							"errors":  "Wrong username or password",
							"success": false,
						}
						log.Debug("loginval; ", loginval)
						newadr, _ := fimpgo.NewAddressFromString("pt:j1/mt:rsp/rt:cloud/rn:remote-client/ad:smarthome-app")
						msg := fimpgo.NewMessage("evt.pd7.response", "vinculum", fimpgo.VTypeObject, loginval, nil, nil, newMsg.Payload)
						fc.mqt.Publish(newadr, msg)
						log.Debug("loginval; ", loginval)
					}
				}
				//loginMsg := fimpgo.NewMessage("evt.auth.login_report", model.ServiceName, fimpgo.VTypeString, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)

			} else if fc.appLifecycle.AuthState() == edgeapp.AuthStateInProgress {
				log.Info("cmd.auth.login - auth state in progress ")
			} else if fc.appLifecycle.AuthState() == edgeapp.AuthStateAuthenticated {
				log.Info("cmd.auth.login - app is already authenticated")
			} else {
				log.Info("cmd.auth.login - missing auth state")
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
		case "cmd.auth.logout":
			// Exclud all products
			// Remove config and products.json
			// Clear tokens in memory
			// Send fimp message with auth state
			if fc.easee.IsConfigured() {
				fc.SendExclusionReportForAllChargers()
				fc.easee.ClearProducts()
			}

			// This was previously in the same if-statement as above. Changed to always excecute, as if "cmd.auth.logout" is sent, the app clearly thinks that the user
			// is logger in to easee, even though it may not be. Thus the integration needs to be reconfigured to proper states, aka not logged in. Also, evt.auth.status_report
			// needs to be returned to avoid the app from getting stuck in loading.
			fc.configs.ClearTokens()
			fc.appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
			fc.appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
			fc.appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)
			fc.appLifecycle.SetAppState(edgeapp.AppStateNotConfigured, nil)

			msg := fimpgo.NewMessage("evt.auth.status_report", model.ServiceName, fimpgo.VTypeObject, fc.appLifecycle.GetAllStates(), nil, nil, newMsg.Payload)
			msg.Source = model.ServiceName
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				// if response topic is not set , sending back to default application event topic
				fc.mqt.Publish(adr, msg)
			}
			log.Info("Logged out successfully")

		case "cmd.app.uninstall":
			// Excldue all products
			fc.SendExclusionReportForAllChargers()
			fc.easee.ClearProducts()
			fc.configs.ClearTokens()
		}

	}

}
