package easee_test

import (
	"testing"
	"time"

	"github.com/michalkurzeja/go-clock"
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

var (
	now     = time.Date(2022, time.September, 10, 8, 00, 12, 00, time.UTC) //nolint:gofumpt
	yearAgo = now.Add(-365 * 24 * time.Hour)
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

func TestController_ChargepointCableLockReport(t *testing.T) { //nolint:paralleltest
	clock.Mock(now)
	t.Cleanup(clock.Restore)

	tests := []struct {
		name       string
		mockClient func(c *mocks.Client)
		want       bool
		wantErr    bool
	}{
		{
			name: "controller should send lock report successfully",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(103), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     true,
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     false,
							Timestamp: now.Add(-time.Hour),
						},
						{
							Value:     true,
							Timestamp: now,
						},
					}, nil)
			},
			want: true,
		},
		{
			name: "easee client error",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(103), yearAgo, now).
					Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
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

func TestController_ChargepointStateReport(t *testing.T) { //nolint:paralleltest
	clock.Mock(now)
	t.Cleanup(clock.Restore)

	tests := []struct {
		name       string
		mockClient func(c *mocks.Client)
		want       string
		wantErr    bool
	}{
		{
			name: "reported state: unavailable",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(0),
							Timestamp: now,
						},
					}, nil)
			},
			want: "unavailable",
		},
		{
			name: "reported state: disconnected",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(0),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(1),
							Timestamp: now,
						},
					}, nil)
			},
			want: "disconnected",
		},
		{
			name: "reported state: ready_to_charge",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(2),
							Timestamp: now,
						},
					}, nil)
			},
			want: "ready_to_charge",
		},
		{
			name: "reported state: charging",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(3),
							Timestamp: now,
						},
					}, nil)
			},
			want: "charging",
		},
		{
			name: "reported state: finished",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(4),
							Timestamp: now,
						},
					}, nil)
			},
			want: "finished",
		},
		{
			name: "reported state: error",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(5),
							Timestamp: now,
						},
					}, nil)
			},
			want: "error",
		},
		{
			name: "reported state: requesting",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(6),
							Timestamp: now,
						},
					}, nil)
			},
			want: "requesting",
		},
		{
			name: "unknown state",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(999),
							Timestamp: now,
						},
					}, nil)
			},
			want: "unknown",
		},
		{
			name: "easee client error",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
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

func TestController_ElectricityMeterReport(t *testing.T) { //nolint:paralleltest
	clock.Mock(now)
	t.Cleanup(clock.Restore)

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
				c.On("Observations", testChargerID, easee.ObservationID(120), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(2),
							Timestamp: now,
						},
					}, nil)
			},
			want: 2000,
		},
		{
			name: "correct report for kWh",
			unit: "kWh",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(124), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1111),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(1234),
							Timestamp: now,
						},
					}, nil)
			},
			want: 1234,
		},
		{
			name:    "error on unsupported unit",
			unit:    "dummy",
			wantErr: true,
		},
		{
			name: "easee client error",
			unit: "W",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(120), yearAgo, now).
					Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
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

func TestController_ChargepointCurrentSessionReport(t *testing.T) { //nolint:paralleltest
	clock.Mock(now)
	t.Cleanup(clock.Restore)

	tests := []struct {
		name       string
		mockClient func(c *mocks.Client)
		want       float64
		wantErr    bool
	}{
		{
			name: "charger should return data if the state == charging",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(3),
							Timestamp: now,
						},
					}, nil)
				c.On("Observations", testChargerID, easee.ObservationID(121), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(123),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(234),
							Timestamp: now,
						},
					}, nil)
			},
			want: 234,
		},
		{
			name: "charger should return data if the state == finished",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(4),
							Timestamp: now,
						},
					}, nil)
				c.On("Observations", testChargerID, easee.ObservationID(121), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(123),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(234),
							Timestamp: now,
						},
					}, nil)
			},
			want: 234,
		},
		{
			name: "charger should not return data if the state == unavailable",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(0),
							Timestamp: now,
						},
					}, nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data if the state == disconnected",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(0),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(1),
							Timestamp: now,
						},
					}, nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data if the state == error",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(5),
							Timestamp: now,
						},
					}, nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data if the state == requesting",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(6),
							Timestamp: now,
						},
					}, nil)
			},
			want: 0,
		},
		{
			name: "charger should not return data on unknown state",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return([]easee.Observation{
						{
							Value:     float64(1),
							Timestamp: now.Add(-2 * time.Hour),
						},
						{
							Value:     float64(999),
							Timestamp: now,
						},
					}, nil)
			},
			want: 0,
		},
		{
			name: "easee client error",
			mockClient: func(c *mocks.Client) {
				c.On("Observations", testChargerID, easee.ObservationID(109), yearAgo, now).
					Return(nil, errors.New("test error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) {
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
