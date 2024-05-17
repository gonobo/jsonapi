package jsonapi_test

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/stretchr/testify/assert"
)

type testcase[T any] struct {
	in            T
	wantJSON      string
	wantErr       bool
	skipUnmarshal bool
}

func runMarshalJSONTest[T any](t *testing.T, tc testcase[T]) {
	gotJSON, err := json.MarshalIndent(tc.in, "", "  ")
	if tc.wantErr && assert.Error(t, err) {
		return
	}
	assert.NoError(t, err)
	if !assert.JSONEq(t, tc.wantJSON, string(gotJSON)) {
		t.Log(string(gotJSON))
	}

	if tc.skipUnmarshal {
		return
	}

	t.Run("unmarshal", func(t *testing.T) {
		var got T
		err = json.Unmarshal(gotJSON, &got)
		assert.NoError(t, err)
		assert.EqualValues(t, tc.in, got)
	})
}

func TestNewSingleDocument(t *testing.T) {
	doc := jsonapi.NewSingleDocument(&jsonapi.Resource{ID: "1", Type: "item"})
	assert.EqualValues(t, &jsonapi.Document{
		Data: jsonapi.One{
			Value: &jsonapi.Resource{ID: "1", Type: "item"},
		},
	}, doc)
}

func TestNewMultiDocument(t *testing.T) {
	doc := jsonapi.NewMultiDocument(
		&jsonapi.Resource{ID: "1", Type: "item"},
		&jsonapi.Resource{ID: "2", Type: "item"},
	)
	assert.EqualValues(t, &jsonapi.Document{
		Data: jsonapi.Many{
			Value: []*jsonapi.Resource{
				{ID: "1", Type: "item"},
				{ID: "2", Type: "item"},
			},
		},
	}, doc)
}

func TestLinksMarshalJSON(t *testing.T) {
	t.Run("null link", func(t *testing.T) {
		runMarshalJSONTest(t, testcase[jsonapi.Links]{
			in:       jsonapi.Links{"self": nil},
			wantJSON: `{"self": null}`,
		})
	})

	t.Run("simple link", func(t *testing.T) {
		runMarshalJSONTest(t, testcase[jsonapi.Links]{
			in: jsonapi.Links{
				"self": &jsonapi.Link{Href: "/items/1"},
			},
			wantJSON: `{"self": "/items/1"}`,
		})
	})

	t.Run("complex link", func(t *testing.T) {
		runMarshalJSONTest(t, testcase[jsonapi.Links]{
			in: jsonapi.Links{
				"self": &jsonapi.Link{
					Href:     "/items/1",
					HrefLang: jsonapi.HrefLang{"en"},
				},
			},
			wantJSON: `{
				"self": {
					"href": "/items/1",
					"hreflang": "en"
				}
			}`,
		})
	})

	t.Run("complex link (many href lang)", func(t *testing.T) {
		runMarshalJSONTest(t, testcase[jsonapi.Links]{
			in: jsonapi.Links{
				"self": &jsonapi.Link{
					Href:     "/items/1",
					HrefLang: jsonapi.HrefLang{"en", "fr"},
				},
			},
			wantJSON: `{
				"self": {
					"href": "/items/1",
					"hreflang": ["en", "fr"]
				}
			}`,
		})
	})
}

