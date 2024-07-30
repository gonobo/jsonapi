package visitor_test

import (
	"encoding/json"
	"testing"

	"github.com/gonobo/jsonapi/v1"
	"github.com/gonobo/jsonapi/v1/extra/visitor"
	"github.com/gonobo/jsonapi/v1/jsonapitest"
	"github.com/stretchr/testify/assert"
)

func TestVisitor(t *testing.T) {
	type testcase struct {
		name     string
		document jsonapi.Document
		wantErr  bool
	}

	partial := visitor.PartialVisitor{
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
							"foo": jsonapitest.MarshalRaw(t, "bar"),
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
			err := visitor.VisitDocument(partial.Visitor(), &tc.document)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
