package model

import (
	"github.com/futurehomeno/cliffhanger/types"
	log "github.com/sirupsen/logrus"
)

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
