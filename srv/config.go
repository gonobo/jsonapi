package srv

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gonobo/jsonapi"
)

type Config struct {
	contextResolver jsonapi.RequestContextResolver
	documentOptions []DocumentOptions
	jsonapiMarshal  jsonapiMarshalFunc
	jsonMarshal     jsonMarshalFunc
	middlewares     []Middleware
}

type jsonapiMarshalFunc = func(any) (jsonapi.Document, error)

type jsonMarshalFunc = func(any) ([]byte, error)

type DocumentOptions = func(context.Context, *jsonapi.Document)

type Middleware = func(http.Handler) http.Handler

type Options func(*Config)

type WriteOptions func(*Config)

func (c Config) Apply(options ...Options) Config {
	for _, apply := range options {
		apply(&c)
	}
	return c
}

func (c Config) ApplyWriteOptions(options ...WriteOptions) Config {
	for _, apply := range options {
		apply(&c)
	}
	return c
}

func (c Config) applyDocumentOptions(ctx context.Context, doc *jsonapi.Document) {
	for _, apply := range c.documentOptions {
		apply(ctx, doc)
	}
}

func (c Config) applyMiddleware(handler http.Handler) http.Handler {
	h := handler
	for _, mw := range c.middlewares {
		h = mw(h)
	}
	return h
}

func DefaultConfig() Config {
	return Config{
		jsonapiMarshal:  jsonapi.Marshal,
		jsonMarshal:     json.Marshal,
		contextResolver: jsonapi.DefaultRequestContextResolver(),
	}
}

func ResolvesContext(resolver jsonapi.RequestContextResolver) Options {
	return func(c *Config) {
		c.contextResolver = resolver
	}
}

func UseMiddleware(middleware Middleware) Options {
	return func(c *Config) {
		c.middlewares = append(c.middlewares, middleware)
	}
}
