package option_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/srv"
	"github.com/gonobo/jsonapi/srv/option"
	"github.com/stretchr/testify/assert"
)

type writeOptionTestCase struct {
	name          string
	options       []srv.WriteOptions
	doc           jsonapi.Document
	wantDocument  jsonapi.Document
	wantJSON      string
	wantStatusErr int
}

func (tc writeOptionTestCase) Run(t *testing.T) {
	t.Run(tc.name, func(t *testing.T) {
		recorder := httptest.NewRecorder()
		srv.Write(recorder, &tc.doc, tc.options...)
		if tc.wantStatusErr > 0 {
			assert.Equal(t, tc.wantStatusErr, recorder.Result().StatusCode)
			return
		}
		got := jsonapi.Document{}
		err := json.NewDecoder(recorder.Result().Body).Decode(&got)
		if err != nil {
			t.Errorf("unexpected error: %s", err)
			return
		}
		if tc.wantJSON != "" {
			data, err := json.Marshal(got)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}
			assert.JSONEq(t, tc.wantJSON, string(data), "got json: %s", string(data))
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
				option.Meta("mystring", "string_value"),
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
				option.Link("test", "http://www.example.com/things/1"),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Links: jsonapi.Links{
					"test": &jsonapi.Link{Href: "http://www.example.com/things/1"},
				},
			},
		},
		{
			name: "fails with invalid link",
			options: []srv.WriteOptions{
				option.Link("test", "this is not a url"),
			},
			wantStatusErr: http.StatusInternalServerError,
		},
	} {
		tc.Run(t)
	}
}

func TestSelfLink(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a self link",
			options: []srv.WriteOptions{
				option.SelfLink(httptest.NewRequest("GET", "http://www.example.com/things/1", nil)),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Links: jsonapi.Links{
					"self": &jsonapi.Link{Href: "http://www.example.com/things/1"},
				},
			},
		},
	} {
		tc.Run(t)
	}
}

type decodedURL string

func (d decodedURL) String() string {
	value, err := url.ParseRequestURI(string(d))
	if err != nil {
		panic(err)
	}
	value.RawQuery = value.Query().Encode()
	return value.String()
}

func TestNextPageCursorLink(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a next page link",
			options: []srv.WriteOptions{
				option.NextPageCursorLink(
					httptest.NewRequest("GET", "http://www.example.com/things/1", nil),
					"page-cursor",
					100,
				),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Links: jsonapi.Links{
					"next": &jsonapi.Link{
						Href: decodedURL("http://www.example.com/things/1?page[cursor]=page-cursor&page[limit]=100").String(),
					},
				},
			},
		},
	} {
		tc.Run(t)
	}
}

func TestResourceLinks(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a next page link",
			options: []srv.WriteOptions{
				option.ResourceLinks("http://www.example.com/api", jsonapi.DefaultURLResolver()),
			},
			doc: *jsonapi.NewSingleDocument(&jsonapi.Resource{
				Type: "things",
				ID:   "1",
				Relationships: jsonapi.RelationshipsNode{
					"parent": &jsonapi.Relationship{},
				},
			}),
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"data": {
					"type": "things",
					"id": "1",
					"links": {
						"self": "http://www.example.com/api/things/1"
					},
					"relationships": {
						"parent": {
							"links": {
								"related": "http://www.example.com/api/things/1/parent",
								"self": "http://www.example.com/api/things/1/relationships/parent"
							}
						}
					}
				}
			}`,
		},
	} {
		tc.Run(t)
	}
}
