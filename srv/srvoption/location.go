package srvoption

import (
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/srv"
)

const (
	HeaderKeyLocation = "Location"
)

func WriteLocationHeader(baseURL string, resolver jsonapi.URLResolver) srv.WriteOptions {
	return srv.UseDocumentOptions(writeLocationHeader(baseURL, resolver))
}

func writeLocationHeader(baseURL string, resolver jsonapi.URLResolver) srv.DocumentOptions {
	return func(w http.ResponseWriter, r *http.Request, d *jsonapi.Document) {
		data := d.Data.First()
		location := resolver.ResolveURL(jsonapi.RequestContext{
			ResourceType: data.Type,
			ResourceID:   data.ID,
		}, baseURL)
		w.Header().Add(HeaderKeyLocation, location)
	}
}
