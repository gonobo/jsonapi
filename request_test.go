package jsonapi_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/gonobo/jsonapi/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewRequest(t *testing.T) {
	client := jsonapi.NewRequest("http://api.foo.com", func(r *jsonapi.Client) {
		r.URLResolver = jsonapi.DefaultURLResolver()
	})
	assert.NotNil(t, client)
}

type doer func(req *http.Request) (*http.Response, error)

func (d doer) Do(req *http.Request) (*http.Response, error) {
	return d(req)
}

func doerWithResponse(statusCode int, response string) doer {
	return doer(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: statusCode,
			Body:       io.NopCloser(bytes.NewBufferString(response)),
		}, nil
	})
}

func doerWithError(err error) doer {
	return doer(func(req *http.Request) (*http.Response, error) {
		return nil, err
	})
}

func assertHTTPRequest(t *testing.T, req *http.Request, wantMethod, wantURL, wantBody string) {
	assert.Equal(t, wantMethod, req.Method)
	assert.Equal(t, wantURL, req.URL.String())
	if wantBody != "" {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, wantBody, string(body))
	}
}

func TestRequestFetch(t *testing.T) {
	type testcase struct {
		name         string
		resourceType string
		id           string
		options      []func(*http.Request)
		wantURL      string
		wantMethod   string
		wantBody     string
		wantErr      bool
	}

	for _, tc := range []testcase{
		{
			name:         "fetches a resource",
			resourceType: "items",
			id:           "1",
			wantURL:      "http://api.foo.com/items/1",
			wantMethod:   http.MethodGet,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := jsonapi.NewRequest("http://api.foo.com")
			request.Doer = doerWithResponse(http.StatusOK, "")
			_, err := request.Get(tc.resourceType, tc.id, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				},
			)...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestFetchRef(t *testing.T) {
	type testcase struct {
		name         string
		resourceType string
		id           string
		ref          string
		options      []func(*http.Request)
		wantURL      string
		wantMethod   string
		wantBody     string
		wantErr      bool
	}

	for _, tc := range []testcase{
		{
			name:         "fetches a resource",
			resourceType: "items",
			id:           "1",
			ref:          "owner",
			wantURL:      "http://api.foo.com/items/1/relationships/owner",
			wantMethod:   http.MethodGet,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.GetRef(tc.resourceType, tc.id, tc.ref, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestFetchRelated(t *testing.T) {
	type testcase struct {
		name         string
		resourceType string
		id           string
		ref          string
		options      []func(*http.Request)
		wantURL      string
		wantMethod   string
		wantBody     string
		wantErr      bool
	}

	for _, tc := range []testcase{
		{
			name:         "fetches a resource",
			resourceType: "items",
			id:           "1",
			ref:          "owner",
			wantURL:      "http://api.foo.com/items/1/owner",
			wantMethod:   http.MethodGet,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.GetRelated(tc.resourceType, tc.id, tc.ref, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestSearch(t *testing.T) {
	type testcase struct {
		name         string
		resourceType string
		options      []func(*http.Request)
		wantURL      string
		wantMethod   string
		wantBody     string
		wantErr      bool
	}

	for _, tc := range []testcase{
		{
			name:         "searches for resources",
			resourceType: "items",
			wantURL:      "http://api.foo.com/items",
			wantMethod:   http.MethodGet,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.List(tc.resourceType, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestCreate(t *testing.T) {
	type testcase struct {
		name       string
		data       jsonapi.Resource
		options    []func(*http.Request)
		wantURL    string
		wantMethod string
		wantBody   string
		wantErr    bool
	}

	for _, tc := range []testcase{
		{
			name: "creates a resource",
			data: jsonapi.Resource{
				Type: "items",
			},
			wantURL:    "http://api.foo.com/items",
			wantMethod: http.MethodPost,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.Create(tc.data, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestUpdate(t *testing.T) {
	type testcase struct {
		name       string
		data       jsonapi.Resource
		options    []func(*http.Request)
		wantURL    string
		wantMethod string
		wantBody   string
		wantErr    bool
	}

	for _, tc := range []testcase{
		{
			name: "updates a resource",
			data: jsonapi.Resource{
				ID:   "1",
				Type: "items",
			},
			wantURL:    "http://api.foo.com/items/1",
			wantMethod: http.MethodPatch,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.Update(tc.data, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestDelete(t *testing.T) {
	type testcase struct {
		name         string
		resourceType string
		id           string
		options      []func(*http.Request)
		wantURL      string
		wantMethod   string
		wantBody     string
		wantErr      bool
	}

	for _, tc := range []testcase{
		{
			name:         "deletes a resource",
			resourceType: "items",
			id:           "1",
			wantURL:      "http://api.foo.com/items/1",
			wantMethod:   http.MethodDelete,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.Delete(tc.resourceType, tc.id, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestUpdateRef(t *testing.T) {
	type testcase struct {
		name       string
		data       jsonapi.Resource
		ref        string
		options    []func(*http.Request)
		wantURL    string
		wantMethod string
		wantBody   string
		wantErr    bool
	}

	for _, tc := range []testcase{
		{
			name: "updates a resource relationship",
			ref:  "foo",
			data: jsonapi.Resource{
				Type: "items",
				ID:   "1",
				Relationships: map[string]*jsonapi.Relationship{
					"foo": {
						Data: jsonapi.One{Value: &jsonapi.Resource{
							Type: "foos",
							ID:   "2",
						}},
					},
				},
			},
			wantURL:    "http://api.foo.com/items/1/relationships/foo",
			wantMethod: http.MethodPatch,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.UpdateRef(tc.data, tc.ref, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestAddRefsToMany(t *testing.T) {
	type testcase struct {
		name       string
		data       jsonapi.Resource
		ref        string
		options    []func(*http.Request)
		wantURL    string
		wantMethod string
		wantBody   string
		wantErr    bool
	}

	for _, tc := range []testcase{
		{
			name: "adds a resource ref to a many relationship",
			ref:  "foo",
			data: jsonapi.Resource{
				Type: "items",
				ID:   "1",
				Relationships: map[string]*jsonapi.Relationship{
					"foo": {
						Data: jsonapi.Many{Value: []*jsonapi.Resource{{
							Type: "foos",
							ID:   "2",
						}}},
					},
				},
			},
			wantURL:    "http://api.foo.com/items/1/relationships/foo",
			wantMethod: http.MethodPost,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.AddRefsToMany(tc.data, tc.ref, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestRequestRemoveRefsFromMany(t *testing.T) {
	type testcase struct {
		name       string
		data       jsonapi.Resource
		ref        string
		options    []func(*http.Request)
		wantURL    string
		wantMethod string
		wantBody   string
		wantErr    bool
	}

	for _, tc := range []testcase{
		{
			name: "adds a resource ref to a many relationship",
			ref:  "foo",
			data: jsonapi.Resource{
				Type: "items",
				ID:   "1",
				Relationships: map[string]*jsonapi.Relationship{
					"foo": {
						Data: jsonapi.Many{Value: []*jsonapi.Resource{{
							Type: "foos",
							ID:   "2",
						}}},
					},
				},
			},
			wantURL:    "http://api.foo.com/items/1/relationships/foo",
			wantMethod: http.MethodDelete,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := jsonapi.NewRequest("http://api.foo.com")
			client.Doer = doerWithResponse(http.StatusOK, "")
			_, err := client.RemoveRefsFromMany(tc.data, tc.ref, append(
				tc.options,
				func(r *http.Request) {
					assertHTTPRequest(t, r, tc.wantMethod, tc.wantURL, tc.wantBody)
				})...)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestSend(t *testing.T) {
	request := jsonapi.NewRequest("http://api.foo.com")
	request.Context.Document = &jsonapi.Document{}

	t.Run("returns an error when json encoder fails", func(t *testing.T) {
		oldEncoder := request.JSONEncoder

		t.Cleanup(func() {
			request.JSONEncoder = oldEncoder
		})

		request.JSONEncoder = jsonEncoderError{}
		_, err := jsonapi.Do(request)
		assert.Error(t, err)
	})

	t.Run("returns an error when the request fails", func(t *testing.T) {
		request.Doer = doerWithError(errors.New("error"))
		_, err := jsonapi.Do(request)
		assert.Error(t, err)
	})
}

type jsonEncoderError struct{}

func (jsonEncoderError) EncodeJSON(w io.Writer, value any) error {
	return errors.New("error")
}

func TestRequestWithContext(t *testing.T) {
	jsonapictx := jsonapi.RequestContext{ResourceType: "items"}
	req, err := http.NewRequest(http.MethodGet, "www.example.com", nil)
	assert.NoError(t, err)
	req = jsonapi.RequestWithContext(req, &jsonapictx)

	got, ok := jsonapi.FromContext(req.Context())
	assert.True(t, ok)
	assert.Equal(t, jsonapictx, *got)
}
