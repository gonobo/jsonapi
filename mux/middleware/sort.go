package middleware

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/query/sort"
	"github.com/gonobo/jsonapi/response"
)

// SortQueryParser is used to parse sort query parameters.
type SortQueryParser interface {
	// ParseSortQuery parses the sort query parameters from the request.
	ParseSortQuery(*http.Request) ([]query.Sort, error)
}

// ResolvesSortParamsWithParser resolves the sort parameters from the request.
func ResolvesSortParamsWithParser(parser SortQueryParser) jsonapi.Middleware {
	return func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
		return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
			criteria, err := parser.ParseSortQuery(r)

			if err != nil {
				err = fmt.Errorf("failed to parse sort query: %s", err.Error())
				return response.Error(http.StatusBadRequest, err)
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Sort = criteria

			return next.ServeJSONAPI(jsonapi.RequestWithContext(r, ctx))
		})
	}
}

// ResolvesSortParams resolves the sort parameters from the request using
// the default sort query parser.
func ResolvesSortParams() jsonapi.Middleware {
	return ResolvesSortParamsWithParser(sort.NewRequestQueryParser())
}
