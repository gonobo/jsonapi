package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gonobo/jsonapi/v1"
	"github.com/gonobo/jsonapi/v1/extra/visitor"
	"github.com/gonobo/jsonapi/v1/query"
)

const (
	HeaderKeyLocation    = "Location"
	LinkAttributeSelf    = "self"
	LinkAttributeRelated = "related"
)

// ResponseRecorder rr implements [http.ResponseWriter], capturing
// WriteHeader() and Write() calls into its instance members. To send recorded data
// to a response writer w, call rr.Flush(w).
//
// The zero value of ResponseRecorder is uninitialized; use the NewRecorder()
// function to create a new instance.
type ResponseRecorder struct {
	Status        int               // The response status code.
	HeaderMap     http.Header       // The response headers.
	Document      *jsonapi.Document // The response document payload.
	JSONUnmarshal jsonUnmarshalFunc // A function that unmarshals JSON documents. Defaults to [json.Unmarshal].
	JSONMarshal   jsonMarshalFunc   // A function that marshals JSON documents. Defaults to [json.Marshal].
}

// NewRecorder returns an initialized [ResponseRecorder].
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		HeaderMap:     make(http.Header),
		JSONUnmarshal: json.Unmarshal,
		JSONMarshal:   json.Marshal,
	}
}

// Write stores the content of p into an internal buffer.
func (ww *ResponseRecorder) Write(p []byte) (int, error) {
	ww.Document = &jsonapi.Document{}
	err := ww.JSONUnmarshal(p, ww.Document)
	return len(p), err
}

// Header returns a header map. It is the same header map
// used in a standard http response writer.
func (ww *ResponseRecorder) Header() http.Header {
	return ww.HeaderMap
}

// WriteHeader stores the status value in memory.
func (ww *ResponseRecorder) WriteHeader(status int) {
	ww.Status = status
}

// Flush writes the status code, headers, and document to w.
func (ww ResponseRecorder) Flush(w http.ResponseWriter) {
	for k, v := range ww.Header() {
		w.Header()[k] = v
	}
	if ww.Status != 0 {
		w.WriteHeader(ww.Status)
	}
	if ww.Document != nil {
		body, err := ww.JSONMarshal(ww.Document)
		if err != nil {
			panic(fmt.Errorf("memory writer: failed to marshal body: %w", err))
		}
		swallowWriteResult(w.Write(body))
	}
}

// Write writes the http response to the response writer. If data is a struct,
// it is marshaled into the JSON:API document format as the primary data. If data is
// of type jsonapi.Document, the document is written as is. Provide [WriteOptions] to
// alter default behavior.
//
// As with ResponseWriter.Write(), the caller should ensure no other calls are
// made to w after Write() is called.
func Write(w http.ResponseWriter, data any, status int, options ...WriteOptions) {
	cfg := DefaultConfig()
	cfg.ApplyWriteOptions(options...)

	if data == nil {
		// no data to serialize; write header and return early
		w.WriteHeader(status)
		return
	}

	doc, err := cfg.jsonapiMarshal(data)

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
	payload, err := cfg.jsonMarshal(doc)

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
	swallowWriteResult(w.Write(payload))
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

// WriteLink adds a URL to the response document's links attribute with the provided key.
func WriteLink(key string, href string) WriteOptions {
	return WithDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			if d.Links == nil {
				d.Links = jsonapi.Links{}
			}
			uri, err := url.ParseRequestURI(href)
			if err != nil {
				return fmt.Errorf("add link: failed to parse href: %s: %w", href, err)
			}
			d.Links[key] = &jsonapi.Link{Href: uri.String()}
			return nil
		})
}

// WriteSelfLink adds the full request URL to the response document's links attribute.
func WriteSelfLink(r *http.Request) WriteOptions {
	href := r.URL.String()
	return WriteLink(LinkAttributeSelf, href)
}

func pageCursorLink(r *http.Request, name string, cursor string, limit int) WriteOptions {
	requestURL := *r.URL
	query := requestURL.Query()
	query.Del("page[cursor]")
	query.Set("page[cursor]", cursor)

	if limit > 0 {
		query.Del("page[limit]")
		query.Set("page[limit]", strconv.Itoa(limit))
	}

	requestURL.RawQuery = query.Encode()
	return WriteLink(name, requestURL.String())
}

func pageNumberLink(r *http.Request, name string, page int, limit int) WriteOptions {
	requestURL := *r.URL
	query := requestURL.Query()
	query.Del("page[number]")
	query.Set("page[number]", strconv.Itoa(page))

	if limit > 0 {
		query.Del("page[limit]")
		query.Set("page[limit]", strconv.Itoa(limit))
	}

	requestURL.RawQuery = query.Encode()
	return WriteLink(name, requestURL.String())
}

