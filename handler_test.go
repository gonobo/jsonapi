package jsonapi_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/jsonapitest"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	type testcase struct {
		name       string
		req        *http.Request
		handler    jsonapi.RequestHandler
		options    []func(*jsonapi.H)
		wantStatus int
	}

	for _, tc := range []testcase{
		{
			name: "handler with default resolver returns error - empty path",
			req:  httptest.NewRequest("GET", "http://example.com", nil),
			handler: jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
				res := jsonapi.NewResponse(http.StatusOK)
				return res
			}),
			options:    nil,
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "handler with default resolver returns ok - resolver override",
			req:  httptest.NewRequest("GET", "http://example.com", nil),
			handler: jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
				res := jsonapi.NewResponse(http.StatusOK)
				return res
			}),
			options: []func(*jsonapi.H){func(h *jsonapi.H) {
				h.RequestContextResolver = jsonapi.RequestContextResolverFunc(
					func(r *http.Request) (jsonapi.RequestContext, error) {
						return jsonapi.RequestContext{}, nil
					},
				)
			}},
			wantStatus: http.StatusOK,
		},
		{
			name: "handler func returns ok - get request",
			req:  httptest.NewRequest("GET", "http://example.com/items/1", nil),
			handler: jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
				res := jsonapi.NewResponse(http.StatusOK)
				return res
			}),
			options:    nil,
			wantStatus: http.StatusOK,
		},
		{
			name: "handler func returns ok - create request",
			req: jsonapitest.NewRequest("POST", "http://example.com/items",
				*jsonapi.NewSingleDocument(&jsonapi.Resource{ID: "1", Type: "items"}),
			),
			handler: jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
				res := jsonapi.NewResponse(http.StatusCreated)
				return res
			}),
			options:    nil,
			wantStatus: http.StatusCreated,
		},
		{
			name: "handler func returns bad request - invalid payload",
			req:  httptest.NewRequest("POST", "http://example.com/items", bytes.NewBufferString("[]")),
			handler: jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
				res := jsonapi.NewResponse(http.StatusCreated)
				return res
			}),
			options:    nil,
			wantStatus: http.StatusBadRequest,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := jsonapi.Handler(tc.handler, tc.options...)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, tc.req)
			if !assert.Equal(t, tc.wantStatus, w.Code) {
				if w.Body != nil {
					t.Logf(w.Body.String())
				}
			}
		})
	}
}

func TestRouteJSONAPI(t *testing.T) {
	handler := jsonapi.HandlerFunc(func(r *http.Request) jsonapi.Response {
		res := jsonapi.NewResponse(http.StatusOK)
		return res
	})
	r, err := http.NewRequest("GET", "http://example.com/foos/42", nil)

	if err != nil {
		t.Fatal(err)
		return
	}

	res := handler.RouteJSONAPI(&jsonapi.RequestContext{
		ResourceType: "foos",
		ResourceID:   "42",
	}, r)

	assert.Equal(t, http.StatusOK, res.Code)
}
