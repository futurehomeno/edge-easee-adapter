package cmd

import (
	"io"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/test/suite"
	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/signalr"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/mocks"
)

const (
	cmdDeviceChargepointTopic = "pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1"
	evtDeviceChargepointTopic = "pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1"
	evtDeviceMeterElecTopic   = "pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:meter_elec/ad:1"
)

func TestEaseeAdapter(t *testing.T) { //nolint:paralleltest
	mqttAddr := test.SetupMQTTContainer(t)
	testContainer := newTestContainer(t)

	s := &suite.Suite{
		Config: suite.Config{
			MQTTServerURI: mqttAddr,
		},
		Cases: []*suite.Case{
			{
				Name: "Adapter is capable of reacting to incoming observations",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateAwaitingStart)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.TotalPower,
								Timestamp: time.Now(),
								Value:     "0",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Timestamp: time.Now(),
								Value:     "12.34",
							},
						})
						s.MockObservations(300*time.Millisecond, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateCharging)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.TotalPower,
								Timestamp: time.Now(),
								Value:     "1",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Timestamp: time.Now().Add(time.Hour),
								Value:     "13.45",
							},
							{
								// This observation should be skipped, as it's outdated.
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Timestamp: time.Now().Add(30 * time.Minute),
								Value:     "14.44",
							},
							{
								// This observation should be skipped, as it doesn't have a valid timestamp.
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Value:     "15.55",
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Expectations: []*suite.Expectation{
							// Initial batch
							suite.ExpectString(evtDeviceChargepointTopic, "evt.state.report", "chargepoint", "ready_to_charge"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 0).ExpectProperty("unit", "W"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 12.34).ExpectProperty("unit", "kWh"),

							// Update
							suite.ExpectString(evtDeviceChargepointTopic, "evt.state.report", "chargepoint", "charging"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 1000).ExpectProperty("unit", "W"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 13.45).ExpectProperty("unit", "kWh"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 14.44).ExpectProperty("unit", "kWh").
								Never(),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 15.55).
								Never(),
						},
					},
				},
			},
			{
				// Regression: we've encountered cases where chargers reported power usage and a state other than "charging".
				Name: "Adapter reports a charger state as charging based on power consumption",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateAwaitingStart)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.TotalPower,
								Timestamp: time.Now(),
								Value:     "0",
							},
						})
						s.MockObservations(300*time.Millisecond, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.TotalPower,
								Timestamp: time.Now(),
								Value:     "1.23",
							},
						})
						s.MockObservations(300*time.Millisecond, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateReadyToCharge)),
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Expectations: []*suite.Expectation{
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 0).ExpectProperty("unit", "W"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 1230).ExpectProperty("unit", "W"),
							suite.ExpectString(evtDeviceChargepointTopic, "evt.state.report", "chargepoint", "ready_to_charge").ExactlyOnce(),
							suite.ExpectString(evtDeviceChargepointTopic, "evt.state.report", "chargepoint", "charging").ExactlyOnce(),
						},
					},
				},
			},
			{
				Name: "Adapter should not report data if signalR connection is lost/not established",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup("localhost:1111", nil)),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Command:       suite.NullMessage(cmdDeviceChargepointTopic, "cmd.state.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError(evtDeviceChargepointTopic, "chargepoint"),
						},
					},
				},
			},
			{
				Name: "Lower lifetime energy reading should be skipped",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateCharging)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Timestamp: time.Now(),
								Value:     "12.34",
							},
						})
						s.MockObservations(200*time.Millisecond, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Timestamp: time.Now(),
								Value:     "11",
							},
						})
						s.MockObservations(200*time.Millisecond, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Timestamp: time.Now(),
								Value:     "13.45",
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Expectations: []*suite.Expectation{
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 12.34).ExpectProperty("unit", "kWh"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 13.45).ExpectProperty("unit", "kWh"),
							suite.ExpectFloat(evtDeviceMeterElecTopic, "evt.meter.report", "meter_elec", 11).Never(),
						},
					},
				},
			},
			{
				Name: "Get max current report",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{
							RatedCurrent: 32,
						}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateAwaitingStart)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.MaxChargerCurrent,
								Timestamp: time.Now(),
								Value:     "32",
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					suite.SleepNode(100 * time.Millisecond),
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Command:       suite.NullMessage(cmdDeviceChargepointTopic, "cmd.max_current.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt(evtDeviceChargepointTopic, "evt.max_current.report", "chargepoint", 32),
						},
					},
				},
			},
			{
				Name: "Set max current, too low value",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, nil)),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Command:       suite.IntMessage(cmdDeviceChargepointTopic, "cmd.max_current.set", "chargepoint", 0),
						Expectations: []*suite.Expectation{
							suite.ExpectError(evtDeviceChargepointTopic, "chargepoint"),
						},
					},
				},
			},
			{
				Name: "Extend Report Meter",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateCharging)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.TotalPower,
								Timestamp: time.Now(),
								Value:     "12",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.LifetimeEnergy,
								Timestamp: time.Now(),
								Value:     "13.45",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.InCurrentT3,
								Timestamp: time.Now(),
								Value:     "1",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.InCurrentT4,
								Timestamp: time.Now(),
								Value:     "2",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeDouble,
								ID:        signalr.InCurrentT5,
								Timestamp: time.Now(),
								Value:     "12.3",
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Expectations: []*suite.Expectation{
							extendMeterReportExpectation(map[string]float64{
								"i1": 1,
							}),
							extendMeterReportExpectation(map[string]float64{
								"i2": 2,
							}),
							extendMeterReportExpectation(map[string]float64{
								"i3": 12.3,
							}),
							extendMeterReportExpectation(map[string]float64{
								"e_import": 13.45,
							}),
							extendMeterReportExpectation(map[string]float64{
								"p_import": 12000,
							}),
						},
					},
				},
			},
			{
				Name: "Start session report after observation, no previous session",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{
							DetectedPowerGridType: api.GridTypeUnknown,
						}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{
							RatedCurrent: 32,
						}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateAwaitingStart)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeString,
								ID:        signalr.ChargingSessionStart,
								Timestamp: time.Now(),
								Value:     `{ "Auth": "", "AuthReason": 0, "Id": 435, "MeterValue": 1277.872637, "Start": "2025-01-22T12:51:47.000Z"}`,
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.current_session.report", "chargepoint", 0).
								ExpectProperty("offered_current", "0").
								ExpectProperty("started_at", "2025-01-22T12:51:47Z"),
						},
					},
				},
			},
			{
				Name: "Get sessions report",
				Setup: serviceSetup(
					testContainer,
					"configured",
					mqttAddr,
					func(client *mocks.APIClient) {
						client.On("ChargerConfig", "XX12345").Return(&api.ChargerConfig{
							DetectedPowerGridType: api.GridTypeUnknown,
						}, nil)
						client.On("ChargerSiteInfo", "XX12345").Return(&api.ChargerSiteInfo{
							RatedCurrent: 32,
						}, nil)
						client.On("Ping").Return(nil)
					},
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []signalr.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeInteger,
								ID:        signalr.ChargerOPState,
								Timestamp: time.Now(),
								Value:     strconv.Itoa(int(signalr.ChargerStateAwaitingStart)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  signalr.ObservationDataTypeString,
								ID:        signalr.ChargingSessionStop,
								Timestamp: time.Now(),
								Value: `{
										  "Auth": "",
										  "AuthReason": 0,
										  "EnergyKwh": 0.411273,
										  "Id": 435,
										  "MeterValueStart": 1277.872637,
										  "MeterValueStop": 1278.28391,
										  "Start": "2025-01-22T12:51:47.000Z",
										  "Stop": "2025-01-22T13:05:38.000Z"
										}`,
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{
							waitForRunning(),
							func(_ *testing.T) {
								time.Sleep(10 * time.Millisecond)
							},
						},
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "cmd.current_session.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.current_session.report", "chargepoint", 0).
								ExpectProperty("offered_current", "0").
								ExpectProperty("started_at", "2025-01-22T12:51:47Z").
								ExpectProperty("finished_at", "2025-01-22T13:05:38Z"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

//nolint:unparam
func serviceSetup(tc *testContainer, configSet, mqttAddr string, mockClientFn func(c *mocks.APIClient), opts ...func(tc *testContainer)) suite.ServiceSetup {
	return func(t *testing.T) (service suite.Service, _ []suite.Mock) {
		t.Helper()

		tearDown(configSet)(t)

		config := configSetup(t, configSet, mqttAddr)
		loggerSetup(t)

		for _, o := range opts {
			o(tc)
		}

		tc.SetUp()

		client := mocks.NewAPIClient(t)
		mockClientFn(client)

		services.easeeAPIClient = client

		app, err := Build(config)
		if err != nil {
			t.Fatalf("failed to build app: %s", err)
		}

		return app, nil
	}
}

func signalRSetup(addr string, setupServer func(s *test.SignalRServer)) func(tc *testContainer) {
	return func(tc *testContainer) {
		tc.signalRAddress = addr
		tc.signalRSetupFn = setupServer
	}
}

func tearDown(configSet string) suite.Callback {
	return func(t *testing.T) {
		t.Helper()

		resetContainer()
		cleanUpTestData(t, configSet)
	}
}

func cleanUpTestData(t *testing.T, configSet string) {
	t.Helper()

	workDir := path.Join("../../testdata/testing/", configSet)
	dataPath := path.Join(workDir, "/data/")
	defaultsPath := path.Join(workDir, "/defaults/")

	// clean up 'data' path
	err := os.RemoveAll(dataPath)
	if err != nil {
		t.Fatalf("failed to clean up after previous tests: %s", err)
	}

	// recreate 'data' path
	if err = os.Mkdir(dataPath, 0o755); err != nil {
		t.Fatalf("failed to clean up after previous tests: %s", err)
	}

	// copy 'adapter.json' from 'defaults' to 'data'
	fin, err := os.Open(path.Join(defaultsPath, "adapter.json"))
	if err != nil {
		t.Fatalf("failed to clean up after previous tests: %s", err)
	}
	defer fin.Close()

	fout, err := os.Create(path.Join(dataPath, "adapter.json"))
	if err != nil {
		t.Fatalf("failed to clean up after previous tests: %s", err)
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)
	if err != nil {
		t.Fatalf("failed to clean up after previous tests: %s", err)
	}

	err = os.Remove(path.Join(workDir, "data.db"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed to clean up after previous tests: %s", err)
	}
}

func configSetup(t *testing.T, configSet, mqttAddr string) *config.Config {
	t.Helper()

	cfgDir := path.Join("./../../testdata/testing/", configSet)
	cfg := config.New(cfgDir)
	storage := cliffConfig.NewStorage(cfg, cfgDir)

	service := config.NewService(storage)

	if err := service.Load(); err != nil {
		t.Fatalf("failed to load configuration: %s", err)
	}

	service.Model().MQTTServerURI = mqttAddr
	services.configService = service

	return service.Model()
}

func loggerSetup(t *testing.T) {
	t.Helper()

	cfg := getConfigService().Model()
	bootstrap.InitializeLogger(cfg.LogFile, cfg.LogLevel, cfg.LogFormat)
}

func waitForRunning() suite.Callback {
	return func(t *testing.T) {
		t.Helper()

		getLifecycle().WaitFor("test_suite", lifecycle.StateTypeAppState, lifecycle.AppStateRunning)
	}
}

type testContainer struct {
	t *testing.T

	signalRAddress string
	signalRSetupFn func(s *test.SignalRServer)

	signalRServer *test.SignalRServer
}

func newTestContainer(t *testing.T) *testContainer {
	t.Helper()

	return &testContainer{
		t: t,
	}
}

func extendMeterReportExpectation(expectation map[string]float64) *suite.Expectation {
	return suite.ExpectFloatMap(evtDeviceMeterElecTopic, "evt.meter_ext.report", "meter_elec", expectation)
}

func (c *testContainer) SetUp() {
	c.t.Helper()

	if c.signalRAddress == "" {
		c.signalRAddress = test.DefaultSignalRAddr
	}

	c.signalRServer = test.NewSignalRServer(c.t, c.signalRAddress)

	if c.signalRSetupFn != nil {
		c.signalRSetupFn(c.signalRServer)
	}

	c.signalRServer.Start()
}

func (c *testContainer) TearDown() suite.Callback {
	return func(t *testing.T) {
		t.Helper()

		c.signalRServer.Close()
		c.signalRAddress = ""
	}
}
