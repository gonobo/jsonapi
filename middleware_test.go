package jsonapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	type testcase struct {
		name       string
		middleware jsonapi.Middleware
		req        *http.Request
		want       int
	}

	for _, tc := range []testcase{
		{
			name:       "passthrough",
			middleware: jsonapi.Passthrough(),
			req:        httptest.NewRequest(http.MethodGet, "/", nil),
			want:       http.StatusOK,
		},
		{
			name: "use middleware that manipulates the response",
			middleware: jsonapi.Passthrough().Use(
				func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
					return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
						res := next.ServeJSONAPI(r)
						res.Code = http.StatusOK
						return res
					})
				},
			),
			req:  httptest.NewRequest(http.MethodGet, "/", nil),
			want: http.StatusOK,
		},
		{
			name: "use middleware that returns the response early",
			middleware: jsonapi.Passthrough().Use(
				func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
					return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
						res := jsonapi.NewResponse(http.StatusAccepted)
						return res
					})
				},
			),
			req:  httptest.NewRequest(http.MethodGet, "/", nil),
			want: http.StatusAccepted,
		},
		{
			name: "use middleware that modifies the request",
			middleware: jsonapi.Passthrough().Use(
				func(next jsonapi.RequestHandler) jsonapi.RequestHandler {
					return jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
						type key string
						ctx := context.WithValue(r.Context(), key("key"), "value")
						return next.ServeJSONAPI(r.WithContext(ctx))
					})
				},
			),
			req:  httptest.NewRequest(http.MethodGet, "/", nil),
			want: http.StatusOK,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var handler jsonapi.RequestHandler = jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
				return jsonapi.NewResponse(http.StatusOK)
			})
			handler = tc.middleware.Wrap(handler)
			got := handler.ServeJSONAPI(tc.req)
			assert.EqualValues(t, tc.want, got.Code)
		})
	}
}
