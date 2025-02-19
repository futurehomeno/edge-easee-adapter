package model

import (
	"strconv"
	"time"
)

type StartChargingSession struct {
	ID         int64     `json:"Id"`
	MeterValue float64   `json:"MeterValue"`
	Start      time.Time `json:"Start"`
}

func (s *StartChargingSession) IDString() string {
	return strconv.FormatInt(s.ID, 10)
}

type StopChargingSession struct {
	ID              int64     `json:"Id"`
	Energy          float64   `json:"EnergyKwh"`
	MeterValueStart float64   `json:"MeterValueStart"`
	MeterValueStop  float64   `json:"MeterValueStop"`
	Start           time.Time `json:"Start"`
	Stop            time.Time `json:"Stop"`
}

func (s *StopChargingSession) IDString() string {
	return strconv.FormatInt(s.ID, 10)
}
