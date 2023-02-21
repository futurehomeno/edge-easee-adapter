package cmd

import (
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/bootstrap"
	cliffConfig "github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/test/suite"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/test"
)

func TestEaseeEdgeApp(t *testing.T) { //nolint:paralleltest
	testContainer := newTestContainer(t)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Adapter is capable of reacting to incoming observations",
				Setup: serviceSetup(
					testContainer,
					"configured",
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []easee.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Integer,
								ID:        easee.ChargerOPState,
								Value:     strconv.Itoa(int(easee.ReadyToCharge)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.SessionEnergy,
								Value:     "0",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Boolean,
								ID:        easee.CableLocked,
								Value:     "false",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.TotalPower,
								Value:     "0",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.LifetimeEnergy,
								Value:     "12.34",
							},
						})
						s.MockObservations(300*time.Millisecond, []easee.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Integer,
								ID:        easee.ChargerOPState,
								Value:     strconv.Itoa(int(easee.Charging)),
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.SessionEnergy,
								Value:     "1.23",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Boolean,
								ID:        easee.CableLocked,
								Value:     "true",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.TotalPower,
								Value:     "1",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.LifetimeEnergy,
								Value:     "13.45",
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Expectations: []*suite.Expectation{
							// Initial batch
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.state.report", "chargepoint", "ready_to_charge"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.current_session.report", "chargepoint", 0),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.cable_lock.report", "chargepoint", false),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:meter_elec/ad:1", "evt.meter.report", "meter_elec", 0).ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:meter_elec/ad:1", "evt.meter.report", "meter_elec", 12.34).ExpectProperty("unit", "kWh"),

							// Update
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.state.report", "chargepoint", "charging"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.current_session.report", "chargepoint", 1.23),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.cable_lock.report", "chargepoint", true),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:meter_elec/ad:1", "evt.meter.report", "meter_elec", 1000).ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:meter_elec/ad:1", "evt.meter.report", "meter_elec", 13.45).ExpectProperty("unit", "kWh"),
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
					signalRSetup(test.DefaultSignalRAddr, func(s *test.SignalRServer) {
						s.MockObservations(0, []easee.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.TotalPower,
								Value:     "0",
							},
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Integer,
								ID:        easee.ChargerOPState,
								Value:     strconv.Itoa(int(easee.ReadyToCharge)),
							},
						})
						s.MockObservations(300*time.Millisecond, []easee.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Double,
								ID:        easee.TotalPower,
								Value:     "1.23",
							},
						})
						s.MockObservations(300*time.Millisecond, []easee.Observation{
							{
								ChargerID: test.ChargerID,
								DataType:  easee.Integer,
								ID:        easee.ChargerOPState,
								Value:     strconv.Itoa(int(easee.ReadyToCharge)),
							},
						})
					})),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:meter_elec/ad:1", "evt.meter.report", "meter_elec", 0).ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:meter_elec/ad:1", "evt.meter.report", "meter_elec", 1230).ExpectProperty("unit", "W"),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.state.report", "chargepoint", "ready_to_charge").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "evt.state.report", "chargepoint", "charging").ExactlyOnce(),
						},
					},
				},
			},
			{
				Name:     "Adapter should not report data if signalR connection is lost/not established",
				Setup:    serviceSetup(testContainer, "configured", signalRSetup("localhost:1111", nil)),
				TearDown: []suite.Callback{tearDown("configured"), testContainer.TearDown()},
				Nodes: []*suite.Node{
					{
						InitCallbacks: []suite.Callback{waitForRunning()},
						Command:       suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "cmd.state.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:easee/ad:1/sv:chargepoint/ad:1", "chargepoint"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func serviceSetup(tc *testContainer, configSet string, opts ...func(tc *testContainer)) suite.ServiceSetup {
	return func(t *testing.T) (service suite.Service, mocks []suite.Mock) {
		t.Helper()

		tearDown(configSet)(t)

		configSetup(t, configSet)
		loggerSetup(t)

		for _, o := range opts {
			o(tc)
		}

		tc.SetUp()

		app, err := buildEdgeApp()
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

		err := os.RemoveAll(path.Join("../../testdata/testing/", configSet, "/data/"))
		if err != nil {
			t.Fatalf("failed to clean up after previous tests: %s", err)
		}
	}
}

func configSetup(t *testing.T, configSet string) {
	t.Helper()

	cfgDir := path.Join("./../../testdata/testing/", configSet)
	cfg := config.New(cfgDir)

	services.configStorage = cliffConfig.NewStorage(cfg, cfgDir)
}

func loggerSetup(t *testing.T) {
	t.Helper()

	cfg := getConfigService().Model().(*config.Config) //nolint:forcetypeassert
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
