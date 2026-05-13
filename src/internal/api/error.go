package api

import (
	"fmt"
	"net/http"
)

// HTTPError provides a way to pass more meaningful information regarding http errors without breaking interfaces.
type HTTPError struct {
	Message    string
	StatusCode int
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("%s, status code: %d", e.Message, e.StatusCode)
}

// Is lets callers use errors.Is(httpErr, ErrUnauthorized) etc. so HTTPError values returned
// from mocked HTTP clients match the same sentinels production code wraps via fmt.Errorf("%w ...").
func (e HTTPError) Is(target error) bool {
	switch target {
	case ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized
	case ErrTimeout:
		return e.StatusCode == http.StatusBadGateway ||
			e.StatusCode == http.StatusServiceUnavailable ||
			e.StatusCode == http.StatusGatewayTimeout
	case ErrServer:
		return e.StatusCode == http.StatusInternalServerError
	}
	return false
}
