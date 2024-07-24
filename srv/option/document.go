package option

import (
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/srv"
)

const (
	HeaderKeyLocation = "Location"
)

// UseSortParams orders resources in the response document according to the specified
// sort criterion.
func UseSortParams(cmp jsonapi.Comparator, criterion []query.Sort) srv.WriteOptions {
	return srv.UseDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			d.Sort(cmp, criterion)
			return nil
		},
	)
}

// WithMetaValue adds a key/value pair to the response document's meta attribute.
func WithMetaValue(key string, value any) srv.WriteOptions {
	return srv.UseDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			if d.Meta == nil {
				d.Meta = jsonapi.Meta{}
			}
			d.Meta[key] = value
			return nil
		})
}

// UseDocumentVisitor applies the visitor to the response document. Visitors can traverse and modify
// a document's nodes.
func UseDocumentVisitor(visitor jsonapi.PartialVisitor) srv.WriteOptions {
	return srv.UseDocumentOptions(func(w http.ResponseWriter, d *jsonapi.Document) error {
		return d.ApplyVisitor(visitor.Visitor())
	})
}

// WithLocationHeader adds the "Location" http header to the response. The resulting
// URL is based on the primary data resource's type and id.
func WithLocationHeader(baseURL string, resolver jsonapi.URLResolver) srv.WriteOptions {
	return srv.UseDocumentOptions(writeLocationHeader(baseURL, resolver))
}

func writeLocationHeader(baseURL string, resolver jsonapi.URLResolver) srv.DocumentOptions {
	return func(w http.ResponseWriter, d *jsonapi.Document) error {
		data := d.Data.First()
		location := resolver.ResolveURL(jsonapi.RequestContext{
			ResourceType: data.Type,
			ResourceID:   data.ID,
		}, baseURL)
		w.Header().Add(HeaderKeyLocation, location)
		return nil
	}
}
