package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"slices"
	"sort"
	"strings"

	"github.com/gonobo/jsonapi/internal/jsontest"
	"github.com/gonobo/jsonapi/query"
)

const LatestSupportedVersion = "1.1"

// Document is the highest order node, containing either a single resource
// or collection of resources in response to a client request. Clients also
// send documents to a server to create or modify existing resources.
type Document struct {
	Jsonapi           JSONAPI                     // The JSON:API object.
	Data              PrimaryData                 // The primary data.
	Meta              Meta                        // Top-level metadata.
	Links             Links                       // Top-level links.
	Errors            []*Error                    // Server response errors.
	Included          []*Resource                 // Included resources associated with the primary data.
	Extensions        map[string]*json.RawMessage // Optional JSON:API extensions.
	ValidateOnMarshal bool                        // Optional verification according to the JSON:API specification
}

// NewSingleDocument creates a new document with the provided resource as primary data.
func NewSingleDocument(data *Resource) *Document {
	return &Document{
		Data: One{Value: data},
	}
}

// NewMultiDocument creates a new document with the provided resources as primary data.
func NewMultiDocument(data ...*Resource) *Document {
	return &Document{
		Data: Many{Value: data},
	}
}

// Decode reads the JSON-encoded document from its input and stores it in the input document.
func Decode(r io.Reader, doc *Document) error {
	return json.NewDecoder(r).Decode(&doc)
}

// Encode writes the JSON encoding of v to the stream, followed by a newline character.
func Encode(w io.Writer, doc *Document) error {
	return json.NewEncoder(w).Encode(doc)
}

// Validate checks the document for correctness.

// ApplyVisitor allows the provided visitor to traverse this document.
func (d *Document) ApplyVisitor(v *Visitor) error {
	return applyDocumentVisitor(d, v)
}

// Error calls errors.Join() on all errors within the document.
// It returns a single error if any errors were present or nil otherwise.
func (d Document) Error() error {
	errs := make([]error, 0)
	for _, err := range d.Errors {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// MarshalJSON serializes the document as JSON.
func (d Document) MarshalJSON() ([]byte, error) {
	if err := ValidateSpec(&d); err != nil {
		return nil, fmt.Errorf("marshal: document failed spec: %w", err)
	}

	type out struct {
		Jsonapi  JSONAPI     `json:"jsonapi,omitempty"`
		Data     PrimaryData `json:"data,omitempty"`
		Meta     Meta        `json:"meta,omitempty"`
		Links    Links       `json:"links,omitempty"`
		Errors   []*Error    `json:"errors,omitempty"`
		Included []*Resource `json:"included,omitempty"`
	}

	node := node[out]{
		value: out{
			Jsonapi:  d.Jsonapi,
			Data:     d.Data,
			Meta:     d.Meta,
			Errors:   d.Errors,
			Links:    d.Links,
			Included: d.Included,
		},
		ext: d.Extensions,
	}

	return json.Marshal(node)
}

// UnmarshalJSON deserializes the document from JSON, populating its contents.
func (d *Document) UnmarshalJSON(data []byte) error {
	type in struct {
		Jsonapi  JSONAPI          `json:"jsonapi,omitempty"`
		Data     *json.RawMessage `json:"data,omitempty"`
		Meta     Meta             `json:"meta,omitempty"`
		Links    Links            `json:"links,omitempty"`
		Errors   []*Error         `json:"errors,omitempty"`
		Included []*Resource      `json:"included,omitempty"`
	}

	node := node[in]{}

	errs := make([]error, 0)
	errs = append(errs, json.Unmarshal(data, &node))

	d.Jsonapi = node.value.Jsonapi
	d.Meta = node.value.Meta
	d.Errors = node.value.Errors
	d.Links = node.value.Links
	d.Included = node.value.Included
	d.Extensions = node.ext

	primaryData := node.value.Data

	if node.ContainsAttribute("data") && primaryData == nil {
		// the "data" attribute was set to null; use the zero value
		// for one node.
		d.Data = One{}
	} else if primaryData == nil {
		// nothing more to process, quit early
		return errors.Join(errs...)
	} else if jsontest.IsJSONArray(primaryData) {
		// the primary data contains multiple resources.
		many := Many{}
		errs = append(errs, json.Unmarshal(*primaryData, &many))
		d.Data = many
	} else if jsontest.IsJSONObject(primaryData) {
		// the primary data contains a single resource.
		one := One{}
		errs = append(errs, json.Unmarshal(*primaryData, &one))
		d.Data = one
	}

	return errors.Join(errs...)
}

// Comparator compares two resources and determines ordinality by comparing
// the values of a specified attribute.
type Comparator interface {
	// Compare compares the values of the specified attribute for the
	// two resources at the specified indexes. If the two values are
	// equivalent, then Compare() should return 0. If the first value is
	// greater than the second value, then Compare() should return a
	// positive value. If the first value is less than the second value,
	// then Compare() should return a negative value.
	Compare(i, j int, attribute string) int
}

// ComparatorFunc functions implement Comparator.
type ComparatorFunc func(i, j int, attribute string) int

// Compare compares items indexed at i and j, returning negative if less,
// positive if greater, or zero if equal.
func (f ComparatorFunc) Compare(i, j int, attribute string) int {
	return f(i, j, attribute)
}

// Comparer determines the order of two items.
type Comparer[T any] func(a, b T) int

// ResourceComparator returns a Comparator that compares two resources a and b
// based on an attribute specified by the key index of map m.
func ResourceComparator[S ~[]T, T any](arr S, m map[string]Comparer[T]) Comparator {
	return ComparatorFunc(func(i, j int, attribute string) int {
		compare, ok := m[attribute]
		if !ok {
			// attribute cannot be compared, assume a is less than b
			// in this regard.
			return -1
		}
		a, b := arr[i], arr[j]
		return compare(a, b)
	})
}

// Sort sorts the document's primary data by comparing the resources
// against the provided sort criterion.
func (d *Document) Sort(cmp Comparator, criterion []query.Sort) {
	// do not attempt to sort if there aren't multiple records of primary
	// data or if there is no sort criteria specified.
	if d.Data == nil {
		return
	} else if !d.Data.IsMany() {
		return
	} else if len(criterion) == 0 {
		return
	}

	// use the comparator to determine resource order; if the items are
	// equivalent, then perform the comparison against the next
	// sort criteria.
	sort.Slice(d.Data.Items(), func(i, j int) bool {
		result := 0
		for _, criteria := range criterion {
			if criteria.Descending {
				result = cmp.Compare(j, i, criteria.Property)
			} else {
				result = cmp.Compare(i, j, criteria.Property)
			}
			if result == 0 {
				continue
			}
		}
		return result < 0
	})
}

// Version contains information regarding the JSON:API version supported by the server.
type Version string

// Value returns the associated version, or the last version supported by this library if
// zero-value.
func (v Version) Value() string {
	version := string(v)
	if version == "" {
		version = LatestSupportedVersion
	}
	return version
}

// MarshalJSON serializes the version into JSON.
func (v Version) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.Value())
}

