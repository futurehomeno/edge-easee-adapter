package api

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	log "github.com/sirupsen/logrus"
)

// Credentials stands for Easee API credentials.
type Credentials struct {
	AccessToken  string   `json:"accessToken"`
	ExpiresIn    int      `json:"expiresIn"`
	AccessClaims []string `json:"accessClaims"`
	TokenType    string   `json:"tokenType"`
	RefreshToken string   `json:"refreshToken"`
}

// Charger represents charger data.
type Charger struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Color         int       `json:"color"`
	CreatedOn     string    `json:"createdOn"`
	UpdatedOn     string    `json:"updatedOn"`
	BackPlate     BackPlate `json:"backPlate"`
	LevelOfAccess int       `json:"levelOfAccess"`
	ProductCode   int       `json:"productCode"`
}

// BackPlate represents charger's back plate.
type BackPlate struct {
	ID                string `json:"id"`
	MasterBackPlateID string `json:"masterBackPlateId"`
}

// ChargerConfig represents charger config.
type ChargerConfig struct {
	MaxChargerCurrent     float64  `json:"maxChargerCurrent"`
	DetectedPowerGridType GridType `json:"detectedPowerGridType"`
}

// GridType represents a grdi type.
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

func (g GridType) ToFimpGridType() (chargepoint.GridType, int) {
	if g >= GridTypeFirstInvalid {
		log.Warnf("Invalid grid type state %v", g)
	}

	switch g {
	case GridTypeUnknown,
		GridTypeNotYetDetected,
		GridTypeErrorNoValidPowerGridFound:
		return "", 0
	case GridTypeTN3Phase,
		GridTypeWarningTN3PhaseGNDFault:
		return chargepoint.GridTypeTN, 3
	case GridTypeTN2PhasePin23,
		GridTypeWarningTN2PhasePin235,
		GridTypeWarningTN2PhasePIN234,
		GridTypeWarningTN2PhaseGNDFault:
		return chargepoint.GridTypeTN, 2
	case GridTypeTN1Phase,
		GridTypeWarningTN1PhaseNeutralOnPin3:
		return chargepoint.GridTypeTN, 1
	case GridTypeIT3Phase,
		GridTypeWarningIT3PhaseGNDFault,
		GridTypeWarningIT3PhaseGNDFaultL3:
		return chargepoint.GridTypeIT, 3
	case GridTypeIT1Phase,
		GridTypeWarningIT1PhaseGNDFault,
		GridTypeWarningIT1PhaseGNDFaultL3:
		return chargepoint.GridTypeIT, 1
	case GridTypeErrorTN400VNeutralOnWrongPin:
		return chargepoint.GridTypeTN, 0
	case GridTypeErrorITGroundConnectedToPin2Or3:
		return chargepoint.GridTypeIT, 0
	default:
		log.Warnf("Unknown grid type: %v", g)

		return "", 0
	}
}

// loginBody represents a login request body.
type loginBody struct {
	Username string `json:"userName"`
	Password string `json:"password"`
}

// refreshBody represents a token refresh request body.
type refreshBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// cableLockBody represents a cable lock request body.
type cableLockBody struct {
	State bool `json:"state"`
}

// maxCurrentBody represents a charger max current request body.
type maxCurrentBody struct {
	MaxChargerCurrent float64 `json:"maxChargerCurrent"`
}

// dynamicCurrentBody represents a charger dynamic current request body.
type dynamicCurrentBody struct {
	DynamicChargerCurrent float64 `json:"dynamicChargerCurrent"`
}
