package jsonapitest

import (
	"encoding/json"
	"testing"

	"github.com/gonobo/jsonapi/v2"
	"github.com/stretchr/testify/assert"
)

func MarshalRaw(t *testing.T, value any) *json.RawMessage {
	raw, err := jsonapi.MarshalRaw(value)
	if err != nil {
		t.Fatalf("json value: %s", err)
		return nil
	}
	return raw
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

func AssertJSONAPIEq(t *testing.T, want string, got string, msgAndArgs ...any) bool {
	var wantDoc, gotDoc jsonapi.Document

	err := json.Unmarshal([]byte(want), &wantDoc)
	if fail := !assert.NoError(t, err, "want failed to unmarshal"); fail {
		return fail
	}

	err = json.Unmarshal([]byte(got), &gotDoc)
	if fail := !assert.NoError(t, err, "got failed to unmarshal"); fail {
		return fail
	}

	wantData, err := json.Marshal(wantDoc)
	if fail := !assert.NoError(t, err, "want failed to marshal"); fail {
		return fail
	}

	gotData, err := json.Marshal(gotDoc)
	if fail := !assert.NoError(t, err, "got failed to marshal"); fail {
		return fail
	}

	return assert.JSONEq(t, string(wantData), string(gotData), msgAndArgs...)
}