// JSONAPI includes information about the server's implementation.
// See https://jsonapi.org/format/#document-jsonapi-object for details.
type JSONAPI struct {
	Version Version        `json:"version"`           // The highest specification version supported by the server.
	Ext     []string       `json:"ext,omitempty"`     // An array of URIs for all applied extensions.
	Profile []string       `json:"profile,omitempty"` // An array of URIs for all applied profiles.
	Meta    map[string]any `json:"meta,omitempty"`    // Metadata containing non-standard information.
}

// Meta contains non-standard information within a document.
type Meta = map[string]any

// RelationshipsNode contains the relationships defined within a resource.
type RelationshipsNode = map[string]*Relationship

// ExtensionsNode contains data defined by JSON:API extensions. Since they can be arbitrary,
// they are stored as raw JSON messages to be serialized by the caller.
type ExtensionsNode = map[string]*json.RawMessage

// Resource represents a server resource. Resources can appear in documents as primary data,
// included resources, or referenced in other resources' relationships.
// See https://jsonapi.org/format/#document-resource-objects for details.
type Resource struct {
	ID            string            // ID is the unique identifier of the resource.
	LocalID       string            // LocalID is the unique identifier of the resource within the context of the document.
	Type          string            // Type is the type of resource.
	Attributes    map[string]any    // Attributes are the resource's attributes.
	Relationships RelationshipsNode // Relationships are the resource's relationships.
	Links         Links             // Links are the resource's associated URL links.
	Meta          Meta              // Meta contains any non-standard information about the resource.
	Extensions    ExtensionsNode    // Extensions contain any JSON:API extensions associated with the resource.
}

// MarshalJSON serializes this resource into JSON.
func (r Resource) MarshalJSON() ([]byte, error) {
	type out struct {
		ID            string            `json:"id,omitempty"`
		LocalID       string            `json:"lid,omitempty"`
		Type          string            `json:"type"`
		Attributes    map[string]any    `json:"attributes,omitempty"`
		Relationships RelationshipsNode `json:"relationships,omitempty"`
		Links         Links             `json:"links,omitempty"`
		Meta          Meta              `json:"meta,omitempty"`
	}

	node := node[out]{
		value: out{
			ID:            r.ID,
			LocalID:       r.LocalID,
			Type:          r.Type,
			Attributes:    r.Attributes,
			Relationships: r.Relationships,
			Links:         r.Links,
			Meta:          r.Meta,
		},
		ext: r.Extensions,
	}

	return json.Marshal(node)
}

