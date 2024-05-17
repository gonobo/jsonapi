package jsonapi

// Middleware is a function that takes a Handler and returns a Handler.
type Middleware func(next RequestHandler) RequestHandler

// Passthrough creates a middleware function that returns the next handler.
func Passthrough() Middleware {
	return func(next RequestHandler) RequestHandler {
		return next
	}
}

// Use appends the provided middleware to the current middleware chain.
func (fn Middleware) Use(middleware Middleware) Middleware {
	return func(next RequestHandler) RequestHandler {
		return middleware(fn.Wrap(next))
	}
}

// Wrap wraps the provided handler with the current middleware chain.
func (fn Middleware) Wrap(handler RequestHandler) RequestHandler {
	return fn(handler)
}
