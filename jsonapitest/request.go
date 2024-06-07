package jsonapitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gonobo/jsonapi"
)

// NewRequest creates a new http.Request using a jsonapi.Document as payload,
// suitable for passing to an [http.Handler] for testing.
func NewRequest(method string, target string, doc jsonapi.Document) *http.Request {
	return httptest.NewRequest(method, target, requestPayload(doc))
}

type requestPayload jsonapi.Document

// Read implements the io.Reader interface.
func (p requestPayload) Read(b []byte) (int, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}

	copy(b, data)
	return len(data), nil
}
