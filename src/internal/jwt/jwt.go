package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// ExpirationDate returns the expiration date (UTC) of the JWT token.
func ExpirationDate(jwtToken string) (time.Time, error) {
	var claims jwt.RegisteredClaims

	_, _, err := jwt.NewParser().ParseUnverified(jwtToken, &claims)
	if err != nil {
		return time.Time{}, err
	}

	if claims.ExpiresAt == nil {
		return time.Time{}, errors.New("no expiration date found in the token")
	}

	return claims.ExpiresAt.Time.UTC(), nil
}
