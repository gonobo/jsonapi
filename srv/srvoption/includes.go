package srvoption

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/srv"
)

const (
	QueryParamInclude = "include"
)

var (
	JSONUnmarshal func([]byte, any) error = json.Unmarshal
)

func ResolvesIncludedResources(resolver jsonapi.URLResolver) srv.Options {
	return srv.UseMiddleware(resolveIncludedResources(resolver))
}

func resolveIncludedResources(resolver jsonapi.URLResolver) srv.Middleware {
	return func(next http.Handler) http.Handler {
		return resolveIncluded{next, resolver}
	}
}

type resolveIncluded struct {
	http.Handler
	jsonapi.URLResolver
}

func (rr resolveIncluded) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// allow downstream handler to resolve request, but store response instead of sending it
	// to the stream.
	mem := srv.MemoryWriter{}
	rr.Handler.ServeHTTP(&mem, r)

	if rr.skip(r, &mem) {
		// nothing to do, skip request. send response and return
		mem.Flush(w)
		return
	}

	include := strings.Split(r.URL.Query().Get(QueryParamInclude), ",")
	if len(include) == 0 {
		// nothing to do, included resources were not requested
		mem.Flush(w)
		return
	}

	// unmarshal the response; extract the resource item(s) from the document's primary data
	doc := jsonapi.Document{}
	err := JSONUnmarshal(mem.Body, &doc)
	if err != nil {
		srv.Error(w, fmt.Errorf("resolve included: failed to unmarshal body: %w", err), http.StatusInternalServerError)
		return
	}

	if doc.Data == nil {
		// nothing to do, no resources to include
		mem.Flush(w)
		return
	}

	// memoize included resources; some relationships might share the same resource -- no need
	// to include it multiple times.
	memo := make(map[string]*jsonapi.Resource)
	mux := srv.GetResourceMux(r)

	// iterate through each resource, fetching the related resource associated with the target relationship.
	for _, item := range doc.Data.Items() {
		for _, relationship := range include {
			err := rr.fetchRelated(r, mux, relationship, item, memo)
			if err != nil {
				srv.Error(w, fmt.Errorf("resolve included: failed to fetch related: %w", err), http.StatusInternalServerError)
				return
			}
		}
	}

	// add the collected resources to the included field of the document.
	for _, item := range memo {
		doc.Included = append(doc.Included, item)
	}

	mem.Body, err = json.Marshal(doc)
	if err != nil {
		srv.Error(w, fmt.Errorf("resolve included: failed to marshal body: %w", err), http.StatusInternalServerError)
		return
	}

	// flush updated document to the response stream
	mem.Flush(w)
}

func (rr resolveIncluded) fetchRelated(r *http.Request,
	mux *srv.ResourceMux, relationship string, item *jsonapi.Resource, memo map[string]*jsonapi.Resource) error {
	if item.Relationships == nil {
		// nothing to include, return early
		return nil
	}

	ref, ok := item.Relationships[relationship]
	if !ok {
		// no relationship data contained in the document, return early
		return nil
	} else if ref.Data == nil {
		// no relationship data contained in the document, return early
		return nil
	}

	// create the request context to be used in the fetch request
	ctx := rr.createRequestContext(ref.Data)
	mem := srv.MemoryWriter{}

	mux.ServeResourceHTTP(&mem, jsonapi.RequestWithContext(r, &ctx))

	if mem.Status != http.StatusOK {
		return fmt.Errorf("failed to retrieve included resources: %s", relationship)
	}

	doc := jsonapi.Document{}
	err := JSONUnmarshal(mem.Body, &doc)
	if err != nil {
		return fmt.Errorf("failed to unmarshal included resources: %s: %w", relationship, err)
	}

	if doc.Data == nil {
		return nil
	}

	for _, item := range doc.Data.Items() {
		key := fmt.Sprintf("%s:%s", item.Type, item.ID)
		memo[key] = item
	}

	return nil
}

func (resolveIncluded) createRequestContext(ref jsonapi.PrimaryData) jsonapi.RequestContext {
	// collect the unique identifiers for all resources identified in the relationship
	var resourceType string
	ids := make([]string, 0)
	items := ref.Items()

	for idx, item := range items {
		if idx == 0 {
			resourceType = item.Type
		}
		ids = append(ids, item.ID)
	}

	return jsonapi.RequestContext{
		ResourceType: resourceType,
		Related:      true,
		FetchIDs:     ids,
	}
}

func (resolveIncluded) skip(r *http.Request, mem *srv.MemoryWriter) bool {
	return r.Method != http.MethodGet || // is not a fetch request
		mem.Status != http.StatusOK || // is not a 200 return status
		len(mem.Body) == 0 // has no a document payload
}
