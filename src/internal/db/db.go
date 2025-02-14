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
	// LatestSessionsByChargerID returns latest and previous charging sessions by chargerID.
	LatestSessionsByChargerID(chargerID string) (ChargingSessions, error)
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
	sessions, err := s.LatestSessionsByChargerID(chargerID)
	if err != nil {
		return errors.Wrap(err, "register start session: can't get last charging session")
	}

	bucket := s.bucketName(chargerID)
	latest := sessions.Latest()

	if latest != nil && latest.Stop.IsZero() {
		latest.Stop = session.Start

		err = s.db.Set(bucket, latest.IDString(), latest)
		if err != nil {
			return errors.Wrap(err, "register start session: can't update previous charging session")
		}
	}

	return s.db.Set(bucket, session.IDString(), ChargingSession{
		ID:    session.ID,
		Start: session.Start,
	})
}

func (s *sessionStorage) RegisterSessionStop(chargerID string, session model.StopChargingSession) error {
	return s.db.Set(s.bucketName(chargerID), session.IDString(), ChargingSession{
		ID:     session.ID,
		Start:  session.Start,
		Stop:   session.Stop,
		Energy: session.Energy,
	})
}

func (s *sessionStorage) LatestSessionsByChargerID(chargerID string) (ChargingSessions, error) {
	bucket := s.bucketName(chargerID)

	stringKeys, err := s.db.Keys(bucket)
	if err != nil {
		return nil, err
	}

	keys := make([]int, 0, len(stringKeys))

	for _, k := range stringKeys {
		key, _ := strconv.Atoi(k)
		keys = append(keys, key)
	}

	// Sort keys (session IDs) in descending order.
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	sessions := make(ChargingSessions, 0, 2) // latest and previous

	for _, k := range keys {
		var session *ChargingSession

		ok, err := s.db.Get(bucket, strconv.Itoa(k), &session)
		if !ok || err != nil {
			return nil, err
		}

		sessions = append(sessions, session)

		if len(sessions) == 2 {
			break
		}
	}

	return sessions, nil
}

func (s *sessionStorage) bucketName(chargerID string) string {
	return bucketNamePrefix + chargerID
}

type ChargingSession struct {
	ID     int64     `json:"id"`
	Start  time.Time `json:"start"`
	Stop   time.Time `json:"stop"`
	Energy float64   `json:"energy"`
}

func (s *ChargingSession) IDString() string {
	return strconv.FormatInt(s.ID, 10)
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
