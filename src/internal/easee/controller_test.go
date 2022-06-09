package easee_test

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

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
			name: "correct report for kWh",
			unit: "kWh",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(defaultChargerState(t), nil)
			},
			want: 1234,
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
			name: "error on unsupported unit",
			unit: "dummy",
			mockClient: func(c *mocks.Client) {
				c.On("ChargerState", testChargerID).Return(defaultChargerState(t), nil)
			},
			wantErr: true,
		},
		{
			name: "easee client error",
			unit: "W",
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

			storage := fakes.NewConfigStorage(&config.Config{PollingInterval: "30s"}, config.Factory)
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
