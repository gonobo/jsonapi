package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/server"
)

const (
	QueryParamInclude = "include"
)

// FieldsetQueryParser is a function that parses the fieldset query parameters.
type FieldsetQueryParser interface {
	ParseFieldsetQuery(*http.Request) ([]query.Fieldset, error)
}

// UseFieldsetQueryParser is a middleware that resolves the fieldset parameters from the request
// and stores them within the JSON:API request context.
func UseFieldsetQueryParser(parser FieldsetQueryParser) server.Options {
	return server.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fields, err := parser.ParseFieldsetQuery(r)

			if err != nil {
				server.Error(w, fmt.Errorf("failed to parse fieldset params: %s", err), http.StatusBadRequest)
				return
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Fields = fields
			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	})
}

// PageQueryParser is used to parse sort query parameters.
type PageQueryParser interface {
	// ParsePageQuery parses the sort query parameters from the request.
	ParsePageQuery(*http.Request) (query.Page, error)
}

// UsePaginationQueryParser is a middleware that parses the sort parameters from the URL query and
// stores them within the JSON:API context.
func UsePaginationQueryParser(parser PageQueryParser) server.Options {
	return server.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			page, err := parser.ParsePageQuery(r)

			if err != nil {
				server.Error(w, fmt.Errorf("sort: failed to parse query params: %w", err), http.StatusBadRequest)
				return
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Pagination = page

			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	})
}

// FilterQueryParser is a parser for JSON:API filter query parameters.
type FilterQueryParser interface {
	// ParseFilterQuery parses the filter query parameter from the request.
	ParseFilterQuery(*http.Request) (query.FilterExpression, error)
}

// UseFilterQueryParser is a middleware that parses and extracts any filter parameters in the
// request query and generates a filter expression stored within
// the JSON:API request context.
func UseFilterQueryParser(parser FilterQueryParser) server.Options {
	return server.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			filter, err := parser.ParseFilterQuery(r)

			if err != nil {
				server.Error(w, fmt.Errorf("failed to parse filter params: %s", err), http.StatusBadRequest)
				return
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Filter = filter
			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	})
}

// SortQueryParser is used to parse sort query parameters.
type SortQueryParser interface {
	// ParseSortQuery parses the sort query parameters from the request.
	ParseSortQuery(*http.Request) ([]query.Sort, error)
}

// UseSortQueryParser parses the sort parameters from the URL query and
// stores them within the JSON:API context.
func UseSortQueryParser(parser SortQueryParser) server.Options {
	return server.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sort, err := parser.ParseSortQuery(r)

			if err != nil {
				server.Error(w, fmt.Errorf("sort: failed to parse query params: %w", err), http.StatusBadRequest)
				return
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Sort = sort

			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	})
}

// UseIncludeQueryParser is a middleware that parses the list of included
// resources requested by the client and adds them to the JSON:API context.
func UseIncludeQueryParser() server.Options {
	return server.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			include := strings.Split(r.URL.Query().Get(QueryParamInclude), ",")
			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Include = include
			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	})
}
