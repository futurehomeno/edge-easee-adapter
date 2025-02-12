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

const chargerID = "chargerID"

func (s *SessionStorageSuite) TestRegisterStopSession() {
	database, err := database.NewDatabase(s.T().TempDir())
	require.NoError(s.T(), err)

	sessionStorage := db.NewSessionStorage(database)

	timeStart := time.Date(1997, 0o2, 17, 18, 0, 0, 0, time.UTC)
	timeStop := timeStart.Add(time.Hour)

	err = sessionStorage.RegisterSessionStop(chargerID, model.StopChargingSession{
		ID:              1,
		Energy:          10,
		MeterValueStart: 0,
		MeterValueStop:  0,
		Start:           timeStart,
		Stop:            timeStop,
	})

	require.NoError(s.T(), err)

	result, err := sessionStorage.LatestSessionsByChargerID(chargerID, uint(1))

	require.NoError(s.T(), err)

	assert.NotEmpty(s.T(), result)

	assert.Equal(s.T(), &db.ChargingSession{
		EnergyKwh: 10,
		ID:        1,
		Start:     timeStart,
		Stop:      timeStop,
	}, result[0])

	err = sessionStorage.Reset()
	require.NoError(s.T(), err)
}

func (s *SessionStorageSuite) TestRegisterStartSession() {
	database, err := database.NewDatabase(s.T().TempDir())
	require.NoError(s.T(), err)

	sessionStorage := db.NewSessionStorage(database)

	timeStart1 := time.Date(1997, 0o2, 17, 18, 0, 0, 0, time.UTC)
	timeStart2 := timeStart1.Add(time.Hour)

	err = sessionStorage.RegisterSessionStart(chargerID, model.StartChargingSession{
		ID:         1,
		Start:      timeStart1,
		MeterValue: 10,
	})

	require.NoError(s.T(), err)

	result, err := sessionStorage.LatestSessionsByChargerID(chargerID, uint(1))

	require.NoError(s.T(), err)

	assert.NotEmpty(s.T(), result)

	assert.Equal(s.T(), &db.ChargingSession{
		EnergyKwh: 0,
		ID:        1,
		Start:     timeStart1,
	}, result[0])

	err = sessionStorage.RegisterSessionStart(chargerID, model.StartChargingSession{
		ID:         2,
		Start:      timeStart2,
		MeterValue: 10,
	})

	require.NoError(s.T(), err)

	result, err = sessionStorage.LatestSessionsByChargerID(chargerID, uint(2))

	require.NoError(s.T(), err)

	assert.NotEmpty(s.T(), result)

	assert.Equal(s.T(), &db.ChargingSession{
		EnergyKwh: 0,
		ID:        1,
		Start:     timeStart1,
		Stop:      timeStart2,
	}, result[1])

	assert.Equal(s.T(), &db.ChargingSession{
		ID:    2,
		Start: timeStart2,
	}, result[0])

	err = sessionStorage.Reset()
	require.NoError(s.T(), err)
}

func (s *SessionStorageSuite) TestGetSessionNonExistChargerID() {
	database, err := database.NewDatabase(s.T().TempDir())
	require.NoError(s.T(), err)

	sessionStorage := db.NewSessionStorage(database)

	result, err := sessionStorage.LatestSessionsByChargerID(chargerID, uint(1))
	require.NoError(s.T(), err)

	assert.Empty(s.T(), result)

	err = sessionStorage.Reset()
	require.NoError(s.T(), err)
}

type SessionStorageSuite struct {
	suite.Suite
}

func TestSessionStorageSuite(t *testing.T) { //nolint:paralleltest
	suite.Run(t, new(SessionStorageSuite))
}
