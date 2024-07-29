package server_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/server"
	"github.com/gonobo/jsonapi/server/servertest"
)

type fixture struct {
	options []server.Options
	wrapped http.Handler
}

func (f fixture) Handler(t *testing.T) http.Handler {
	handler := server.Handle(f.wrapped, f.options...)
	return handler
}

type fixtureopts = servertest.FixtureOption[fixture]

func writesError(status int, err error, opts ...server.WriteOptions) fixtureopts {
	return func(t *testing.T, f *fixture) {
		f.wrapped = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			server.Error(w, err, status, opts...)
		})
	}
}

func writes(status int, doc *jsonapi.Document, opts ...server.WriteOptions) fixtureopts {
	return func(t *testing.T, f *fixture) {
		f.wrapped = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if doc == nil {
				server.Write(w, nil, status, opts...)
			} else {
				server.Write(w, doc, status, opts...)
			}
		})
	}
}

func wrapsHandler(handler http.Handler) fixtureopts {
	return func(t *testing.T, f *fixture) {
		f.wrapped = handler
	}
}

func withOption(option server.Options) fixtureopts {
	return func(t *testing.T, f *fixture) {
		f.options = append(f.options, option)
	}
}

func TestHandle(t *testing.T) {
	type testcase = servertest.Case[fixture]
	servertest.Run(t, []testcase{
		{
			Name: "returns 200 OK",
			Req:  httptest.NewRequest("GET", "/things", nil),
			Options: []fixtureopts{
				writes(http.StatusOK, jsonapi.NewMultiDocument()),
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"jsonapi": {"version": "1.1"}, "data": []}`,
		},
		{
			Name: "fails to resolve context",
			Req:  httptest.NewRequest("GET", "/things", nil),
			Options: []fixtureopts{
				withOption(
					server.WithContextResolver(jsonapi.RequestContextResolverFunc(
						func(r *http.Request) (jsonapi.RequestContext, error) {
							return jsonapi.RequestContext{}, errors.New("failed")
						},
					)),
				),
			},
			WantStatus: http.StatusInternalServerError,
		},
	}...)
}

func TestResourceMux(t *testing.T) {
	type testcase = servertest.Case[fixture]
	servertest.Run(t, []testcase{
		{
			Name: "routes to correct handler",
			Req:  httptest.NewRequest("GET", "/this", nil),
			Options: []fixtureopts{
				wrapsHandler(server.ResourceMux{
					"this": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						server.Write(w, jsonapi.NewMultiDocument(), http.StatusOK)
					}),
					"that": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						server.Error(w, errors.New("oops!"), http.StatusBadGateway)
					}),
				}),
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"jsonapi": {"version": "1.1"}, "data": []}`,
		},
		{
			Name: "routes to correct handler (use handle method)",
			Req:  httptest.NewRequest("GET", "/this", nil),
			Options: []fixtureopts{
				wrapsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mux := server.ResourceMux{}
					mux.Handle("this", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						server.Write(w, jsonapi.NewMultiDocument(), http.StatusOK)
					}))
					mux.Handle("that", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						server.Error(w, errors.New("oops!"), http.StatusBadGateway)
					}))
					mux.ServeHTTP(w, r)
				})),
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"jsonapi": {"version": "1.1"}, "data": []}`,
		},
		{
			Name: "fails when request context is missing",
			Req:  httptest.NewRequest("GET", "/this", nil),
			Options: []fixtureopts{
				wrapsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mux := server.ResourceMux{}
					mux.Handle("this", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						server.Write(w, jsonapi.NewMultiDocument(), http.StatusOK)
					}))
					mux.ServeHTTP(w, jsonapi.RequestWithContext(r, nil))
				})),
			},
			WantStatus: http.StatusInternalServerError,
		},
		{
			Name: "returns 404 on unknown resource",
			Req:  httptest.NewRequest("GET", "/that", nil),
			Options: []fixtureopts{
				wrapsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mux := server.ResourceMux{}
					mux.Handle("this", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						server.Write(w, jsonapi.NewMultiDocument(), http.StatusOK)
					}))
					mux.ServeHTTP(w, r)
				})),
			},
			WantStatus: http.StatusNotFound,
		},
	}...)
}

type node struct {
	ID       string  `jsonapi:"primary,nodes"`
	Value    string  `jsonapi:"attr,value"`
	Children []*node `jsonapi:"rel,children"`
}

type nodeResource map[string]*node

func (n nodeResource) createNode(w http.ResponseWriter, r *http.Request) {
	ctx, _ := jsonapi.GetContext(r.Context())
	data := node{}
	err := jsonapi.Unmarshal(ctx.Document, &data)

	if err != nil {
		server.Error(w, err, http.StatusBadRequest)
		return
	}

	data.ID = "new"
	server.Write(w, data, http.StatusCreated)
}

func (n nodeResource) listNodes(w http.ResponseWriter, r *http.Request) {
	items := make([]node, 0, len(n))
	for _, item := range n {
		items = append(items, item)
	}
	server.Write(w, items, http.StatusOK)
}

func (n nodeResource) getNode(w http.ResponseWriter, r *http.Request) {
	ctx, _ := jsonapi.GetContext(r.Context())
	item, ok := n[ctx.ResourceID]
	if !ok {
		server.Error(w, errors.New("not found"), http.StatusNotFound)
		return
	}
	server.Write(w, item, http.StatusOK)
}

func (n nodeResource) getNodeChildren(w http.ResponseWriter, r *http.Request) {
	ctx, _ := jsonapi.GetContext(r.Context())
	item, ok := n[ctx.ResourceID]
	if !ok {
		server.Error(w, errors.New("not found"), http.StatusNotFound)
		return
	}
	server.Write(w, item, http.StatusOK, server.WriteRef("children"))
}

func (n nodeResource) updateNode(w http.ResponseWriter, r *http.Request) {
	ctx, _ := jsonapi.GetContext(r.Context())
	item, ok := n[ctx.ResourceID]
	if !ok {
		server.Error(w, errors.New("not found"), http.StatusNotFound)
		return
	}

	err := jsonapi.Unmarshal(ctx.Document, item)
	if err != nil {
		server.Error(w, err, http.StatusBadRequest)
		return
	}

	n[ctx.ResourceID] = item
	server.Write(w, item, http.StatusOK)
}

func (n nodeResource) deleteNode(w http.ResponseWriter, r *http.Request) {
	ctx, _ := jsonapi.GetContext(r.Context())
	delete(n, ctx.ResourceID)
	server.Write(w, nil, http.StatusNoContent)
}

func newNodeResourceHandler(n nodeResource) server.Resource {
	return server.Resource{
		Refs: server.RelationshipMux{
			"children": server.Relationship{
				GetRef: http.HandlerFunc(n.getNodeChildren),
			},
		},
		Create: http.HandlerFunc(n.createNode),
		List:   http.HandlerFunc(n.listNodes),
		Get:    http.HandlerFunc(n.getNode),
		Update: http.HandlerFunc(n.updateNode),
		Delete: http.HandlerFunc(n.deleteNode),
	}
}

func servesResource(n nodeResource) fixtureopts {
	return wrapsHandler(newNodeResourceHandler(n))
}

func TestResource(t *testing.T) {
	type testcase = servertest.Case[fixture]

	servertest.Run(t, []testcase{
		{
			Name: "routes to list handler",
			Req:  httptest.NewRequest("GET", "/this", nil),
			Options: []fixtureopts{
				servesResource(nodeResource{
					"42": {"42", "forty-two", nil},
				}),
			},
			WantStatus: http.StatusOK,
			WantBody: `{
				"jsonapi": {"version": "1.1"},
				"data": [{"id": "42", "type": "nodes", "attributes": {"value": "forty-two"}}]
			}`,
		},
		{
			Name: "fails when request context is missing",
			Req:  httptest.NewRequest("GET", "/nodes", nil),
			Options: []fixtureopts{
				wrapsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handler := newNodeResourceHandler(nodeResource{})
					handler.ServeHTTP(w, jsonapi.RequestWithContext(r, nil))
				})),
			},
			WantStatus: http.StatusInternalServerError,
		},
		{
			Name: "returns 404 on unhandled endpoint",
			Req:  httptest.NewRequest("GET", "/nodes", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusNotFound,
		},
	}...)
}
