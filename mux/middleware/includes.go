package middleware

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/mux"
	"github.com/gonobo/jsonapi/response"
)

// ResolvesIncludedResources adds middleware that resolves client include requests.
func ResolvesIncludedResources() mux.Options {
	return func(m *mux.Mux) {
		resolver := IncludeResolverMiddleware{Router: m}
		mux.WithMiddleware(resolver.Middleware())(m)
	}
}

// IncludeResolverMiddleware is a middleware that resolves included resources.
type IncludeResolverMiddleware struct {
	Router
}

// Middleware returns the middleware function.
func (i IncludeResolverMiddleware) Middleware() jsonapi.Middleware {
	return func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
		return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
			return i.resolveIncluded(next, r)
		})
	}
}

func (i IncludeResolverMiddleware) resolveIncluded(next jsonapi.RequestHandler, r *http.Request) jsonapi.Response {
	// allow downstream handler to resolve request
	res := next.ServeJSONAPI(r)

	ctx, _ := jsonapi.GetContext(r.Context())

	if r.Method != http.MethodGet {
		// nothing to do, not a fetch request
		return res
	} else if res.Code != http.StatusOK {
		// nothing to do, request was unsuccessful or does not return a resource
		return res
	} else if res.Body == nil || res.Body.Data == nil {
		// nothing to do, request payload is missing primary data
		return res
	}

	memo := make(map[string]*jsonapi.Resource)
	for _, item := range res.Body.Data.Items() {
		for _, include := range ctx.Include {

			err := i.fetchIncluded(r, include, memo, item)
			if err != nil {
				return response.Error(http.StatusInternalServerError, err)
			}
		}
	}

	for _, item := range memo {
		res.Body.Included = append(res.Body.Included, item)
	}

	return res
}

func (i IncludeResolverMiddleware) fetchIncluded(
	r *http.Request,
	include string,
	memo map[string]*jsonapi.Resource,
	item *jsonapi.Resource) error {

	if item.Relationships == nil {
		return fmt.Errorf("no relationships object")
	}

	ids := make([]string, 0)
	refs, ok := item.Relationships[include]
	if !ok {
		return fmt.Errorf("relationship %s not found", include)
	}

	var resourceType string

	items := refs.Data.Items()
	for idx, item := range items {
		if idx == 0 {
			resourceType = item.Type
		}
		ids = append(ids, item.ID)
	}

	ctx := jsonapi.RequestContext{
		ResourceType: resourceType,
		Related:      true,
		FetchIDs:     ids,
	}

	includeResponse := i.RouteJSONAPI(&ctx, jsonapi.RequestWithContext(r, &ctx))

	if includeResponse.Code != http.StatusOK {
		return fmt.Errorf("failed to retrieve included resources")
	} else if includeResponse.Body.Data != nil {
		return fmt.Errorf("failed to retrieve included resources")
	}

	for _, item := range includeResponse.Body.Data.Items() {
		key := fmt.Sprintf("%s:%s", item.Type, item.ID)
		memo[key] = item
	}

	return nil
}
