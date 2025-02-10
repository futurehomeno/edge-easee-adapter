package db

import (
	"sort"
	"strconv"
	"time"

	"github.com/futurehomeno/cliffhanger/database"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/internal/model"
)

const (
	bucketName = "charging-sessions:"
)

type SessionStorage interface {
	Start() error
	Stop() error

	RegisterStopSession(chargerID string, session model.StopChargingSession) error
	RegisterStartSession(chargerID string, session model.StartChargingSession) error
	GetLastChargingSessionsByChargerID(chargerID string, sessionNumber uint) (ChargingSessions, error)
}

type sessionStorage struct {
	db database.Database
}

func NewSessionStorage(workdir string) SessionStorage {
	db, err := database.NewDatabase(workdir)
	if err != nil {
		log.WithError(err).Error("can't create db")
	}

	return &sessionStorage{db}
}

func (s *sessionStorage) Start() error {
	return s.db.Start()
}

func (s *sessionStorage) Stop() error {
	return s.db.Stop()
}

func (s *sessionStorage) RegisterStartSession(chargerID string, session model.StartChargingSession) error {
	lastSession, err := s.GetLastChargingSessionsByChargerID(chargerID, 1)
	if err != nil {
		return errors.Wrap(err, "register start session: can't get last charging session:")
	}

	if len(lastSession) != 0 && lastSession[0].Stop.IsZero() {
		lastSession[0].Stop = session.Start

		err = s.db.Set(bucketName+chargerID, string(lastSession[0].ID), lastSession[0])
		if err != nil {
			return errors.Wrap(err, "register start session: can't update previous charging session:")
		}
	}

	return s.db.Set(bucketName+chargerID, string(session.ID), ChargingSession{
		EnergyKwh: session.MeterValue,
		ID:        session.ID,
		Start:     session.Start,
	})
}

func (s *sessionStorage) RegisterStopSession(chargerID string, session model.StopChargingSession) error {
	return s.db.Set(bucketName+chargerID, string(session.ID), ChargingSession{
		EnergyKwh: session.Energy,
		ID:        session.ID,
		Start:     session.Start,
		Stop:      session.Stop,
	})
}

func (s *sessionStorage) GetLastChargingSessionsByChargerID(chargerID string, sessionNumber uint) (ChargingSessions, error) {
	var sessions ChargingSessions

	keys, err := s.db.Keys(bucketName + chargerID)
	if err != nil {
		return ChargingSessions{}, err
	}

	if len(keys) != 0 {
		return ChargingSessions{}, errors.New("session not found for charger:" + chargerID)
	}

	sort.Slice(keys, func(i, j int) bool {
		key1, _ := strconv.Atoi(keys[i])
		key2, _ := strconv.Atoi(keys[j])

		return key1 > key2
	})

	for i := uint(1); i <= sessionNumber; i++ {
		var session *ChargingSession

		ok, err := s.db.Get(bucketName+chargerID, keys[i], session)
		if !ok || err != nil {
			return ChargingSessions{}, err
		}

		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[j].Stop.After(sessions[i].Stop)
	})

	return sessions, nil
}

type ChargingSession struct {
	EnergyKwh float64
	ID        int64
	Start     time.Time
	Stop      time.Time
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
