package jsonapi_test

import (
	"net/http"
	"testing"

	"github.com/gonobo/jsonapi/v2"
	"github.com/stretchr/testify/assert"
)

func TestURLResolver(t *testing.T) {
	type testcase struct {
		name     string
		resolver jsonapi.URLResolver
		context  jsonapi.RequestContext
		baseURL  string
		want     string
	}

	for _, tc := range []testcase{
		{
			name:     "default resource collection",
			resolver: jsonapi.DefaultURLResolver(),
			context:  jsonapi.RequestContext{ResourceType: "items"},
			baseURL:  "http://api.foo.com",
			want:     "http://api.foo.com/items",
		},
		{
			name:     "default resource item",
			resolver: jsonapi.DefaultURLResolver(),
			context:  jsonapi.RequestContext{ResourceType: "items", ResourceID: "1"},
			baseURL:  "http://api.foo.com",
			want:     "http://api.foo.com/items/1",
		},
		{
			name:     "default resource relationship",
			resolver: jsonapi.DefaultURLResolver(),
			context:  jsonapi.RequestContext{ResourceType: "items", ResourceID: "1", Relationship: "refs"},
			baseURL:  "http://api.foo.com",
			want:     "http://api.foo.com/items/1/relationships/refs",
		},
		{
			name:     "default related resource",
			resolver: jsonapi.DefaultURLResolver(),
			context:  jsonapi.RequestContext{ResourceType: "items", ResourceID: "1", Relationship: "refs", Related: true},
			baseURL:  "http://api.foo.com",
			want:     "http://api.foo.com/items/1/refs",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.resolver.ResolveURL(tc.context, tc.baseURL)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestContextResolver(t *testing.T) {
	type testcase struct {
		name     string
		resolver jsonapi.ContextResolver
		url      string
		want     jsonapi.RequestContext
		wantErr  bool
	}

	for _, tc := range []testcase{
		{
			name:     "default empty path",
			resolver: jsonapi.DefaultContextResolver(),
			url:      "/",
			wantErr:  true,
		},
		{
			name:     "default empty path",
			resolver: jsonapi.DefaultContextResolver(),
			url:      "",
			wantErr:  true,
		},
		{
			name:     "default resource collection",
			resolver: jsonapi.DefaultContextResolver(),
			url:      "/items",
			want:     jsonapi.RequestContext{ResourceType: "items"},
		},
		{
			name:     "default resource",
			resolver: jsonapi.DefaultContextResolver(),
			url:      "/items/1",
			want:     jsonapi.RequestContext{ResourceType: "items", ResourceID: "1"},
		},
		{
			name:     "default resource relationship",
			resolver: jsonapi.DefaultContextResolver(),
			url:      "/items/1/relationships/foo",
			want:     jsonapi.RequestContext{ResourceType: "items", ResourceID: "1", Relationship: "foo"},
		},
		{
			name:     "default related resource",
			resolver: jsonapi.DefaultContextResolver(),
			url:      "/items/1/foo",
			want:     jsonapi.RequestContext{ResourceType: "items", ResourceID: "1", Relationship: "foo", Related: true},
		},
		{
			name:     "default resource relationship with absolute url",
			resolver: jsonapi.DefaultContextResolver(),
			url:      "http://api.foo.com/items/1/relationships/foo",
			want:     jsonapi.RequestContext{ResourceType: "items", ResourceID: "1", Relationship: "foo"},
		},
		{
			name:     "prefixed resource relationship with absolute url",
			resolver: jsonapi.ContextResolverWithPrefix("/v2"),
			url:      "http://api.foo.com/v2/items/1/relationships/foo",
			want:     jsonapi.RequestContext{ResourceType: "items", ResourceID: "1", Relationship: "foo"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.url, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := tc.resolver.ResolveContext(req)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.EqualValues(t, &tc.want, got)
		})
	}
}
