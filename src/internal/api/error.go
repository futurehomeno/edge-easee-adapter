package api

import (
	"fmt"
	"io"
)

// HTTPError provides a way to pass more meaningful information regarding http errors without breaking interfaces.
type HTTPError struct {
	Err    error
	Status int
	Body   io.ReadCloser
}

func (e HTTPError) Error() string {
	body := ""

	if e.Body != nil {
		if bts, err := io.ReadAll(e.Body); err != nil {
			body = string(bts)
		}
	}

	return fmt.Sprintf("%s, status code: %d, body: %s", e.Err, e.Status, body)
}
