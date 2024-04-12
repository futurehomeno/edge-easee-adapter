package helper

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

func SupportedPhaseModes(gridType chargepoint.GridType, phaseMode int, phases int) []chargepoint.PhaseMode { //nolint:cyclop
	if phases == 1 {
		if gridType == chargepoint.GridTypeTN {
			return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1}
		}

		if gridType == chargepoint.GridTypeIT || gridType == chargepoint.GridTypeTT {
			return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2}
		}
	}

	if phases == 3 {
		if gridType == chargepoint.GridTypeTN {
			switch phaseMode {
			case 1:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3}
			case 2:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3, chargepoint.PhaseModeNL1L2L3}
			case 3:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeNL1L2L3}
			}
		}

		if gridType == chargepoint.GridTypeIT || gridType == chargepoint.GridTypeTT {
			switch phaseMode {
			case 1:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1}
			case 2:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2, chargepoint.PhaseModeL2L3, chargepoint.PhaseModeL3L1, chargepoint.PhaseModeL1L2L3}
			case 3:
				return []chargepoint.PhaseMode{chargepoint.PhaseModeL1L2L3}
			}
		}
	}

	log.Errorf("can't set supported phase modes")

	return []chargepoint.PhaseMode{}
}
