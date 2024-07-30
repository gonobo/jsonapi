package jsonapitest

import (
	"encoding/json"
	"testing"

	"github.com/gonobo/jsonapi"
)

func MarshalRaw(t *testing.T, value any) *json.RawMessage {
	raw, err := jsonapi.MarshalRaw(value)
	if err != nil {
		t.Fatalf("json value: %s", err)
		return nil
	}
	return raw
}
