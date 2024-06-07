package middleware

import (
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/mux"
	"github.com/gonobo/jsonapi/response"
)

// ResolvesRelatedResources adds middleware that resolves related resource requests.
func ResolvesRelatedResources() mux.Options {
	return func(m *mux.Mux) {
		resolver := RelatedResourceResolverMiddleware{Router: m}
		mux.WithMiddleware(resolver.middleware())(m)
	}
}

type Router interface {
	RouteJSONAPI(*jsonapi.RequestContext, *http.Request) jsonapi.Response
}

// RelatedResourceResolverMiddleware is a middleware that resolves related resources.
type RelatedResourceResolverMiddleware struct {
	Router
}

// Middleware returns the middleware function.
func (r RelatedResourceResolverMiddleware) middleware() jsonapi.Middleware {
	return func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
		return jsonapi.HandlerFunc(func(req *http.Request) jsonapi.Response {
			return r.resolveRelated(next, req)
		})
	}
}

func (r RelatedResourceResolverMiddleware) resolveRelated(next jsonapi.RequestHandler, req *http.Request) jsonapi.Response {
	ctx, _ := jsonapi.GetContext(req.Context())

	if !ctx.Related {
		// this is not a related resource request; quit early and
		// let downstream handlers resolve.
		return next.ServeJSONAPI(req)
	}

	// create a new context that fetches the target resource
	resourceCtx := ctx.EmptyChild()
	resourceCtx.ResourceType = ctx.ResourceType
	resourceCtx.ResourceID = ctx.ResourceID
	resourceCtx.Relationship = ctx.Relationship

	// retrieve the resource resourceHandler
	// fetch the resource relationship references
	parent := r.RouteJSONAPI(ctx, jsonapi.RequestWithContext(req, resourceCtx))

	if parent.Code != http.StatusOK {
		// the request failed, there is nothing to do.
		return parent
	} else if parent.Body == nil || parent.Body.Data == nil {
		// if the primary data is missing or empty,
		// there is nothing to do.
		return parent
	}

	// inspect the primary data
	ids := make([]string, 0)
	resourceType := ""

	// extract ids and resource type from the relationship
	for idx, item := range parent.Body.Data.Items() {
		if idx == 0 {
			resourceType = item.Type
		}
		ids = append(ids, item.ID)
	}

	if len(ids) == 0 {
		// there are no references to resolve, return an empty resource list
		return response.Ok(&jsonapi.Document{
			Links: parent.Body.Links,
			Meta:  parent.Body.Meta,
		})
	}

	// create a new context that includes the related ids to search for.
	// clear the relationship and resource id, while updating
	// the resource type so it corresponds to the related resource.
	relatedCtx := ctx.Child()
	relatedCtx.FetchIDs = ids
	relatedCtx.ResourceType = resourceType
	relatedCtx.Relationship = ""
	relatedCtx.ResourceID = ""
	relatedCtx.Related = false

	// serve a find related request
	return r.RouteJSONAPI(relatedCtx, jsonapi.RequestWithContext(req, relatedCtx))
}
