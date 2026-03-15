package model

import (
	"github.com/futurehomeno/cliffhanger/types"
	log "github.com/sirupsen/logrus"
)

func SupportedPhaseModes(gridType types.GridType, easeePhaseMode EaseePhaseModeT, phases int) []types.PhaseMode {
	if gridType == "" || easeePhaseMode == 0 || phases == 0 {
		log.Warnf("phase modes mapper: unset grid type or phases: %s", gridType)
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

	phaseModeMap, ok := phasesMap[easeePhaseMode]
	if !ok {
		log.Errorf("phase modes mapper: unknown easee phase mode: %d", easeePhaseMode)
		return nil
	}

	return phaseModeMap
}

//gridtype:phases:phaseMode(1: 1phase, 2: auto, 3:3phase)
var phaseModeMatrix = map[types.GridType]map[int]map[EaseePhaseModeT][]types.PhaseMode{
	types.GridTypeTN: {
		1: {
			EaseePhaseMode1Phase: []types.PhaseMode{types.PhaseModeNL1},
			EaseePhaseModeAuto:   []types.PhaseMode{types.PhaseModeNL1},
		},
		3: {
			EaseePhaseMode1Phase: []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3},
			EaseePhaseModeAuto:   []types.PhaseMode{types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3, types.PhaseModeNL1L2, types.PhaseModeNL2L3, types.PhaseModeNL1L2L3},
			EaseePhaseMode3Phase: []types.PhaseMode{types.PhaseModeNL1L2L3},
		},
	},
	types.GridTypeTT: {
		1: {
			EaseePhaseMode1Phase: []types.PhaseMode{types.PhaseModeL1L2},
			EaseePhaseModeAuto:   []types.PhaseMode{types.PhaseModeL1L2},
		},
		3: {
			EaseePhaseMode1Phase: []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1},
			EaseePhaseModeAuto:   []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1 /* forbidden types.PhaseModeL1L2L3*/},
			// forbidden EaseePhaseMode3Phase: []types.PhaseMode{types.PhaseModeL1L2L3},
		},
	},
	types.GridTypeIT: {
		1: {
			EaseePhaseMode1Phase: []types.PhaseMode{types.PhaseModeL1L2},
			EaseePhaseModeAuto:   []types.PhaseMode{types.PhaseModeL1L2},
		},
		3: {
			1:                  []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1},
			EaseePhaseModeAuto: []types.PhaseMode{types.PhaseModeL1L2, types.PhaseModeL2L3, types.PhaseModeL3L1 /* forbidden types.PhaseModeL1L2L3*/},
			// forbidden  EaseePhaseMode3Phase: []types.PhaseMode{types.PhaseModeL1L2L3},
		},
	},
}
