package middleware

import (
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/query/fieldset"
	"github.com/gonobo/jsonapi/response"
)

// FieldsetQueryParser is a function that parses the fieldset query parameters.
type FieldsetQueryParser interface {
	// ParseFieldsetQuery parses the fieldset query parameters from the request.
	ParseFieldsetQuery(r *http.Request) ([]query.Fieldset, error)
}

// ResolvesFieldsetParamsWithParser resolves the fieldset parameters from the request.
func ResolvesFieldsetParamsWithParser(parser FieldsetQueryParser) jsonapi.Middleware {
	return func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
		return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
			criteria, err := parser.ParseFieldsetQuery(r)

			if err != nil {
				err = fmt.Errorf("failed to parse fieldset: %s", err)
				return response.Error(http.StatusBadRequest, err)
			}

			ctx, _ := jsonapi.GetContext(r.Context())
			ctx.Fields = criteria

			return next.ServeJSONAPI(jsonapi.RequestWithContext(r, ctx))
		})
	}
}

// ResolvesFieldsetParams resolves the fieldset parameters from the request
// using the default fieldset query parser.
func ResolvesFieldsetParams() jsonapi.Middleware {
	return ResolvesFieldsetParamsWithParser(fieldset.Parser{})
}
