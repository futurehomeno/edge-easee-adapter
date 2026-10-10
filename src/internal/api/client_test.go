package api_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/edge-easee-adapter/internal/api"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
	mockedapi "github.com/futurehomeno/edge-easee-adapter/internal/test/mocks/api"
)

// Every Client method fetches a token and hands it to the HTTP client; a token failure is
// wrapped and nothing is sent.
func TestAPIClient_DelegatesWithTheToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		expect func(h *mockedapi.HTTPClient)
		call   func(c api.Client) error
	}{
		{
			name:   "UpdateMaxCurrent",
			expect: func(h *mockedapi.HTTPClient) { h.On("UpdateMaxCurrent", "token", "id", 16.0).Return(nil) },
			call:   func(c api.Client) error { return c.UpdateMaxCurrent("id", 16) },
		},
		{
			name:   "UpdateDynamicCurrent",
			expect: func(h *mockedapi.HTTPClient) { h.On("UpdateDynamicCurrent", "token", "id", 10.0).Return(nil) },
			call:   func(c api.Client) error { return c.UpdateDynamicCurrent("id", 10) },
		},
		{
			name:   "StopCharging",
			expect: func(h *mockedapi.HTTPClient) { h.On("StopCharging", "token", "id").Return(nil) },
			call:   func(c api.Client) error { return c.StopCharging("id") },
		},
		{
			name:   "SetCableAlwaysLocked",
			expect: func(h *mockedapi.HTTPClient) { h.On("SetCableAlwaysLocked", "token", "id", true).Return(nil) },
			call:   func(c api.Client) error { return c.SetCableAlwaysLocked("id", true) },
		},
		{
			name:   "SetPhaseMode",
			expect: func(h *mockedapi.HTTPClient) { h.On("SetPhaseMode", "token", "id", 2).Return(nil) },
			call:   func(c api.Client) error { return c.SetPhaseMode("id", 2) },
		},
		{
			name: "ChargerConfig",
			expect: func(h *mockedapi.HTTPClient) {
				h.On("ChargerConfig", "token", "id").Return(&model.ChargerConfig{}, nil)
			},
			call: func(c api.Client) error { _, err := c.ChargerConfig("id"); return err },
		},
		{
			name: "ChargerSiteInfo",
			expect: func(h *mockedapi.HTTPClient) {
				h.On("ChargerSiteInfo", "token", "id").Return(&model.ChargerSiteInfo{}, nil)
			},
			call: func(c api.Client) error { _, err := c.ChargerSiteInfo("id"); return err },
		},
		{
			name:   "Chargers",
			expect: func(h *mockedapi.HTTPClient) { h.On("Chargers", "token").Return([]model.Charger{}, nil) },
			call:   func(c api.Client) error { _, err := c.Chargers(); return err },
		},
		{
			name: "ChargerDetails",
			expect: func(h *mockedapi.HTTPClient) {
				h.On("ChargerDetails", "token", "id").Return(model.ChargerDetails{}, nil)
			},
			call: func(c api.Client) error { _, err := c.ChargerDetails("id"); return err },
		},
		{
			name:   "Ping",
			expect: func(h *mockedapi.HTTPClient) { h.On("Ping", "token").Return(nil) },
			call:   func(c api.Client) error { return c.Ping() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			httpClient := mockedapi.NewHTTPClient(t)
			tt.expect(httpClient)

			auth := mockedapi.NewAuthenticator(t)
			auth.On("AccessToken").Return("token", nil).Once()

			assert.NoError(t, tt.call(api.NewAPIClient(httpClient, auth)))
		})

		t.Run(tt.name+" without a token", func(t *testing.T) {
			t.Parallel()

			httpClient := mockedapi.NewHTTPClient(t)

			auth := mockedapi.NewAuthenticator(t)
			auth.On("AccessToken").Return("", api.ErrNotLoggedIn).Once()

			err := tt.call(api.NewAPIClient(httpClient, auth))
			require.ErrorIs(t, err, api.ErrNotLoggedIn)
			httpClient.AssertNotCalled(t, tt.name, mock.Anything, mock.Anything, mock.Anything)
		})
	}
}
