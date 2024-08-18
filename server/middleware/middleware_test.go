package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi/v1"
	"github.com/gonobo/jsonapi/v1/query"
	"github.com/gonobo/jsonapi/v1/query/page"
	"github.com/gonobo/jsonapi/v1/server"
	"github.com/gonobo/jsonapi/v1/server/middleware"
	"github.com/stretchr/testify/assert"
)

type testcase struct {
	name       string
	options    []server.Options
	req        *http.Request
	muxconfig  func(*testing.T, *server.ResourceMux)
	wantStatus int
	wantBody   string
}

func (tc testcase) run(t *testing.T) {
	t.Run(tc.name, func(t *testing.T) {
		mux := server.ResourceMux{}
		handler := server.Handle(mux, tc.options...)
		tc.muxconfig(t, &mux)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, tc.req)

		gotStatus := w.Result().StatusCode
		assert.Equal(t, tc.wantStatus, gotStatus)

		if tc.wantBody != "" {
			data, err := io.ReadAll(w.Result().Body)
			assert.NoError(t, err, "unexpected error during payload decoding")
			got := string(data)
			assert.JSONEq(t, tc.wantBody, got, "got body: %s", got)
		}
	})
}

func TestPageQueryParser(t *testing.T) {
	for _, tc := range []testcase{
		{
			name: "parses page query params",
			options: []server.Options{
				middleware.UsePageQueryParser(page.CursorNavigationParser{}),
			},
			muxconfig: func(t *testing.T, rm *server.ResourceMux) {
				rm.Handle("things", http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						ctx := jsonapi.FromContext(r.Context())
						assert.EqualValues(t, ctx.Pagination, query.Page{
							Cursor: "abc",
							Limit:  100,
						})
						w.WriteHeader(http.StatusOK)
						server.Write(w, jsonapi.NewMultiDocument(), http.StatusOK)
					}))
			},
			req:        httptest.NewRequest("GET", "https://example.com/things?page[cursor]=abc&page[limit]=100", nil),
			wantStatus: http.StatusOK,
		},
	} {
		tc.run(t)
	}
}
