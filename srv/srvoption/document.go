package srvoption

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/srv"
)

// Meta adds a key/value pair to the response document's meta attribute.
func Meta(key string, value any) srv.WriteOptions {
	return srv.UseDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			if d.Meta == nil {
				d.Meta = jsonapi.Meta{}
			}
			d.Meta[key] = value
			return nil
		})
}

// Link adds a URL to the response document's links attribute.
func Link(key string, href string) srv.WriteOptions {
	return srv.UseDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			if d.Links == nil {
				d.Links = jsonapi.Links{}
			}
			_, err := url.Parse(href)
			if err != nil {
				return fmt.Errorf("add link: failed to parse href: %s: %w", href, err)
			}
			d.Links[key] = &jsonapi.Link{Href: href}
			return nil
		})
}

// SelfLink adds the full request URL to the response document's links attribute.
func SelfLink(r *http.Request) srv.WriteOptions {
	href := r.URL.String()
	return Link("self", href)
}

// NextPageCursorLink adds the next pagination URL to the response document's links attribute.
func NextPageCursorLink(r *http.Request, cursor string, limit int) srv.WriteOptions {
	requestURL := *r.URL
	query := requestURL.Query()
	query.Del("page[cursor]")
	query.Set("page[cursor]", cursor)

	if limit > 0 {
		query.Del("page[limit]")
		query.Set("page[limit]", strconv.Itoa(limit))
	}

	requestURL.RawQuery = query.Encode()
	return Link("next", requestURL.String())
}

// VisitDocument applies the visitor to the response document. Visitors can traverse and modify
// a document's nodes.
func VisitDocument(visitor jsonapi.PartialVisitor) srv.WriteOptions {
	return srv.UseDocumentOptions(func(w http.ResponseWriter, d *jsonapi.Document) error {
		return d.ApplyVisitor(visitor.Visitor())
	})
}

// ResourceLinks applies the "self" and "related" links to all resources and resource relationships
// embedded in the response document.
func ResourceLinks(baseURL string, resolver jsonapi.URLResolver) srv.WriteOptions {
	visitor := jsonapi.PartialVisitor{
		Resource: func(r *jsonapi.Resource) error {
			self := resolver.ResolveURL(jsonapi.RequestContext{
				ResourceType: r.Type,
				ResourceID:   r.ID,
			}, baseURL)

			if r.Links == nil {
				r.Links = jsonapi.Links{}
			}

			r.Links["self"] = &jsonapi.Link{Href: self}

			for name, rel := range r.Relationships {
				if rel.Meta == nil {
					rel.Meta = jsonapi.Meta{}
				}

				rel.Meta["$parent_type"] = r.Type
				rel.Meta["$parent_id"] = r.ID
				rel.Meta["$rel_name"] = name
			}

			return nil
		},
		Relationship: func(r *jsonapi.Relationship) error {
			resourceType := r.Meta["$parent_type"].(string)
			resourceID := r.Meta["$parent_id"].(string)
			relationship := r.Meta["rel_name"].(string)

			self := resolver.ResolveURL(jsonapi.RequestContext{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Relationship: relationship,
			}, baseURL)

			related := resolver.ResolveURL(jsonapi.RequestContext{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Relationship: relationship,
				Related:      true,
			}, baseURL)

			if r.Links == nil {
				r.Links = jsonapi.Links{}
			}

			r.Links["self"] = &jsonapi.Link{Href: self}
			r.Links["related"] = &jsonapi.Link{Href: related}

			delete(r.Meta, "$parent_type")
			delete(r.Meta, "$parent_id")
			delete(r.Meta, "$rel_name")

			return nil
		},
	}

	return VisitDocument(visitor)
}
