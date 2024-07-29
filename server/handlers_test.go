package server_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/jsonapitest"
	"github.com/gonobo/jsonapi/server"
	"github.com/gonobo/jsonapi/server/middleware"
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
	Children []*node `jsonapi:"relation,children"`
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
	items := make([]*node, 0, len(n))
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
			Req:  httptest.NewRequest("GET", "/nodes", nil),
			Options: []fixtureopts{
				servesResource(nodeResource{
					"42": {"42", "forty-two", nil},
				}),
			},
			WantStatus: http.StatusOK,
			WantBody: `
			{
				"data":[
					{
						"attributes":{"value":"forty-two"},
						"id":"42",
						"relationships":{
							"children":{"data":[]}
						},
						"type":"nodes"
					}
				],
				"jsonapi":{"version":"1.1"}
			}`,
		},
		{
			Name: "routes to create handler",
			Req: httptest.NewRequest("POST", "/nodes", jsonapitest.Body{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						Type: "nodes",
						Attributes: map[string]any{
							"value": "forty-two",
						},
					},
				},
			}),
			Options: []fixtureopts{
				servesResource(nodeResource{}),
				withOption(middleware.UseRequestBodyParser()),
			},
			WantStatus: http.StatusCreated,
			WantBody: `{
				"data":{
					"attributes":{"value":"forty-two"},
					"id":"new",
					"relationships":{
						"children":{"data":[]}
					},
					"type":"nodes"
					},
				"jsonapi":{"version":"1.1"}
			}`,
		},
		{
			Name: "routes to get handler",
			Req:  httptest.NewRequest("GET", "/nodes/42", nil),
			Options: []fixtureopts{
				servesResource(nodeResource{"42": {"42", "forty-two", nil}}),
			},
			WantStatus: http.StatusOK,
			WantBody: `{
				"data":{
					"attributes":{"value":"forty-two"},
					"id":"42",
					"relationships":{
						"children":{"data":[]}
					},
					"type":"nodes"
					},
				"jsonapi":{"version":"1.1"}
			}`,
		},
		{
			Name: "routes to get ref handler",
			Req:  httptest.NewRequest("GET", "/nodes/42/relationships/children", nil),
			Options: []fixtureopts{
				servesResource(nodeResource{"42": {
					ID:       "42",
					Value:    "forty-two",
					Children: []*node{{"24", "twenty-four", nil}},
				}}),
			},
			WantStatus: http.StatusOK,
			WantBody: `{
				"jsonapi": {"version": "1.1"},
				"data": [{"id": "24", "type": "nodes"}]
			}`,
		},
		{
			Name: "routes to update handler",
			Req: httptest.NewRequest("PATCH", "/nodes/42", jsonapitest.Body{
				Data: jsonapi.One{
					Value: &jsonapi.Resource{
						Type: "nodes",
						ID:   "42",
						Attributes: map[string]any{
							"value": "updated",
						},
					},
				},
			}),
			Options: []fixtureopts{
				servesResource(nodeResource{"42": {"42", "forty-two", nil}}),
				withOption(middleware.UseRequestBodyParser()),
			},
			WantStatus: http.StatusOK,
			WantBody: `{
				"data":{
					"attributes":{"value":"updated"},
					"id":"42",
					"relationships":{
						"children":{"data":[]}
					},
					"type":"nodes"
					},
				"jsonapi":{"version":"1.1"}
			}`,
		},
		{
			Name: "routes to delete handler",
			Req:  httptest.NewRequest("DELETE", "/nodes/42", nil),
			Options: []fixtureopts{
				servesResource(nodeResource{"42": {"42", "forty-two", nil}}),
			},
			WantStatus: http.StatusNoContent,
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
			Name: "relationship mux fails when request context is missing",
			Req:  httptest.NewRequest("GET", "/nodes/42/relationships/children", nil),
			Options: []fixtureopts{
				wrapsHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					handler := server.RelationshipMux{}
					handler.ServeHTTP(w, jsonapi.RequestWithContext(r, nil))
				})),
			},
			WantStatus: http.StatusInternalServerError,
		},
		{
			Name: "returns 404 on unknown ref endpoint",
			Req:  httptest.NewRequest("GET", "/nodes/42/relationships/unknown", nil),
			Options: []fixtureopts{
				servesResource(nodeResource{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled list resources endpoint",
			Req:  httptest.NewRequest("GET", "/nodes", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled create resource endpoint",
			Req:  httptest.NewRequest("POST", "/nodes", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled get resource endpoint",
			Req:  httptest.NewRequest("GET", "/nodes/42", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled update resource endpoint",
			Req:  httptest.NewRequest("PATCH", "/nodes/42", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled delete resource endpoint",
			Req:  httptest.NewRequest("DELETE", "/nodes/42", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled get ref endpoint",
			Req:  httptest.NewRequest("GET", "/nodes/42/relationships/unknown", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Relationship{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled update ref endpoint",
			Req:  httptest.NewRequest("PATCH", "/nodes/42/relationships/children", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Relationship{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled add ref endpoint",
			Req:  httptest.NewRequest("POST", "/nodes/42/relationships/children", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Relationship{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 404 on unhandled remove ref endpoint",
			Req:  httptest.NewRequest("DELETE", "/nodes/42/relationships/children", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Relationship{}),
			},
			WantStatus: http.StatusNotFound,
		},
		{
			Name: "returns 405 on invalid resource endpoint",
			Req:  httptest.NewRequest("PUT", "/nodes/42", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusMethodNotAllowed,
		},
		{
			Name: "returns 405 on invalid collection endpoint",
			Req:  httptest.NewRequest("PUT", "/nodes", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Resource{}),
			},
			WantStatus: http.StatusMethodNotAllowed,
		},
		{
			Name: "returns 405 on invalid ref endpoint",
			Req:  httptest.NewRequest("PUT", "/nodes/42/relationships/children", nil),
			Options: []fixtureopts{
				wrapsHandler(server.Relationship{}),
			},
			WantStatus: http.StatusMethodNotAllowed,
		},
	}...)
}
