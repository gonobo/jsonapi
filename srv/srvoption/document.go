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
			uri, err := url.ParseRequestURI(href)
			if err != nil {
				return fmt.Errorf("add link: failed to parse href: %s: %w", href, err)
			}
			d.Links[key] = &jsonapi.Link{Href: uri.String()}
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
	const keyParentType = "$__parenttype"
	const keyParentID = "$__parentid"
	const keyRelName = "$__relname"

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

				rel.Meta[keyParentType] = r.Type
				rel.Meta[keyParentID] = r.ID
				rel.Meta[keyRelName] = name
			}

			return nil
		},
		Relationship: func(r *jsonapi.Relationship) error {
			resourceType := r.Meta[keyParentType].(string)
			resourceID := r.Meta[keyParentID].(string)
			relationship := r.Meta[keyRelName].(string)

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

			delete(r.Meta, keyParentType)
			delete(r.Meta, keyParentID)
			delete(r.Meta, keyRelName)

			return nil
		},
	}

	return VisitDocument(visitor)
}
