package jsonapitest

import (
	"encoding/json"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/stretchr/testify/assert"
)

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
