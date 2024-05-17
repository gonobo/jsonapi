package jsonapi_test

import (
	"encoding/json"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/stretchr/testify/assert"
)

func TestVisitor(t *testing.T) {
	type testcase struct {
		name     string
		document jsonapi.Document
		wantErr  bool
	}

	partial := jsonapi.PartialVisitor{
		Document: func(d *jsonapi.Document) error { return nil },
	}

	for _, tc := range []testcase{
		{name: "empty document"},
		{
			name: "error document",
			document: jsonapi.Document{
				Errors: []*jsonapi.Error{
					{
						Detail: "error",
						Meta:   jsonapi.Meta{},
					},
				}},
		},
		{
			name: "one document",
			document: jsonapi.Document{
				Links: jsonapi.Links{
					"foo": &jsonapi.Link{Href: "http://foo.com"},
				},
				Meta: jsonapi.Meta{
					"foo": "bar",
				},
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						ID:         "1",
						Type:       "nodes",
						Attributes: map[string]any{"foo": "bar"},
						Extensions: map[string]*json.RawMessage{
							"foo": MarshalRaw(t, "bar"),
						},
						Links: jsonapi.Links{},
						Meta:  jsonapi.Meta{},
						Relationships: jsonapi.RelationshipsNode{
							"foo": &jsonapi.Relationship{
								Links: jsonapi.Links{},
								Meta:  jsonapi.Meta{},
								Data: jsonapi.Many{
									Value: []*jsonapi.Resource{
										{ID: "1", Type: "nodes", Meta: jsonapi.Meta{}},
										{ID: "2", Type: "nodes", Meta: jsonapi.Meta{}},
									},
								},
							},
						},
					},
				},
				Included: []*jsonapi.Resource{
					{ID: "1", Type: "nodes"},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.document.ApplyVisitor(partial.Visitor())
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
