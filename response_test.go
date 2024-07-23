package jsonapi_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/stretchr/testify/assert"
)

func TestNewResponse(t *testing.T) {
	type testcase struct {
		name    string
		status  int
		options []jsonapi.ResponseOption
		want    jsonapi.Response
	}

	for _, tc := range []testcase{
		{
			name:   "default",
			status: http.StatusNoContent,
			want: jsonapi.Response{
				Body:    nil,
				Code:    http.StatusNoContent,
				Headers: make(map[string]string),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := jsonapi.NewResponse(tc.status, tc.options...)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResponseWrite(t *testing.T) {
	type testcase struct {
		name      string
		response  jsonapi.Response
		options   []jsonapi.ResponseOption
		writer    http.ResponseWriter
		wantErr   bool
		wantPanic bool
	}

	for _, tc := range []testcase{
		{
			name:      "panics if zero value",
			writer:    httptest.NewRecorder(),
			wantPanic: true,
		},
		{
			name:      "panics if writer is nil",
			response:  jsonapi.NewResponse(http.StatusOK),
			wantPanic: true,
		},
		{
			name:     "writes default to writer",
			response: jsonapi.NewResponse(http.StatusOK),
			writer:   httptest.NewRecorder(),
			wantErr:  false,
		},
		{
			name: "writes custom to writer",
			response: jsonapi.NewResponse(http.StatusOK, func(r *jsonapi.Response) {
				r.Body = &jsonapi.Document{ValidateOnMarshal: true}
				r.Headers["x-app-id"] = "1234"
			}),
			writer:  httptest.NewRecorder(),
			wantErr: true,
		},
		{
			name: "writes error to writer",
			response: jsonapi.NewResponse(http.StatusInternalServerError, func(r *jsonapi.Response) {
				r.AppendError(errors.New("bad request"))
			}),
			writer:  httptest.NewRecorder(),
			wantErr: false,
		},
		{
			name: "writes error node to writer",
			response: jsonapi.NewResponse(http.StatusInternalServerError, func(r *jsonapi.Response) {
				r.AppendError(jsonapi.Error{})
			}),
			writer:  httptest.NewRecorder(),
			wantErr: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error

			if tc.wantPanic {
				assert.Panics(t, func() {
					_, err = tc.response.WriteResponse(tc.writer, tc.options...)
				})
				return
			}

			assert.NotPanics(t, func() {
				_, err = tc.response.WriteResponse(tc.writer, tc.options...)
			})

			if tc.wantErr {
				assert.Error(t, err)
				return
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
