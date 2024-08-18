package jsonapi_test

import (
	"encoding/json"
	"testing"

	"github.com/gonobo/jsonapi/v1"
	"github.com/gonobo/jsonapi/v1/jsonapitest"
	"github.com/stretchr/testify/assert"
)

type typedef string

type SimpleItem struct {
	ID     string  `jsonapi:"primary,items"`
	Value1 string  `jsonapi:"attr,value1"`
	Value2 string  `jsonapi:"attr,value2,omitempty"`
	Value3 typedef `jsonapi:"attr,value3,omitempty"`
	Omit   any
}

func (SimpleItem) MarshalLinksJSONAPI() jsonapi.Links {
	return nil
}

func (SimpleItem) MarshalMetaJSONAPI() jsonapi.Meta {
	return nil
}

func (*SimpleItem) UnmarshalLinksJSONAPI(node jsonapi.Links) {}

func (*SimpleItem) UnmarshalMetaJSONAPI(node jsonapi.Meta) {}

type ComplexItem struct {
	ID         string        `jsonapi:"primary,items"`
	Value      any           `jsonapi:"attr,value,omitempty"`
	SimpleOmit *SimpleItem   `jsonapi:"attr,simpleomit,omitempty"`
	Simple     *SimpleItem   `jsonapi:"attr,simple"`
	Obj        SimpleItem    `jsonapi:"attr,obj,omitempty"`
	Arr        []*SimpleItem `jsonapi:"attr,arr,omitempty"`
	Arr2       []SimpleItem  `jsonapi:"attr,arr2,omitempty"`
	Omit       any
}

type RelatedItem struct {
	ID       string        `jsonapi:"primary,items"`
	One      *SimpleItem   `jsonapi:"relation,one"`
	Many     []*SimpleItem `jsonapi:"relation,many"`
	Nullable *SimpleItem   `jsonapi:"relation,nullable,omitempty"`
	ManyObj  []SimpleItem  `jsonapi:"relation,manyobj,omitempty"`
}

func (RelatedItem) MarshalRelatedMetaJSONAPI(name string) jsonapi.Meta {
	return nil
}

func (RelatedItem) MarshalRelatedLinksJSONAPI(name string) jsonapi.Links {
	return nil
}

func (*RelatedItem) UnmarshalRelatedMetaJSONAPI(name string, meta jsonapi.Meta) {}

func (*RelatedItem) UnmarshalRelatedLinksJSONAPI(name string, links jsonapi.Links) {}

type ExtensionItem struct {
	ID          string  `jsonapi:"primary,items"`
	Version     string  `jsonapi:"ext,version,foo"`
	VersionOmit string  `jsonapi:"ext,versionomit,foo,omitempty"`
	VersionNil  *string `jsonapi:"ext,versionnil,foo"`
}

type MarshalOverrideItem struct {
	ID    string
	Value string
	Item  *MarshalOverrideItem `jsonapi:"relation,item"`
}

func (m MarshalOverrideItem) MarshalJSONAPI() (*jsonapi.Resource, error) {
	return &jsonapi.Resource{
		ID:   m.ID,
		Type: "override-items",
		Attributes: map[string]any{
			"value": m.Value,
		},
	}, nil
}

func (m *MarshalOverrideItem) UnmarshalJSONAPI(node *jsonapi.Resource) error {
	m.ID = node.ID
	m.Value = node.Attributes["value"].(string)
	return nil
}

type InvalidID struct {
	ID string `jsonapi:"primary"`
}

type AnyID struct {
	ID any `jsonapi:"primary,items"`
}

func (AnyID) String() string { return "anyid" }