// WriteNavigationLinks adds pagination URLs to the response document's links attribute.
func WriteNavigationLinks(r *http.Request, info map[string]query.Page) WriteOptions {
	options := make([]WriteOptions, 0, len(info))
	for name, page := range info {
		if page.Cursor != "" {
			options = append(options, pageCursorLink(r, name, page.Cursor, page.Limit))
		} else {
			options = append(options, pageNumberLink(r, name, page.PageNumber, page.Limit))
		}
	}
	return func(c *Config) {
		c.ApplyWriteOptions(options...)
	}
}

// WriteResourceLinks applies the "self" and "related" links to all resources and resource relationships
// embedded in the response document.
func WriteResourceLinks(baseURL string, resolver jsonapi.URLResolver) WriteOptions {
	const keyParentType = "$__parenttype"
	const keyParentID = "$__parentid"
	const keyRelName = "$__relname"

	v := visitor.SectionVisitor{
		Resource: func(r *jsonapi.Resource) error {
			self := resolver.ResolveURL(jsonapi.RequestContext{
				ResourceType: r.Type,
				ResourceID:   r.ID,
			}, baseURL)

			if r.Links == nil {
				r.Links = jsonapi.Links{}
			}

			r.Links[LinkAttributeSelf] = &jsonapi.Link{Href: self}

			for name, rel := range r.Relationships {
				if rel.Meta == nil {
					rel.Meta = jsonapi.Meta{}
				}

				rel.Meta[keyParentType] = r.Type
				rel.Meta[keyParentID] = r.ID
				rel.Meta[keyRelName] = name
			}

			return nil
		},
		Relationship: func(r *jsonapi.Relationship) error {
			resourceType := r.Meta[keyParentType].(string)
			resourceID := r.Meta[keyParentID].(string)
			relationship := r.Meta[keyRelName].(string)

			self := resolver.ResolveURL(jsonapi.RequestContext{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Relationship: relationship,
			}, baseURL)

			related := resolver.ResolveURL(jsonapi.RequestContext{
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Relationship: relationship,
				Related:      true,
			}, baseURL)

			if r.Links == nil {
				r.Links = jsonapi.Links{}
			}

			r.Links[LinkAttributeSelf] = &jsonapi.Link{Href: self}
			r.Links[LinkAttributeRelated] = &jsonapi.Link{Href: related}

			delete(r.Meta, keyParentType)
			delete(r.Meta, keyParentID)
			delete(r.Meta, keyRelName)

			return nil
		},
	}

	return writeWithDocumentVisitor(v)
}

// writeWithDocumentVisitor applies the visitor to the response document. Visitors can traverse and modify
// a document's nodes.
func writeWithDocumentVisitor(v visitor.SectionVisitor) WriteOptions {
	return WithDocumentOptions(func(w http.ResponseWriter, d *jsonapi.Document) error {
		return visitor.VisitDocument(v.Visitor(), d)
	})
}

// WriteSortedPrimaryData orders resources in the response document according to the specified
// sort criterion.
func WriteSortedPrimaryData(cmp jsonapi.Comparator, criterion []query.Sort) WriteOptions {
	return WithDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			d.Sort(cmp, criterion)
			return nil
		},
	)
}

// WriteMeta adds a key/value pair to the response document's meta attribute.
func WriteMeta(key string, value any) WriteOptions {
	return WithDocumentOptions(
		func(w http.ResponseWriter, d *jsonapi.Document) error {
			if d.Meta == nil {
				d.Meta = jsonapi.Meta{}
			}
			d.Meta[key] = value
			return nil
		})
}

// WriteLocationHeader adds the "Location" http header to the response. The resulting
// URL is based on the primary data resource's type and id.
func WriteLocationHeader(baseURL string, resolver jsonapi.URLResolver) WriteOptions {
	return WithDocumentOptions(setLocationHeader(baseURL, resolver))
}

func setLocationHeader(baseURL string, resolver jsonapi.URLResolver) DocumentOptions {
	return func(w http.ResponseWriter, d *jsonapi.Document) error {
		data := d.Data.First()
		location := resolver.ResolveURL(jsonapi.RequestContext{
			ResourceType: data.Type,
			ResourceID:   data.ID,
		}, baseURL)
		w.Header().Add(HeaderKeyLocation, location)
		return nil
	}
}

// WriteRef overwrites the primary data with the primary data's relationship,
// referenced by name.
func WriteRef(name string) WriteOptions {
	return WithDocumentOptions(func(w http.ResponseWriter, d *jsonapi.Document) error {
		first := d.Data.First()
		ref, ok := first.Relationships[name]
		if !ok {
			return fmt.Errorf("write ref: unknown relationship %s", name)
		}

		d.Links = ref.Links
		d.Data = ref.Data
		d.Meta = ref.Meta
		d.Included = nil

		return nil
	})
}
