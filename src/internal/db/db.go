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
	bucketNamePrefix = "charging-sessions:"
)

// ChargingSessionStorage stores charging sessions per charger.
type ChargingSessionStorage interface {
	Start() error
	Stop() error
	Reset() error

	RegisterSessionStart(chargerID string, session model.StartChargingSession) error
	RegisterSessionStop(chargerID string, session model.StopChargingSession) error
	// LatestSessionsByChargerID returns the latest and the previous session.
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

		err = s.db.Set(bucket, idString(latest.ID), latest)
		if err != nil {
			return errors.Wrap(err, "register start session: can't update previous charging session")
		}
	}

	return s.db.Set(bucket, idString(session.ID), ChargingSession{
		ID:    session.ID,
		Start: session.Start,
	})
}

func (s *sessionStorage) RegisterSessionStop(chargerID string, session model.StopChargingSession) error {
	return s.db.Set(s.bucketName(chargerID), idString(session.ID), ChargingSession{
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

	keys := make([]int64, 0, len(stringKeys))

	for _, k := range stringKeys {
		key, err := strconv.ParseInt(k, 10, 64)
		if err != nil {
			log.Errorf("session storage: skipping unparsable session key %q in bucket %s: %v", k, bucket, err)

			continue
		}

		keys = append(keys, key)
	}

	// Session IDs, newest first.
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	sessions := make(ChargingSessions, 0, 2) // latest and previous

	for _, k := range keys {
		var session *ChargingSession

		ok, err := s.db.Get(bucket, strconv.FormatInt(k, 10), &session)
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

func idString(id int64) string {
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
