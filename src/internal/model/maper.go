package model

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

func SupportedPhaseModes(gridType chargepoint.GridType, phaseMode int, phases int) []chargepoint.PhaseMode {
	gridTypeMap, ok := phaseModeMatrix[gridType]
	if !ok {
		log.Errorf("can't set supported phase modes for gridType: %v", gridType)

		return []chargepoint.PhaseMode{}
	}

	phasesMap, ok := gridTypeMap[phases]
	if !ok {
		log.Errorf("can't set supported phase modes for phases: %v", phases)

		return []chargepoint.PhaseMode{}
	}

	phaseModeMap, ok := phasesMap[phaseMode]
	if !ok {
		log.Errorf("can't set supported phase modes for easee phase mode: %v", phaseMode)

		return []chargepoint.PhaseMode{}
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

// GridType represents a grid type.
type GridType int

const (
	GridTypeUnknown                         GridType = -1
	GridTypeNotYetDetected                  GridType = 0
	GridTypeTN3Phase                        GridType = 1
	GridTypeTN2PhasePin23                   GridType = 2
	GridTypeTN1Phase                        GridType = 3
	GridTypeIT3Phase                        GridType = 4
	GridTypeIT1Phase                        GridType = 5
	GridTypeWarningTN2PhasePin235           GridType = 30
	GridTypeWarningTN1PhaseNeutralOnPin3    GridType = 31
	GridTypeWarningIT3PhaseGNDFault         GridType = 32
	GridTypeWarningIT1PhaseGNDFault         GridType = 33
	GridTypeErrorNoValidPowerGridFound      GridType = 50
	GridTypeErrorTN400VNeutralOnWrongPin    GridType = 51
	GridTypeErrorITGroundConnectedToPin2Or3 GridType = 52
	GridTypeWarningIT3PhaseGNDFaultL3       GridType = 34
	GridTypeWarningIT1PhaseGNDFaultL3       GridType = 35
	GridTypeWarningTN2PhasePIN234           GridType = 36
	GridTypeWarningTN3PhaseGNDFault         GridType = 37
	GridTypeWarningTN2PhaseGNDFault         GridType = 38

	GridTypeFirstInvalid = GridTypeWarningTN2PhasePin235
)

type networkType struct {
	gridType chargepoint.GridType
	phase    int
}

var easeeNetworkTypeMap = map[GridType]networkType{
	GridTypeUnknown:                         {"", 0},
	GridTypeNotYetDetected:                  {"", 0},
	GridTypeTN3Phase:                        {chargepoint.GridTypeTN, 3},
	GridTypeTN2PhasePin23:                   {chargepoint.GridTypeTN, 2},
	GridTypeTN1Phase:                        {chargepoint.GridTypeTN, 1},
	GridTypeIT3Phase:                        {chargepoint.GridTypeIT, 3},
	GridTypeIT1Phase:                        {chargepoint.GridTypeIT, 1},
	GridTypeWarningTN2PhasePin235:           {chargepoint.GridTypeTN, 2},
	GridTypeWarningTN1PhaseNeutralOnPin3:    {chargepoint.GridTypeTN, 1},
	GridTypeWarningIT3PhaseGNDFault:         {chargepoint.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFault:         {chargepoint.GridTypeIT, 1},
	GridTypeErrorNoValidPowerGridFound:      {"", 0},
	GridTypeErrorTN400VNeutralOnWrongPin:    {chargepoint.GridTypeTN, 0},
	GridTypeErrorITGroundConnectedToPin2Or3: {chargepoint.GridTypeIT, 0},
	GridTypeWarningIT3PhaseGNDFaultL3:       {chargepoint.GridTypeIT, 3},
	GridTypeWarningIT1PhaseGNDFaultL3:       {chargepoint.GridTypeIT, 1},
	GridTypeWarningTN2PhasePIN234:           {chargepoint.GridTypeTN, 2},
	GridTypeWarningTN3PhaseGNDFault:         {chargepoint.GridTypeTN, 3},
	GridTypeWarningTN2PhaseGNDFault:         {chargepoint.GridTypeTN, 2},
}

// ToFimpGridType returns grid type and phases.
func (g GridType) ToFimpGridType() (chargepoint.GridType, int) {
	if g >= GridTypeFirstInvalid {
		log.Warnf("Invalid grid type state %v", g)
	}

	if networkType, ok := easeeNetworkTypeMap[g]; ok {
		return networkType.gridType, networkType.phase
	}

	log.Warnf("Unknown grid type: %v", g)

	return "", 0
}
