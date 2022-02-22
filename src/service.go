package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/discovery"
	"github.com/futurehomeno/fimpgo/edgeapp"
	log "github.com/sirupsen/logrus"

	easee2 "github.com/futurehomeno/edge-easee-adapter/_old/easee"
	model2 "github.com/futurehomeno/edge-easee-adapter/_old/model"
	"github.com/futurehomeno/edge-easee-adapter/_old/router"
	"github.com/futurehomeno/edge-easee-adapter/_old/utils"
)

func _main() {
	var workDir string
	flag.StringVar(&workDir, "c", "", "Work dir")
	flag.Parse()
	if workDir == "" {
		workDir = "./"
	} else {
		fmt.Println("Work dir ", workDir)
	}
	appLifecycle := edgeapp.NewAppLifecycle()
	configs := model2.NewConfigs(workDir)
	err := configs.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load config file.")
	}

	utils.SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("--------------Starting easee----------------")
	log.Info("Work directory : ", configs.WorkDir)
	appLifecycle.SetAppState(edgeapp.AppStateStarting, nil)
	appLifecycle.SetAuthState(edgeapp.AuthStateNotAuthenticated)
	appLifecycle.SetConfigState(edgeapp.ConfigStateNotConfigured)
	appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)

	mqtt := fimpgo.NewMqttTransport(configs.MqttServerURI, configs.MqttClientIdPrefix, configs.MqttUsername, configs.MqttPassword, true, 1, 1)
	err = mqtt.Start()
	defer mqtt.Stop()

	responder := discovery.NewServiceDiscoveryResponder(mqtt)
	responder.RegisterResource(model2.GetDiscoveryResource())
	responder.Start()

	userToken := easee2.UserToken{}
	client, err := easee2.NewClient(&userToken)
	easee := easee2.NewEasee(client, configs.WorkDir)
	err = easee.LoadProductsFromFile()
	if err != nil {
		log.Debug("Can't load easee state file.")
		log.Error(err)
	}

	fimpRouter := router.NewFromFimpRouter(mqtt, easee, appLifecycle, configs)
	fimpRouter.Start()
	//------------------ Remote API check -- !!!!IMPORTANT!!!!-------------
	// The app MUST perform remote API availability check.
	// During gateway boot process the app might be started before network is initialized or another local app booted.
	if err := edgeapp.NewSystemCheck().WaitForInternet(5 * time.Minute); err == nil {
		log.Info("<main> Internet connection - OK")
	} else {
		log.Error("<main> Internet connection - ERROR")
	}
	//--------------------------------------------------------------------
	// Check if adapter is configured
	if configs.IsConfigured() {
		userToken.AccessToken = configs.AccessToken
		userToken.RefreshToken = configs.RefreshToken
		easee.SetUserToken(&userToken)
		appLifecycle.SetAuthState(edgeapp.AuthStateAuthenticated)
		// Check if token is expired
		if configs.IsTokenExpired() {
			// Get new tokens with refreshtoken
			log.Info("Refreshing tokens")
			newUserToken, err := client.RefreshTokens()
			if err != nil || newUserToken == nil {
				log.Debug("Did not manage to refeshtokens")
				log.Error(err)
				appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)
			} else {
				configs.AccessToken = newUserToken.AccessToken
				configs.RefreshToken = newUserToken.RefreshToken
				configs.SetExpiresAt(newUserToken.ExpiresIn)
				configs.SaveToFile()
				easee.SetUserToken(newUserToken)
			}
		}
		if easee.IsConfigured() {
			log.Debug("Easee is configured")
			appLifecycle.SetConfigState(edgeapp.ConfigStateConfigured)
			appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)
			appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)

			err := fimpRouter.SendStateForAllChargers()
			if err != nil {
				log.Error(err)
			}
			err = fimpRouter.SendWattReportForAllProducts()
			if err != nil {
				log.Error(err)
			}
			err = fimpRouter.SendLifetimeEnergyReportIfValueChanged()
			if err != nil {
				log.Error(err)
			}
			err = fimpRouter.SendCableReportForAllProducts()
			if err != nil {
				log.Error(err)
			}
			err = fimpRouter.SendSessionEnergyReportForAllProducts()
			if err != nil {
				log.Error(err)
			}

		} else {
			// Need to configure Easee and set correct state. What if it doesn't work?
			err = easee.GetProducts()
			if err != nil {
				log.Error(err)
			}
			err = easee.GetConfigForAllProducts()
			if err != nil {
				log.Error(err)
			}
			err = easee.GetStateForAllProducts()
			if err != nil {
				log.Error(err)
			}
			easee.SaveProductsToFile()
			fimpRouter.SendInclusionReports()
			appLifecycle.SetConfigState(edgeapp.ConfigStateConfigured)
			appLifecycle.SetConnectionState(edgeapp.ConnStateConnected)
			appLifecycle.SetAppState(edgeapp.AppStateRunning, nil)
		}

	} else {
		appLifecycle.SetAppState(edgeapp.AppStateNotConfigured, nil)
	}

	//------------------ Sample code --------------------------------------

	for {
		appLifecycle.WaitForState("main", edgeapp.SystemEventTypeConfigState, edgeapp.ConfigStateConfigured)
		log.Info("<main> Starting ticker")
		ticker := time.NewTicker(time.Duration(configs.PollTimeSec) * time.Second)
		for ; true; <-ticker.C {
			if appLifecycle.AuthState() == edgeapp.AuthStateAuthenticated {
				if configs.IsTokenExpired() {
					newUserToken, err := client.RefreshTokens()
					if err != nil || newUserToken == nil {
						log.Debug("Did not manage to refeshtokens")
						log.Error(err)
						appLifecycle.SetConnectionState(edgeapp.ConnStateDisconnected)
						continue
					} else {
						configs.AccessToken = newUserToken.AccessToken
						configs.RefreshToken = newUserToken.RefreshToken
						configs.SetExpiresAt(newUserToken.ExpiresIn)
						configs.SaveToFile()
						easee.SetUserToken(newUserToken)
					}
				}

				err := easee.GetStateForAllProducts()
				if err != nil {
					log.Error(err)
				}

				err = fimpRouter.SendChangedStateForAllChargers()
				if err != nil {
					log.Error(err)
				}

				err = fimpRouter.SendWattReportIfValueChanged()
				if err != nil {
					log.Error(err)
				}

				err = fimpRouter.SendLifetimeEnergyReportIfValueChanged()
				if err != nil {
					log.Error(err)
				}

				err = fimpRouter.SendCableReportIfChanged()
				if err != nil {
					log.Error(err)
				}

				err = fimpRouter.SendSessionEnergyReportIfValueChanged()
				if err != nil {
					log.Error(err)
				}

				// TODO: improve ticker
				log.Debug("stop ticker and start new one")
				ticker.Stop()
				ticker = time.NewTicker(time.Duration(configs.PollTimeSec) * time.Second)
				easee.SaveProductsToFile()
			}
		}
		//TODO: Add logic here
		appLifecycle.WaitForState("main", edgeapp.SystemEventTypeConfigState, edgeapp.ConfigStateNotConfigured)
	}
}