// UnmarshalJSON deserializes this resource from JSON.
func (r *Resource) UnmarshalJSON(data []byte) error {
	type in struct {
		ID            string                   `json:"id,omitempty"`
		LocalID       string                   `json:"lid,omitempty"`
		Type          string                   `json:"type"`
		Attributes    map[string]any           `json:"attributes,omitempty"`
		Relationships map[string]*Relationship `json:"relationships,omitempty"`
		Links         Links                    `json:"links,omitempty"`
		Meta          Meta                     `json:"meta,omitempty"`
	}

	node := node[in]{}
	err := json.Unmarshal(data, &node)

	r.ID = node.value.ID
	r.LocalID = node.value.LocalID
	r.Type = node.value.Type
	r.Attributes = node.value.Attributes
	r.Relationships = node.value.Relationships
	r.Links = node.value.Links
	r.Meta = node.value.Meta

	return err
}

// Ref returns a reference to this resource. Any non essential information --
// information that is not required to identify the resource -- is omitted.
// Any resource metadata, however is included.
func (r Resource) Ref() *Resource {
	return &Resource{
		ID:   r.ID,
		Type: r.Type,
		Meta: r.Meta,
	}
}

// MarshalJSONAPI returns a shallow copy of this resource.
func (r Resource) MarshalJSONAPI() (*Resource, error) {
	return &r, nil
}

// UnmarshalJSONAPI extracts information from a resource node and populates itself.
func (r *Resource) UnmarshalJSONAPI(other *Resource) error {
	r.Attributes = other.Attributes
	r.Extensions = other.Extensions
	r.ID = other.ID
	r.Type = other.Type
	r.Links = other.Links
	r.LocalID = other.LocalID
	r.Meta = other.Meta
	r.Relationships = other.Relationships
	return nil
}

func (r Resource) nodeid() string {
	return fmt.Sprint(r.Type, r.ID)
}

// Relationship describes a resource's relationships. Relationships are
// contain either "to-many" or "to-one" associations with the parent resource.
// See https://jsonapi.org/format/#document-resource-object-relationships
// for details.
type Relationship struct {
	Data  PrimaryData `json:"data,omitempty"`  // Relationship data containing associated references.
	Links Links       `json:"links,omitempty"` // URL links related to the relationship.
	Meta  Meta        `json:"meta,omitempty"`  // Non-standard information related to the relationship.
}

// UnmarshalJSON deserializes this relationship from JSON.
func (r *Relationship) UnmarshalJSON(data []byte) error {
	type in struct {
		Data  *json.RawMessage `json:"data,omitempty"`
		Meta  Meta             `json:"meta,omitempty"`
		Links Links            `json:"links,omitempty"`
	}

	node := node[in]{}

	errs := make([]error, 0)
	errs = append(errs, json.Unmarshal(data, &node))

	r.Meta = node.value.Meta
	r.Links = node.value.Links

	ref := node.value.Data

	if node.ContainsAttribute("data") && ref == nil {
		// the "data" attribute was set to null; use the zero value
		// for one node.
		r.Data = One{}
	} else if ref == nil {
		// nothing more to process, quit early
		return errors.Join(errs...)
	} else if jsontest.IsJSONArray(ref) {
		// the primary data contains multiple resources.
		many := Many{}
		errs = append(errs, json.Unmarshal(*ref, &many))
		r.Data = many
	} else if jsontest.IsJSONObject(ref) {
		// the primary data contains a single resource.
		one := One{}
		errs = append(errs, json.Unmarshal(*ref, &one))
		r.Data = one
	}

	return errors.Join(errs...)
}

// PrimaryData interfaces provide document primary data or resource relationship data.
// Since data can be either a single resource or a collection of resources, PrimaryData
// has helper functions to both identify and iterate over said resources.
type PrimaryData interface {
	// Items returns the contained items as a collection of resources.
	//	- If the data node represents a null "to-one" relationship, then the slice will be empty.
	//	- If the data node represents a "to-one" relationship, then the slice will contain the
	//		associated resource at the first index.
	//	- If the data node represents a "to-many" relationship, then the slice will contain
	//		the associated resources.
	Items() []*Resource
	// IsMany returns true if the data represents a "to-many" relationship or collection of primary data.
	IsMany() bool
	// First returns the first item in a primary data node -- the node itself for single or "one"
	// primary data, or the first element in multi or "many" primary data.
	//
	// If the data node is set to "null" (a jsonapi.One instance with a nil value) then nil is returned.
	// If the data node itself is nil, First() panics.
	First() *Resource
}

