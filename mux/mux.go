package mux

import (
	"errors"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/response"
)

// Options are options for configuring a Muxer.
type Options = func(*Mux)

// Mux is a multiplexer for JSONAPI requests. Routes are determined
// by the request resource type. Mux implements http.Handler and can be
// used directly with http.ListenAndServe() or any function that accepts
// an http.Handler.
//
// The muxer is configured with a set of default options. To modify these
// options, use the MuxOptions:
//
//	router := server.New(
//		server.WithRequestContextResolver(CustomContextResolver{}),
//		server.WithMiddleware(LoggingMiddleware{}),
//		server.WithMiddleware(middleware.ResolvesSortParams()),
//		server.WithMiddleware(middleware.ResolvesFilterParams()),
//		server.WithMiddleware(middleware.ResolvesPagination()),
//		middleware.ResolvesIncludes(),
//		middleware.ResolvesRelatedResources(),
//		server.WithRoute("posts", PostHandler{}),
//		server.WithRoute("users", server.ResourceRoute{
//			Create: UserCreateHandler{},
//			Update: UserUpdateHandler{},
//			Delete: UserDeleteHandler{},
//			Get:    UserGetHandler{},
//			List:   UserListHandler{},
//		}),
//	)
type Mux struct {
	middleware jsonapi.Middleware                // Optional middleware stack.
	handlers   map[string]jsonapi.RequestHandler // The internal mapping of resource types to their associated handler.
}

// New creates a new Mux with the given options.
//
// By default, Mux determines context from the request url:
//
//	":base/:type"                        // for resource type
//	":base/:type/:id"                    // for resource type and resource id
//	":base/:type/:id/relationships/:ref" // for resource type, id, and relationship name
//	":base/:type/:id/:ref"               // for type, id, relationship, and if the request is for related resources.
//
// This and other behaviors can be modified via configuration options.
// Request instance.
func New(options ...Options) Mux {
	router := Mux{
		handlers:   make(map[string]jsonapi.RequestHandler),
		middleware: jsonapi.Passthrough(),
	}
	for _, option := range options {
		option(&router)
	}
	return router
}

// WithRoute associates the handler with the supplied resource type.
func WithRoute(resourceType string, handler jsonapi.RequestHandler) Options {
	return func(m *Mux) {
		m.handlers[resourceType] = handler
	}
}

// WithMiddleware adds middleware to the muxer.
func WithMiddleware(middleware jsonapi.Middleware) Options {
	return func(m *Mux) {
		m.middleware = m.middleware.Use(middleware)
	}
}

// ServeJSONAPI handles a JSON:API request.
func (m Mux) ServeJSONAPI(r *http.Request) jsonapi.Response {
	handler := m.middleware.Wrap(jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
		ctx, ok := jsonapi.GetContext(r.Context())
		if !ok {
			return response.InternalError(errors.New("missing jsonapi context"))
		}
		return m.RouteJSONAPI(ctx, r)
	}))
	return handler.ServeJSONAPI(r)
}

// RouteJSONAPI routes a JSON:API request to the appropriate handler.
func (m Mux) RouteJSONAPI(ctx *jsonapi.RequestContext, r *http.Request) jsonapi.Response {
	// choose the handler corresponding to the request resource type
	handler, ok := m.handlers[ctx.ResourceType]
	if !ok {
		return response.ResourceNotFound(ctx)
	}

	// finally, handle the request.
	return handler.ServeJSONAPI(r)
}
