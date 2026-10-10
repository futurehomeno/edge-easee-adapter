package model_test

import (
	"testing"

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
			phases:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, model.SettablePhaseModes(tt.gridType, tt.phases))
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

			got, err := model.ToEaseePhaseMode(tt.gridType, tt.phases, tt.mode)
			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAdvertisedPhaseModes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		gridType    types.GridType
		phases      int
		outputPhase types.PhaseMode
		want        []types.PhaseMode
	}{
		{
			name:     "TN 3-phase without a reading keeps the first phase only",
			gridType: types.GridTypeTN,
			phases:   3,
			want:     []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL1L2L3},
		},
		{
			name:        "TN 3-phase advertises the observed phase",
			gridType:    types.GridTypeTN,
			phases:      3,
			outputPhase: types.PhaseModeNL3,
			want:        []types.PhaseMode{types.PhaseModeNL3, types.PhaseModeNL1L2L3},
		},
		{
			name:        "IT 3-phase advertises the observed pair",
			gridType:    types.GridTypeIT,
			phases:      3,
			outputPhase: types.PhaseModeL2L3,
			want:        []types.PhaseMode{types.PhaseModeL2L3, types.PhaseModeL1L2L3},
		},
		{
			name:        "a three-phase reading does not pick a phase",
			gridType:    types.GridTypeTT,
			phases:      3,
			outputPhase: types.PhaseModeL1L2L3,
			want:        []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL1L2L3},
		},
		{
			name:        "a phase the grid does not offer is ignored",
			gridType:    types.GridTypeTN,
			phases:      3,
			outputPhase: types.PhaseModeL2L3,
			want:        []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL1L2L3},
		},
		{
			name:        "single-phase installs are unchanged",
			gridType:    types.GridTypeTN,
			phases:      1,
			outputPhase: types.PhaseModeNL3,
			want:        []types.PhaseMode{types.PhaseModeNL1},
		},
		{
			name:     "unknown grid type",
			gridType: types.GridTypeUnknown,
			phases:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, model.AdvertisedPhaseModes(tt.gridType, tt.phases, tt.outputPhase))
		})
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
