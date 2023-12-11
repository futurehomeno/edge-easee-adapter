package api

import (
	"time"

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

// ChargerSiteInfo represents charger rate current.
type ChargerSiteInfo struct {
	RatedCurrent float64 `json:"ratedCurrent"`
}

// ChargeSessions represents charge sessions.
type ChargeSessions []*ChargeSession

// Latest returns latest charge session.
func (c ChargeSessions) Latest() *ChargeSession {
	if len(c) < 1 {
		return nil
	}

	return c[0]
}

// Previous returns previous charge session.
func (c ChargeSessions) Previous() *ChargeSession {
	if len(c) < 2 {
		return nil
	}

	return c[1]
}

// ChargeSession represents charger session.
type ChargeSession struct {
	CarConnected    time.Time `json:"carConnected"`
	CarDisconnected time.Time `json:"carDisconnected"`
	KiloWattHours   float64   `json:"kiloWattHours"`
	IsComplete      bool      `json:"isComplete"`
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
