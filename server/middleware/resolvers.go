package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gonobo/jsonapi/v2"
	"github.com/gonobo/jsonapi/v2/server"
)

// UseRelatedResourceResolver is a middleware that handles incoming requests
// for related resources.
func UseRelatedResourceResolver() server.Options {
	return server.WithMiddleware(func(next http.Handler) http.Handler {
		resolver := relatedResourceResolver{next}
		return http.HandlerFunc(resolver.retrieveRelated)
	})
}

// UseIncludedResourceResolver is a middleware that retrieves the client-request
// server resources associated with the response document's primary data,
// and adds it to the "included" array.
//
// UseIncludedResourceResolver currently supports inclusion requests only one level deep;
// dot notation for multiple inclusions is not supported.
func UseIncludedResourceResolver() server.Options {
	return server.WithMiddleware(
		func(next http.Handler) http.Handler {
			resolver := relatedResourceResolver{next}
			return http.HandlerFunc(resolver.includeRelated)
		},
	)
}

type relatedResourceResolver struct {
	handler http.Handler
}

func (rr relatedResourceResolver) includeRelated(w http.ResponseWriter, r *http.Request) {
	ctx := jsonapi.FromContext(r.Context())

	// skip if the request is not for a single resource.

	if !rr.isFetchResourceRequest(r, ctx) {
		rr.handler.ServeHTTP(w, r)
		return
	}

	// capture downstream response using a recorder

	mem := server.NewRecorder()
	rr.handler.ServeHTTP(mem, r)

	// if the result is not OK, or if there is no data to parse,
	// return the result to the client.

	if mem.Status != http.StatusOK {
		mem.Flush(w)
	} else if mem.Document == nil || mem.Document.Data == nil {
		mem.Flush(w)
	}

	data := mem.Document.Data.First()
	memo := make(map[string]*jsonapi.Resource)

	// parse the data from downstream; resolve each requested relationship
	// and store in a lookup table (to prevent multiple references to the
	// same resource). once finished, dump the memo to the list of included
	// resources and return.

	for _, include := range ctx.Include {
		// check and handle nested include requests represented with dot notation.
		names := strings.Split(include, ".")
		// if an error is generated during fetch, halt and return the error
		// back to the client.
		if err := rr.fetchResourceRelationships(r, names, data, memo); err != nil {
			server.Error(w, fmt.Errorf("include resources: %w", err), http.StatusInternalServerError)
			return
		}
	}

	for _, included := range memo {
		mem.Document.Included = append(mem.Document.Included, included)
	}

	mem.Flush(w)
}

func (rr relatedResourceResolver) retrieveRelated(w http.ResponseWriter, r *http.Request) {
	ctx := jsonapi.FromContext(r.Context())

	// skip if the request is not for related resources
	if !rr.isRelatedResourceRequest(ctx) {
		rr.handler.ServeHTTP(w, r)
		return
	}

	// use a recorder to capture a downstream request to retrieve
	// the parent resource.

	mem := server.NewRecorder()
	ctx = ctx.Child()
	ctx.Related = false

	rr.handler.ServeHTTP(mem, jsonapi.RequestWithContext(r, ctx))

	// if the request is not OK, or if there is no data to be parsed
	// return early.

	if mem.Status != http.StatusOK {
		mem.Flush(w)
		return
	} else if mem.Document == nil || mem.Document.Data == nil {
		mem.Flush(w)
		return
	}

	// capture the response data's relationships, and and send additional
	// request to get the related resources. use a memo to avoid duplicate
	// resources in the response.

	memo := make(map[string]*jsonapi.Resource)
	data := mem.Document.Data.First()
	names := []string{ctx.Relationship}

	if err := rr.fetchResourceRelationships(r, names, data, memo); err != nil {
		// if the request fails, return the error back to the client.
		server.Error(w, fmt.Errorf("related resources: %w", err), http.StatusInternalServerError)
		return
	}

	// dump the memo into a new multi document and return to the client.
	items := make([]*jsonapi.Resource, len(memo))

	for _, item := range memo {
		items = append(items, item)
	}

	server.Write(w, jsonapi.NewMultiDocument(items...),
		http.StatusOK,
		server.WriteSelfLink(r),
		server.WriteMeta("count", len(items)),
	)
}

func (relatedResourceResolver) isFetchResourceRequest(r *http.Request, ctx *jsonapi.RequestContext) bool {
	return r.Method == http.MethodGet &&
		ctx.ResourceID != "" &&
		ctx.Relationship == "" &&
		ctx.ResourceType != "" &&
		!ctx.Related
}

func (relatedResourceResolver) isRelatedResourceRequest(ctx *jsonapi.RequestContext) bool {
	return ctx.Related &&
		ctx.Relationship != "" &&
		ctx.ResourceID != "" &&
		ctx.ResourceType != ""
}

func (rr relatedResourceResolver) fetchResourceRelationships(r *http.Request,
	names []string, data *jsonapi.Resource, memo map[string]*jsonapi.Resource) error {
	if len(names) > 0 {
		name := names[0]

		if data.Relationships == nil {
			return nil
		}

		ref, ok := data.Relationships[name]

		if !ok || ref.Data == nil {
			return nil
		}

		items := ref.Data.Items()

		if len(items) == 0 {
			return nil
		}

		ctx := jsonapi.FromContext(r.Context())
		ctx = ctx.EmptyChild()
		ctx.FetchIDs = make([]string, 0, len(items))

		for idx, item := range items {
			if idx == 0 {
				ctx.ResourceType = item.Type
			}
			ctx.FetchIDs = append(ctx.FetchIDs, item.ID)
		}

		mem := server.NewRecorder()
		rr.handler.ServeHTTP(mem, jsonapi.RequestWithContext(r, ctx))

		if mem.Status != http.StatusOK {
			return nil
		} else if mem.Document == nil || mem.Document.Data == nil {
			return nil
		} else if len(mem.Document.Errors) > 0 {
			return mem.Document.Error()
		}

		items = mem.Document.Data.Items()

		if len(names) > 1 {
			names = names[1:]
		} else {
			names = []string{}
		}

		for _, item := range items {
			if err := rr.fetchResourceRelationships(r, names, item, memo); err != nil {
				return fmt.Errorf("include resources: %v: %w", names, err)
			}
		}
	} else {
		// names is empty, add data to the memo
		key := fmt.Sprintf("%s:%s", data.Type, data.ID)
		memo[key] = data
	}

	return nil
}