func TestDocumentMarshalJSON(t *testing.T) {
	type tc = testcase[jsonapi.Document]
	t.Run("empty document", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in:            jsonapi.Document{},
			wantJSON:      `{"jsonapi": {"version": "1.1"}}`,
			skipUnmarshal: true,
		})
	})

	t.Run("document with meta", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Meta:    jsonapi.Meta{"foo": "bar"},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"meta": {"foo": "bar"}
			}`,
		})
	})

	t.Run("document with extensions", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Extensions: map[string]*json.RawMessage{
					"foo:version": MarshalRaw(t, "2"),
				},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"foo:version": "2"
			}`,
		})
	})

	t.Run("document with one primary data", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:   "1",
						Type: "items",
					},
				},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"data": {
					"type": "items",
					"id": "1"
				}
			}`,
		})
	})

	t.Run("document with one primary data and relationships", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:   "1",
						Type: "items",
						Relationships: jsonapi.RelationshipsNode{
							"linksonly": &jsonapi.Relationship{
								Links: jsonapi.Links{
									"self": &jsonapi.Link{Href: "http://api.foo.com/items/1"},
								},
							},
							"null": &jsonapi.Relationship{
								Data: jsonapi.One{},
							},
							"one": &jsonapi.Relationship{
								Data: jsonapi.One{
									Value: &jsonapi.Resource{
										ID:   "1",
										Type: "items",
									},
								},
							},
							"many": &jsonapi.Relationship{
								Data: jsonapi.Many{
									Value: []*jsonapi.Resource{
										{ID: "1", Type: "items"},
										{ID: "2", Type: "items"},
									},
								},
							},
						},
					},
				},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"data": {
					"type": "items",
					"id": "1",
					"relationships": {
						"linksonly": {
							"links": {
								"self": "http://api.foo.com/items/1"
							}
						},
						"null": { "data": null },
						"one": {
							"data": { "id": "1", "type": "items" }
						},
						"many": {
							"data": [
								{ "id": "1", "type": "items" },
								{ "id": "2", "type": "items" }
							]
						}
					}
				}
			}`,
		})
	})

	t.Run("document with null primary data", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Data:    jsonapi.One{},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"data": null
			}`,
		})
	})

	t.Run("document with many primary data", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Data: jsonapi.Many{
					Value: []*jsonapi.Resource{
						{ID: "1", Type: "items"},
					},
				},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"data": [{
					"type": "items",
					"id": "1"
				}]
			}`,
		})
	})

	t.Run("document with empty primary data", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Data:    jsonapi.Many{},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"data": []
			}`,
			skipUnmarshal: true,
		})
	})

	t.Run("document with inclusions", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Data: jsonapi.Many{
					Value: []*jsonapi.Resource{
						{ID: "1", Type: "items"},
					},
				},
				Included: []*jsonapi.Resource{
					{ID: "1", Type: "items"},
				},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"data": [{"id": "1", "type": "items"}],
				"included": [{"id": "1", "type": "items"}]
			}`,
			skipUnmarshal: true,
		})
	})

	t.Run("document with errors", func(t *testing.T) {
		runMarshalJSONTest(t, tc{
			in: jsonapi.Document{
				Jsonapi: jsonapi.JSONAPI{Version: "1.1"},
				Errors: []*jsonapi.Error{
					{
						Status: "500",
						Code:   "ERR",
						Title:  "Unknown Error",
						Detail: "An unknown error occurred.",
					},
				},
			},
			wantJSON: `{
				"jsonapi": {"version": "1.1"},
				"errors": [{
					"code": "ERR",
					"status": "500",
					"title": "Unknown Error",
					"detail": "An unknown error occurred."
				}]
			}`,
		})
	})
}

func TestErrorNode(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		node := jsonapi.Error{Title: "Bad Request", Detail: "bad request error"}
		var err error = &node
		assert.Equal(t, err.Error(), "Bad Request: bad request error")

		node.Title = ""
		assert.Equal(t, err.Error(), "bad request error")
	})
}

func TestDocumentJoinErrors(t *testing.T) {
	t.Run("with errors", func(t *testing.T) {
		document := jsonapi.Document{Errors: []*jsonapi.Error{
			{Detail: "error 1"},
			{Detail: "error 2"},
		}}
		assert.Error(t, document.Error())
	})

	t.Run("no errors", func(t *testing.T) {
		document := jsonapi.Document{Errors: []*jsonapi.Error{}}
		assert.NoError(t, document.Error())
	})
}

type comparator func(int, int, string) int

func (c comparator) Compare(i, j int, attribute string) int {
	return c(i, j, attribute)
}

