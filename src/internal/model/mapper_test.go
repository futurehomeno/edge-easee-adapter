package model_test

import (
	"testing"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/types"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

func TestSettablePhaseModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		gridType types.GridType
		phases   int
		want     []types.PhaseMode
	}{
		{
			name:     "TN 3-phase offers both single and three phase, whatever mode the charger sits in",
			gridType: types.GridTypeTN,
			phases:   3,
			want:     []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3, types.PhaseModeNL1L2L3},
		},
		{
			name:     "TN 1-phase has nothing to switch between",
			gridType: types.GridTypeTN,
			phases:   1,
			want:     []types.PhaseMode{types.PhaseModeNL1},
		},
		{
			name:     "TN 2-phase offers both wired legs and the two-phase mode",
			gridType: types.GridTypeTN,
			phases:   2,
			want:     []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL1L2},
		},
		{
			name:     "IT 3-phase",
			gridType: types.GridTypeIT,
			phases:   3,
			want:     []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1, types.PhaseModeL1L2L3},
		},
		{
			name:     "TT 3-phase",
			gridType: types.GridTypeTT,
			phases:   3,
			want:     []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1, types.PhaseModeL1L2L3},
		},
		{
			name:     "unknown grid type",
			gridType: types.GridTypeUnknown,
			phases:   3,
		},
		{
			name:     "unsupported phase count",
			gridType: types.GridTypeTN,
			phases:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, model.SettablePhaseModes(tt.gridType, tt.phases))
		})
	}
}

// Grid type 30 drives L1 and L3: the nominal row's NL2 and NL1L2 are out of its reach and FIMP has
// no NL1L3, so it advertises nothing. The other TN 2-phase types keep the row.
func TestSettablePhaseModesFor(t *testing.T) {
	t.Parallel()

	nominal := []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL1L2}

	tests := []struct {
		name     string
		detected model.GridType
		want     []types.PhaseMode
	}{
		{"pins 2, 3, 5", model.GridTypeWarningTN2PhasePin235, nil},
		{"nominal TN 2-phase", model.GridTypeTN2PhasePin23, nominal},
		{"warning on pins 2, 3, 4", model.GridTypeWarningTN2PhasePIN234, nominal},
		{"ground fault", model.GridTypeWarningTN2PhaseGNDFault, nominal},
		{"raw type unknown, as in a state stored by an older adapter", model.GridTypeNotYetDetected, nominal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, model.SettablePhaseModesFor(tt.detected, types.GridTypeTN, 2))
		})
	}
}

func TestToEaseePhaseMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		gridType types.GridType
		phases   int
		mode     types.PhaseMode
		lock3    bool
		want     int
		wantErr  bool
	}{
		{
			name:     "TN 3-phase, NL1 locks to a single phase",
			gridType: types.GridTypeTN,
			phases:   3,
			mode:     types.PhaseModeNL1,
			want:     1,
		},
		{
			name:     "TN 3-phase, NL3 locks to a single phase too - Easee cannot pick which one",
			gridType: types.GridTypeTN,
			phases:   3,
			mode:     types.PhaseModeNL3,
			want:     1,
		},
		{
			name:     "TN 3-phase, NL1L2L3 maps to auto rather than a hard three-phase lock",
			gridType: types.GridTypeTN,
			phases:   3,
			mode:     types.PhaseModeNL1L2L3,
			want:     2,
		},
		{
			name:     "IT 3-phase, L2L3 locks to a single phase",
			gridType: types.GridTypeIT,
			phases:   3,
			mode:     types.PhaseModeL2L3,
			want:     1,
		},
		{
			name:     "IT 3-phase, L1L2L3 maps to auto",
			gridType: types.GridTypeIT,
			phases:   3,
			mode:     types.PhaseModeL1L2L3,
			want:     2,
		},
		{
			name:     "TN 1-phase, NL1 is the only reachable mode",
			gridType: types.GridTypeTN,
			phases:   1,
			mode:     types.PhaseModeNL1,
			want:     1,
		},
		{
			name:     "TN 2-phase, the reported NL2 locks to a single phase",
			gridType: types.GridTypeTN,
			phases:   2,
			mode:     types.PhaseModeNL2,
			want:     1,
		},
		{
			name:     "TN 2-phase, NL1L2 maps to auto",
			gridType: types.GridTypeTN,
			phases:   2,
			mode:     types.PhaseModeNL1L2,
			want:     2,
		},
		{
			name:     "lock: TN 3-phase, NL1L2L3 locks to three phases",
			gridType: types.GridTypeTN,
			phases:   3,
			mode:     types.PhaseModeNL1L2L3,
			lock3:    true,
			want:     3,
		},
		{
			name:     "lock: TN 2-phase, NL1L2 locks to three phases",
			gridType: types.GridTypeTN,
			phases:   2,
			mode:     types.PhaseModeNL1L2,
			lock3:    true,
			want:     3,
		},
		{
			name:     "lock: IT 3-phase, L1L2L3 locks to three phases",
			gridType: types.GridTypeIT,
			phases:   3,
			mode:     types.PhaseModeL1L2L3,
			lock3:    true,
			want:     3,
		},
		{
			name:     "lock: TT 3-phase, L1L2L3 locks to three phases",
			gridType: types.GridTypeTT,
			phases:   3,
			mode:     types.PhaseModeL1L2L3,
			lock3:    true,
			want:     3,
		},
		{
			name:     "lock: TN 3-phase, NL2 stays single phase",
			gridType: types.GridTypeTN,
			phases:   3,
			mode:     types.PhaseModeNL2,
			lock3:    true,
			want:     1,
		},
		{
			name:     "lock: IT 3-phase, L3L1 stays single phase",
			gridType: types.GridTypeIT,
			phases:   3,
			mode:     types.PhaseModeL3L1,
			lock3:    true,
			want:     1,
		},
		{
			name:     "lock: TN 1-phase, NL1 stays single phase",
			gridType: types.GridTypeTN,
			phases:   1,
			mode:     types.PhaseModeNL1,
			lock3:    true,
			want:     1,
		},
		{
			name:     "TN 1-phase cannot go three phase",
			gridType: types.GridTypeTN,
			phases:   1,
			mode:     types.PhaseModeNL1L2L3,
			wantErr:  true,
		},
		{
			name:     "mode from the wrong grid type",
			gridType: types.GridTypeTN,
			phases:   3,
			mode:     types.PhaseModeL1L2,
			wantErr:  true,
		},
		{
			name:     "unknown grid type",
			gridType: types.GridTypeUnknown,
			phases:   3,
			mode:     types.PhaseModeNL1,
			wantErr:  true,
		},
		{
			name:     "empty mode",
			gridType: types.GridTypeTN,
			phases:   3,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := model.ToEaseePhaseMode(tt.gridType, tt.phases, tt.mode, tt.lock3)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionInProgress(t *testing.T) {
	t.Parallel()

	for state, want := range map[chargepoint.State]bool{
		chargepoint.StateReadyToCharge:   true,
		chargepoint.StateCharging:        true,
		chargepoint.StateSuspendedByEV:   true,
		chargepoint.StateDisconnected:    false,
		chargepoint.StateRequesting:      false,
		chargepoint.StateFinished:        false,
		chargepoint.StateError:           false,
		chargepoint.StateUnknown:         false,
		chargepoint.StateSuspendedByEVSE: false,
	} {
		assert.Equal(t, want, model.SessionInProgress(state), state)
	}
}

func TestPhaseModeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		phaseMode int
		want      string
	}{
		{"single-phase lock", 1, "single phase"},
		{"auto", 2, "auto"},
		{"three-phase lock", 3, "three phase"},
		{"below the range", 0, "unknown"},
		{"above the range", 4, "unknown"},
		{"negative", -1, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, model.PhaseModeName(tt.phaseMode))
		})
	}
}

func TestOutputPhaseType_ToFimpState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		phase    model.OutputPhaseType
		gridType types.GridType
		want     types.PhaseMode
	}{
		{"TN suffix on an IT grid names the IT legs", model.P1T2T3TN, types.GridTypeIT, types.PhaseModeL1L2},
		{"TN suffix on a TT grid names the IT legs", model.P1T2T4TN, types.GridTypeTT, types.PhaseModeL3L1},
		{"two-phase TN suffix on an IT grid", model.P2T2T3T4TN, types.GridTypeIT, types.PhaseModeL1L2L3},
		{"IT suffix on a TN grid names the TN leg", model.P1T2T3IT, types.GridTypeTN, types.PhaseModeNL1},
		{"matching suffix is kept", model.P1T2T5TN, types.GridTypeTN, types.PhaseModeNL3},
		{"no twin keeps its suffix", model.P1T2T5TN, types.GridTypeIT, types.PhaseModeNL3},
		{"unknown grid keeps the suffix", model.P1T2T3TN, "", types.PhaseModeNL1},
		{"unassigned stays empty", model.Unassigned, types.GridTypeIT, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.phase.ToFimpState(tt.gridType))
		})
	}
}

func TestSettablePhaseModes_ReturnsACopy(t *testing.T) {
	t.Parallel()

	modes := model.SettablePhaseModes(types.GridTypeIT, 3)
	want := append([]types.PhaseMode(nil), modes...)
	modes[0] = ""

	assert.Equal(t, want, model.SettablePhaseModes(types.GridTypeIT, 3))
}
