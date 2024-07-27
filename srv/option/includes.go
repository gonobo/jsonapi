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

// UseIncludeQueryParser is a middleware that parses the list of included
// resources requested by the client and adds them to the JSON:API context.
func UseIncludeQueryParser() srv.Options {
	return srv.WithMiddleware(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				include := strings.Split(r.URL.Query().Get(QueryParamInclude), ",")
				ctx, _ := jsonapi.GetContext(r.Context())
				ctx.Include = include
				next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
			})
		},
	)
}

func UseRelatedResourceResolver() srv.Options {
	return srv.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				resolver := includeResolver{}
				resolver.resolveRelatedResources(next, w, r)
			},
		)
	})
}

// UseIncludedResourceResolver is a middleware that retrieves the client-request
// server resources associated with the response document's primary data,
// and adds it to the "included" array.
//
// UseIncludedResourceResolver currently supports inclusion requests only one level deep;
// dot notation for multiple inclusions is not supported.
func UseIncludedResourceResolver() srv.Options {
	return srv.WithMiddleware(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				resolver := includeResolver{}
				resolver.resolveIncludedQuery(next, w, r)
			})
		},
	)
}

type includeResolver struct{}

func (ir includeResolver) resolveIncludedQuery(next http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx, _ := jsonapi.GetContext(r.Context())

	// if client did not request an individual resource, skip this middleware.
	if !ir.isFetchResourceRequest(r, ctx) {
		next.ServeHTTP(w, r)
		return
	}

	// serve the request with a memory writer to get the resource document.
	mem := srv.MemoryWriter{}
	next.ServeHTTP(&mem, r)

	if mem.Status != http.StatusOK {
		// nothing to do, no OK response
		mem.Flush(w)
		return
	}

	body := jsonapi.Document{}
	err := JSONUnmarshal(mem.Body, &body)

	if err != nil {
		srv.Error(w, fmt.Errorf("resolve include: failed to marshal payload: %w", err), http.StatusInternalServerError)
		return
	}

	include := ctx.Include

	if len(include) == 0 {
		// nothing to do, included resources were not requested
		mem.Flush(w)
		return
	}

	// memoize included resources; some relationships might share the same resource -- no need
	// to include it multiple times.
	memo := make(map[string]*jsonapi.Resource)

	// iterate through each resource, fetching the related resource associated with the target relationship.
	for _, item := range body.Data.Items() {
		for _, relationship := range include {
			err := ir.fetchRelatedIncludes(ctx, next, r, relationship, item, memo)
			if err != nil {
				err = fmt.Errorf("resolve included: failed to fetch related: %w", err)
				srv.Error(w, err, http.StatusInternalServerError)
				return
			}
		}
	}

	for _, item := range memo {
		body.Included = append(body.Included, item)
	}

	srv.Write(w, body)
}

func (ir includeResolver) fetchRelatedIncludes(
	ctx *jsonapi.RequestContext,
	handler http.Handler,
	r *http.Request,
	relationship string,
	item *jsonapi.Resource,
	memo map[string]*jsonapi.Resource) error {
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
	ctx = ir.createRequestContext(ctx, ref.Data)
	mem := srv.MemoryWriter{}

	handler.ServeHTTP(&mem, jsonapi.RequestWithContext(r, ctx))

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

func (includeResolver) createRequestContext(parent *jsonapi.RequestContext, ref jsonapi.PrimaryData) *jsonapi.RequestContext {
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

	ctx := parent.Child()
	ctx.ResourceType = resourceType
	ctx.Related = true
	ctx.FetchIDs = ids

	return ctx
}

func (includeResolver) isRelatedResourceRequest(ctx *jsonapi.RequestContext) bool {
	return ctx.Related &&
		ctx.Relationship != "" &&
		ctx.ResourceID != "" &&
		ctx.ResourceType != ""
}

func (includeResolver) isFetchResourceRequest(r *http.Request, ctx *jsonapi.RequestContext) bool {
	return r.Method == http.MethodGet &&
		ctx.ResourceID != "" &&
		ctx.Relationship == "" &&
		ctx.Related == false &&
		ctx.ResourceType != ""
}

func (rr includeResolver) resolveRelatedResources(next http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx, _ := jsonapi.GetContext(r.Context())

	// skip if client is not requesting related resources
	if !rr.isRelatedResourceRequest(ctx) {
		next.ServeHTTP(w, r)
		return
	}

	// otherwise, retrieve individual resource relationship references first,
	// then fetch the the resources referenced by the relationship.

	ctx = ctx.Child()
	ctx.Related = false

	mem := srv.MemoryWriter{}
	next.ServeHTTP(&mem, jsonapi.RequestWithContext(r, ctx))

	if mem.Status != http.StatusOK {
		mem.Flush(w)
		return
	}

	body := jsonapi.Document{}
	err := JSONUnmarshal(mem.Body, &body)

	if err != nil {
		srv.Error(w, fmt.Errorf("fetch related: failed to unmarshal payload: %w", err), http.StatusInternalServerError)
		return
	} else if body.Data == nil {
		srv.Error(w, fmt.Errorf("fetch related: missing primary data from payload"), http.StatusInternalServerError)
		return
	}

	// create a request context for related resources
	ctx = rr.createRequestContext(ctx, body.Data)

	// if there are no ids to fetch, return an empty document
	if len(ctx.FetchIDs) == 0 {
		srv.Write(w, jsonapi.NewMultiDocument())
		return
	}

	// serve the request
	next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
}
