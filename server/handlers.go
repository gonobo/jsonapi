package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi/v2"
)

var (
	errNotFound = errors.New("resource not found")
)

// Handler wraps an http handler, providing JSON:API context to downstream
// consumers.
//
// By default, Handler determines context from the request url:
//
//	":base/:type"                        // for resource type
//	":base/:type/:id"                    // for resource type and resource id
//	":base/:type/:id/relationships/:ref" // for resource type, id, and relationship name
//	":base/:type/:id/:ref"               // for type, id, relationship, and if the request is for related resources.
//
// This behavior and others can be modified via configuration [Options].
// Handler implements [http.Handler] and can be used anywhere http handlers are accepted.
//
// Handler zero values are not initialized; use the [Handle] function to create
// a new instance.
type Handler struct {
	Config               // Configuration options for the handler.
	wrapped http.Handler // The underlying http handler.
}

// ServeHTTP handles incoming http requests, and injects the JSON:API request context
// into the http request instance.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, err := h.contextResolver.ResolveContext(r)

	if err != nil {
		Error(w, fmt.Errorf("failed to resolve context: %w", err), http.StatusInternalServerError)
		return
	}

	h.wrapped.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
}

// Handle returns a [Handler], which wraps the provided http handler. The provided
// resolver supplies the JSON:API request context, which is injected into the http request's
// context when the handler is invoked. This context can then be retrieved downstream
// via jsonapi.GetContext().
func Handle(handler http.Handler, options ...Options) Handler {
	cfg := DefaultConfig()
	cfg.Apply(options...)

	wrapped := cfg.applyMiddleware(handler)
	return Handler{cfg, wrapped}
}

// ResourceMux is similar to [http.ServeMux], but instead of routing requests directly from URL patterns,
// routes are determined by JSON:API context resource type. Add resource handlers explicitly or
// via the ResourceMux's Handle() method. When an incoming request
// is received, and the JSON:API request context is resolved, the handler that resolves
// the request will be determined by the request context's resource type. Undefined
// resource requests will return 404 responses.
//
// ResourceMux must have be wrapped by or have a parent handler wrapped by [Handle]
// to provide JSON:API request context.
type ResourceMux map[string]http.Handler

// Handle adds a request handler to the mux. All requests associated with the provided resource type will be
// served by the provided handler.
func (m ResourceMux) Handle(resource string, handler http.Handler) {
	m[resource] = handler
}

// ServeHTTP uses the embedded JSON:API request context to forward requests
// to their associated handler. If no handler is found, a 404 is returned to the client.
func (m ResourceMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := jsonapi.FromContext(r.Context())
	resource, ok := m[ctx.ResourceType]
	serveIfNotNil(w, r, resource, !ok)
}

// Resource is a collection of handlers for resource endpoints.
// Each handler corresponds to a specific JSON:API resource operation,
// such as Create, List, Get, Update, etc. The request type is determined
// by the HTTP method and the JSON:API context. Resource instances
// are intended to be used in conjunction with [ResourceMux].
//
// Resource must be wrapped by or have a parent handler wrapped by [Handle]
// to provide JSON:API request context.
//
// The handler members are optional. If the corresponding request
// handler is nil, the request is rejected with a 404 Not Found response.
//
// If the request method does not conform to the JSON:API specification,
// the request is rejected with a 405 Method Not Allowed response.
type Resource struct {
	Relationships http.Handler // Relationships handles requests to resource relationships.
	Create        http.Handler // Create handles requests to create new resources.
	List          http.Handler // List handles requests to list resource collections.
	Get           http.Handler // Get handles requests to fetch a specific resource.
	Update        http.Handler // Update handles requests to update a specific resource.
	Delete        http.Handler // Delete handles requests to delete a specific resource.
}

// ServeHTTP routes incoming JSON:API requests to the appropriate resource operation.
func (h Resource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := jsonapi.FromContext(r.Context())

	// in order of specificity:
	// 1) handle resource relationship requests
	// 2) handle resource requests
	// 3) handle resource collection requests

	if ctx.Relationship != "" && h.Relationships != nil {
		h.Relationships.ServeHTTP(w, r)
		return
	}

	if ctx.ResourceID != "" {
		h.serveResource(w, r)
		return
	}

	h.serveCollection(w, r)
}

// serveCollection handles incoming JSON:API requests for collections of resources.
func (h Resource) serveCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serveIfNotNil(w, r, h.List, h.List == nil)
		return
	case http.MethodPost:
		serveIfNotNil(w, r, h.Create, h.Create == nil)
		return
	}

	// no matches, return 405
	methodNotAllowed(w)
}

// serveResource handles incoming JSON:API requests for individual resources.
func (h Resource) serveResource(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serveIfNotNil(w, r, h.Get, h.Get == nil)
		return
	case http.MethodPatch:
		serveIfNotNil(w, r, h.Update, h.Update == nil)
		return
	case http.MethodDelete:
		serveIfNotNil(w, r, h.Delete, h.Delete == nil)
		return
	}

	// no matches, return 405
	methodNotAllowed(w)
}

func serveIfNotNil(w http.ResponseWriter, r *http.Request, h http.Handler, isNil bool) {
	if isNil {
		notFound(w)
		return
	}
	h.ServeHTTP(w, r)
}

// notFound returns a 404 error.
func notFound(w http.ResponseWriter) {
	Error(w, errNotFound, http.StatusNotFound)
}

func methodNotAllowed(w http.ResponseWriter) {
	Error(w, errors.New("method not allowed"), http.StatusMethodNotAllowed)
}

// Relationship handlers route requests that correspond to a resource's relationships.
// Supported requests include GetRef, UpdateRef, AddRef, and RemoveRef. If the request does
// not match the JSON:API specifications for the above handlers, a 404 error is returned
// to the client
//
// Relationship instances can be used alone or as a handler to a [Resource] instance's Ref field.
type Relationship struct {
	Get       http.Handler // Get handles requests to fetch a specific resource relationship.
	Update    http.Handler // Update handles requests to update a specific resource relationship.
	AddRef    http.Handler // AddRef handles requests to add references to a specific resource relationship.
	RemoveRef http.Handler // RemoveRef handles requests to remove references from a specific resource relationship.
}

// ServeHTTP handles incoming JSON:API requests for resource relationships.
func (h Relationship) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		serveIfNotNil(w, r, h.Get, h.Get == nil)
		return
	case http.MethodPost:
		serveIfNotNil(w, r, h.AddRef, h.AddRef == nil)
		return
	case http.MethodPatch:
		serveIfNotNil(w, r, h.Update, h.Update == nil)
		return
	case http.MethodDelete:
		serveIfNotNil(w, r, h.RemoveRef, h.RemoveRef == nil)
		return
	}

	// no matches, return 405
	methodNotAllowed(w)
}

// RelationshipMux is a http handler multiplexer for a resource's relationships.
// Each handler within the mux corresponds to a specific relationship, indexed
// the relationship name. If the relationship is not defined, a 404 error is
// returned to the client.
//
// RelationshipMux must be wrapped by or have a parent handler wrapped by [Handle]
// to provide JSON:API request context.
type RelationshipMux map[string]http.Handler

// ServeHTTP handles incoming JSON:API requests for resource relationships.
func (h RelationshipMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := jsonapi.FromContext(r.Context())
	handler, ok := h[ctx.Relationship]
	serveIfNotNil(w, r, handler, !ok)
}
