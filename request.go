package jsonapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	MediaType = "application/vnd.api+json"
)

// Request is a JSON:API http client. It uses an underlying standard library
// http client to send requests that adhere to the JSON:API specification.
//
// By default, Request generates the following urls (given a base url):
//
//	":base/:type" for search, create
//	":base/:type/:id" for fetch, update, delete
//	":base/:type/:id/relationships/:ref" for fetchRef
//	":base/:type/:id/:ref" for fetchRelated
//
// This behavior can be modified by updating the URLResolver field of a
// Request instance.
type Request struct {
	JSONEncoder                // The JSON encoder.
	URLResolver                // The URL resolver.
	BaseURL     string         // The base url of the JSON:API server.
	Client      Doer           // The underlying http client.
	Context     RequestContext // The JSON:API request context.
	Method      string         // The http method to use.
}

// NewRequest creates a new JSON:API request instance.
func NewRequest(baseURL string, options ...func(*Request)) Request {
	r := Request{
		BaseURL:     baseURL,
		Client:      http.DefaultClient,
		URLResolver: DefaultURLResolver(),
		JSONEncoder: DefaultJSONEncoder{},
	}

	for _, option := range options {
		option(&r)
	}

	return r
}

// HTTPRequest generates a http.Request from a Request instance.
func (r Request) HTTPRequest() (*http.Request, error) {
	url := r.URLResolver.ResolveURL(r.Context, r.BaseURL)
	var req *http.Request = nil
	var body *bytes.Buffer = nil
	var err error = nil

	if r.Context.Document == nil {
		return http.NewRequest(r.Method, url, nil)
	}

	body = &bytes.Buffer{}
	err = r.JSONEncoder.EncodeJSON(body, r.Context.Document)

	if err != nil {
		err = fmt.Errorf("failed to JSON encode request body: %s", err)
	}

	if err == nil {
		req, err = http.NewRequest(r.Method, url, body)
	}

	return req, err
}

// Get retrieves a single resource from the server.
func (r Request) Get(resourceType, id string, options ...func(*http.Request)) (*http.Response, error) {
	r.Method = http.MethodGet
	r.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
	}
	return Send(r, options...)
}

// GetRef retrieves a single resource's relationship from the server.
func (r Request) GetRef(resourceType, id, ref string, options ...func(*http.Request)) (*http.Response, error) {
	r.Method = http.MethodGet
	r.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
		Relationship: ref,
	}
	return Send(r, options...)
}

// GetRelated retrieves all server resources that are referenced in a resource's relationship.
func (r Request) GetRelated(resourceType, id, relation string, options ...func(*http.Request)) (*http.Response, error) {
	r.Method = http.MethodGet
	r.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
		Relationship: relation,
		Related:      true,
	}
	return Send(r, options...)
}

// List retrieves a collection of server resources associated with the resource type.
func (r Request) List(resourceType string, options ...func(*http.Request)) (*http.Response, error) {
	r.Method = http.MethodGet
	r.Context = RequestContext{
		ResourceType: resourceType,
	}
	return Send(r, options...)
}

// Create creates a new server resource. If the resource has an empty string resource ID,
// then the server will assign it; otherwise, the id will be submitted in the request
// payload.
func (r Request) Create(data Resource, options ...func(*http.Request)) (*http.Response, error) {
	r.Method = http.MethodPost
	r.Context = RequestContext{
		ResourceType: data.Type,
		Document:     &Document{Data: One{Value: &data}},
	}
	return Send(r, options...)
}

// Update updates a new server resource.
func (r Request) Update(data Resource, options ...func(*http.Request)) (*http.Response, error) {
	r.Method = http.MethodPatch
	r.Context = RequestContext{
		ResourceType: data.Type,
		ResourceID:   data.ID,
		Document:     &Document{Data: One{Value: &data}},
	}
	return Send(r, options...)
}

// Delete removes a resource from the server.
func (r Request) Delete(resourceType, id string, options ...func(*http.Request)) (*http.Response, error) {
	r.Method = http.MethodDelete
	r.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
	}
	return Send(r, options...)
}

// UpdateRef replaces a single resource's relationship with the request data.
func (r Request) UpdateRef(data Resource, ref string, options ...func(*http.Request)) (*http.Response, error) {
	return Send(r.replaceResourceRef(data, ref, http.MethodPatch), options...)
}

// AddRefsToMany adds a server resource reference to the provided resource's "to-many" relationship.
func (r Request) AddRefsToMany(data Resource, ref string, options ...func(*http.Request)) (*http.Response, error) {
	return Send(r.replaceResourceRef(data, ref, http.MethodPost), options...)
}

// RemoveRefsFromMany removes a server resource reference to the provided resource's "to-many" relationship.
func (r Request) RemoveRefsFromMany(data Resource, ref string, options ...func(*http.Request)) (*http.Response, error) {
	return Send(r.replaceResourceRef(data, ref, http.MethodDelete), options...)
}

func (r Request) replaceResourceRef(data Resource, ref string, method string) Request {
	r.Method = method

	relation := data.Relationships[ref]
	document := &Document{}
	document.Meta = relation.Meta
	document.Links = relation.Links
	document.Data = relation.Data

	r.Context.ResourceType = data.Type
	r.Context.ResourceID = data.ID
	r.Context.Relationship = ref
	r.Context.Document = document
	return r
}

// Doer is responsible for sending HTTP requests. The Golang http.Client struct
// implements this interface.
type Doer interface {
	// Do sends an HTTP request and returns an HTTP response.
	Do(*http.Request) (*http.Response, error)
}

// JSONEncoder is responsible for encoding JSON data. It is primarily
// used for dependency injection during unit testing.
type JSONEncoder interface {
	// EncodeJSON encodes the provided value to JSON.
	EncodeJSON(w io.Writer, value any) error
}

// DefaultJSONEncoder is a wrapper that implements JSONEncoder by calling:
//
//	json.NewEncoder(w).Encode(value).
type DefaultJSONEncoder struct{}

// EncodeJSON encodes the provided value to JSON.
func (DefaultJSONEncoder) EncodeJSON(w io.Writer, value any) error {
	return json.NewEncoder(w).Encode(value)
}

// Send sends a request to the server.
func Send(r Request, options ...func(*http.Request)) (*http.Response, error) {
	req, err := r.HTTPRequest()
	if err != nil {
		return nil, jsonapiError("create http request: %s", err)
	}

	for _, option := range options {
		option(req)
	}

	res, err := r.Client.Do(req)

	if err != nil {
		return nil, jsonapiError("send http request: %s", err)
	}

	return res, nil
}

// RequestWithContext sets the request context on the provided request.
func RequestWithContext(r *http.Request, c *RequestContext) *http.Request {
	ctx := SetContext(r.Context(), c)
	return r.WithContext(ctx)
}