func TestMarshalResource(t *testing.T) {
	type testcase struct {
		in      any
		want    jsonapi.Resource
		wantErr bool
	}

	run := func(t *testing.T, tc testcase) {
		got, err := jsonapi.MarshalResource(tc.in)
		if tc.wantErr && assert.Error(t, err) {
			return
		}
		assert.EqualValues(t, &tc.want, got)
	}

	t.Run("marshaler interface", func(t *testing.T) {
		run(t, testcase{
			in: MarshalOverrideItem{ID: "1", Value: "foo"},
			want: jsonapi.Resource{
				ID:         "1",
				Type:       "override-items",
				Attributes: map[string]any{"value": "foo"},
			},
		})
	})

	t.Run("extensions", func(t *testing.T) {
		run(t, testcase{
			in: ExtensionItem{ID: "1", Version: "2"},
			want: jsonapi.Resource{
				ID:            "1",
				Type:          "items",
				Attributes:    map[string]any{},
				Relationships: map[string]*jsonapi.Relationship{},
				Extensions: map[string]*json.RawMessage{
					"foo:version":    jsonapitest.MarshalRaw(t, "2"),
					"foo:versionnil": nil,
				},
			},
		})
	})

	t.Run("relationships", func(t *testing.T) {
		run(t, testcase{
			in: RelatedItem{
				ID:   "1",
				One:  &SimpleItem{ID: "2"},
				Many: []*SimpleItem{{ID: "3"}},
			},
			want: jsonapi.Resource{
				ID:         "1",
				Type:       "items",
				Attributes: map[string]any{},
				Extensions: map[string]*json.RawMessage{},
				Relationships: map[string]*jsonapi.Relationship{
					"one": {
						Data: jsonapi.One{
							Value: &jsonapi.Resource{
								ID:   "2",
								Type: "items",
							},
						},
					},
					"many": {
						Data: jsonapi.Many{
							Value: []*jsonapi.Resource{
								{ID: "3", Type: "items"},
							},
						},
					},
					"nullable": {},
					"manyobj":  {},
				},
			},
		})
	})

	t.Run("relationships with empty refs", func(t *testing.T) {
		run(t, testcase{
			in: RelatedItem{
				ID:   "1",
				Many: []*SimpleItem{{ID: "3"}},
			},
			want: jsonapi.Resource{
				ID:         "1",
				Type:       "items",
				Attributes: map[string]any{},
				Extensions: map[string]*json.RawMessage{},
				Relationships: map[string]*jsonapi.Relationship{
					"one": {
						Data: jsonapi.One{},
					},
					"many": {
						Data: jsonapi.Many{
							Value: []*jsonapi.Resource{
								{ID: "3", Type: "items"},
							},
						},
					},
					"nullable": {},
					"manyobj":  {},
				},
			},
		})
	})

	t.Run("failed relationships", func(t *testing.T) {
		run(t, testcase{
			in: RelatedItem{
				ID:   "1",
				One:  &SimpleItem{ID: "2"},
				Many: []*SimpleItem{{ID: "3"}, nil},
			},
			wantErr: true,
		})
	})

	t.Run("stringer primary tag", func(t *testing.T) {
		run(t, testcase{
			in: AnyID{ID: AnyID{}},
			want: jsonapi.Resource{
				ID:            "anyid",
				Type:          "items",
				Attributes:    map[string]any{},
				Relationships: map[string]*jsonapi.Relationship{},
				Extensions:    map[string]*json.RawMessage{},
			},
		})
	})

	t.Run("stringer primary tag", func(t *testing.T) {
		run(t, testcase{
			in: AnyID{ID: &AnyID{}},
			want: jsonapi.Resource{
				ID:            "anyid",
				Type:          "items",
				Attributes:    map[string]any{},
				Relationships: map[string]*jsonapi.Relationship{},
				Extensions:    map[string]*json.RawMessage{},
			},
		})
	})

	t.Run("nil primary tag", func(t *testing.T) {
		run(t, testcase{
			in: AnyID{},
			want: jsonapi.Resource{
				ID:            "<nil>",
				Type:          "items",
				Attributes:    map[string]any{},
				Relationships: map[string]*jsonapi.Relationship{},
				Extensions:    map[string]*json.RawMessage{},
			},
		})
	})

	t.Run("numeric primary tag", func(t *testing.T) {
		run(t, testcase{
			in: AnyID{ID: 1},
			want: jsonapi.Resource{
				ID:            "1",
				Type:          "items",
				Attributes:    map[string]any{},
				Relationships: map[string]*jsonapi.Relationship{},
				Extensions:    map[string]*json.RawMessage{},
			},
		})
	})

	t.Run("invalid primary tag", func(t *testing.T) {
		run(t, testcase{
			in:      InvalidID{ID: "1"},
			wantErr: true,
		})
	})

	t.Run("simple attributes", func(t *testing.T) {
		run(t, testcase{
			in: SimpleItem{ID: "1", Value1: "foo", Value2: "bar"},
			want: jsonapi.Resource{
				ID:            "1",
				Type:          "items",
				Relationships: map[string]*jsonapi.Relationship{},
				Attributes: map[string]any{
					"value1": "foo",
					"value2": "bar",
				},
				Extensions: map[string]*json.RawMessage{},
			},
		})
	})

	t.Run("pointer to simple attributes", func(t *testing.T) {
		run(t, testcase{
			in: &SimpleItem{ID: "1", Value1: "foo", Value2: "bar"},
			want: jsonapi.Resource{
				ID:            "1",
				Type:          "items",
				Relationships: map[string]*jsonapi.Relationship{},
				Attributes: map[string]any{
					"value1": "foo",
					"value2": "bar",
				},
				Extensions: map[string]*json.RawMessage{},
			},
		})
	})

	t.Run("complex attribute", func(t *testing.T) {
		run(t, testcase{
			in: &ComplexItem{
				ID:    "1",
				Value: map[string]any{"foo": "bar"},
			},
			want: jsonapi.Resource{
				ID:            "1",
				Type:          "items",
				Relationships: map[string]*jsonapi.Relationship{},
				Attributes: map[string]any{
					"value":  map[string]any{"foo": "bar"},
					"simple": nil,
				},
				Extensions: map[string]*json.RawMessage{},
			},
		})
	})

	t.Run("omit if attribute empty", func(t *testing.T) {
		run(t, testcase{
			in: &ComplexItem{ID: "1"},
			want: jsonapi.Resource{
				ID:            "1",
				Type:          "items",
				Relationships: map[string]*jsonapi.Relationship{},
				Attributes:    map[string]any{"simple": nil},
				Extensions:    map[string]*json.RawMessage{},
			},
		})
	})
}

