package option

import (
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/srv"
)

// Sort orders resources in the response document according to the specified
// sort criterion.
func Sort(cmp jsonapi.Comparator, criterion []query.Sort) srv.WriteOptions {
	return srv.UseDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			d.Sort(cmp, criterion)
			return nil
		},
	)
}

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

// VisitDocument applies the visitor to the response document. Visitors can traverse and modify
// a document's nodes.
func VisitDocument(visitor jsonapi.PartialVisitor) srv.WriteOptions {
	return srv.UseDocumentOptions(func(w http.ResponseWriter, d *jsonapi.Document) error {
		return d.ApplyVisitor(visitor.Visitor())
	})
}
