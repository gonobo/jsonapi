package middleware

import (
	"net/http"

	"github.com/gonobo/jsonapi"
)

// LocationHeaderProvider provides a location header for a resource node.
type LocationHeaderProvider interface {
	// LocationHeader returns the location header for the provided resource node.
	LocationHeader(*jsonapi.Resource) string
}

// ProvidesResourceLocationHeader returns a middleware that provides the location header for
// a newly created resource node.
func ProvidesResourceLocationHeader(provider LocationHeaderProvider) jsonapi.Middleware {
	return func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
		return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
			res := next.ServeJSONAPI(r)
			if res.Code == http.StatusCreated && res.Body != nil && res.Body.Data != nil {
				item := res.Body.Data.Items()[0]
				location := provider.LocationHeader(item)
				res.Headers["Location"] = location
			}
			return res
		})
	}
}

// LocationProviderFunc is a function that returns a location header for a resource node.
type LocationProviderFunc func(*jsonapi.Resource) string

// LocationHeader returns the location header for the provided resource node.
func (f LocationProviderFunc) LocationHeader(node *jsonapi.Resource) string {
	return f(node)
}

// DefaultLocationProvider returns a default location provider that
// uses the provided base URL and JSONAPI URL resolver to generate a Location
// header in responses. Given a base url https://api.example.com
// with resource type "users" and id "123" the location header will be:
//
//	"https://api.example.com/users/123"
func DefaultLocationProvider(baseURL string) LocationProviderFunc {
	return func(rn *jsonapi.Resource) string {
		resolver := jsonapi.DefaultURLResolver()
		return resolver.ResolveURL(jsonapi.RequestContext{
			ResourceType: rn.Type,
			ResourceID:   rn.ID,
		}, baseURL)
	}
}
