package server

import (
	"encoding/json"
	"net/http"
	"slices"

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

type jsonUnmarshalFunc = func([]byte, any) error

type DocumentOptions = func(http.ResponseWriter, *jsonapi.Document) error

type Middleware = func(next http.Handler) http.Handler

type Options func(*Config)

type WriteOptions func(*Config)

func (c *Config) Apply(options ...Options) {
	for _, apply := range options {
		apply(c)
	}
}

func (c *Config) ApplyWriteOptions(options ...WriteOptions) {
	for _, apply := range options {
		apply(c)
	}
}

func (c Config) applyDocumentOptions(w http.ResponseWriter, doc *jsonapi.Document) error {
	for _, apply := range c.documentOptions {
		if err := apply(w, doc); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) applyMiddleware(handler http.Handler) http.Handler {
	h := handler

	// reverse the list of middleware such that:
	//   1) middleware declared first is applied first (farthest from handler)
	//   2) middleware declared last is applied last (closest to handler)
	// aka, FIFO order.
	slices.Reverse(c.middlewares)

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

func WithJSONMarshaler(marshal jsonMarshalFunc) WriteOptions {
	return func(c *Config) {
		c.jsonMarshal = marshal
	}
}

func WithContextResolver(resolver jsonapi.RequestContextResolver) Options {
	return func(c *Config) {
		c.contextResolver = resolver
	}
}

func WithMiddleware(middleware Middleware) Options {
	return func(c *Config) {
		c.middlewares = append(c.middlewares, middleware)
	}
}

func WithDocumentOptions(options DocumentOptions) WriteOptions {
	return func(c *Config) {
		c.documentOptions = append(c.documentOptions, options)
	}
}
