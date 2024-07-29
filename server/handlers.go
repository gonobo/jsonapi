package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
)

type ctxkey string

const (
	ctxKeyResourceMux ctxkey = "resourceMuxCtxKey"
)

var (
	errMissingContext = errors.New("missing jsonapi context")
)

// Handler wraps an http handler, providing JSON:API context to downstream
// consumers.
type Handler struct {
	resolver jsonapi.RequestContextResolver
	handler  http.Handler
}

// ServeHTTP handles incoming http requests, and injects the JSON:API request context
// into the http request instance.
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, err := h.resolver.ResolveContext(r)

	if err != nil {
		Error(w, fmt.Errorf("failed to resolve context: %w", err), http.StatusInternalServerError)
		return
	}

	h.handler.ServeHTTP(w, jsonapi.RequestWithContext(r, &ctx))
}

// Handle returns a JSON:API handler, which wraps the provided http handler. The provided
// resolver supplies the JSON:API request context, which is injected into the http request's
// context when the handler is invoked. This context can then be retrieved downstream
// via jsonapi.GetContext().
func Handle(resolver jsonapi.RequestContextResolver, handler http.Handler) Handler {
	return Handler{resolver, handler}
}

// ResourceMux is similar to http.ServeMux, but instead of routing requests from URL patterns,
// routes are resolved by JSON:API resource type. Add resource handlers
// via the ResourceMux's HandleResource() method. When an incoming request
// is received, and the JSON:API request context is resolved, the handler that resolves
// the request will be determined by the request context's resource type.
//
// ResourceMux implements http.Handler and can be
// used directly with http.ListenAndServe() or any function that accepts
// an http.Handler.
type ResourceMux struct {
	Config
	handlers map[string]http.Handler
}

// New returns a newly initialized JSON:API server multiplexer.
// By default, ServeMux determines context from the request url:
//
//	":base/:type"                        // for resource type
//	":base/:type/:id"                    // for resource type and resource id
//	":base/:type/:id/relationships/:ref" // for resource type, id, and relationship name
//	":base/:type/:id/:ref"               // for type, id, relationship, and if the request is for related resources.
//
// This and other behaviors can be modified via configuration options.
func New(options ...Options) ResourceMux {
	cfg := DefaultConfig()
	cfg.Apply(options...)
	mux := ResourceMux{
		Config:   cfg,
		handlers: make(map[string]http.Handler),
	}
	return mux
}

// GetResourceMux returns the root resource mux stored within the request context.
// If the mux is not found, GetResourceMux() panics.
func GetResourceMux(r *http.Request) *ResourceMux {
	value := r.Context().Value(ctxKeyResourceMux)
	mux := value.(*ResourceMux)
	return mux
}

// SetResourceMux sets the provided resource mux to the request context.
func SetResourceMux(r *http.Request, m *ResourceMux) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKeyResourceMux, m)
	return r.WithContext(ctx)
}

// HandleResource adds a request handler to the mux. All requests associated with the provided resource type will be
// served by the provided handler.
func (m *ResourceMux) HandleResource(resource string, handler http.Handler) {
	m.handlers[resource] = handler
}

// ServeHTTP is the main entry point for incoming http requests.
func (m ResourceMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler http.Handler = http.HandlerFunc(m.ServeResourceHTTP)
	handler = m.applyMiddleware(handler)
	Handle(m.contextResolver, handler).ServeHTTP(w, SetResourceMux(r, &m))
}

// ServeResourceHTTP uses the embedded JSON:API request context to forward requests
// to their associated handler. If no handler is found, a 404 is returned to the client.
// ServeResourceHTTP is exported for test scenarios; it should not be invoked directly by
// library consumers.
func (m ResourceMux) ServeResourceHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, ok := jsonapi.GetContext(r.Context())
	if !ok {
		Error(w, errors.New("missing jsonapi context"), http.StatusInternalServerError)
	}

	resource, ok := m.handlers[ctx.ResourceType]
	if !ok {
		Error(w, errors.New("resource not found"), http.StatusNotFound)
	}

	resource.ServeHTTP(w, r)
}

