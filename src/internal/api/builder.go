package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type requestBuilder struct {
	method  string
	url     string
	body    interface{}
	headers map[string]string
}

func newRequestBuilder(method, url string) *requestBuilder {
	return &requestBuilder{
		method:  method,
		url:     url,
		headers: make(map[string]string),
	}
}

func (r *requestBuilder) withBody(body interface{}) *requestBuilder {
	r.body = body

	return r
}

func (r *requestBuilder) addHeader(key, value string) *requestBuilder {
	r.headers[key] = value

	return r
}

func (r *requestBuilder) build() (*http.Request, error) {
	var body io.Reader

	if r.body != nil {
		b, err := json.Marshal(r.body)
		if err != nil {
			return nil, err
		}

		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(r.method, r.url, body) //nolint:noctx
	if err != nil {
		return nil, err
	}

	for key, value := range r.headers {
		req.Header.Add(key, value)
	}

	return req, nil
}