func TestUnmarshal(t *testing.T) {
	t.Run("error when document primary data is null", func(t *testing.T) {
		in := jsonapi.NewSingleDocument(nil)
		var out SimpleItem
		err := jsonapi.Unmarshal(in, &out)
		assert.Error(t, err)
	})

	t.Run("error when document fields are zero value", func(t *testing.T) {
		in := jsonapi.Document{}
		var out SimpleItem
		err := jsonapi.Unmarshal(&in, &out)
		assert.Error(t, err)
	})

	t.Run("error when passing in a non-pointer", func(t *testing.T) {
		in := jsonapi.NewSingleDocument(&jsonapi.Resource{
			ID:         "1",
			Type:       "items",
			Attributes: map[string]any{"value": "foo"},
			Links:      jsonapi.Links{},
			Meta:       jsonapi.Meta{},
		})
		var out SimpleItem
		err := jsonapi.Unmarshal(in, out)
		assert.Error(t, err)
	})

	t.Run("single struct", func(t *testing.T) {
		in := jsonapi.NewSingleDocument(&jsonapi.Resource{
			ID:         "1",
			Type:       "override-items",
			Attributes: map[string]any{"value": "foo"},
			Links:      jsonapi.Links{},
			Meta:       jsonapi.Meta{},
			Relationships: jsonapi.RelationshipsNode{
				"item": &jsonapi.Relationship{
					Links: jsonapi.Links{},
					Meta:  jsonapi.Meta{},
				},
			},
		})

		out := MarshalOverrideItem{}
		err := jsonapi.Unmarshal(in, &out)
		assert.NoError(t, err)

		want := MarshalOverrideItem{ID: "1", Value: "foo"}
		assert.EqualValues(t, want, out)
	})

	t.Run("slice of structs", func(t *testing.T) {
		in := jsonapi.NewMultiDocument(&jsonapi.Resource{
			ID:         "1",
			Type:       "override-items",
			Attributes: map[string]any{"value": "foo"},
			Links:      jsonapi.Links{},
			Meta:       jsonapi.Meta{},
			Relationships: jsonapi.RelationshipsNode{
				"item": &jsonapi.Relationship{
					Links: jsonapi.Links{},
					Meta:  jsonapi.Meta{},
				},
			},
		})

		out := []MarshalOverrideItem{}
		err := jsonapi.Unmarshal(in, &out)
		assert.NoError(t, err)

		want := []MarshalOverrideItem{{ID: "1", Value: "foo"}}
		assert.ElementsMatch(t, want, out)
	})
}

