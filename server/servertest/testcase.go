package servertest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi/v1/jsonapitest"
	"github.com/stretchr/testify/assert"
)

type Fixture interface {
	Handler(*testing.T) http.Handler
}

type FixtureOption[F Fixture] func(*testing.T, *F)

type Case[F Fixture] struct {
	Name       string
	Req        *http.Request
	Options    []FixtureOption[F]
	Skip       bool
	WantStatus int
	WantBody   string
	WantPanic  bool
	Assert     []func(*testing.T, *http.Response)
}

func Run[F Fixture](t *testing.T, testcases ...Case[F]) {
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip {
				t.Skip("skip is enabled on test case; skipping")
				return
			}

			var f F
			for _, apply := range tc.Options {
				apply(t, &f)
			}

			w := httptest.NewRecorder()
			h := f.Handler(t)

			if tc.WantPanic {
				assert.Panics(t, func() {
					h.ServeHTTP(w, tc.Req)
				})
				return
			}

			assert.NotPanics(t, func() {
				h.ServeHTTP(w, tc.Req)
			})

			res := w.Result()
			assert.Equal(t, tc.WantStatus, res.StatusCode, "status code not equal")

			if tc.WantBody != "" {
				data, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("failed to read response body: %s", err.Error())
					return
				}
				gotBody := string(data)
				jsonapitest.AssertJSONAPIEq(t, tc.WantBody, gotBody, "got JSON: %s", gotBody)
			}

			for _, assertion := range tc.Assert {
				assertion(t, res)
			}
		})
	}
}