// First returns the first item in a primary data node -- the node itself for single or "one"
// primary data, or the first element in multi or "many" primary data.
//
// If the data node is set to "null" (a jsonapi.One instance with a nil value) then nil is returned.
// If the data node itself is nil, First() panics.
//
// Deprecated: First is deprecated and will be removed in the next major release. Use the First()
// method on the PrimaryData interface instead.
func First(data PrimaryData) *Resource {
	log.Println("The First() function is deprecated and will be removed in the next major release. Use the PrimaryData.First() method instead.")
	items := data.Items()
	if len(items) == 0 {
		return nil
	}
	return items[0]
}

// IsMany returns false, as this is a "to-one" relationship.
func (One) IsMany() bool { return false }

// IsMany returns true, as this is a "to-many" relationship.
func (Many) IsMany() bool { return true }

// One is a data node that represents either a "to-one" relationship or a document's primary data.
type One struct {
	Value *Resource `json:"-"` // Value is the single resource.
}

// First returns the resource value (if present) or nil.
func (o One) First() *Resource {
	return o.Value
}

// MarshalJSON serializes the node to JSON.
func (o One) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.Value)
}

// UnmarshalJSON deserializes the node from JSON.
func (o *One) UnmarshalJSON(data []byte) error {
	o.Value = &Resource{}
	return json.Unmarshal(data, o.Value)
}

// IsNull returns true if the node value is nil.
func (o One) IsNull() bool {
	return o.Value == nil
}

// Items returns the underlying value in a collection.
func (o One) Items() []*Resource {
	if !o.IsNull() {
		return []*Resource{o.Value}
	}
	return nil
}

// Many is a data node that represents a "to-many" relationship or a document's primary data.
type Many struct {
	Value []*Resource `json:"-"` // Value is the collection of resources.
}

// First returns the first item in the array of resources.
func (m Many) First() *Resource {
	return m.Value[0]
}

// MarshalJSON serializes the node to JSON.
func (m Many) MarshalJSON() ([]byte, error) {
	if m.Value == nil {
		m.Value = []*Resource{}
	}
	return json.Marshal(m.Value)
}

// UnmarshalJSON deserializes the node from JSON.
func (m *Many) UnmarshalJSON(data []byte) error {
	m.Value = make([]*Resource, 0)
	return json.Unmarshal(data, &m.Value)
}

// Items returns the underlying value collection.
func (m Many) Items() []*Resource {
	return m.Value
}

// Links contains the links defined within a resource, document, or error.
type Links = map[string]*Link

// Link represents a single link.
type Link struct {
	Href     string   // URI-reference pointing to the link’s target.
	Rel      string   // The link’s relation type.
	Type     string   // The media type of the link’s target.
	Title    string   // Human-readable link destination.
	HrefLang HrefLang // Array of strings indicating the link's target language(s).
	Meta     Meta     // Contains non-standard meta-information about the link.
}

// MarshalJSON serializes this link to JSON.
func (l Link) MarshalJSON() ([]byte, error) {
	if l.isHrefOnly() {
		return json.Marshal(l.Href)
	}

	type out struct {
		Href     string   `json:"href"`
		Rel      string   `json:"rel,omitempty"`
		Type     string   `json:"type,omitempty"`
		Title    string   `json:"title,omitempty"`
		HrefLang HrefLang `json:"hreflang,omitempty"`
		Meta     Meta     `json:"meta,omitempty"`
	}

	return json.Marshal(out(l))
}

// UnmarshalJSON deserializes this link from JSON.
func (l *Link) UnmarshalJSON(data []byte) error {
	raw := json.RawMessage(data)
	if !jsontest.IsJSONObject(&raw) {
		// likely a string, marshal to the href field
		return json.Unmarshal(data, &l.Href)
	}

	type in struct {
		Href     string   `json:"href"`
		Rel      string   `json:"rel,omitempty"`
		Type     string   `json:"type,omitempty"`
		Title    string   `json:"title,omitempty"`
		HrefLang HrefLang `json:"hreflang,omitempty"`
		Meta     Meta     `json:"meta,omitempty"`
	}

	obj := in{}
	err := json.Unmarshal(data, &obj)

	l.Href = obj.Href
	l.Rel = obj.Rel
	l.Type = obj.Type
	l.Title = obj.Title
	l.HrefLang = obj.HrefLang
	l.Meta = obj.Meta

	return err
}