func TestUnmarshalResource(t *testing.T) {
	t.Run("identity", func(t *testing.T) {
		in := jsonapi.Resource{
			ID:         "1",
			Type:       "items",
			Attributes: map[string]any{"value1": "foo", "value2": "bar"},
			Links:      jsonapi.Links{},
			Meta:       jsonapi.Meta{},
		}
		out := jsonapi.Resource{}
		err := jsonapi.UnmarshalResource(&in, &out)
		assert.NoError(t, err)
		assert.EqualValues(t, in, out)
	})

	t.Run("nil value", func(t *testing.T) {
		in := jsonapi.Resource{ID: "1", Type: "items"}
		var out *SimpleItem = nil
		err := jsonapi.UnmarshalResource(&in, out)
		assert.Error(t, err)
	})

	t.Run("unmarshal interface", func(t *testing.T) {
		in := jsonapi.Resource{
			ID:         "1",
			Type:       "override-items",
			Attributes: map[string]any{"value": "foo"},
			Links:      jsonapi.Links{},
			Meta:       jsonapi.Meta{},
			Relationships: jsonapi.RelationshipsNode{
				"item": &jsonapi.Relationship{
					Links: jsonapi.Links{},
					Meta:  jsonapi.Meta{},
				},
			},
		}

		out := MarshalOverrideItem{}
		err := jsonapi.UnmarshalResource(&in, &out)
		assert.NoError(t, err)

		want := MarshalOverrideItem{ID: "1", Value: "foo"}
		assert.EqualValues(t, want, out)
	})

	t.Run("extensions", func(t *testing.T) {
		in := jsonapi.Resource{
			ID:            "1",
			Type:          "items",
			Attributes:    map[string]any{},
			Relationships: map[string]*jsonapi.Relationship{},
			Extensions: map[string]*json.RawMessage{
				"foo:version":    jsonapitest.MarshalRaw(t, "2"),
				"foo:versionnil": nil,
			},
		}
		out := ExtensionItem{}
		err := jsonapi.UnmarshalResource(&in, &out)
		assert.NoError(t, err)

		want := ExtensionItem{ID: "1", Version: "2"}
		assert.EqualValues(t, want, out)
	})

	t.Run("simple item", func(t *testing.T) {
		in := jsonapi.Resource{
			ID:   "1",
			Type: "items",
			Attributes: map[string]any{
				"value1": "foo",
				"value3": "bar",
			},
			Links: jsonapi.Links{},
			Meta:  jsonapi.Meta{},
		}
		out := SimpleItem{}
		err := jsonapi.UnmarshalResource(&in, &out)
		assert.NoError(t, err)

		want := SimpleItem{ID: "1", Value1: "foo", Value3: typedef("bar")}
		assert.EqualValues(t, want, out)
	})

	t.Run("item with invalid type", func(t *testing.T) {
		in := jsonapi.Resource{
			ID:   "1",
			Type: "orders",
		}
		out := SimpleItem{}
		err := jsonapi.UnmarshalResource(&in, &out)
		assert.Error(t, err)
	})

	t.Run("complex item", func(t *testing.T) {
		in := jsonapi.Resource{
			ID:   "1",
			Type: "items",
			Attributes: map[string]any{
				"simple": map[string]any{"value1": "foo"},
				"obj":    map[string]any{"value1": "bar"},
				"arr":    []map[string]any{{"value1": "baz"}},
				"arr2":   []map[string]any{{"value1": "duz"}},
			},
		}
		out := ComplexItem{}
		err := jsonapi.UnmarshalResource(&in, &out)
		assert.NoError(t, err)

		want := ComplexItem{
			ID:     "1",
			Simple: &SimpleItem{Value1: "foo"},
			Obj:    SimpleItem{Value1: "bar"},
			Arr:    []*SimpleItem{{Value1: "baz"}},
			Arr2:   []SimpleItem{{Value1: "duz"}},
		}
		assert.EqualValues(t, want, out)
	})

	t.Run("item with relationships", func(t *testing.T) {
		in := jsonapi.Resource{
			ID:   "1",
			Type: "items",
			Relationships: map[string]*jsonapi.Relationship{
				"one": {
					Links: jsonapi.Links{},
					Meta:  jsonapi.Meta{},
					Data: jsonapi.One{
						Value: &jsonapi.Resource{ID: "2", Type: "items"},
					},
				},
				"many": {
					Data: jsonapi.Many{
						Value: []*jsonapi.Resource{{ID: "3", Type: "items"}},
					},
				},
				"nullable": {
					Data: jsonapi.One{},
				},
				"manyobj": {
					Data: jsonapi.Many{
						Value: []*jsonapi.Resource{{ID: "4", Type: "items"}},
					},
				},
			},
		}

		out := RelatedItem{}
		err := jsonapi.UnmarshalResource(&in, &out)
		assert.NoError(t, err)

		want := RelatedItem{
			ID:      "1",
			One:     &SimpleItem{ID: "2"},
			Many:    []*SimpleItem{{ID: "3"}},
			ManyObj: []SimpleItem{{ID: "4"}},
		}

		assert.EqualValues(t, want, out)

		t.Run("nullable relationships", func(t *testing.T) {
			in.Relationships["nullable"].Data = nil
			out := RelatedItem{}
			err := jsonapi.UnmarshalResource(&in, &out)
			assert.NoError(t, err)
			want := RelatedItem{
				ID:      "1",
				One:     &SimpleItem{ID: "2"},
				Many:    []*SimpleItem{{ID: "3"}},
				ManyObj: []SimpleItem{{ID: "4"}},
			}
			assert.EqualValues(t, want, out)
		})

		t.Run("missing relationship", func(t *testing.T) {
			delete(in.Relationships, "nullable")
			out := RelatedItem{}
			err := jsonapi.UnmarshalResource(&in, &out)
			assert.NoError(t, err)
			want := RelatedItem{
				ID:      "1",
				One:     &SimpleItem{ID: "2"},
				Many:    []*SimpleItem{{ID: "3"}},
				ManyObj: []SimpleItem{{ID: "4"}},
			}
			assert.EqualValues(t, want, out)
		})

		t.Run("empty relationships", func(t *testing.T) {
			in.Relationships = nil
			out := RelatedItem{}
			err := jsonapi.UnmarshalResource(&in, &out)
			assert.NoError(t, err)
			want := RelatedItem{ID: "1"}
			assert.EqualValues(t, want, out)
		})
	})
}

