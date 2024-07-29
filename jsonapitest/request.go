package jsonapitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gonobo/jsonapi"
)

// NewRequest creates a new http.Request using a jsonapi.Document as payload,
// suitable for passing to an [http.Handler] for testing.
//
// Deprecated: Use httptest.NewRequest() directly, with jsonapitest.Body as a payload.
func NewRequest(method string, target string, doc jsonapi.Document) *http.Request {
	return httptest.NewRequest(method, target, Body(doc))
}

// Body is a wrapper around jsonapi.Document that implements the io.Reader
// interface. Use as a payload in httptest.NewRequest() calls.
type Body jsonapi.Document

// Read implements the io.Reader interface.
func (p Body) Read(b []byte) (int, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}

	return copy(b, data), nil
}
