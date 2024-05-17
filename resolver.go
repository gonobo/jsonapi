package jsonapi

import (
	"fmt"
	"net/http"
	"path"
	"strings"
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

// RequestContextResolver resolves JSON:API context information from an incoming http request.
type RequestContextResolver interface {
	// ResolveContext resolves the JSON:API context from the provided http request.
	// The context informs downstream dependencies; at minimum, the context should
	// populate the following fields (if applicable):
	//	- ResourceType (the resource type being requested)
	//	- ResourceID (the unique identifier of the resource being requested)
	//	- Relationship (the relationship being requested)
	//	- Related (whether the relationship is a related resource)
	ResolveContext(*http.Request) (RequestContext, error)
}

// RequestContextResolverFunc functions implement ContextResolver.
type RequestContextResolverFunc func(*http.Request) (RequestContext, error)

// ResolveContext resolves the JSON:API context from the provided http request.
func (fn RequestContextResolverFunc) ResolveContext(r *http.Request) (RequestContext, error) {
	return fn(r)
}

// DefaultRequestContextResolver returns a resolver that populates a JSON:API context
// based on the URL path examples given by the JSON:API specification:
//
//	"/:type"                         // ResourceType
//	"/:type/:id"                     // ResourceType, ResourceID
//	"/:type/:id/relationships/:ref"  // ResourceType, ResourceID, Relationship
//	"/:type/:id/:ref"                // ResourceType, ResourceID, Relationship, Related
func DefaultRequestContextResolver() RequestContextResolverFunc {
	return func(r *http.Request) (RequestContext, error) {
		// remove the leading slash from the path so we can count segments
		urlPath := strings.TrimLeft(r.URL.Path, "/")
		segments := strings.Split(urlPath, "/")

		var ctx RequestContext

		if urlPath == "" {
			return ctx, jsonapiError("empty path")
		}

		if ok, _ := path.Match("*/*/relationships/*", urlPath); ok {
			// :type/:id/relationships/:relationship
			ctx.ResourceType = segments[0]
			ctx.ResourceID = segments[1]
			ctx.Relationship = segments[3]
		} else if ok, _ := path.Match("*/*/*", urlPath); ok {
			// :type/:id/:relationship
			ctx.ResourceType = segments[0]
			ctx.ResourceID = segments[1]
			ctx.Relationship = segments[2]
			ctx.Related = true
		} else if ok, _ := path.Match("*/*", urlPath); ok {
			// :type/:id
			ctx.ResourceType = segments[0]
			ctx.ResourceID = segments[1]
		}

		ctx.ResourceType = segments[0]
		return ctx, nil
	}
}
