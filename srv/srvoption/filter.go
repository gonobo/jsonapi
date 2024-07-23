package srvoption

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/srv"
)

// FilterQueryParser is a parser for JSON:API filter query parameters.
type FilterQueryParser interface {
	// ParseFilterQuery parses the filter query parameter from the request.
	ParseFilterQuery(*http.Request) (query.FilterExpression, error)
}

func ResolvesFilterParams(parser FilterQueryParser) srv.Options {
	return srv.UseMiddleware(resolveFilterParams(parser))
}

func resolveFilterParams(parser FilterQueryParser) srv.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filter, err := parser.ParseFilterQuery(r)

			if err != nil {
				srv.Error(w, fmt.Errorf("failed to parse filter params: %s", err), http.StatusBadRequest)
				return
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Filter = filter
			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	}
}
