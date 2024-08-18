package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gonobo/jsonapi/v1"
	"github.com/gonobo/jsonapi/v1/server"
)

func UseRequestBodyParser() server.Options {
	return server.WithMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()

			document := jsonapi.Document{}

			if err := json.NewDecoder(r.Body).Decode(&document); errors.Is(err, io.EOF) {
				// no document inside the payload; execute the next handler
				next.ServeHTTP(w, r)
				return
			} else if err != nil {
				// document parsing failed; return error to client
				server.Error(w, fmt.Errorf("request document error: %w", err), http.StatusBadRequest)
				return
			}

			// save document to context and continue
			ctx, _ := jsonapi.FromContext(r.Context())
			ctx.Document = &document
			next.ServeHTTP(w, jsonapi.RequestWithContext(r, ctx))
		})
	})
}
