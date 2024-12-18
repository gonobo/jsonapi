package jsonapi

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi/v2/internal/null"
)

// URLResolver resolves urls based on the JSON:API request context.
type URLResolver interface {
	// ResolveURL creates a url based on the provided request context and base URL.
	ResolveURL(ctx RequestContext, baseURL string) string
}

// URLResolverFunc functions implement URLResolver.
type URLResolverFunc func(RequestContext, string) string

// ResolveURL creates a url based on the provided request context and base URL.
func (fn URLResolverFunc) ResolveURL(ctx RequestContext, baseURL string) string {
	return fn(ctx, baseURL)
}

// DefaultURLResolver returns a resolver that generates the following urls
// based on the request context (given a base url):
//
//	":base/:type" for search, create
//	":base/:type/:id" for fetch, update, delete
//	":base/:type/:id/relationships/:ref" for fetchRef
//	":base/:type/:id/:ref" for fetchRelated
func DefaultURLResolver() URLResolverFunc {
	return func(ctx RequestContext, baseURL string) string {
		if ctx.Relationship != "" && ctx.Related {
			return fmt.Sprintf("%s/%s/%s/%s",
				baseURL,
				ctx.ResourceType,
				ctx.ResourceID,
				ctx.Relationship,
			)
		}
		if ctx.Relationship != "" {
			return fmt.Sprintf("%s/%s/%s/relationships/%s",
				baseURL,
				ctx.ResourceType,
				ctx.ResourceID,
				ctx.Relationship,
			)
		}
		if ctx.ResourceID != "" {
			return fmt.Sprintf("%s/%s/%s",
				baseURL,
				ctx.ResourceType,
				ctx.ResourceID,
			)
		}
		return fmt.Sprintf("%s/%s", baseURL, ctx.ResourceType)
	}
}

// ContextResolver resolves JSON:API context information from an incoming http request.
type ContextResolver interface {
	// ResolveContext resolves the JSON:API context from the provided http request.
	// The context informs downstream dependencies; at minimum, the context should
	// populate the following fields (if applicable):
	//	- ResourceType (the resource type being requested)
	//	- ResourceID (the unique identifier of the resource being requested)
	//	- Relationship (the relationship being requested)
	//	- Related (whether the relationship is a related resource)
	ResolveContext(*http.Request) (*RequestContext, error)
}

// ContextResolverFunc functions implement ContextResolver.
type ContextResolverFunc func(*http.Request) (*RequestContext, error)

// ResolveContext resolves the JSON:API context from the provided http request.
func (fn ContextResolverFunc) ResolveContext(r *http.Request) (*RequestContext, error) {
	return fn(r)
}

// DefaultContextResolver returns a resolver that populates a JSON:API context
// based on the URL path examples given by the JSON:API specification:
//
//	"/:type"                         // ResourceType
//	"/:type/:id"                     // ResourceType, ResourceID
//	"/:type/:id/relationships/:ref"  // ResourceType, ResourceID, Relationship
//	"/:type/:id/:ref"                // ResourceType, ResourceID, Relationship, Related
func DefaultContextResolver() ContextResolverFunc {
	return ContextResolverWithPrefix("")
}

// ContextResolverWithPrefix returns a resolver that populates a JSON:API context
// based on the URL path examples given by the JSON:API specification:
//
//	"/prefix/:type"                         // ResourceType
//	"/prefix/:type/:id"                     // ResourceType, ResourceID
//	"/prefix/:type/:id/relationships/:ref"  // ResourceType, ResourceID, Relationship
//	"/prefix/:type/:id/:ref"                // ResourceType, ResourceID, Relationship, Related
//
// The specified prefix should start -- but not end -- with a forward slash.
func ContextResolverWithPrefix(prefix string) ContextResolverFunc {
	return func(r *http.Request) (*RequestContext, error) {
		var (
			mux = http.NewServeMux()
			w   = null.Writer{}
			h   = null.Handler{}
			ctx = RequestContext{}
		)

		// use the serve mux path parser to extract path values from the request url.
		mux.Handle(fmt.Sprintf("%s/{jsonapi_type}", prefix), h)
		mux.Handle(fmt.Sprintf("%s/{jsonapi_type}/{jsonapi_id}", prefix), h)
		mux.Handle(fmt.Sprintf("%s/{jsonapi_type}/{jsonapi_id}/relationships/{jsonapi_relationship}", prefix), h)
		mux.Handle(fmt.Sprintf("%s/{jsonapi_type}/{jsonapi_id}/{jsonapi_related}", prefix), h)
		mux.ServeHTTP(w, r)

		ctx.ResourceType = r.PathValue("jsonapi_type")
		if ctx.ResourceType == "" {
			return nil, jsonapiError("unspecified resource type")
		}

		ctx.ResourceID = r.PathValue("jsonapi_id")
		ctx.Relationship = r.PathValue("jsonapi_relationship")

		related := r.PathValue("jsonapi_related")
		if related != "" {
			ctx.Related = true
			ctx.Relationship = related
		}

		return &ctx, nil
	}
}
