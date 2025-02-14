package db_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/futurehomeno/edge-easee-adapter/internal/db"
	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

type SessionStorageSuite struct {
	suite.Suite

	chargerID string

	storage db.ChargingSessionStorage
}

func TestSessionStorageSuite(t *testing.T) { //nolint:paralleltest
	suite.Run(t, new(SessionStorageSuite))
}

func (suite *SessionStorageSuite) SetupSuite() {
	suite.chargerID = "XX12345"
}

func (suite *SessionStorageSuite) SetupTest() {
	fileDB, err := database.NewDatabase(suite.T().TempDir())
	suite.Require().NoError(err)

	suite.storage = db.NewSessionStorage(fileDB)
}

func (suite *SessionStorageSuite) TestRegister_StartAndStopSession() {
	startTime := time.Date(1997, time.February, 17, 18, 0, 0, 0, time.UTC)
	stopTime := startTime.Add(time.Hour)

	err := suite.storage.RegisterSessionStart(suite.chargerID, model.StartChargingSession{
		ID:         1,
		Start:      startTime,
		MeterValue: 10,
	})
	suite.NoError(err)

	got, err := suite.storage.LatestSessionsByChargerID(suite.chargerID)
	suite.NoError(err)

	suite.NotEmpty(got)
	suite.Equal(&db.ChargingSession{
		ID:    1,
		Start: startTime,
	}, got.Latest())

	err = suite.storage.RegisterSessionStop(suite.chargerID, model.StopChargingSession{
		ID:              1,
		Start:           startTime,
		Stop:            stopTime,
		MeterValueStart: 10,
		MeterValueStop:  20,
		Energy:          10,
	})
	suite.NoError(err)

	got, err = suite.storage.LatestSessionsByChargerID(suite.chargerID)
	suite.NoError(err)

	suite.NotEmpty(got)
	suite.Equal(&db.ChargingSession{
		ID:     1,
		Start:  startTime,
		Stop:   stopTime,
		Energy: 10,
	}, got.Latest())

	suite.Nil(got.Previous())
}

func (suite *SessionStorageSuite) TestRegister_TwoConsecutiveSessionStarts() {
	startTime1 := time.Date(1997, time.February, 17, 18, 0, 0, 0, time.UTC)
	startTime2 := startTime1.Add(time.Hour)

	err := suite.storage.RegisterSessionStart(suite.chargerID, model.StartChargingSession{
		ID:         1,
		Start:      startTime1,
		MeterValue: 10,
	})
	suite.NoError(err)

	got, err := suite.storage.LatestSessionsByChargerID(suite.chargerID)
	suite.NoError(err)

	suite.NotEmpty(got)
	suite.Equal(&db.ChargingSession{
		ID:    1,
		Start: startTime1,
	}, got.Latest())

	err = suite.storage.RegisterSessionStart(suite.chargerID, model.StartChargingSession{
		ID:         2,
		Start:      startTime2,
		MeterValue: 10,
	})
	suite.NoError(err)

	got, err = suite.storage.LatestSessionsByChargerID(suite.chargerID)
	suite.NoError(err)

	suite.NotEmpty(got)
	suite.Equal(&db.ChargingSession{
		ID:    1,
		Start: startTime1,
		Stop:  startTime2, // Start time of a new session is treated as a stop time of the previous session.
	}, got.Previous())

	suite.Equal(&db.ChargingSession{
		ID:    2,
		Start: startTime2,
	}, got.Latest())
}

func (suite *SessionStorageSuite) TestRegister_TwoConsecutiveSessionStops() {
	timeStart := time.Date(1997, time.February, 17, 18, 0, 0, 0, time.UTC)
	timeStop := timeStart.Add(time.Hour)

	err := suite.storage.RegisterSessionStop(suite.chargerID, model.StopChargingSession{
		ID:              1,
		Energy:          10,
		MeterValueStart: 10,
		MeterValueStop:  20,
		Start:           timeStart,
		Stop:            timeStop,
	})
	suite.NoError(err)

	got, err := suite.storage.LatestSessionsByChargerID(suite.chargerID)
	suite.NoError(err)

	suite.NotEmpty(got)
	suite.Equal(&db.ChargingSession{
		ID:     1,
		Start:  timeStart,
		Stop:   timeStop,
		Energy: 10,
	}, got.Latest())

	err = suite.storage.RegisterSessionStop(suite.chargerID, model.StopChargingSession{
		ID:              2,
		Energy:          5,
		MeterValueStart: 20,
		MeterValueStop:  25,
		Start:           timeStart.Add(2 * time.Hour),
		Stop:            timeStop.Add(2 * time.Hour),
	})
	suite.NoError(err)

	got, err = suite.storage.LatestSessionsByChargerID(suite.chargerID)
	suite.NoError(err)

	suite.NotEmpty(got)
	suite.Equal(&db.ChargingSession{
		ID:     1,
		Start:  timeStart,
		Stop:   timeStop,
		Energy: 10,
	}, got.Previous())

	suite.Equal(&db.ChargingSession{
		ID:     2,
		Start:  timeStart.Add(2 * time.Hour),
		Stop:   timeStop.Add(2 * time.Hour),
		Energy: 5,
	}, got.Latest())
}

func (suite *SessionStorageSuite) TestGetSessionNonExistChargerID() {
	result, err := suite.storage.LatestSessionsByChargerID(suite.chargerID)

	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), result)
}
