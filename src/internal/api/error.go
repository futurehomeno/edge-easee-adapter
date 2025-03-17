package api

import (
	"fmt"
)

// HTTPError provides a way to pass more meaningful information regarding http errors without breaking interfaces.
type HTTPError struct {
	Message    string
	StatusCode int
}

func (e HTTPError) Error() string {
	return fmt.Sprintf("%s, status code: %d", e.Message, e.StatusCode)
}
