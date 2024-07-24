package option

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

// ResourceMuxer serves incoming requests associated with a server resource type.
// It is implemented by the srv.ResourceMux struct.
type ResourceMuxer interface {
	// ServeResourceHTTP handles incoming requests associated with a server resource type.
	ServeResourceHTTP(http.ResponseWriter, *http.Request)
}

// WithIncludedResources uses the provided resource mux to lookup the client-request
// server resources associated with the response document's primary data,
// and add it to the "included" array.
//
// WithIncludedResources currently supports inclusion requests only one level deep;
// dot notation for multiple inclusions is not supported.
func WithIncludedResources(r *http.Request, mux ResourceMuxer) srv.WriteOptions {
	return srv.WithDocumentOptions(resolveIncludes(r, mux))
}

func resolveIncludes(r *http.Request, mux ResourceMuxer) srv.DocumentOptions {
	return func(w http.ResponseWriter, d *jsonapi.Document) error {
		resolver := includeResolver{d, mux}
		return resolver.Resolve(w, r)
	}
}

type includeResolver struct {
	Document *jsonapi.Document
	Mux      ResourceMuxer
}

func (ir includeResolver) Resolve(w http.ResponseWriter, r *http.Request) error {
	include := strings.Split(r.URL.Query().Get(QueryParamInclude), ",")
	if len(include) == 0 {
		// nothing to do, included resources were not requested
		return nil
	}

	// memoize included resources; some relationships might share the same resource -- no need
	// to include it multiple times.
	memo := make(map[string]*jsonapi.Resource)
	mux := srv.GetResourceMux(r)

	// iterate through each resource, fetching the related resource associated with the target relationship.
	for _, item := range ir.Document.Data.Items() {
		for _, relationship := range include {
			err := ir.fetchRelated(r, mux, relationship, item, memo)
			if err != nil {
				return fmt.Errorf("resolve included: failed to fetch related: %w", err)
			}
		}
	}
	return nil
}

func (ir includeResolver) fetchRelated(r *http.Request,
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
	ctx := ir.createRequestContext(ref.Data)
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

func (includeResolver) createRequestContext(ref jsonapi.PrimaryData) jsonapi.RequestContext {
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
