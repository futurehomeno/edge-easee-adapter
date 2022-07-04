package easee_test

import (
	"testing"
	"time"

	"github.com/michalkurzeja/go-clock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/config"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee"
	"github.com/futurehomeno/edge-easee-adapter/internal/easee/mocks"
	"github.com/futurehomeno/edge-easee-adapter/internal/test/fakes"
)

const (
	testChargerID  = "123456"
	testMaxCurrent = 32.0
)

func TestController_StartChargepointCharging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mode       string
		mockClient func(c *mocks.Client)
		wantErr    bool
	}{
		{
			name: "easee client should start charging session for a particular charger with default charging mode",
			mode: "",
			mockClient: func(c *mocks.Client) {
				c.On("StartCharging", testChargerID, 32.0).Return(nil)
			},
		},
		{
			name: "start charging session with normal mode",
			mode: "normal",
			mockClient: func(c *mocks.Client) {
				c.On("StartCharging", testChargerID, 32.0).Return(nil)
			},
		},
		{
			name: "start charging session with slow mode",
			mode: "Slow",
			mockClient: func(c *mocks.Client) {
				c.On("StartCharging", testChargerID, 10.0).Return(nil)
			},
		},
		{
			name: "ignore unknown charging mode when starting charging session",
			mode: "dummy",
			mockClient: func(c *mocks.Client) {
				c.On("StartCharging", testChargerID, 32.0).Return(nil)
			},
		},
		{
			name: "error when easee client returns an error on starting charging",
			mode: "",
			mockClient: func(c *mocks.Client) {
				c.On("StartCharging", testChargerID, 32.0).Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mocks.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			cfg := &config.Config{
				PollingInterval:              "30s",
				SlowChargingCurrentInAmperes: 10,
				EaseeBackoff:                 "0s",
			}
			storage := fakes.NewConfigStorage(cfg, config.Factory)
			cfgService := config.NewService(storage)

			c := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

			err := c.StartChargepointCharging(tt.mode)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestController_StopChargepointCharging(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockClient func(c *mocks.Client)
		wantErr    bool
	}{
		{
			name: "easee client should stop charging session for a particular charger",
			mockClient: func(c *mocks.Client) {
				c.On("StopCharging", testChargerID).Return(nil)
			},
		},
		{
			name: "error when easee client returns an error",
			mockClient: func(c *mocks.Client) {
				c.On("StopCharging", testChargerID).Return(errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mocks.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			cfg := &config.Config{
				PollingInterval: "30s",
				EaseeBackoff:    "0s",
			}
			storage := fakes.NewConfigStorage(cfg, config.Factory)
			cfgService := config.NewService(storage)

			c := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

			err := c.StopChargepointCharging()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestController_ChargepointCableLockReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockClient func(c *mocks.Client)
		want       bool
		wantErr    bool
	}{
		{
			name: "controller should send lock report successfully",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{CableLocked: true}, nil)
			},
			want: true,
		},
		{
			name: "easee client error",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mocks.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			cfg := &config.Config{
				PollingInterval: "30s",
				EaseeBackoff:    "0s",
			}
			storage := fakes.NewConfigStorage(cfg, config.Factory)
			cfgService := config.NewService(storage)

			c := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

			got, err := c.ChargepointCableLockReport()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestController_ChargepointStateReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockClient func(c *mocks.Client)
		want       string
		wantErr    bool
	}{
		{
			name: "reported state: unavailable",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 0}, nil)
			},
			want: "unavailable",
		},
		{
			name: "reported state: disconnected",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 1}, nil)
			},
			want: "disconnected",
		},
		{
			name: "reported state: ready_to_charge",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 2}, nil)
			},
			want: "ready_to_charge",
		},
		{
			name: "reported state: charging",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 3}, nil)
			},
			want: "charging",
		},
		{
			name: "reported state: finished",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 4}, nil)
			},
			want: "finished",
		},
		{
			name: "reported state: error",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 5}, nil)
			},
			want: "error",
		},
		{
			name: "reported state: requesting",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 6}, nil)
			},
			want: "requesting",
		},
		{
			name: "unknown state",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{ChargerOpMode: 999}, nil)
			},
			want: "unknown",
		},
		{
			name: "easee client error",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(&easee.ChargerState{}, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mocks.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			storage := fakes.NewConfigStorage(&config.Config{PollingInterval: "30s"}, config.Factory)
			cfgService := config.NewService(storage)

			c := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

			got, err := c.ChargepointStateReport()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestController_ElectricityMeterReport_kWh(t *testing.T) {
	t.Cleanup(clock.Restore)

	clientMock := mocks.NewClient(t)
	storage := fakes.NewConfigStorage(&config.Config{PollingInterval: "0s"}, config.Factory)
	cfgService := config.NewService(storage)
	controller := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

	checkFirstEnergyReportNoPreviousData(t, cfgService, clientMock, controller)
	checkReportNoDataAvailableForProvidedHour(t, clientMock, controller, cfgService)
	checkReportNewestDataAvailable(t, clientMock, controller, cfgService)
	checkReportAfterHoursOfClientErrors(t, clientMock, controller, cfgService)
	checkReportValueAlreadyReported(t, controller, cfgService)
}

func checkReportValueAlreadyReported(t *testing.T, controller easee.Controller, cfgService *config.Service) {
	got, err := controller.ElectricityMeterReport("kWh")
	assert.NoError(t, err)
	assert.Equal(t, float64(0), got, "value already sent, should report zero")
	assert.Equal(t, config.EnergyReport{
		Value:     0.65528804063797,
		Timestamp: parse(t, "2022-09-10T13:00:00Z"),
	}, cfgService.GetLastEnergyReport())
}

func checkReportAfterHoursOfClientErrors(t *testing.T, clientMock *mocks.Client, controller easee.Controller, cfgService *config.Service) {
	now := parse(t, "2022-09-10T10:15:12Z")
	clock.Mock(now)

	clientMock.
		On("EnergyPerHour", testChargerID, parse(t, "2022-09-10T09:00:01Z"), now.Truncate(time.Hour)).
		Return(nil, errors.New("oops")).
		Once()

	got, err := controller.ElectricityMeterReport("kWh")
	assert.Error(t, err)

	// we assume the controller was not able to send the report since the last call
	now = parse(t, "2022-09-10T13:13:12Z")
	clock.Mock(now)

	clientMock.
		On("EnergyPerHour", testChargerID, parse(t, "2022-09-10T09:00:01Z"), now.Truncate(time.Hour)). // last Value is from 08:00:00
		Return([]easee.Measurement{
			{
				Value:     0,
				Timestamp: parse(t, "2022-09-10T10:00:00Z"),
			},
			{
				Value:     0,
				Timestamp: parse(t, "2022-09-10T11:00:00Z"),
			},
			{
				Value:     0.5093480348587036,
				Timestamp: parse(t, "2022-09-10T12:00:00Z"),
			},
			{
				Value:     0.14594000577926636,
				Timestamp: parse(t, "2022-09-10T13:00:00Z"),
			},
		}, nil).
		Once()

	got, err = controller.ElectricityMeterReport("kWh")
	assert.NoError(t, err)
	assert.Equal(t, 0.65528804063797, got)
	assert.Equal(t, config.EnergyReport{
		Value:     0.65528804063797,
		Timestamp: parse(t, "2022-09-10T13:00:00Z"),
	}, cfgService.GetLastEnergyReport())
}

func checkReportNewestDataAvailable(t *testing.T, clientMock *mocks.Client, controller easee.Controller, cfgService *config.Service) {
	now := parse(t, "2022-09-10T09:11:12Z")
	clock.Mock(now)

	clientMock.
		On("EnergyPerHour", testChargerID, parse(t, "2022-09-10T08:00:01Z"), now.Truncate(time.Hour)).
		Return([]easee.Measurement{
			{
				Value:     1.7233669328689575,
				Timestamp: parse(t, "2022-09-10T09:00:00Z"),
			},
		}, nil).
		Once()

	got, err := controller.ElectricityMeterReport("kWh")
	assert.NoError(t, err)
	assert.Equal(t, 1.7233669328689575, got)
	assert.Equal(t, config.EnergyReport{
		Value:     1.7233669328689575,
		Timestamp: parse(t, "2022-09-10T09:00:00Z"),
	}, cfgService.GetLastEnergyReport())
}

func checkReportNoDataAvailableForProvidedHour(t *testing.T, clientMock *mocks.Client, controller easee.Controller, cfgService *config.Service) {
	t.Helper()

	now := parse(t, "2022-09-10T09:00:12Z")
	clock.Mock(now)

	clientMock.
		On("EnergyPerHour", testChargerID, parse(t, "2022-09-10T08:00:01Z"), now.Truncate(time.Hour)).
		Return([]easee.Measurement{}, nil).
		Once()

	got, err := controller.ElectricityMeterReport("kWh")
	assert.NoError(t, err)
	assert.Equal(t, float64(0), got)
	assert.Equal(t, config.EnergyReport{
		Value:     1.6433669328689575,
		Timestamp: parse(t, "2022-09-10T08:00:00Z"),
	}, cfgService.GetLastEnergyReport())
}

func checkFirstEnergyReportNoPreviousData(t *testing.T, cfgService *config.Service, clientMock *mocks.Client, controller easee.Controller) {
	t.Helper()

	now := parse(t, "2022-09-10T08:15:12Z")
	clock.Mock(now)

	assert.Equal(t, config.EnergyReport{}, cfgService.GetLastEnergyReport())

	clientMock.
		On("EnergyPerHour", testChargerID, now.Truncate(time.Hour).Add(-2*time.Hour), now.Truncate(time.Hour)).
		Return([]easee.Measurement{
			{
				Value:     0,
				Timestamp: parse(t, "2022-09-10T06:00:00Z"),
			},
			{
				Value:     1.100541353225708,
				Timestamp: parse(t, "2022-09-10T07:00:00Z"),
			},
			{
				Value:     1.6433669328689575,
				Timestamp: parse(t, "2022-09-10T08:00:00Z"),
			},
		}, nil).
		Once()

	got, err := controller.ElectricityMeterReport("kWh")
	assert.NoError(t, err)
	assert.Equal(t, 1.6433669328689575, got)
	assert.Equal(t, config.EnergyReport{
		Value:     1.6433669328689575,
		Timestamp: parse(t, "2022-09-10T08:00:00Z"),
	}, cfgService.GetLastEnergyReport())
}

func TestController_ElectricityMeterReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		unit       string
		mockClient func(c *mocks.Client)
		want       float64
		wantErr    bool
	}{
		{
			name: "correct report for W",
			unit: "W",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(defaultChargerState(t), nil)
			},
			want: 2000,
		},
		{
			name: "correct report for V",
			unit: "V",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(defaultChargerState(t), nil)
			},
			want: 200,
		},
		{
			name:    "error on unsupported unit",
			unit:    "dummy",
			wantErr: true,
		},
		{
			name: "easee client error - W",
			unit: "W",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
		{
			name: "easee client error - V",
			unit: "V",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mocks.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			storage := fakes.NewConfigStorage(&config.Config{PollingInterval: "0s"}, config.Factory)
			cfgService := config.NewService(storage)

			c := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

			got, err := c.ElectricityMeterReport(tt.unit)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestController_ChargepointCurrentSessionReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mockClient func(c *mocks.Client)
		want       float64
		wantErr    bool
	}{
		{
			name: "charger should return data if the state == charging",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(chargerStateWithMode(t, 3), nil)
			},
			want: 234,
		},
		{
			name: "charger should return data if the state == finished",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(chargerStateWithMode(t, 4), nil)
			},
			want: 234,
		},
		{
			name: "charger should not return data if the state == unavailable",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(chargerStateWithMode(t, 0), nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data if the state == disconnected",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(chargerStateWithMode(t, 1), nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data if the state == error",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(chargerStateWithMode(t, 5), nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data if the state == requesting",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(chargerStateWithMode(t, 6), nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data on unknown state",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(chargerStateWithMode(t, 999), nil)
			},
			want: 0,
		},
		{
			name: "easee client error",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mocks.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			storage := fakes.NewConfigStorage(&config.Config{PollingInterval: "30s"}, config.Factory)
			cfgService := config.NewService(storage)

			c := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

			got, err := c.ChargepointCurrentSessionReport()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestController_SetChargepointCableLock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		chargerLocked bool
		mockClient    func(c *mocks.Client)
		wantErr       bool
	}{
		{
			name:          "locked device successfully",
			chargerLocked: true,
			mockClient: func(c *mocks.Client) {
				c.On("SetCableLock", testChargerID, true).Return(nil)
			},
		},
		{
			name:          "unlocked device successfully",
			chargerLocked: false,
			mockClient: func(c *mocks.Client) {
				c.On("SetCableLock", testChargerID, false).Return(nil)
			},
		},
		{
			name:          "easee client error",
			chargerLocked: true,
			mockClient: func(c *mocks.Client) {
				c.On("SetCableLock", testChargerID, true).Return(errors.New("oops"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			clientMock := mocks.NewClient(t)
			if tt.mockClient != nil {
				tt.mockClient(clientMock)
			}

			storage := fakes.NewConfigStorage(&config.Config{PollingInterval: "0s", EaseeBackoff: "0s"}, config.Factory)
			cfgService := config.NewService(storage)

			c := easee.NewController(clientMock, cfgService, testChargerID, testMaxCurrent)

			err := c.SetChargepointCableLock(tt.chargerLocked)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
		})
	}
}

func defaultChargerState(t *testing.T) *easee.ChargerState {
	t.Helper()

	return &easee.ChargerState{
		TotalPower:     2,
		LifetimeEnergy: 1234,
		SessionEnergy:  234,
		Voltage:        200,
	}
}

func chargerStateWithMode(t *testing.T, mode int) *easee.ChargerState {
	t.Helper()

	s := defaultChargerState(t)
	s.ChargerOpMode = easee.ChargerMode(mode)

	return s
}

func parse(t *testing.T, tm string) time.Time {
	parsed, err := time.Parse(time.RFC3339, tm)
	require.NoError(t, err)

	return parsed
}
