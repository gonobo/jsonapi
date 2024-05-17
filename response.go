package jsonapi

import (
	"errors"
	"net/http"
)

// Response objects contain the HTTP response status code,
// headers, and an optional body, formatted as a JSONAPI document.
//
// A Response's zero value is usually invalid; use the NewResponse() function
// to properly initialize a Response object.
//
// Response objects are used by the server to format the response to a
// client. A Response object can be marshaled to JSON and written to the
// HTTP response via the Write() method. Example:
//
//	r := jsonapi.NewResponse(func(res *jsonapi.Response) {
//		res.Body = jsonapi.NewSingleDocument(...)
//		res.Code = http.StatusOK
//		res.Headers["my-header-key"] = "my-header-value"
//	})
//
//	r.Write(w) // flushed to http response
//
// Response objects can also be used to format errors:
//
//	r := jsonapi.NewResponse(func(res *jsonapi.Response) {
//		res.AppendError(errors.New("some error"))
//		res.Code = http.StatusInternalServerError
//	})
type Response struct {
	Code    int               // The http status code.
	Headers map[string]string // Optional status headers.
	Body    *Document         // Optional response body.
}

type ResponseOption = func(*Response)

// NewResponse returns a new JSONAPIResponse object. Use response options
// to modify or format the response header, status code, and body.
// Example:
//
//	response := jsonapi.NewResponse(
//		server.Ok(&jsonapi.Document{ ... }),
//		server.Header("my-header-key", "my-header-value"),
//	)
//
// An unmodified response will return 204 No Content to the client.
func NewResponse(status int, opts ...ResponseOption) Response {
	r := Response{
		Code:    status,
		Headers: make(map[string]string),
		Body:    nil,
	}

	r.ApplyOptions(opts...)
	return r
}

// WriteResponse writes the JSON:API response to the http response writer. The number value
// returned is always zero, and not indicative of the number of bytes written.
func (r Response) WriteResponse(w http.ResponseWriter, opts ...ResponseOption) (int, error) {
	r.ApplyOptions(opts...)

	for h, v := range r.Headers {
		w.Header().Set(h, v)
	}

	w.WriteHeader(r.Code)

	var err error = nil
	if r.Body != nil {
		w.Header().Set("content-type", MediaType)
		err = Encode(w, r.Body)
	}

	return 0, err
}

func (r *Response) ApplyOptions(opts ...ResponseOption) {
	for _, opt := range opts {
		opt(r)
	}
}

func (r *Response) AppendError(cause error) {
	if r.Body == nil {
		r.Body = &Document{}
	}

	var node Error
	if errors.As(cause, &node) {
		r.Body.Errors = append(r.Body.Errors, &node)
		return
	}

	r.Body.Errors = append(r.Body.Errors, &Error{
		Status: http.StatusText(r.Code),
		Title:  "JSONAPI Error",
		Detail: cause.Error(),
	})
}
