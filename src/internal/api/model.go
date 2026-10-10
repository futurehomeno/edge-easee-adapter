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

type maxCurrentBody struct {
	MaxChargerCurrent float64 `json:"maxChargerCurrent"`
}

type dynamicCurrentBody struct {
	DynamicChargerCurrent float64 `json:"dynamicChargerCurrent"`
}

type cableLockStateBody struct {
	State bool `json:"state"`
}

type phaseModeBody struct {
	PhaseMode int `json:"phaseMode"`
}
