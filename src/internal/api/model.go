package api

// loginBody represents a login request body.
type loginBody struct {
	Username string `json:"userName"`
	Password string `json:"password"`
}

// refreshBody represents a token refresh request body.
type refreshBody struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// The following struct is currently commented out because it is not needed for the current functionality.
// However, it may be useful in implementing SetCableAlwaysLock method.
// cableLockBody represents a cable lock request body.
// type cableLockBody struct {
// 	State bool `json:"state"`
// }

// maxCurrentBody represents a charger max current request body.
type maxCurrentBody struct {
	MaxChargerCurrent float64 `json:"maxChargerCurrent"`
}

// dynamicCurrentBody represents a charger dynamic current request body.
type dynamicCurrentBody struct {
	DynamicChargerCurrent float64 `json:"dynamicChargerCurrent"`
}
