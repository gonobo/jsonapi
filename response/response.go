package response

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gonobo/jsonapi"
)

// WithErrors populates the response document errors field with the provided error messages.
// If the message is of type jsonapi.ErrorNode, the error will be added directly to the list;
// otherwise, the error message will be wrapped in a jsonapi.ErrorNode.
func WithErrors(causes ...error) jsonapi.ResponseOption {
	return func(j *jsonapi.Response) {
		for _, cause := range causes {
			j.AppendError(cause)
		}
	}
}

// WithHeader sets the response header.
func WithHeader(key, value string) jsonapi.ResponseOption {
	return func(r *jsonapi.Response) {
		r.Headers[key] = value
	}
}

// Write writes a JSONAPI document to the response.
// The input value is marshaled into the primary data of the JSONAPI document.
func Write(status int, in any, opts ...jsonapi.ResponseOption) jsonapi.Response {
	doc, err := jsonapi.Marshal(in)
	if err != nil {
		return InternalError(fmt.Errorf("response failed: %s", err))
	}

	opts = append([]jsonapi.ResponseOption{func(j *jsonapi.Response) {
		j.Body = &doc
	}}, opts...)

	return jsonapi.NewResponse(status, opts...)
}

// Ok returns a 200 response containing a JSONAPI document. The input value is marshaled
// into the primary data of the JSONAPI document.
func Ok(in any, opts ...jsonapi.ResponseOption) jsonapi.Response {
	return Write(http.StatusOK, in, opts...)
}

// Created returns a 201 response containing a JSONAPI document.
// The input value is marshaled into the primary data of the JSONAPI document.
func Created(in any, opts ...jsonapi.ResponseOption) jsonapi.Response {
	return Write(http.StatusCreated, in, opts...)
}

// NoContent returns a 204 response.
func NoContent() jsonapi.Response {
	return jsonapi.NewResponse(http.StatusNoContent)
}

// Error returns a response containing one or more errors.
func Error(statusCode int, cause error, opts ...jsonapi.ResponseOption) jsonapi.Response {
	opts = append([]jsonapi.ResponseOption{func(r *jsonapi.Response) {
		r.AppendError(cause)
	}}, opts...)

	return jsonapi.NewResponse(statusCode, opts...)
}

// ResourceNotFound sends a JSONAPI formatted 404 error response to the client.
// Deprecated: Use response.NotFound instead.
func ResourceNotFound(ctx *jsonapi.RequestContext) jsonapi.Response {
	log.Println("response.ResourceNotFound() is deprecated and will be removed in a future release. Use response.NotFound instead.")
	return Error(http.StatusNotFound, jsonapi.Error{
		Status: http.StatusText(http.StatusNotFound),
		Title:  "Resource Not Found",
		Detail: "The requested resource was not found.",
		Meta: map[string]interface{}{
			"resource":     ctx.ResourceType,
			"id":           ctx.ResourceID,
			"relationship": ctx.Relationship,
		},
	})
}

// NotFound sends a JSONAPI formatted 404 error response to the client.
func NotFound() jsonapi.Response {
	return Error(http.StatusNotFound, jsonapi.Error{
		Status: http.StatusText(http.StatusNotFound),
		Title:  "Resource Not Found",
		Detail: "The requested resource was not found.",
	})
}

// InternalError sends a JSONAPI formatted 500 error response to the client.
func InternalError(err error) jsonapi.Response {
	return Error(http.StatusInternalServerError, jsonapi.Error{
		Status: http.StatusText(http.StatusInternalServerError),
		Title:  "Internal Server Error",
		Detail: err.Error(),
	})
}
