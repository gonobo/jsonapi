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

// Client is a JSON:API http client. It uses an underlying standard library
// http client to send requests that adhere to the JSON:API specification.
//
// By default, Client generates the following urls (given a base url):
//
//	":base/:type" for search, create
//	":base/:type/:id" for fetch, update, delete
//	":base/:type/:id/relationships/:ref" for fetchRef
//	":base/:type/:id/:ref" for fetchRelated
//
// This behavior can be modified by updating the URLResolver field of a
// Client instance.
type Client struct {
	JSONEncoder                // The JSON encoder.
	URLResolver                // The URL resolver.
	Doer                       // The underlying http client.
	BaseURL     string         // The base url of the JSON:API server.
	Context     RequestContext // The JSON:API request context.
	Method      string         // The http method to use.
}

// NewRequest creates a new JSON:API request instance.
func NewRequest(baseURL string, options ...func(*Client)) Client {
	r := Client{
		BaseURL:     baseURL,
		Doer:        http.DefaultClient,
		URLResolver: DefaultURLResolver(),
		JSONEncoder: DefaultJSONEncoder{},
	}

	for _, option := range options {
		option(&r)
	}

	return r
}

// httpRequest generates a http.Request from a Request instance.
func (c Client) httpRequest() (*http.Request, error) {
	url := c.ResolveURL(c.Context, c.BaseURL)
	var req *http.Request = nil
	var body *bytes.Buffer = nil
	var err error = nil

	if c.Context.Document == nil {
		return http.NewRequest(c.Method, url, nil)
	}

	body = &bytes.Buffer{}
	err = c.JSONEncoder.EncodeJSON(body, c.Context.Document)

	if err != nil {
		err = fmt.Errorf("failed to JSON encode request body: %s", err)
	}

	if err == nil {
		req, err = http.NewRequest(c.Method, url, body)
	}

	return req, err
}

// Get retrieves a single resource from the server.
func (c Client) Get(resourceType, id string, options ...func(*http.Request)) (*http.Response, error) {
	c.Method = http.MethodGet
	c.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
	}
	return Do(c, options...)
}

// GetRef retrieves a single resource's relationship from the server.
func (c Client) GetRef(resourceType, id, ref string, options ...func(*http.Request)) (*http.Response, error) {
	c.Method = http.MethodGet
	c.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
		Relationship: ref,
	}
	return Do(c, options...)
}

// GetRelated retrieves all server resources that are referenced in a resource's relationship.
func (c Client) GetRelated(resourceType, id, relation string, options ...func(*http.Request)) (*http.Response, error) {
	c.Method = http.MethodGet
	c.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
		Relationship: relation,
		Related:      true,
	}
	return Do(c, options...)
}

// List retrieves a collection of server resources associated with the resource type.
func (c Client) List(resourceType string, options ...func(*http.Request)) (*http.Response, error) {
	c.Method = http.MethodGet
	c.Context = RequestContext{
		ResourceType: resourceType,
	}
	return Do(c, options...)
}

// Create creates a new server resource. If the resource has an empty string resource ID,
// then the server will assign it; otherwise, the id will be submitted in the request
// payload.
func (c Client) Create(data Resource, options ...func(*http.Request)) (*http.Response, error) {
	c.Method = http.MethodPost
	c.Context = RequestContext{
		ResourceType: data.Type,
		Document:     &Document{Data: One{Value: &data}},
	}
	return Do(c, options...)
}

// Update updates a new server resource.
func (c Client) Update(data Resource, options ...func(*http.Request)) (*http.Response, error) {
	c.Method = http.MethodPatch
	c.Context = RequestContext{
		ResourceType: data.Type,
		ResourceID:   data.ID,
		Document:     &Document{Data: One{Value: &data}},
	}
	return Do(c, options...)
}

// Delete removes a resource from the server.
func (c Client) Delete(resourceType, id string, options ...func(*http.Request)) (*http.Response, error) {
	c.Method = http.MethodDelete
	c.Context = RequestContext{
		ResourceType: resourceType,
		ResourceID:   id,
	}
	return Do(c, options...)
}

// UpdateRef replaces a single resource's relationship with the request data.
func (c Client) UpdateRef(data Resource, ref string, options ...func(*http.Request)) (*http.Response, error) {
	return Do(c.replaceResourceRef(data, ref, http.MethodPatch), options...)
}

// AddRefsToMany adds a server resource reference to the provided resource's "to-many" relationship.
func (c Client) AddRefsToMany(data Resource, ref string, options ...func(*http.Request)) (*http.Response, error) {
	return Do(c.replaceResourceRef(data, ref, http.MethodPost), options...)
}

// RemoveRefsFromMany removes a server resource reference to the provided resource's "to-many" relationship.
func (c Client) RemoveRefsFromMany(data Resource, ref string, options ...func(*http.Request)) (*http.Response, error) {
	return Do(c.replaceResourceRef(data, ref, http.MethodDelete), options...)
}

func (c Client) replaceResourceRef(data Resource, ref string, method string) Client {
	c.Method = method

	relation := data.Relationships[ref]
	document := &Document{}
	document.Meta = relation.Meta
	document.Links = relation.Links
	document.Data = relation.Data

	c.Context.ResourceType = data.Type
	c.Context.ResourceID = data.ID
	c.Context.Relationship = ref
	c.Context.Document = document
	return c
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

// Do sends a request to the server.
func Do(c Client, options ...func(*http.Request)) (*http.Response, error) {
	req, err := c.httpRequest()
	if err != nil {
		return nil, jsonapiError("create http request: %s", err)
	}

	for _, option := range options {
		option(req)
	}

	res, err := c.Doer.Do(req)

	if err != nil {
		return nil, jsonapiError("send http request: %s", err)
	}

	return res, nil
}

// RequestWithContext sets the request context on the provided request.
func RequestWithContext(r *http.Request, c *RequestContext) *http.Request {
	ctx := WithContext(r.Context(), c)
	return r.WithContext(ctx)
}
