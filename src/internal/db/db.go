package db

import (
	"sort"
	"strconv"
	"time"

	"github.com/futurehomeno/cliffhanger/database"
	"github.com/pkg/errors"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

const (
	bucketNamePrefix = "charging-sessions:"
)

// ChargingSessionStorage is service used to store charging sessions.
type ChargingSessionStorage interface {
	// Start starts ChargingSessionStorage service.
	Start() error
	// Stop stops ChargingSessionStorage service.
	Stop() error
	// Reset ChargingSessionStorage service.
	Reset() error

	// RegisterSessionStart registers start charging session for charger with chargerID.
	RegisterSessionStart(chargerID string, session model.StartChargingSession) error
	// RegisterSessionStop registers stop charging session for charger with chargerID.
	RegisterSessionStop(chargerID string, session model.StopChargingSession) error
	// LatestSessionsByChargerID returns sessionNumber last ChargingSessions for charger with chargerID.
	LatestSessionsByChargerID(chargerID string, sessionNumber uint) (ChargingSessions, error)
}

type sessionStorage struct {
	db database.Database
}

func NewSessionStorage(db database.Database) ChargingSessionStorage {
	return &sessionStorage{db}
}

func (s *sessionStorage) Start() error {
	return s.db.Start()
}

func (s *sessionStorage) Stop() error {
	return s.db.Stop()
}

func (s *sessionStorage) Reset() error {
	return s.db.Reset()
}

func (s *sessionStorage) RegisterSessionStart(chargerID string, session model.StartChargingSession) error {
	lastSession, err := s.LatestSessionsByChargerID(chargerID, 1)
	if err != nil {
		return errors.Wrap(err, "register start session: can't get last charging session")
	}

	if len(lastSession) != 0 && lastSession.Latest().Stop.IsZero() {
		lastSession.Latest().Stop = session.Start

		err = s.db.Set(bucketName(chargerID), IDString(lastSession.Latest().ID), lastSession.Latest())
		if err != nil {
			return errors.Wrap(err, "register start session: can't update previous charging session")
		}
	}

	return s.db.Set(bucketName(chargerID), IDString(session.ID), ChargingSession{
		ID:    session.ID,
		Start: session.Start,
	})
}

func (s *sessionStorage) RegisterSessionStop(chargerID string, session model.StopChargingSession) error {
	return s.db.Set(bucketName(chargerID), IDString(session.ID), ChargingSession{
		EnergyKwh: session.Energy,
		ID:        session.ID,
		Start:     session.Start,
		Stop:      session.Stop,
	})
}

func (s *sessionStorage) LatestSessionsByChargerID(chargerID string, sessionNumber uint) (ChargingSessions, error) {
	var sessions ChargingSessions

	keys, err := s.db.Keys(bucketName(chargerID))
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, nil
	}

	sort.Slice(keys, func(i, j int) bool {
		key1, _ := strconv.Atoi(keys[i])
		key2, _ := strconv.Atoi(keys[j])

		return key1 > key2
	})

	for i := 0; uint(i) < sessionNumber && i < len(keys); i++ {
		var session *ChargingSession

		ok, err := s.db.Get(bucketName(chargerID), keys[i], &session)
		if !ok || err != nil {
			return nil, err
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

type ChargingSession struct {
	EnergyKwh float64   `json:"EnergyKwh"`
	ID        int64     `json:"ID"`
	Start     time.Time `json:"Start"`
	Stop      time.Time `json:"Stop"`
}

func IDString(id int64) string {
	return strconv.FormatInt(id, 10)
}

type ChargingSessions []*ChargingSession

func (c ChargingSessions) Latest() *ChargingSession {
	if len(c) < 1 {
		return nil
	}

	return c[0]
}

func (c ChargingSessions) Previous() *ChargingSession {
	if len(c) < 2 {
		return nil
	}

	return c[1]
}

func bucketName(chargerID string) string {
	return bucketNamePrefix + chargerID
}
