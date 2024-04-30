package model

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

func SupportedPhaseModes(gridType chargepoint.GridType, phaseMode, phases int) []chargepoint.PhaseMode {
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

var phaseModeMatrix = map[chargepoint.GridType]map[int]map[int][]chargepoint.PhaseMode{
	chargepoint.GridTypeTN: {
		1: {
			1: []chargepoint.PhaseMode{chargepoint.PhaseModeNL1},
			2: []chargepoint.PhaseMode{chargepoint.PhaseModeNL1},
		},
		3: {
			1: []chargepoint.PhaseMode{chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3},
			2: []chargepoint.PhaseMode{chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3, chargepoint.PhaseModeNL1L2L3},
			3: []chargepoint.PhaseMode{chargepoint.PhaseModeNL1L2L3},
		},
	},
	chargepoint.GridTypeTT: {
		1: {
			1: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2},
			2: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2},
		},
		3: {
			1: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1},
			2: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1, chargepoint.PhaseModeL1L2L3},
			3: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2L3},
		},
	},
	chargepoint.GridTypeIT: {
		1: {
			1: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2},
			2: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2},
		},
		3: {
			1: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1},
			2: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1, chargepoint.PhaseModeL1L2L3},
			3: []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2L3},
		},
	},
}
