package srv

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
)

// MemoryWriter mw wraps and implements http.ResponseWriter, capturing
// WriteHeader() and Write() into an internal buffer. To send data
// to the underlying response writer, call mw.Flush().
type MemoryWriter struct {
	http.ResponseWriter        // The underlying response writer.
	Status              int    // The return status code.
	Body                []byte // The return payload.
}

// Write stores the content of p into an internal buffer.
func (m *MemoryWriter) Write(p []byte) (int, error) {
	m.Body = make([]byte, 0, len(p))
	return copy(m.Body, p), nil
}

// WriteHeader stores the status value in memory.
func (m *MemoryWriter) WriteHeader(status int) {
	m.Status = status
}

// Flush sends the status code and body buffer to the underlying writer.
func (m MemoryWriter) Flush() {
	if m.Status != 0 {
		m.ResponseWriter.WriteHeader(m.Status)
	}
	if len(m.Body) > 0 {
		swallowWriteResult(m.ResponseWriter.Write(m.Body))
	}
}

// Write writes the http response to the response writer. If input in is a struct,
// it is marshaled into the JSON:API document format as the primary data. If in is
// of type jsonapi.Document, the document is written as is. Provide write options
// alter default behavior.
//
// As with ResponseWriter.Write(), the caller should ensure no other calls are
// made to w after Write() is called.
func Write(w http.ResponseWriter, in any, options ...WriteOptions) {
	cfg := DefaultConfig().ApplyWriteOptions(options...)

	doc, err := cfg.jsonapiMarshal(in)

	if err != nil {
		errmsg := fmt.Sprintf("jsonapi: failed to marshal response: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		return
	}

	data, err := cfg.jsonMarshal(doc)

	if err != nil {
		errmsg := fmt.Sprintf("jsonapi: failed to marshal response: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		return
	}

	// ResponseWriter.Write() fulfills the io.Writer interface.
	swallowWriteResult(w.Write(data))
}

// Error returns a JSON:API formatted document containing the provided error. If
// the error is of type jsonapi.Error, it is marshaled as is; otherwise the error
// text is marshaled into the document payload.
//
// As with ResponseWriter.Write(), the caller should ensure no other calls are
// made to w after Write() is called.
func Error(w http.ResponseWriter, err error, status int, options ...WriteOptions) {
	var doc jsonapi.Document
	var jsonapierr jsonapi.Error

	w.WriteHeader(status)

	if errors.As(err, &jsonapierr) {
		doc.Errors = append(doc.Errors, &jsonapierr)
	} else {
		jsonapierr = jsonapi.NewError(err, "Error")
		doc.Errors = append(doc.Errors, &jsonapierr)
	}

	Write(w, doc, options...)
}

func swallowWriteResult(int, error) {}