// Resource is a collection of handlers for a single resource type.
// Each handler corresponds to a specific JSON:API resource operation,
// such as Create, List, Get, Update, etc. The request type is determined
// by the HTTP method and the JSON:API context. Resource instances
// should be used in conjunction with server.ServeMux, which resolves the
// JSON:API context before calling the ServeHTTP() method.
//
// The handlers are optional. If the corresponding request
// handler is nil, the request is rejected with a 404 Not Found response.
type Resource struct {
	Relationship http.Handler // Relationship handles requests to resource relationships.
	Create       http.Handler // Create handles requests to create new resources.
	List         http.Handler // List handles requests to list resource collections.
	Get          http.Handler // Get handles requests to fetch a specific resource.
	Update       http.Handler // Update handles requests to update a specific resource.
	Delete       http.Handler // Delete handles requests to delete a specific resource.
}

// ServeHTTP routes incoming JSON:API requests to the appropriate resource operation.
func (h Resource) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, ok := jsonapi.GetContext(r.Context())
	if !ok {
		Error(w, errors.New("missing jsonapi context"), http.StatusInternalServerError)
	}

	// in order of specificity:
	// 1) handle resource relationship requests
	// 2) handle resource requests
	// 3) handle resource collection requests

	if ctx.Relationship != "" && h.Relationship != nil {
		h.Relationship.ServeHTTP(w, r)
	}

	if ctx.ResourceID != "" {
		h.serveResource(w, r)
	}

	h.serveCollection(w, r)
}

// serveCollection handles incoming JSON:API requests for collections of resources.
func (h Resource) serveCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.List != nil {
			h.List.ServeHTTP(w, r)
			return
		}
	case http.MethodPost:
		if h.Create != nil {
			h.Create.ServeHTTP(w, r)
			return
		}
	}

	// no matches, return 404
	serveNotFound(w)
}

// serveResource handles incoming JSON:API requests for individual resources.
func (h Resource) serveResource(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.Get != nil {
			h.Get.ServeHTTP(w, r)
			return
		}
	case http.MethodPatch:
		if h.Update != nil {
			h.Update.ServeHTTP(w, r)
			return
		}
	case http.MethodDelete:
		if h.Delete != nil {
			h.Delete.ServeHTTP(w, r)
			return
		}
	}

	// no matches, return 404
	serveNotFound(w)
}

// serveNotFound returns a 404 error.
func serveNotFound(w http.ResponseWriter) {
	Error(w, errors.New("resource not found"), http.StatusNotFound)
}

// Relationship handlers route requests that correspond to a resource's relationships.
// Supported requests include GetRef, UpdateRef, AddRef, and RemoveRef. If the request does
// not match the JSON:API specifications for the above handlers, a 404 error is returned
// to the client.
type Relationship struct {
	GetRef    http.Handler // GetRef handles requests to fetch a specific resource relationship.
	UpdateRef http.Handler // UpdateRef handles requests to update a specific resource relationship.
	AddRef    http.Handler // AddRef handles requests to add a specific resource relationship.
	RemoveRef http.Handler // RemoveRef handles requests to remove a specific resource relationship.
}

// ServeHTTP handles incoming JSON:API requests for resource relationships.
func (h Relationship) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if h.GetRef != nil {
			h.GetRef.ServeHTTP(w, r)
			return
		}
	case http.MethodPost:
		if h.AddRef != nil {
			h.AddRef.ServeHTTP(w, r)
			return
		}
	case http.MethodPatch:
		if h.UpdateRef != nil {
			h.UpdateRef.ServeHTTP(w, r)
			return
		}
	case http.MethodDelete:
		if h.RemoveRef != nil {
			h.RemoveRef.ServeHTTP(w, r)
			return
		}
	}

	// no matches, return 404
	serveNotFound(w)
}

// RelationshipMux is a http handler multiplexor for a resource's relationships.
// Each handler within the mux corresponds to a specific relationship, indexed
// the relationship name. If the relationship is not defined, a 404 error is
// returned to the client.
type RelationshipMux map[string]http.Handler

// ServeHTTP handles incoming JSON:API requests for resource relationships.
func (h RelationshipMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, ok := jsonapi.GetContext(r.Context())
	if !ok {
		Error(w, errMissingContext, http.StatusInternalServerError)
		return
	}

	handler, ok := h[ctx.Relationship]
	if !ok {
		serveNotFound(w)
		return
	}

	handler.ServeHTTP(w, r)
}
