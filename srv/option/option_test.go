package option_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/query"
	"github.com/gonobo/jsonapi/query/pagination"
	"github.com/gonobo/jsonapi/srv"
	"github.com/gonobo/jsonapi/srv/option"
	"github.com/stretchr/testify/assert"
)

type optionTestCase struct {
	name       string
	options    []srv.Options
	req        *http.Request
	muxconfig  func(*testing.T, *srv.ResourceMux)
	wantStatus int
	wantBody   string
}

func (tc optionTestCase) run(t *testing.T) {
	t.Run(tc.name, func(t *testing.T) {
		mux := srv.New(tc.options...)
		tc.muxconfig(t, &mux)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, tc.req)

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
	for _, tc := range []optionTestCase{
		{
			name: "parses page query params",
			options: []srv.Options{
				option.WithPaginationQueryParser(pagination.CursorNavigationParser{}),
			},
			muxconfig: func(t *testing.T, rm *srv.ResourceMux) {
				rm.HandleResource("things", http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						ctx, _ := jsonapi.GetContext(r.Context())
						assert.EqualValues(t, ctx.Pagination, query.Page{
							Cursor: "abc",
							Limit:  100,
						})
						w.WriteHeader(http.StatusOK)
						srv.Write(w, jsonapi.NewMultiDocument(), http.StatusOK)
					}))
			},
			req:        httptest.NewRequest("GET", "https://example.com/things?page[cursor]=abc&page[limit]=100", nil),
			wantStatus: http.StatusOK,
		},
	} {
		tc.run(t)
	}
}