func (l Link) isHrefOnly() bool {
	return l.Href != "" &&
		l.Rel == "" &&
		l.Type == "" &&
		l.Title == "" &&
		l.HrefLang.isEmpty() &&
		l.Meta == nil
}

// HrefLang is a string or an array of strings indicating the language(s) of the link’s target.
// An array of strings indicates that the link’s target is available in multiple languages.
// Each string MUST be a valid language tag [RFC5646].
//
// HrefLang is serialized as a string if there is only one element within the slice, or an array
// of strings otherwise.
type HrefLang []string

// UnmarshalJSON provides custom JSON deserialization.
func (h *HrefLang) UnmarshalJSON(data []byte) error {
	raw := json.RawMessage(data)
	errs := make([]error, 0)

	if jsontest.IsJSONArray(&raw) {
		strarray := make([]string, 0)
		errs = append(errs, json.Unmarshal(data, &strarray))
		*h = append(*h, strarray...)
		return errors.Join(errs...)
	}

	lang := ""
	errs = append(errs, json.Unmarshal(data, &lang))
	*h = append(*h, lang)
	return errors.Join(errs...)
}

// MarshalJSON provides custom JSON deserialization.
func (h HrefLang) MarshalJSON() ([]byte, error) {
	if len(h) == 1 {
		return json.Marshal(h[0])
	}
	return json.Marshal([]string(h))
}

// isEmpty returns true if the slice is empty.
func (h HrefLang) isEmpty() bool {
	return len(h) == 0
}

// Error provides additional information about problems encountered
// while performing an operation. Error objects MUST be returned
// as an array keyed by errors in the top level of a JSON:API document.
type Error struct {
	ID     string       `json:"id,omitempty"`     // A unique identifier for this particular occurrence of the problem.
	Links  Links        `json:"links,omitempty"`  // A links object associated with the error.
	Status string       `json:"status,omitempty"` // The HTTP status code applicable to this problem.
	Code   string       `json:"code,omitempty"`   // An application-specific error code.
	Title  string       `json:"title,omitempty"`  // A short summary of the problem.
	Detail string       `json:"detail,omitempty"` // A specific explanation of the problem.
	Source *ErrorSource `json:"source,omitempty"` // An object containing references to the primary source of the error.
	Meta   Meta         `json:"meta,omitempty"`   // Contains non-standard meta-information about the error.
}

// Error returns the combined title and detail as a single message.
func (e Error) Error() string {
	if e.Title != "" {
		return fmt.Sprintf("%s: %s", e.Title, e.Detail)
	}
	return e.Detail
}

// ErrorSource is an object containing references to the primary source of the error.
type ErrorSource struct {
	// A JSON Pointer [RFC6901] to the value in the request document
	// that caused the error [e.g. "/data" for a primary data object,
	// or "/data/attributes/title" for a specific attribute].
	Pointer string `json:"pointer,omitempty"`

	Parameter string `json:"parameter,omitempty"` // URI query parameter that caused the error.
	Header    string `json:"header,omitempty"`    // Name of a single request header which caused the error.
}

type node[T any] struct {
	value    T
	ext      map[string]*json.RawMessage
	contains []string
}

func (n node[T]) ContainsAttribute(key string) bool {
	return slices.Contains(n.contains, key)
}

func (n node[T]) MarshalJSON() ([]byte, error) {
	data, err := json.Marshal(n.value)
	if err == nil {
		errs := make([]error, 0)
		raw := make(map[string]*json.RawMessage)
		errs = append(errs, json.Unmarshal(data, &raw))

		for k, v := range n.ext {
			raw[k] = v
		}

		data, err = json.Marshal(raw)
		errs = append(errs, err)
		err = errors.Join(errs...)
	}
	return data, err
}

func (n *node[T]) UnmarshalJSON(data []byte) error {
	errs := make([]error, 0)
	errs = append(errs, json.Unmarshal(data, &n.value))

	// partially unmarshal the object; remove any keys that do not
	// fit the extensions format (ex., "namespace:key")
	raw := make(map[string]*json.RawMessage)
	errs = append(errs, json.Unmarshal(data, &raw))
	for key := range raw {
		n.contains = append(n.contains, key)
		if !strings.Contains(key, ":") {
			delete(raw, key)
		}
	}

	// take what's left over, serialize and deserialize into the extensions map
	if len(raw) != 0 {
		n.ext = raw
	}

	return errors.Join(errs...)
}
