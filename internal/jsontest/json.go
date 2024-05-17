package jsontest

import (
	"bytes"
	"encoding/json"
)

// IsJSONObject returns true if the serialized JSON data is an object type.
func IsJSONObject(data *json.RawMessage) bool {
	// Get slice of data with optional leading whitespace removed.
	// See RFC 7159, Section 2 for the definition of JSON whitespace.
	x := bytes.TrimLeft(*data, " \t\r\n")
	return bytes.HasPrefix(x, []byte("{"))
}

// IsJSONArray returns true if the serialized JSON data is an array type.
func IsJSONArray(data *json.RawMessage) bool {
	// Get slice of data with optional leading whitespace removed.
	// See RFC 7159, Section 2 for the definition of JSON whitespace.
	x := bytes.TrimLeft(*data, " \t\r\n")
	return bytes.HasPrefix(x, []byte("["))
}

// IsJSONNull returns true if the serialized JSON data is a null type.
func IsJSONNull(data *json.RawMessage) bool {
	x := bytes.TrimLeft(*data, " \t\r\n")

	if len(x) < 4 {
		return false
	}

	token := x[0:3]
	return string(token) == "null"
}