func TestMarshal(t *testing.T) {
	type testcase struct {
		name    string
		in      any
		want    jsonapi.Document
		wantErr bool
	}

	for _, tc := range []testcase{
		{
			name: "marshal document",
			in: jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:            "1",
						Type:          "items",
						Attributes:    map[string]any{"value1": ""},
						Relationships: map[string]*jsonapi.Relationship{},
					},
				},
			},
			want: jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:            "1",
						Type:          "items",
						Attributes:    map[string]any{"value1": ""},
						Relationships: map[string]*jsonapi.Relationship{},
					},
				},
			},
		},
		{
			name: "marshal document pointer",
			in: &jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:            "1",
						Type:          "items",
						Attributes:    map[string]any{"value1": ""},
						Relationships: map[string]*jsonapi.Relationship{},
					},
				},
			},
			want: jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:            "1",
						Type:          "items",
						Attributes:    map[string]any{"value1": ""},
						Relationships: map[string]*jsonapi.Relationship{},
					},
				},
			},
		},
		{
			name: "marshal resource",
			in: &jsonapi.Resource{
				ID:            "1",
				Type:          "items",
				Attributes:    map[string]any{"value1": ""},
				Relationships: map[string]*jsonapi.Relationship{},
			},
			want: jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:            "1",
						Type:          "items",
						Attributes:    map[string]any{"value1": ""},
						Relationships: map[string]*jsonapi.Relationship{},
					},
				},
			},
		},
		{
			name: "marshal one resource",
			in:   SimpleItem{ID: "1"},
			want: jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:            "1",
						Type:          "items",
						Attributes:    map[string]any{"value1": ""},
						Extensions:    map[string]*json.RawMessage{},
						Relationships: map[string]*jsonapi.Relationship{},
					},
				},
			},
		},
		{
			name: "marshal many resource",
			in: []*SimpleItem{
				{ID: "1"},
			},
			want: jsonapi.Document{
				Data: jsonapi.Many{
					Value: []*jsonapi.Resource{
						{
							ID:            "1",
							Type:          "items",
							Attributes:    map[string]any{"value1": ""},
							Extensions:    map[string]*json.RawMessage{},
							Relationships: map[string]*jsonapi.Relationship{},
						},
					},
				},
			},
		},
		{
			name: "marshal many resource with includes",
			in: []*RelatedItem{
				{ID: "1", One: &SimpleItem{ID: "2"}},
			},
			want: jsonapi.Document{
				Data: jsonapi.Many{
					Value: []*jsonapi.Resource{
						{
							ID:         "1",
							Type:       "items",
							Attributes: map[string]any{},
							Extensions: map[string]*json.RawMessage{},
							Relationships: map[string]*jsonapi.Relationship{
								"one": {
									Data: jsonapi.One{Value: &jsonapi.Resource{
										ID:   "2",
										Type: "items",
									}},
								},
								"many":     {Data: jsonapi.Many{Value: []*jsonapi.Resource{}}},
								"manyobj":  {Data: nil},
								"nullable": {Data: nil},
							},
						},
					},
				},
				Included: []*jsonapi.Resource{
					{
						ID:   "2",
						Type: "items",
						Attributes: map[string]any{
							"value1": "",
						},
						Extensions:    map[string]*json.RawMessage{},
						Relationships: map[string]*jsonapi.Relationship{},
					},
				},
			},
		},
		{
			name: "marshal many resource fails",
			in: []*SimpleItem{
				nil,
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := jsonapi.Marshal(tc.in)
			t.Log(got)

			if tc.wantErr && assert.Error(t, err) {
				return
			}

			assert.NoError(t, err)
			assert.EqualValues(t, tc.want, got)
		})
	}
}

