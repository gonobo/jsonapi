package option

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/srv"
)

// PageQueryParser is used to parse sort query parameters.
type PageQueryParser interface {
	// ParsePageQuery parses the sort query parameters from the request.
	ParsePageQuery(*http.Request) (query.Page, error)
}

// WithPaginationQueryParser parses the sort parameters from the URL query and
// stores them within the JSON:API context.
func WithPaginationQueryParser(parser PageQueryParser) srv.Options {
	return srv.WithMiddleware(func(next http.Handler) http.Handler {
		return pageParamResolver{next, parser}
	})
}

type pageParamResolver struct {
	http.Handler
	PageQueryParser
}

func (s pageParamResolver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	page, err := s.ParsePageQuery(r)

	if err != nil {
		srv.Error(w, fmt.Errorf("sort: failed to parse query params: %w", err), http.StatusBadRequest)
		return
	}

	ctx, _ := jsonapi.GetContext(r.Context())
	ctx.Pagination = page

	s.Handler.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
}
