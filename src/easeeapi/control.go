package easeeapi

import log "github.com/sirupsen/logrus"

// StartCharging starts charging on a charger
func (e *Easee) StartCharging(chargerID string) error {
	var err error
	err = e.client.ControlCharger(chargerID, Start)
	if err != nil {
		log.Error("Easee - StartCharging: ", err)
		return err
	}
	return err
}

// StopCharing stops charging on a charger
func (e *Easee) StopCharing(chargerID string) error {
	var err error
	err = e.client.ControlCharger(chargerID, Stop)
	if err != nil {
		log.Error("Easee - StopCharging: ", err)
		return err
	}
	return err
}

// PauseCharging puts the charge session on hold
func (e *Easee) PauseCharging(chargerID string) error {
	var err error
	err = e.client.ControlCharger(chargerID, Pause)
	if err != nil {
		log.Error("Easee - PauseCharging: ", err)
		return err
	}
	return err
}

// ResumeCharging resumes the charge session
func (e *Easee) ResumeCharging(chargerID string) error {
	var err error
	err = e.client.ControlCharger(chargerID, Resume)
	if err != nil {
		log.Error("Easee - ResumeCharging: ", err)
		return err
	}
	return err
}
