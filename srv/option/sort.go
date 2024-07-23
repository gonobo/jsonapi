package option

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/srv"
)

// SortQueryParser is used to parse sort query parameters.
type SortQueryParser interface {
	// ParseSortQuery parses the sort query parameters from the request.
	ParseSortQuery(*http.Request) ([]query.Sort, error)
}

// ResolvesSortQuery parses the sort parameters from the URL query and
// stores them within the JSON:API context.
func ResolvesSortQuery(parser SortQueryParser) srv.Options {
	return srv.UseMiddleware(func(next http.Handler) http.Handler {
		return sortParamResolver{next, parser}
	})
}

type sortParamResolver struct {
	http.Handler
	SortQueryParser
}

func (s sortParamResolver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sort, err := s.ParseSortQuery(r)

	if err != nil {
		srv.Error(w, fmt.Errorf("sort: failed to parse query params: %w", err), http.StatusBadRequest)
		return
	}

	ctx, _ := jsonapi.GetContext(r.Context())
	ctx.Sort = sort

	s.Handler.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
}
