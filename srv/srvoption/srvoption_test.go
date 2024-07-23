package srvoption_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/srv"
	"github.com/gonobo/jsonapi/srv/srvoption"
	"github.com/stretchr/testify/assert"
)

type writeOptionTestCase struct {
	name          string
	options       []srv.WriteOptions
	doc           jsonapi.Document
	wantDocument  jsonapi.Document
	wantStatusErr int
}

func (tc writeOptionTestCase) Run(t *testing.T) {
	t.Run(tc.name, func(t *testing.T) {
		recorder := httptest.NewRecorder()
		srv.Write(recorder, &tc.doc, tc.options...)
		if tc.wantStatusErr > 0 {
			assert.Equal(t, recorder.Result().StatusCode, tc.wantStatusErr)
			return
		}
		got := jsonapi.Document{}
		err := json.NewDecoder(recorder.Result().Body).Decode(&got)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		assert.EqualValues(t, tc.wantDocument, got)
	})
}

func TestMeta(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a meta string value",
			options: []srv.WriteOptions{
				srvoption.Meta("mystring", "string_value"),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Meta:    jsonapi.Meta{"mystring": "string_value"},
			},
		},
	} {
		tc.Run(t)
	}
}

func TestLink(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a link",
			options: []srv.WriteOptions{
				srvoption.Link("test", "http://www.example.com/things/1"),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Links: jsonapi.Links{
					"test": &jsonapi.Link{Href: "http://www.example.com/things/1"},
				},
			},
		},
	} {
		tc.Run(t)
	}
}
