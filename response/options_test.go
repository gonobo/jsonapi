package response_test

import (
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/response"
	"github.com/stretchr/testify/assert"
)

func TestAddTopLinks(t *testing.T) {
	t.Run("document with empty links", func(t *testing.T) {
		document := jsonapi.Document{}
		res := jsonapi.Response{Body: &document}
		options := response.AddTopLinks(jsonapi.Links{
			"foo": &jsonapi.Link{Href: "/bar"},
		})
		assert.Nil(t, document.Links)
		options(&res)
		assert.NotNil(t, document.Links)
		assert.EqualValues(t, jsonapi.Links{
			"foo": &jsonapi.Link{Href: "/bar"},
		}, document.Links)
	})
}

func TestResolveRelativeLinkURLs(t *testing.T) {

	document := jsonapi.Document{
		Links: jsonapi.Links{
			"top": &jsonapi.Link{Href: "/top"},
		},
		Data: jsonapi.One{
			Value: &jsonapi.Resource{
				ID:   "1",
				Type: "items",
				Links: jsonapi.Links{
					"self": &jsonapi.Link{Href: "/items/1"},
					"foo":  &jsonapi.Link{Href: "/foo"},
				},
				Relationships: jsonapi.RelationshipsNode{
					"one": &jsonapi.Relationship{
						Links: jsonapi.Links{
							"self":    &jsonapi.Link{},
							"related": &jsonapi.Link{},
							"foo":     &jsonapi.Link{Href: "/foo"},
						},
					},
					"nolinks": &jsonapi.Relationship{},
				},
			},
		},
	}
	res := jsonapi.Response{Body: &document}
	options := response.ResolveRelativeLinkURLs("http://api.foo.com", jsonapi.DefaultURLResolver())
	options(&res)

	assert.EqualValues(t, jsonapi.Document{
		Links: jsonapi.Links{
			"top": &jsonapi.Link{Href: "http://api.foo.com/top"},
		},
		Data: jsonapi.One{
			Value: &jsonapi.Resource{
				ID:   "1",
				Type: "items",
				Links: jsonapi.Links{
					"self": &jsonapi.Link{Href: "http://api.foo.com/items/1"},
					"foo":  &jsonapi.Link{Href: "http://api.foo.com/foo"},
				},
				Relationships: jsonapi.RelationshipsNode{
					"one": &jsonapi.Relationship{
						Links: jsonapi.Links{
							"self":    &jsonapi.Link{Href: "http://api.foo.com/items/1/relationships/one"},
							"related": &jsonapi.Link{Href: "http://api.foo.com/items/1/one"},
							"foo":     &jsonapi.Link{Href: "http://api.foo.com/foo"},
						},
					},
					"nolinks": &jsonapi.Relationship{},
				},
			},
		},
	}, document)
}
