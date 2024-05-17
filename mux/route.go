package mux

import (
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/response"
)

// Route is a collection of handlers for a single resource type.
// Each handler corresponds to a specific JSON:API resource operation,
// such as Create, List, Get, Update, etc. The request type is determined
// by the HTTP method and the JSON:API context.
//
// The handlers are optional. If the corresponding request
// handler is nil, the request is rejected with a 404 Not Found response.
type Route struct {
	Create    jsonapi.RequestHandler // Create handles requests to create new resources.
	List      jsonapi.RequestHandler // List handles requests to list resource collections.
	Get       jsonapi.RequestHandler // Get handles requests to fetch a specific resource.
	GetRef    jsonapi.RequestHandler // GetRef handles requests to fetch a specific resource relationship.
	Update    jsonapi.RequestHandler // Update handles requests to update a specific resource.
	UpdateRef jsonapi.RequestHandler // UpdateRef handles requests to update a specific resource relationship.
	AddRef    jsonapi.RequestHandler // AddRef handles requests to add a specific resource relationship.
	RemoveRef jsonapi.RequestHandler // RemoveRef handles requests to remove a specific resource relationship.
	Delete    jsonapi.RequestHandler // Delete handles requests to delete a specific resource.
}

// ServeJSONAPI handles incoming JSON:API requests.
func (r Route) ServeJSONAPI(req *http.Request) jsonapi.Response {
	ctx, _ := jsonapi.GetContext(req.Context())
	return r.RouteJSONAPI(ctx, req)
}

// RouteJSONAPI routes incoming JSON:API requests to the appropriate resource operation.
func (r Route) RouteJSONAPI(ctx *jsonapi.RequestContext, req *http.Request) jsonapi.Response {
	if ctx.Relationship != "" {
		return r.serveRelationship(ctx, req)
	}
	if ctx.ResourceID != "" {
		return r.serveResource(ctx, req)
	}
	return r.serveCollection(ctx, req)
}

// serveRelationship handles incoming JSON:API requests for resource relationships.
func (r Route) serveRelationship(ctx *jsonapi.RequestContext, req *http.Request) jsonapi.Response {
	switch req.Method {
	case http.MethodGet:
		if r.GetRef != nil {
			return r.GetRef.ServeJSONAPI(req)
		}
	case http.MethodPost:
		if r.AddRef != nil {
			return r.AddRef.ServeJSONAPI(req)
		}
	case http.MethodPatch:
		if r.UpdateRef != nil {
			r.UpdateRef.ServeJSONAPI(req)
		}
	case http.MethodDelete:
		if r.RemoveRef != nil {
			return r.RemoveRef.ServeJSONAPI(req)
		}
	}
	return response.ResourceNotFound(ctx)
}

// serveResource handles incoming JSON:API requests for individual resources.
func (r Route) serveResource(ctx *jsonapi.RequestContext, req *http.Request) jsonapi.Response {
	switch req.Method {
	case http.MethodGet:
		if r.Get != nil {
			return r.GetRef.ServeJSONAPI(req)
		}
	case http.MethodPatch:
		if r.Update != nil {
			return r.Update.ServeJSONAPI(req)
		}
	case http.MethodDelete:
		if r.Delete != nil {
			return r.Delete.ServeJSONAPI(req)
		}
	}

	return response.ResourceNotFound(ctx)
}

// serveCollection handles incoming JSON:API requests for collections of resources.
func (r Route) serveCollection(ctx *jsonapi.RequestContext, req *http.Request) jsonapi.Response {
	switch req.Method {
	case http.MethodGet:
		if r.List != nil {
			return r.List.ServeJSONAPI(req)
		}
	case http.MethodPost:
		if r.Create != nil {
			return r.Create.ServeJSONAPI(req)
		}
	}
	return response.ResourceNotFound(ctx)
}
