package model

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/_old/utils"
)

// ServiceName is the fimp service name
const ServiceName = "easee"

// Configs is used for config file
type Configs struct {
	path               string
	InstanceAddress    string    `json:"instance_address"`
	MqttServerURI      string    `json:"mqtt_server_uri"`
	MqttUsername       string    `json:"mqtt_server_username"`
	MqttPassword       string    `json:"mqtt_server_password"`
	MqttClientIdPrefix string    `json:"mqtt_client_id_prefix"`
	LogFile            string    `json:"log_file"`
	LogLevel           string    `json:"log_level"`
	LogFormat          string    `json:"log_format"`
	WorkDir            string    `json:"-"`
	ConfiguredAt       string    `json:"configured_at"`
	ConfiguredBy       string    `json:"configured_by"`
	AccessToken        string    `json:"accessToken"`
	RefreshToken       string    `json:"refreshToken"`
	ExpiresAt          time.Time `json:"expiresAt"`
	PollTimeSec        uint16    `json:"poll_time_sec"`
}

// NewConfigs creates a new config
func NewConfigs(workDir string) *Configs {
	conf := &Configs{WorkDir: workDir}
	conf.path = filepath.Join(workDir, "data", "config.json")
	if !utils.FileExists(conf.path) {
		log.Info("Config file doesn't exist.Loading default config")
		defaultConfigFile := filepath.Join(workDir, "defaults", "config.json")
		err := utils.CopyFile(defaultConfigFile, conf.path)
		if err != nil {
			fmt.Print(err)
			panic("Can't copy config file.")
		}
	}
	return conf
}

// LoadFromFile does that
func (cf *Configs) LoadFromFile() error {
	configFileBody, err := ioutil.ReadFile(cf.path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(configFileBody, cf)
	if err != nil {
		return err
	}
	return nil
}

// SaveToFile does that
func (cf *Configs) SaveToFile() error {
	cf.ConfiguredBy = "auto"
	cf.ConfiguredAt = time.Now().Format(time.RFC3339)
	if cf.PollTimeSec < 5 {
		cf.PollTimeSec = 10
	}
	bpayload, err := json.Marshal(cf)
	err = ioutil.WriteFile(cf.path, bpayload, 0664)
	if err != nil {
		return err
	}
	return err
}

// GetDataDir gets data dir in workdir
func (cf *Configs) GetDataDir() string {
	return filepath.Join(cf.WorkDir, "data")
}

// GetDefaultDir gets default dir in workdir
func (cf *Configs) GetDefaultDir() string {
	return filepath.Join(cf.WorkDir, "defaults")
}

// LoadDefaults does that
func (cf *Configs) LoadDefaults() error {
	configFile := filepath.Join(cf.WorkDir, "data", "config.json")
	os.Remove(configFile)
	log.Info("Config file doesn't exist.Loading default config")
	defaultConfigFile := filepath.Join(cf.WorkDir, "defaults", "config.json")
	return utils.CopyFile(defaultConfigFile, configFile)
}

// IsConfigured not used
func (cf *Configs) IsConfigured() bool {
	// TODO : Add logic here
	if cf.AccessToken != "" && cf.RefreshToken != "" {
		return true
	}
	return false
}

// IsTokenExpired checks if token is expired
func (cf *Configs) IsTokenExpired() bool {
	diff := cf.ExpiresAt.Sub(time.Now())
	if diff <= 0 {
		return true
	}
	return false
}

// SetExpiresAt saves token expire date
func (cf *Configs) SetExpiresAt(sec float64) {
	now := time.Now()
	expireTime := now.Add(time.Duration(sec) * time.Second)
	cf.ExpiresAt = expireTime

}

// ClearTokens clears and saves config
func (cf *Configs) ClearTokens() {
	cf.AccessToken = ""
	cf.RefreshToken = ""
	cf.ExpiresAt = time.Time{}
	cf.SaveToFile()
}
