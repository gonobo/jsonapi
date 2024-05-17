package middleware

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/query/filter"
	"github.com/gonobo/jsonapi/response"
)

// FilterQueryParser is a parser for JSON:API filter query parameters.
type FilterQueryParser interface {
	// ParseFilterQuery parses the filter query parameter from the request.
	ParseFilterQuery(*http.Request) (query.FilterExpression, error)
}

// ResolvesFilterParamsWithParser returns a middleware that uses the provided
// parser to parse the filter query parameter, adding it to the request context.
func ResolvesFilterParamsWithParser(parser FilterQueryParser) jsonapi.Middleware {
	return func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
		return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
			expr, err := parser.ParseFilterQuery(r)

			if err != nil {
				err = fmt.Errorf("failed to parse filter query: %w", err)
				return response.Error(http.StatusBadRequest, err)
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Filter = expr

			return next.ServeJSONAPI(jsonapi.RequestWithContext(r, ctx))
		})
	}
}

// ResolvesFilterParams returns a middleware uses the default parser to extract filter query parameters,
// adding it to the request context.
func ResolvesFilterParams() jsonapi.Middleware {
	return ResolvesFilterParamsWithParser(filter.NewRequestQueryParser())
}
