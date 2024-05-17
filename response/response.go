package response

import (
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

// Ok returns a 200 response containing a JSONAPI document.
func Ok(body *jsonapi.Document, opts ...jsonapi.ResponseOption) jsonapi.Response {
	opts = append([]jsonapi.ResponseOption{func(j *jsonapi.Response) {
		j.Body = body
	}}, opts...)

	return jsonapi.NewResponse(http.StatusOK, opts...)
}

// Created returns a 201 response containing a JSONAPI document and location header.
func Created(resource *jsonapi.Resource, opts ...jsonapi.ResponseOption) jsonapi.Response {
	opts = append([]jsonapi.ResponseOption{
		func(j *jsonapi.Response) {
			j.Body = &jsonapi.Document{Data: jsonapi.One{Value: resource}}
		},
	}, opts...)

	return jsonapi.NewResponse(http.StatusCreated, opts...)
}

// Error returns a response containing one or more errors.
func Error(statusCode int, cause error, opts ...jsonapi.ResponseOption) jsonapi.Response {
	opts = append([]jsonapi.ResponseOption{func(r *jsonapi.Response) {
		r.AppendError(cause)
	}}, opts...)

	return jsonapi.NewResponse(statusCode, opts...)
}

// ResourceNotFound sends a JSONAPI formatted 404 error response to the client.
func ResourceNotFound(ctx *jsonapi.RequestContext) jsonapi.Response {
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

// InternalError sends a JSONAPI formatted 500 error response to the client.
func InternalError(err error) jsonapi.Response {
	return Error(http.StatusInternalServerError, jsonapi.Error{
		Status: http.StatusText(http.StatusInternalServerError),
		Title:  "Internal Server Error",
		Detail: err.Error(),
	})
}
