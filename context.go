package jsonapi

import (
	"context"

	"github.com/gonobo/jsonapi/v2/query"
)

// RequestContext contains information about the JSON:API request that defines it.
//
// RequestContext decouples information relevant to the specification, such as
// the resource type, unique identifier, relationships, etc. from the request
// itself.
type RequestContext struct {
	ResourceType string                 // The request resource type, e.g, "users".
	ResourceID   string                 // The request resource ID, e.g, "123".
	Relationship string                 // The request relationship, e.g, "posts".
	Related      bool                   // If true, the server should fetch the resources associated with the relationship.
	FetchIDs     []string               // If nonempty, the server should fetch the resources with these IDs.
	Include      []string               // The list of resources associated with the request resource that should be included in the response.
	Document     *Document              // The JSON:API document that defines the request payload.
	Fields       []query.Fieldset       // The list of fields (attributes, relationships, etc.) that should be included in the response payload
	Filter       query.FilterExpression // The filter expression that was evaluated from the request query.
	Sort         []query.Sort           // The sort criteria that was evaluated from the request query.
	Pagination   query.Page             // The pagination criteria that was evaluated from the request query.
	parent       *RequestContext
}

type contextkey string

const jsonapiContextKey contextkey = "jsonapi_context"

// Child returns a new Context that is a child of the current Context; the new child context
// contains the same information as its parent.
func (c *RequestContext) Child() *RequestContext {
	clone := c.Clone()
	clone.parent = c
	return clone
}

// EmptyChild returns a new Context that is a child of the current Context.
func (c *RequestContext) EmptyChild() *RequestContext {
	ctx := &RequestContext{}
	ctx.parent = c
	return ctx
}

// Clone returns a new Context that is a clone of the current Context.
func (c RequestContext) Clone() *RequestContext {
	return &c
}

// Parent returns the parent Context of the current Context.
func (c RequestContext) Parent() *RequestContext {
	return c.parent
}

// Root returns the root Context of the current Context. That is, the top-most
// parent that has no parent of its own.
func (c RequestContext) Root() *RequestContext {
	current := &c
	for {
		if current.parent == nil {
			return current
		}
		current = current.parent
	}
}

// FromContext returns the JSON:API FromContext from the parent context.
// FromContext panics if the context was never set.
func FromContext(parent context.Context) *RequestContext {
	value := parent.Value(jsonapiContextKey)
	ctx, ok := value.(*RequestContext)
	if !ok {
		panic("parent context does not contain a JSON:API context")
	} else if ctx == nil {
		panic("parent context contains a nil JSON:API context")
	}
	return ctx
}

// WithContext sets the JSON:API Context in the parent context.
func WithContext(ctx context.Context, value *RequestContext) context.Context {
	return context.WithValue(ctx, jsonapiContextKey, value)
}
