package option

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/srv"
)

// FieldsetQueryParser is a function that parses the fieldset query parameters.
type FieldsetQueryParser interface {
	ParseFieldsetQuery(*http.Request) ([]query.Fieldset, error)
}

// WithFieldsetQueryParser is a middleware that resolves the fieldset parameters from the request
// and stores them within the JSON:API request context.
func WithFieldsetQueryParser(parser FieldsetQueryParser) srv.Options {
	return srv.WithMiddleware(resolveFieldsetParams(parser))
}

func resolveFieldsetParams(parser FieldsetQueryParser) srv.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fields, err := parser.ParseFieldsetQuery(r)

			if err != nil {
				srv.Error(w, fmt.Errorf("failed to parse fieldset params: %s", err), http.StatusBadRequest)
				return
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Fields = fields
			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	}
}
