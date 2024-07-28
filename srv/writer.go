package srv

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gonobo/jsonapi"
)

// Recorder mw wraps and implements http.ResponseWriter, capturing
// WriteHeader() and Write() into an internal buffer. To send data
// to the underlying response writer, call mw.Flush().
type Recorder struct {
	Status        int               // The return status code.
	HeaderMap     http.Header       // The return headers.
	Document      *jsonapi.Document // The jsonapi document payload.
	JSONUnmarshal jsonUnmarshalFunc
	JSONMarshal   jsonMarshalFunc
}

// NewRecorder creates and initializes a new memory writer
// for use as a http response writer.
func NewRecorder() *Recorder {
	return &Recorder{
		HeaderMap:     make(http.Header),
		JSONUnmarshal: json.Unmarshal,
		JSONMarshal:   json.Marshal,
	}
}

// Write stores the content of p into an internal buffer.
func (m *Recorder) Write(p []byte) (int, error) {
	m.Document = &jsonapi.Document{}
	err := m.JSONUnmarshal(p, m.Document)
	return len(p), err
}

// Header returns a header map. It is the same header map
// used in a standard http response writer.
func (m *Recorder) Header() http.Header {
	return m.HeaderMap
}

// WriteHeader stores the status value in memory.
func (m *Recorder) WriteHeader(status int) {
	m.Status = status
}

// Flush sends the status code and body buffer to the underlying writer.
func (m Recorder) Flush(w http.ResponseWriter) {
	for k, v := range m.Header() {
		w.Header()[k] = v
	}
	if m.Status != 0 {
		w.WriteHeader(m.Status)
	}
	if m.Document != nil {
		body, err := m.JSONMarshal(m.Document)
		if err != nil {
			panic(fmt.Errorf("memory writer: failed to marshal body: %w", err))
		}
		swallowWriteResult(w.Write(body))
	}
}

// Write writes the http response to the response writer. If input in is a struct,
// it is marshaled into the JSON:API document format as the primary data. If in is
// of type jsonapi.Document, the document is written as is. Provide write options
// alter default behavior.
//
// As with ResponseWriter.Write(), the caller should ensure no other calls are
// made to w after Write() is called.
func Write(w http.ResponseWriter, in any, status int, options ...WriteOptions) {
	cfg := DefaultConfig()
	cfg.ApplyWriteOptions(options...)

	if in == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	doc, err := cfg.jsonapiMarshal(in)

	if err != nil {
		errmsg := fmt.Sprintf("jsonapi: failed to marshal response: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		return
	}

	// apply document options
	err = cfg.applyDocumentOptions(w, &doc)

	if err != nil {
		errmsg := fmt.Sprintf("jsonapi: failed to apply document options: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		return
	}

	// marshal document
	data, err := cfg.jsonMarshal(doc)

	if err != nil {
		errmsg := fmt.Sprintf("jsonapi: failed to marshal response: %s", err)
		http.Error(w, errmsg, http.StatusInternalServerError)
		return
	}

	// add jsonapi header
	w.Header().Add("Content-Type", jsonapi.MediaType)

	// write status header
	w.WriteHeader(status)

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

	if errors.As(err, &jsonapierr) {
		doc.Errors = append(doc.Errors, &jsonapierr)
	} else {
		jsonapierr = jsonapi.NewError(err, "Error")
		doc.Errors = append(doc.Errors, &jsonapierr)
	}

	Write(w, doc, status, options...)
}

func swallowWriteResult(int, error) {}
