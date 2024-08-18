package server_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gonobo/jsonapi/v1"
	"github.com/gonobo/jsonapi/v1/query"
	"github.com/gonobo/jsonapi/v1/server"
	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	type testcase struct {
		name       string
		data       any
		status     int
		options    []server.WriteOptions
		wantStatus int
		wantBody   string
	}

	for _, tc := range []testcase{
		{
			name:       "empty payload",
			status:     http.StatusNoContent,
			data:       nil,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "jsonapi marshaler fails",
			status:     http.StatusOK,
			data:       struct{}{},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "json marshaler fails",
			status: http.StatusOK,
			data:   &jsonapi.Document{},
			options: []server.WriteOptions{
				server.WithJSONMarshaler(func(a any) ([]byte, error) { return nil, errors.New("error") }),
			},
			wantStatus: http.StatusInternalServerError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.Write(recorder, tc.data, tc.status, tc.options...)
			assert.Equal(t, tc.wantStatus, recorder.Result().StatusCode)
			if tc.wantBody != "" {
				data, err := io.ReadAll(recorder.Result().Body)
				if err != nil {
					t.Fatal(err)
					return
				}
				gotBody := string(data)
				assert.JSONEq(t, tc.wantBody, gotBody, "got: %s", gotBody)
			}
		})
	}
}

func TestError(t *testing.T) {
	type testcase struct {
		name       string
		err        error
		status     int
		options    []server.WriteOptions
		wantStatus int
	}

	for _, tc := range []testcase{
		{
			name:       "write any error type",
			err:        errors.New("an error"),
			status:     http.StatusBadGateway,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "passthrough jsonapi errors",
			err:        jsonapi.NewError(errors.New("error detail"), "error title"),
			status:     http.StatusInternalServerError,
			wantStatus: http.StatusInternalServerError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.Error(recorder, tc.err, tc.status, tc.options...)
			assert.Equal(t, tc.wantStatus, recorder.Result().StatusCode)
		})
	}
}

func TestResponseRecorder(t *testing.T) {
	t.Run("flushes to response writer", func(t *testing.T) {
		w := httptest.NewRecorder()
		rr := server.NewRecorder()
		rr.Status = http.StatusCreated
		rr.Document = jsonapi.NewSingleDocument(&jsonapi.Resource{ID: "42", Type: "things"})
		rr.Header().Add("Content-Type", "application/json")
		rr.Flush(w)

		assert.Equal(t, http.StatusCreated, w.Result().StatusCode)
		assert.Equal(t, "application/json", w.Result().Header.Get("Content-Type"))

		data, err := io.ReadAll(w.Result().Body)
		if err != nil {
			t.Fatal(err)
			return
		}

		got := string(data)
		assert.JSONEq(t, `{"data": {"id": "42", "type": "things"}, "jsonapi": {"version": "1.1"}}`, got)
	})

	t.Run("panics if document cannot be marshaled", func(t *testing.T) {
		w := httptest.NewRecorder()
		rr := server.NewRecorder()
		rr.Status = http.StatusCreated
		rr.Document = jsonapi.NewSingleDocument(&jsonapi.Resource{ID: "42", Type: "things"})
		rr.Header().Add("Content-Type", "application/json")
		rr.JSONMarshal = func(a any) ([]byte, error) { return nil, errors.New("error") }

		assert.Panics(t, func() { rr.Flush(w) })
	})
}

type writeOptionTestCase struct {
	name          string
	options       []server.WriteOptions
	doc           jsonapi.Document
	wantDocument  jsonapi.Document
	wantJSON      string
	wantStatusErr int
}

func (tc writeOptionTestCase) run(t *testing.T) {
	t.Run(tc.name, func(t *testing.T) {
		recorder := httptest.NewRecorder()
		server.Write(recorder, &tc.doc, http.StatusOK, tc.options...)
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

func TestWriteMeta(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a meta string value",
			options: []server.WriteOptions{
				server.WriteMeta("mystring", "string_value"),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Meta:    jsonapi.Meta{"mystring": "string_value"},
			},
		},
	} {
		tc.run(t)
	}
}

func TestWriteLink(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a link",
			options: []server.WriteOptions{
				server.WriteLink("test", "http://www.example.com/things/1"),
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
			options: []server.WriteOptions{
				server.WriteLink("test", "this is not a url"),
			},
			wantStatusErr: http.StatusInternalServerError,
		},
	} {
		tc.run(t)
	}
}

func TestWriteSelfLink(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a self link",
			options: []server.WriteOptions{
				server.WriteSelfLink(httptest.NewRequest("GET", "http://www.example.com/things/1", nil)),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Links: jsonapi.Links{
					"self": &jsonapi.Link{Href: "http://www.example.com/things/1"},
				},
			},
		},
	} {
		tc.run(t)
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

func TestWriteNavigationLinks(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a next page link",
			options: []server.WriteOptions{
				server.WriteNavigationLinks(
					httptest.NewRequest("GET", "http://www.example.com/things/1", nil),
					map[string]query.Page{
						"next": {
							Cursor: "page-cursor",
							Limit:  100,
						},
						"prev": {
							PageNumber: 5,
							Limit:      25,
						},
					},
				),
			},
			wantDocument: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Links: jsonapi.Links{
					"next": &jsonapi.Link{
						Href: decodedURL("http://www.example.com/things/1?page[cursor]=page-cursor&page[limit]=100").String(),
					},
					"prev": &jsonapi.Link{
						Href: decodedURL("http://www.example.com/things/1?page[number]=5&page[limit]=25").String(),
					},
				},
			},
		},
	} {
		tc.run(t)
	}
}

func TestWriteResourceLinks(t *testing.T) {
	for _, tc := range []writeOptionTestCase{
		{
			name: "adds a next page link",
			options: []server.WriteOptions{
				server.WriteResourceLinks("http://www.example.com/api", jsonapi.DefaultURLResolver()),
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
		tc.run(t)
	}
}

func TestWriteSortedPrimaryData(t *testing.T) {
	type thing struct {
		ID    string `jsonapi:"primary,things"`
		Value string `jsonapi:"attr,value"`
	}

	things := []thing{
		{ID: "1", Value: "foo"},
		{ID: "2", Value: "bar"},
		{ID: "3", Value: "quux"},
	}

	cmp := jsonapi.ResourceComparator(things, map[string]jsonapi.Comparer[thing]{
		"value": func(a, b thing) int { return strings.Compare(a.Value, b.Value) },
	})

	recorder := server.NewRecorder()

	server.Write(
		recorder,
		things,
		http.StatusOK,
		server.WriteSortedPrimaryData(cmp, []query.Sort{{Property: "value"}}))

	data := recorder.Document.Data.Items()
	assert.Equal(t, "2", data[0].ID)
	assert.Equal(t, "1", data[1].ID)
	assert.Equal(t, "3", data[2].ID)
}

func TestWriteLocationHeader(t *testing.T) {
	type thing struct {
		ID    string `jsonapi:"primary,things"`
		Value string `jsonapi:"attr,value"`
	}

	w := server.NewRecorder()
	server.Write(
		w,
		thing{"42", "foo"},
		http.StatusCreated,
		server.WriteLocationHeader("https://www.example.com/api", jsonapi.DefaultURLResolver()),
	)

	location := w.Header().Get("Location")
	assert.Equal(t, "https://www.example.com/api/things/42", location)
}
