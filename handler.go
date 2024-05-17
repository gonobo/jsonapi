package jsonapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

// RequestHandler handles JSON:API requests.
type RequestHandler interface {
	// ServeJSONAPI handles a JSON:API request.
	ServeJSONAPI(*http.Request) Response
}

// HandlerFunc is an adapter to allow the use of ordinary functions as JSON:API handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(*http.Request) Response

// ServeJSONAPI calls f(r).
func (f HandlerFunc) ServeJSONAPI(r *http.Request) Response {
	return f(r)
}

// RouteJSONAPI routes a JSON:API request to the handler function.
func (f HandlerFunc) RouteJSONAPI(ctx *RequestContext, r *http.Request) Response {
	return f.ServeJSONAPI(r)
}

// Handler wraps JSON:API handlers such that they can be used as
// standard library http.Handler instances.
func Handler(handler RequestHandler, options ...func(*H)) http.Handler {
	adapter := H{
		handler:                handler,
		RequestContextResolver: DefaultRequestContextResolver(),
	}

	for _, option := range options {
		option(&adapter)
	}

	return adapter
}

// H is an adapter to allow the use of JSON:API handlers as HTTP handlers.
type H struct {
	RequestContextResolver                // The JSON:API context resolver.
	handler                RequestHandler // The JSON:API handler.
}

// ServeHTTP implements the http.Handler interface.
func (h H) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// generate the jsonapi context from the request.
	ctx, err := h.ResolveContext(r)
	if err != nil {
		errmsg := fmt.Sprintf("jsonapi: context error: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		return
	}

	body := &Document{}

	// Unmarshal request payload if payload is present.
	if r.Body != nil {
		err = Decode(r.Body, body)
	}

	if errors.Is(err, io.EOF) {
		// there was no request body to parse.
		err = nil
		body = nil
	} else if err != nil {
		// the request body could not be parsed into a JSON:API document.
		errmsg := fmt.Sprintf("jsonapi: decode error: %s", err)
		http.Error(w, errmsg, http.StatusBadRequest)
		return
	}

	// assign the payload to the request context.
	ctx.Document = body

	// wrap the http writer in the jsonapi intermediate writer then serve the request.
	response := h.handler.ServeJSONAPI(RequestWithContext(r, &ctx))
	response.WriteResponse(w)
}
