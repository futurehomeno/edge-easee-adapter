package easee

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/edge-easee-adapter/_old/utils"
)

// Easee structure for products and http client
type Easee struct {
	Products map[string]Product
	client   *Client
	workDir  string
	path     string
}

// NewEasee creates a Easee structure
func NewEasee(client *Client, workDir string) *Easee {
	easee := &Easee{
		client:  client,
		workDir: workDir,
	}
	easee.path = filepath.Join(easee.workDir, "data", "products.json")
	easee.Products = map[string]Product{}
	return easee
}

// Login gets tokens with username and password
func (e *Easee) Login(userPw Login) error {
	userToken, err := e.client.GetTokens(userPw)
	if err != nil {
		return err
	}
	err = e.client.SetUserToken(userToken)
	if err != nil {
		return err
	}
	return nil
}

// SetUserToken sets usertoken on easee client
func (e *Easee) SetUserToken(tokens *UserToken) error {
	var err error
	if tokens == nil {
		err = fmt.Errorf("Tried to update user tokens with empty UserToken")
		return err
	}
	err = e.client.SetUserToken(tokens)
	if err != nil {
		return err
	}
	return nil
}

// GetAccessToken returns client access token
func (e *Easee) GetAccessToken() string {
	return e.client.GetUserToken().AccessToken
}

// GetRefreshToken returns client refres token
func (e *Easee) GetRefreshToken() string {
	return e.client.GetUserToken().RefreshToken
}

// GetExpiresIn returns client refres token
func (e *Easee) GetExpiresIn() float64 {
	return e.client.GetUserToken().ExpiresIn
}

// LoadProductsFromFile does that
func (e *Easee) LoadProductsFromFile() error {
	if !utils.FileExists(e.path) {
		return fmt.Errorf("products.json file doesn't exist")
	}
	productsFileBody, err := ioutil.ReadFile(e.path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(productsFileBody, &e.Products)
	if err != nil {
		return err
	}
	return nil
}

// GetProducts gets chargers for account and creates Product from them
func (e *Easee) GetProducts() error {
	chargers, err := e.client.GetChargers()
	if err != nil {
		return err
	}
	if e.Products == nil {
		e.Products = map[string]Product{}
	}
	for _, c := range chargers {
		charger := c
		e.Products[charger.ID] = Product{
			Charger:       &charger,
			ChargerConfig: nil,
			ChargerState:  nil,
		}
	}
	return nil
}

// GetChargerConfig gets config for one charger and adds it to the product
func (e *Easee) GetChargerConfig(chargerID string) error {
	config, err := e.client.GetChargerConfig(chargerID)
	if err != nil {
		return err
	}
	if product, ok := e.Products[chargerID]; ok {
		product.ChargerConfig = config
		e.Products[chargerID] = product
	} else {
		err := fmt.Errorf("No charger with id: %s", chargerID)
		return err
	}
	return nil
}

// GetChargerState gets the state for one charger and adds it to the product
func (e *Easee) GetChargerState(chargerID string) error {
	log.Debug("Get charger state for : ", chargerID)
	state, err := e.client.GetChargerState(chargerID)
	if err != nil {
		return err
	}
	if product, ok := e.Products[chargerID]; ok {
		product.LastState = product.ChargerState
		product.ChargerState = state
		e.Products[chargerID] = product
	} else {
		err := fmt.Errorf("No charger with id: %s", chargerID)
		return err
	}
	return nil
}

// GetStateForAllProducts uses GetChargerState on all chargers in product
func (e *Easee) GetStateForAllProducts() error {
	var err error
	for _, product := range e.Products {
		err = e.GetChargerState(product.Charger.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetConfigForAllProducts uses GetChargerConfig on all chargers in product
func (e *Easee) GetConfigForAllProducts() error {
	var err error
	for _, product := range e.Products {
		err = e.GetChargerConfig(product.Charger.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

// SaveProductsToFile does that
func (e *Easee) SaveProductsToFile() error {

	//e.ConfiguredAt = time.Now().Format(time.RFC3339)
	bpayload, err := json.Marshal(e.Products)
	err = ioutil.WriteFile(e.path, bpayload, 0664)
	if err != nil {
		return err
	}
	return err
}

// RemoveProduct removes charger from products and save to disk
func (e *Easee) RemoveProduct(chargerID string) error {
	var err error
	if _, ok := e.Products[chargerID]; ok {
		delete(e.Products, chargerID)
		e.SaveProductsToFile()
	} else {
		err := fmt.Errorf("Can't delete charger with id: %s", chargerID)
		return err
	}
	return err
}

// ClearProducts removes products from Easee and saves file
func (e *Easee) ClearProducts() {
	e.Products = nil
	e.SaveProductsToFile()
}

// HasProducts returns true if easee contains products
func (e *Easee) HasProducts() bool {
	if len(e.Products) == 0 {
		return false
	}
	return true
}

// IsConfigured not used
func (e *Easee) IsConfigured() bool {
	if e.HasProducts() {
		return true
	}
	return false
}
