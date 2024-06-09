package response

import (
	"strings"

	"github.com/gonobo/jsonapi"
)

func modifyResponseBody(fn func(*jsonapi.Document)) jsonapi.ResponseOption {
	return func(j *jsonapi.Response) {
		if j.Body != nil {
			fn(j.Body)
		}
	}
}

// AddTopLinks adds links to the top level of the document.
func AddTopLinks(links jsonapi.Links) jsonapi.ResponseOption {
	return modifyResponseBody(func(d *jsonapi.Document) {
		if d.Links == nil {
			d.Links = jsonapi.Links{}
		}
		for k, v := range links {
			d.Links[k] = v
		}
	})
}

// ResolveRelativeLinkURLs traverses the document, affixing the provided base url to any
// links with relative paths.
func ResolveRelativeLinkURLs(baseURL string, resolver jsonapi.URLResolver) jsonapi.ResponseOption {
	return modifyResponseBody(func(d *jsonapi.Document) {
		impl := &resolveRelativeLinkURLs{baseURL: baseURL, resolver: resolver}

		visitor := jsonapi.PartialVisitor{
			Relationships: impl.VisitRelationships,
			Resource:      impl.VisitResource,
			Links:         impl.VisitLinks,
		}

		d.ApplyVisitor(visitor.Visitor())
	})
}

type resolveRelativeLinkURLs struct {
	baseURL    string
	resolver   jsonapi.URLResolver
	currentCtx jsonapi.RequestContext
}

func (a *resolveRelativeLinkURLs) VisitResource(obj *jsonapi.Resource) error {
	if obj.Links != nil {
		a.currentCtx = jsonapi.RequestContext{ResourceType: obj.Type, ResourceID: obj.ID}
		self := a.resolver.ResolveURL(a.currentCtx, a.baseURL)
		links := jsonapi.Links{}

		for key, value := range obj.Links {
			switch key {
			case "self":
				value.Href = self
			}
			links[key] = value
		}

		obj.Links = links
	}
	return nil
}

func (a *resolveRelativeLinkURLs) VisitRelationships(obj jsonapi.RelationshipsNode) error {
	for name, rel := range obj {
		if rel.Links == nil {
			continue
		}

		ctx := a.currentCtx.Clone()
		ctx.Relationship = name
		self := a.resolver.ResolveURL(*ctx, a.baseURL)

		ctx.Related = true
		related := a.resolver.ResolveURL(*ctx, a.baseURL)

		links := jsonapi.Links{}

		for key, value := range rel.Links {
			switch key {
			case "self":
				value.Href = self
			case "related":
				value.Href = related
			}
			links[key] = value
		}

		rel.Links = links
	}
	return nil
}

func (a resolveRelativeLinkURLs) VisitLinks(obj jsonapi.Links) error {
	for key, value := range obj {
		value.Href = a.setRelativeLink(value.Href)
		obj[key] = value
	}
	return nil
}

func (a resolveRelativeLinkURLs) setRelativeLink(href string) string {
	if strings.HasPrefix(href, "/") {
		return a.baseURL + href
	}
	return href
}