func TestDocumentSort(t *testing.T) {
	t.Run("returns if there are no items to sort", func(t *testing.T) {
		doc := jsonapi.Document{}
		doc.Sort(comparator(func(i1, i2 int, s string) int { return 0 }), []query.Sort{})
	})

	t.Run("returns if document does not have multiple primary data", func(t *testing.T) {
		item := &jsonapi.Resource{ID: "1", Type: "item"}
		doc := jsonapi.Document{Data: jsonapi.One{Value: item}}
		doc.Sort(comparator(func(i1, i2 int, s string) int { return 0 }), []query.Sort{})
	})

	t.Run("multiple items", func(t *testing.T) {
		items := []*jsonapi.Resource{
			{ID: "1", Attributes: map[string]any{"foo": "1", "bar": "3"}},
			{ID: "2", Attributes: map[string]any{"foo": "1", "bar": "2"}},
			{ID: "3", Attributes: map[string]any{"foo": "3", "bar": "1"}},
		}

		criterion := []query.Sort{
			{Property: "foo", Descending: true},
			{Property: "bar"},
		}

		doc := jsonapi.Document{Data: jsonapi.Many{Value: items}}

		doc.Sort(comparator(func(i1, i2 int, attribute string) int {
			a, b := items[i1], items[i2]
			v1, v2 := a.Attributes[attribute].(string), b.Attributes[attribute].(string)
			return strings.Compare(v1, v2)
		}), criterion)

		assert.Equal(t, "3", items[0].ID)
		assert.Equal(t, "2", items[1].ID)
		assert.Equal(t, "1", items[2].ID)

		t.Run("returns with no criteria", func(t *testing.T) {
			doc.Sort(comparator(func(i1, i2 int, s string) int { return 0 }), nil)
		})
	})
}

func TestParse(t *testing.T) {
	type testcase struct {
		name    string
		want    jsonapi.Document
		wantErr bool
	}

	for _, tc := range []testcase{
		{
			name: "marshal-single",
			want: jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						Attributes: map[string]any{"value": "some-value"},
						ID:         "42",
						Type:       "nodes",
						Links: jsonapi.Links{
							"self": {Href: "/nodes/42"},
						},
						Relationships: jsonapi.RelationshipsNode{
							"links-only": {
								Links: jsonapi.Links{
									"self":    {Href: "/nodes/42/relationships/links-only"},
									"related": {Href: "/nodes/42/links-only"},
								},
								Meta: jsonapi.Meta{
									"links-only": map[string]any{"foo": "bar"},
								},
							},
							"many": {
								Data: jsonapi.Many{Value: []*jsonapi.Resource{}},
								Links: jsonapi.Links{
									"self":    {Href: "/nodes/42/relationships/many"},
									"related": {Href: "/nodes/42/many"},
								},
								Meta: jsonapi.Meta{
									"many": map[string]any{"foo": "bar"},
								},
							},
							"one": {
								Data: jsonapi.One{},
								Links: jsonapi.Links{
									"self":    {Href: "/nodes/42/relationships/one"},
									"related": {Href: "/nodes/42/one"},
								},
								Meta: jsonapi.Meta{
									"one": map[string]any{"foo": "bar"},
								},
							},
						},
						Meta: jsonapi.Meta{
							"root": map[string]any{"foo": "bar"},
						},
					},
				},
			},
			wantErr: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.Open(fmt.Sprintf("testdata/%s.json", tc.name))
			if err != nil {
				t.Fatal(err)
				return
			}
			defer file.Close()
			doc := &jsonapi.Document{}
			err = jsonapi.Decode(file, doc)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NotNil(t, doc)
			assert.EqualValues(t, &tc.want, doc)
		})
	}
}

func TestWrite(t *testing.T) {
	t.Run("writes to response writer", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		doc := jsonapi.Document{Meta: map[string]any{"foo": "bar"}}
		err := jsonapi.Encode(recorder, &doc)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"jsonapi":{"version": "1.1"}, "meta":{"foo":"bar"}}`, string(recorder.Body.String()))
	})
}
