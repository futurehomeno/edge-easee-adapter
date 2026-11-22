package model

import (
	"fmt"
	"slices"

	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/types"
	log "github.com/sirupsen/logrus"
)

// easeePhaseModeSingle and EaseePhaseModeAuto are Easee's internal phase modes:
// 1 locks the charger to one phase, 2 lets it choose, 3 locks it to three phases.
// Three-phase requests map to auto rather than 3, because most EVs refuse a
// 1->3 phase transition mid-session and would stall on a hard lock, unless
// use_locked_3_phases asks for 3 during a session.
const (
	easeePhaseModeSingle = 1
	// EaseePhaseModeAuto lets the charger pick, so it is not pinned to any single leg.
	EaseePhaseModeAuto  = 2
	EaseePhaseModeLock3 = 3
)

var phaseModeNames = map[int]string{
	easeePhaseModeSingle: "single phase",
	EaseePhaseModeAuto:   "auto",
	EaseePhaseModeLock3:  "three phase",
}

// PhaseModeName names an Easee internal phase mode for a log line. A value Easee adds later is
// named "unknown" rather than guessed at, so the line cannot claim a mode that does not exist.
func PhaseModeName(phaseMode int) string {
	if name, ok := phaseModeNames[phaseMode]; ok {
		return name
	}

	return "unknown"
}

// SettablePhaseModes returns every phase mode the charger can be switched to, regardless
// of the mode it currently sits in. The auto row of the matrix is the union of the others.
func SettablePhaseModes(gridType types.GridType, phases int) []types.PhaseMode {
	return slices.Clone(SupportedPhaseModes(gridType, EaseePhaseModeAuto, phases))
}

// SettablePhaseModesFor is what a charger advertises: SettablePhaseModes of the FIMP pair the raw
// grid type detected maps to. WarningTN2PhasePin235 drives L1 and L3 (pins 3, 5), out of the TN
// 2-phase row's NL2 and NL1L2, and FIMP has no NL1L3 to name its two-phase mode, so it advertises
// none. An unknown raw type - a state stored by an older adapter - keeps the pair's row.
func SettablePhaseModesFor(detected GridType, gridType types.GridType, phases int) []types.PhaseMode {
	if detected == GridTypeWarningTN2PhasePin235 {
		return nil
	}

	return SettablePhaseModes(gridType, phases)
}

// ToEaseePhaseMode maps a FIMP phase mode onto Easee's internal phase mode.
func ToEaseePhaseMode(gridType types.GridType, phases int, mode types.PhaseMode, lock3 bool) (int, error) {
	if slices.Contains(SupportedPhaseModes(gridType, easeePhaseModeSingle, phases), mode) {
		return easeePhaseModeSingle, nil
	}

	if slices.Contains(SettablePhaseModes(gridType, phases), mode) {
		if lock3 {
			return EaseePhaseModeLock3, nil
		}

		return EaseePhaseModeAuto, nil
	}

	return 0, fmt.Errorf("phase modes mapper: mode %s unsupported on a %s grid with %d phases", mode, gridType, phases)
}

// SessionInProgress is the complement of ChargerState.IsSessionFinished on FIMP states.
func SessionInProgress(state chargepoint.State) bool {
	return state == chargepoint.StateReadyToCharge || state == chargepoint.StateCharging || state == chargepoint.StateSuspendedByEV
}

func SupportedPhaseModes(gridType types.GridType, phaseMode, phases int) []types.PhaseMode {
	if gridType == "" || phaseMode == 0 || phases == 0 {
		return nil
	}

	gridTypeMap, ok := phaseModeMatrix[gridType]
	if !ok {
		log.Errorf("phase modes mapper: unknown grid type: %s", gridType)

		return nil
	}

	phasesMap, ok := gridTypeMap[phases]
	if !ok {
		log.Errorf("phase modes mapper: unsupported number of phases: %d", phases)

		return nil
	}

	phaseModeMap, ok := phasesMap[phaseMode]
	if !ok {
		log.Errorf("phase modes mapper: unknown Easee's internal phase mode: %d", phaseMode)

		return nil
	}

	return phaseModeMap
}

var phaseModeMatrix = map[types.GridType]map[int]map[int][]types.PhaseMode{
	types.GridTypeTN: {
		1: {
			1: []types.PhaseMode{types.PhaseModeNL1},
			2: []types.PhaseMode{types.PhaseModeNL1},
		},
		2: {
			1: []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2},
			2: []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL1L2},
			3: []types.PhaseMode{types.PhaseModeNL1L2},
		},
		3: {
			1: []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3},
			2: []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3, types.PhaseModeNL1L2L3},
			3: []types.PhaseMode{types.PhaseModeNL1L2L3},
		},
	},
	types.GridTypeTT: {
		1: {
			1: []types.PhaseMode{types.PhaseModeL1L2},
			2: []types.PhaseMode{types.PhaseModeL1L2},
		},
		3: {
			1: []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1},
			2: []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1, types.PhaseModeL1L2L3},
			3: []types.PhaseMode{types.PhaseModeL1L2L3},
		},
	},
	types.GridTypeIT: {
		1: {
			1: []types.PhaseMode{types.PhaseModeL1L2},
			2: []types.PhaseMode{types.PhaseModeL1L2},
		},
		3: {
			1: []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1},
			2: []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1, types.PhaseModeL1L2L3},
			3: []types.PhaseMode{types.PhaseModeL1L2L3},
		},
	},
}