func TestMarshalRef(t *testing.T) {
	type testcase struct {
		name    string
		in      any
		ref     string
		want    jsonapi.Document
		wantErr bool
	}

	for _, tc := range []testcase{
		{
			name: "marshal ref fails with empty name",
			in: RelatedItem{
				ID:  "1",
				One: &SimpleItem{ID: "2"},
			},
			ref:     "",
			wantErr: true,
		},
		{
			name:    "marshal ref fails with bad input",
			in:      struct{}{},
			ref:     "items",
			wantErr: true,
		},
		{
			name:    "marshal ref fails with many document",
			in:      []RelatedItem{{ID: "1"}, {ID: "2"}},
			ref:     "items",
			wantErr: true,
		},
		{
			name: "marshal ref",
			in: RelatedItem{
				ID:  "1",
				One: &SimpleItem{ID: "2"},
			},
			ref: "one",
			want: jsonapi.Document{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:   "2",
						Type: "items",
					},
				},
				Included: []*jsonapi.Resource{
					{
						ID:   "1",
						Type: "items",
						Relationships: jsonapi.RelationshipsNode{
							"one": &jsonapi.Relationship{
								Data: jsonapi.One{
									Value: &jsonapi.Resource{
										ID:   "2",
										Type: "items",
									},
								},
							},
							"many": &jsonapi.Relationship{
								Data: jsonapi.Many{Value: []*jsonapi.Resource{}},
							},
							"nullable": &jsonapi.Relationship{},
							"manyobj":  &jsonapi.Relationship{},
						},
						Attributes: map[string]any{},
						Extensions: jsonapi.ExtensionsNode{},
					},
					{
						ID:            "2",
						Type:          "items",
						Attributes:    map[string]any{"value1": ""},
						Extensions:    jsonapi.ExtensionsNode{},
						Relationships: jsonapi.RelationshipsNode{},
					},
				},
			},
		},
		{
			name: "marshal ref fails with unknown relationship",
			in: RelatedItem{
				ID:  "1",
				One: &SimpleItem{ID: "2"},
			},
			ref:     "unknown",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := jsonapi.MarshalRef(tc.in, tc.ref)

			if tc.wantErr && assert.Error(t, err) {
				return
			}

			assert.NoError(t, err)

			wantIncluded := tc.want.Included
			tc.want.Included = nil

			gotIncluded := got.Included
			got.Included = nil

			if !assert.EqualValues(t, tc.want, got) {
				wantJSON, _ := json.MarshalIndent(tc.want, "", "  ")
				gotJSON, _ := json.MarshalIndent(got, "", "  ")
				t.Logf("want: %s\n", string(wantJSON))
				t.Logf("got: %s\n", string(gotJSON))
			}

			assert.ElementsMatch(t, wantIncluded, gotIncluded)
		})
	}
}
