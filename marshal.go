package jsonapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

const (
	tagJSONAPI   = "jsonapi"
	tagAttribute = "attr"
	tagPrimary   = "primary"
	tagRelation  = "relation"
	tagExtension = "ext"
	tagOmitEmpty = "omitempty"
	tagValueSkip = "-"
	tagDelimiter = ","
)

// Marshal generates a JSON:API document from the specified value. If the value
// is a struct, then a single document is returned, using the value as primary data.
// If the value is a slice or array, then a many document is returned, using the
// value as primary data.
//
// Marshaling Document structs simply returns a copy of the instance.
func Marshal(in any) (Document, error) {
	// if the input is already a document, return it.
	if doc, ok := in.(Document); ok {
		return doc, nil
	} else if doc, ok := in.(*Document); ok {
		return *doc, nil
	}

	// use reflection to determine if the document has single or multiple primary data.
	rtype := reflect.TypeOf(in)
	switch rtype.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		return marshalManyDocument(in)
	default:
		return marshalOneDocument(in)
	}
}

// MarshalRef generates a JSON:API document, using a resource's relationship as primary data.
// It otherwise functions in the same manner as Marshal().
func MarshalRef(in any, name string) (Document, error) {
	document := Document{}

	if name == "" {
		return document, jsonapiError("marshal ref: name is empty string")
	}

	document, err := Marshal(in)

	if err != nil {
		return document, jsonapiError("marshal ref: %s", err)
	}

	if document.Data.IsMany() {
		return document, jsonapiError("marshal ref: cannot marshal ref from primary data array")
	}

	one := document.Data.Items()[0]
	ref, ok := one.Relationships[name]

	if !ok {
		return document, jsonapiError("marshal ref: unknown relationship: %s", name)
	}

	// hoist ref data to primary data
	document.Links = ref.Links
	document.Meta = ref.Meta
	document.Data = ref.Data
	document.Included = append(document.Included, one)

	return document, nil
}

func marshalOneDocument(in any) (Document, error) {
	document := Document{}

	includes := make(map[string]*Resource)
	node, err := marshalResource(reflect.ValueOf(in), includes)

	if err == nil {
		document.Data = One{Value: node}

		// remove the primary data node from the inclusion list
		delete(includes, node.nodeid())

		for _, item := range includes {
			// add includes to document
			document.Included = append(document.Included, item)
		}
	}

	return document, err
}

func marshalManyDocument(in any) (Document, error) {
	document := Document{}
	slice := reflect.ValueOf(in)
	many := Many{}

	errs := make([]error, 0)
	includes := make(map[string]*Resource)

	for idx := 0; idx < slice.Len(); idx++ {
		node, err := marshalResource(slice.Index(idx), includes)
		if err != nil {
			errs = append(errs, jsonapiError("index %d: %s", idx, err))
		}
		many.Value = append(many.Value, node)
	}

	err := errors.Join(errs...)

	if err == nil {
		document.Data = many

		for _, item := range many.Items() {
			// remove primary data from includes
			delete(includes, item.nodeid())
		}

		for _, item := range includes {
			// add includes to document
			document.Included = append(document.Included, item)
		}
	}

	return document, err
}

// MarshalResource generates a JSON:API resource object based on
// the input struct's fields. Fields must be tagged with "jsonapi:"
// in order to be processed by the marshaler, or the struct type
// can implement MarshalResourceJSONAPI() to override the process.
func MarshalResource(in any) (*Resource, error) {
	includes := make(map[string]*Resource)
	node, err := marshalResource(reflect.ValueOf(in), includes)
	return node, err
}

// ResourceMarshaler can marshal its information into a resource node.
// Structs that implement this interface can override the default marshaling
// process.
type ResourceMarshaler interface {
	// MarshalJSONAPI marshals this instance into a resource node.
	MarshalJSONAPI() (*Resource, error)
}

// LinksMarshaler creates links associated with the instance when marshaled.
type LinksMarshaler interface {
	// MarshalLinksJSONAPI returns links associated with the instance when marshaled.
	MarshalLinksJSONAPI() Links
}

// RelatedLinksMarshaler creates links associated with an instance's relationships when marshaled.
type RelatedLinksMarshaler interface {
	// MarshalRelatedLinksJSONAPI returns links associated with an instance's relationships when marshaled.
	MarshalRelatedLinksJSONAPI(name string) Links
}

// MetaMarshaler creates metadata associated with the instance when marshaled.
type MetaMarshaler interface {
	// MarshalMetaJSONAPI returns metadata associated with the instance when marshaled.
	MarshalMetaJSONAPI() Meta
}

// RelatedMetaMarshaler creates metadata associated with an instance's relationships when marshaled.
type RelatedMetaMarshaler interface {
	// MarshalRelatedMetaJSONAPI returns metadata associated with an instance's relationships when marshaled.
	MarshalRelatedMetaJSONAPI(name string) Meta
}

// MarshalRaw serializes the input value to its JSON equivalent,
// wrapped in a RawMessage type for ease of use. Typically used
// for marshaling extensions within a JSON:API document.
func MarshalRaw(value any) (*json.RawMessage, error) {
	data, err := json.Marshal(value)
	raw := json.RawMessage(data)
	return &raw, err
}

// Unmarshal populates the output struct or slice with information
// stored inside the provided document. Struct fields must either be properly
// tagged with "jsonapi:" or the struct must implement the
// UnmarshalResourceJSONAPI() method.
func Unmarshal(doc *Document, out any) error {
	// use reflection to determine if the document has single or multiple primary data.
	rtype := reflect.TypeOf(out)

	if rtype.Kind() != reflect.Pointer {
		return jsonapiError("unmarshal: output must be a pointer to a struct, slice, or array")
	}

	ptrKind := pointerKind(reflect.ValueOf(out))
	switch ptrKind {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		return unmarshalManyDocument(doc, out)
	default:
		return unmarshalOneDocument(doc, out)
	}
}

func unmarshalOneDocument(doc *Document, out any) error {
	if doc.Data == nil {
		return jsonapiError("unmarshal: primary data is null or missing")
	} else if item := doc.Data.First(); item == nil {
		return jsonapiError("unmarshal: primary data is null or missing")
	} else {
		return unmarshalResource(item, reflect.ValueOf(out))
	}
}

func unmarshalManyDocument(doc *Document, out any) error {
	slice := reflect.Indirect(reflect.ValueOf(out))
	errs := make([]error, 0)

	for _, resource := range doc.Data.Items() {
		itemType := slice.Type().Elem()
		item := newItem(itemType)
		errs = append(errs, unmarshalResource(resource, item))
		slice.Set(appendSlice(itemType, slice, item))
	}

	return errors.Join(errs...)
}

// UnmarshalResource populates the output struct's fields with information
// stored inside the provided resource node. Fields must either be properly
// tagged with "jsonapi:" or the struct must implement the
// UnmarshalJSONAPI() method.
func UnmarshalResource(node *Resource, out any) error {
	return unmarshalResource(node, reflect.ValueOf(out))
}

// ResourceUnmarshaler can extract information from a resource node and populate itself.
// This is an escape hatch from the default unmarshal process.
type ResourceUnmarshaler interface {
	// UnmarshalJSONAPI extracts information from a resource node and populates itself.
	UnmarshalJSONAPI(*Resource) error
}

// LinksUnmarshaler can extract links from a resource node and populate itself.
type LinksUnmarshaler interface {
	// UnmarshalLinksJSONAPI extracts links from a resource node and populates itself.
	UnmarshalLinksJSONAPI(Links)
}

// RelatedLinksUnmarshaler can extract relationship links from a node and populate itself.
type RelatedLinksUnmarshaler interface {
	// UnmarshalRelatedLinksJSONAPI extracts relationship links from a node and populate itself.
	UnmarshalRelatedLinksJSONAPI(name string, links Links)
}

// MetaUnmarshaler can extract metadata from a resource node and populate itself.
type MetaUnmarshaler interface {
	// UnmarshalMetaJSONAPI extracts metadata from a resource node and populates itself.
	UnmarshalMetaJSONAPI(Meta)
}

// RelatedMetaUnmarshaler can extract relationship metadata from a node and populate itself.
type RelatedMetaUnmarshaler interface {
	// UnmarshalRelatedMetaJSONAPI extracts relationship metadata from a node and populates itself.
	UnmarshalRelatedMetaJSONAPI(name string, meta Meta)
}

func unmarshalResource(node *Resource, root reflect.Value) error {
	rtype := root.Type()

	if isNilValue(root) {
		return jsonapiError("cannot unmarshal to nil value")
	} else if root.Kind() == reflect.Pointer {
		root = reflect.Indirect(root)
		rtype = root.Type()
	}

	ptr := root.Addr().Interface()

	// if the value is a resource unmarshaler, defer to it.
	if unmarshaler, ok := ptr.(ResourceUnmarshaler); ok {
		return unmarshaler.UnmarshalJSONAPI(node)
	}

	links := node.Links
	if unmarshaler, ok := ptr.(LinksUnmarshaler); ok && links != nil {
		unmarshaler.UnmarshalLinksJSONAPI(links)
	}

	meta := node.Meta
	if unmarshaler, ok := ptr.(MetaUnmarshaler); ok && meta != nil {
		unmarshaler.UnmarshalMetaJSONAPI(meta)
	}

	errs := make([]error, 0)

	for idx := 0; idx < rtype.NumField(); idx++ {
		field := rtype.Field(idx)
		tag := field.Tag.Get(tagJSONAPI)

		if tag == "" {
			continue
		}

		value := root.Field(idx)
		tokens := strings.Split(tag, tagDelimiter)

		switch tokens[0] {
		case tagPrimary:
			errs = append(errs, unmarshalIdentity(node, value, tokens))
		case tagAttribute:
			errs = append(errs, unmarshalAttribute(node, value, tokens))
		case tagRelation:
			errs = append(errs, unmarshalRelation(node, root, value, tokens))
		case tagExtension:
			errs = append(errs, unmarshalExtension(node, value, tokens))
		}
	}

	return errors.Join(errs...)
}

func isNilValue(value reflect.Value) bool {
	return value.Kind() == reflect.Pointer && value.IsNil()
}

func pointerKind(value reflect.Value) reflect.Kind {
	return reflect.Indirect(value).Type().Kind()
}

func omitEmptyValue(value reflect.Value, omitempty bool) bool {
	if omitempty && isNilValue(value) {
		return true
	} else if omitempty && value.IsZero() {
		return true
	}
	return false
}

func unmarshalIdentity(node *Resource, value reflect.Value, tokens []string) error {
	value.SetString(node.ID)

	wantType := tokens[1]
	err := jsonapiError("unmarshal: want resource type '%s', got '%s'", wantType, node.Type)

	if wantType == node.Type {
		err = nil
	}

	return err
}

func unmarshalAttribute(node *Resource, value reflect.Value, tokens []string) error {
	name := tokens[1]
	attr, ok := node.Attributes[name]
	if !ok {
		return nil
	}

	errs := make([]error, 0)
	attrValue := reflect.ValueOf(attr)
	vtype := value.Type()
	if attrValue.Kind() == reflect.Map || attrValue.Kind() == reflect.Slice {
		// the attribute was a JSON object or array,
		// so leverage JSON marshaling/unmarshaling.
		data, err := json.Marshal(attr)
		errs = append(errs, err)

		ptr := newItem(vtype)
		ptrValue := ptr.Interface()
		errs = append(errs, json.Unmarshal(data, &ptrValue))

		setValue(vtype, value, ptr)
	} else if attrValue.Type().AssignableTo(vtype) {
		value.Set(reflect.ValueOf(attr))
	} else if attrValue.CanConvert(vtype) {
		value.Set(attrValue.Convert(vtype))
	}

	return errors.Join(errs...)
}

func unmarshalExtension(node *Resource, value reflect.Value, tokens []string) error {
	var name, namespace string

	name = tokens[1]
	namespace = tokens[2]

	attrName := fmt.Sprintf("%s:%s", namespace, name)
	data := node.Extensions[attrName]

	if data == nil {
		// nothing to do here, no raw data associated with the extension
		return nil
	}

	vtype := value.Type()
	ptr := newItem(vtype)
	ptrValue := ptr.Interface()

	err := json.Unmarshal(*data, &ptrValue)
	setValue(vtype, value, ptr)

	return err
}

func unmarshalRelation(node *Resource, root, value reflect.Value, tokens []string) error {
	if node.Relationships == nil {
		return nil
	}

	name := tokens[1]
	relation, ok := node.Relationships[name]
	if !ok {
		return nil
	}

	ptr := root.Addr().Interface()

	meta := relation.Meta
	if unmarshaler, ok := ptr.(RelatedMetaUnmarshaler); ok && meta != nil {
		unmarshaler.UnmarshalRelatedMetaJSONAPI(name, meta)
	}

	links := relation.Links
	if unmarshaler, ok := ptr.(RelatedLinksUnmarshaler); ok && links != nil {
		unmarshaler.UnmarshalRelatedLinksJSONAPI(name, links)
	}

	if relation.Data == nil {
		return nil
	}

	errs := make([]error, 0)
	items := relation.Data.Items()

	if relation.Data.IsMany() {
		errs = append(errs, unmarshalMany(items, value))
	} else {
		errs = append(errs, unmarshalOne(items, value))
	}

	return errors.Join(errs...)
}

func unmarshalOne(nodes []*Resource, value reflect.Value) error {
	vtype := value.Type()
	var err error = nil

	if len(nodes) > 0 {
		item := nodes[0]
		ptr := newItem(vtype)
		err = unmarshalResource(item, ptr)
		setValue(vtype, value, ptr)
	} else if vtype.Kind() == reflect.Pointer {
		// set the item to nil, per JSON:API specification
		value.Set(reflect.Zero(vtype))
	}

	return err
}

func unmarshalMany(nodes []*Resource, value reflect.Value) error {
	errs := make([]error, 0)
	// set type to the slice's element type ([]T -> T)
	vtype := value.Type().Elem()
	slicePtr := newSlice(vtype)
	slice := slicePtr.Elem()
	for _, item := range nodes {
		ptr := newItem(vtype)
		errs = append(errs, unmarshalResource(item, ptr))
		slice.Set(appendSlice(vtype, slice, ptr))
	}
	value.Set(slice)
	return errors.Join(errs...)
}

func setValue(vtype reflect.Type, target reflect.Value, ptr reflect.Value) {
	if vtype.Kind() == reflect.Pointer {
		target.Set(ptr)
	} else {
		target.Set(ptr.Elem())
	}
}

func newItem(vtype reflect.Type) reflect.Value {
	if vtype.Kind() == reflect.Pointer {
		return reflect.New(vtype.Elem())
	}
	return reflect.New(vtype)
}

func newSlice(vtype reflect.Type) reflect.Value {
	mkSlice := reflect.MakeSlice(reflect.SliceOf(vtype), 0, 0)
	slice := reflect.New(mkSlice.Type())
	slice.Elem().Set(mkSlice)
	slicePtr := reflect.ValueOf(slice.Interface())
	return slicePtr
}

func appendSlice(elementType reflect.Type, slice reflect.Value, itemPtr reflect.Value) reflect.Value {
	item := reflect.Indirect(itemPtr)
	if elementType.Kind() == reflect.Pointer {
		return reflect.Append(slice, item.Addr())
	} else {
		return reflect.Append(slice, item)
	}
}

func marshalIdentity(value reflect.Value, tokens []string) (rID string, rType string, error error) {
	switch len(tokens) {
	case 2:
		rType = tokens[1]
	default:
		error = jsonapiError("missing resource type from primary tag")
		return
	}

	rID = fmt.Sprintf("%v", value)
	return
}

func marshalResource(rvalue reflect.Value, includes map[string]*Resource) (*Resource, error) {
	if !rvalue.IsValid() {
		return nil, jsonapiError("marshal resource: value is invalid")
	} else if rvalue.Kind() == reflect.Pointer && rvalue.IsNil() {
		return nil, jsonapiError("marshal resource: value is nil")
	} else if rvalue.Kind() == reflect.Pointer {
		// reflect the underlying type
		rvalue = reflect.Indirect(rvalue)
	}

	// if the value is a resource marshaler, defer to it.
	if marshaler, ok := rvalue.Interface().(ResourceMarshaler); ok {
		return marshaler.MarshalJSONAPI()
	}

	rtype := rvalue.Type()

	errs := make([]error, 0)

	node := &Resource{
		Attributes:    make(map[string]any),
		Relationships: make(map[string]*Relationship),
		Extensions:    make(map[string]*json.RawMessage),
	}

	relationshipIndexes := make([]int, 0)

	for idx := 0; idx < rtype.NumField(); idx++ {
		field := rtype.Field(idx)
		tag := field.Tag.Get(tagJSONAPI)

		if tag == "" {
			continue
		}

		value := rvalue.Field(idx)
		tokens := strings.Split(tag, tagDelimiter)

		switch tokens[0] {
		case tagPrimary:
			nodeID, nodeType, err := marshalIdentity(value, tokens)
			node.ID = nodeID
			node.Type = nodeType
			errs = append(errs, err)
		case tagAttribute:
			errs = append(errs, marshalAttribute(value, tokens, node))
		case tagRelation:
			// enqueue relationship index for resolution
			relationshipIndexes = append(relationshipIndexes, idx)
		case tagExtension:
			errs = append(errs, marshalExtension(value, tokens, node))
		}
	}

	if node.Type == "" {
		return nil, jsonapiError("missing primary jsonapi tag")
	}

	// Before iterating, memoize
	// this resource to avoid infinite loops from cyclic
	// references.

	includes[node.nodeid()] = node

	// now that the primary identifier has been resolved,
	// the resource's relationships (and any possible inclusions)
	// can now be resolved as well.

	for _, idx := range relationshipIndexes {
		field := rtype.Field(idx)
		value := rvalue.Field(idx)
		tokens := strings.Split(field.Tag.Get(tagJSONAPI), tagDelimiter)
		errs = append(errs, marshalRelationship(rvalue, value, tokens, node, includes))
	}

	return node, errors.Join(errs...)
}

func marshalAttribute(value reflect.Value, tokens []string, node *Resource) error {
	var name string
	var omitEmpty bool

	switch len(tokens) {
	case 2:
		name = tokens[1]
	case 3:
		name = tokens[1]
		omitEmpty = tokens[2] == tagOmitEmpty
	}

	if omitEmptyValue(value, omitEmpty) {
		return nil
	} else if isNilValue(value) {
		node.Attributes[name] = nil
	} else {
		node.Attributes[name] = value.Interface()
	}

	return nil
}

func marshalExtension(value reflect.Value, tokens []string, node *Resource) error {
	// ex: "jsonapi:ext,<name>,<namespace>[,omitempty]"
	var name, namespace string
	var omitEmpty bool

	switch len(tokens) {
	case 3:
		name = tokens[1]
		namespace = tokens[2]
	case 4:
		name = tokens[1]
		namespace = tokens[2]
		omitEmpty = tokens[3] == tagOmitEmpty
	}

	attribute := fmt.Sprintf("%s:%s", namespace, name)

	if omitEmptyValue(value, omitEmpty) {
		return nil
	} else if isNilValue(value) {
		node.Extensions[attribute] = nil
		return nil
	} else {
		raw, err := MarshalRaw(value.Interface())
		node.Extensions[attribute] = raw
		return err
	}
}

func marshalRelationship(parent reflect.Value,
	value reflect.Value,
	tokens []string,
	node *Resource,
	includes map[string]*Resource) error {
	var name string
	var omitEmpty bool

	switch len(tokens) {
	case 2:
		name = tokens[1]
	case 3:
		name = tokens[1]
		omitEmpty = tokens[2] == tagOmitEmpty
	}

	relationship := &Relationship{}
	node.Relationships[name] = relationship

	model := reflect.Indirect(parent).Interface()
	if marshaler, ok := model.(RelatedLinksMarshaler); ok {
		relationship.Links = marshaler.MarshalRelatedLinksJSONAPI(name)
	}
	if marshaler, ok := model.(RelatedMetaMarshaler); ok {
		relationship.Meta = marshaler.MarshalRelatedMetaJSONAPI(name)
	}

	var err = jsonapiError("relation must be pointer or slice")

	switch value.Kind() {
	case reflect.Pointer:
		err = marshalOneRef(value, relationship, omitEmpty, includes)
	case reflect.Slice:
		err = marshalManyRef(value, relationship, omitEmpty, includes)
	}

	return err
}

func marshalManyRef(value reflect.Value,
	node *Relationship,
	omit bool,
	includes map[string]*Resource) error {
	if omit && value.Len() == 0 {
		return nil
	}
	node.Data = Many{}
	refs := make([]*Resource, 0, value.Len())
	for idx := 0; idx < value.Len(); idx++ {
		item := reflect.Indirect(value.Index(idx))
		include, err := marshalResource(item, includes)
		if err != nil {
			return err
		}
		refs = append(refs, include.Ref())
	}
	node.Data = Many{Value: refs}
	return nil
}

func marshalOneRef(value reflect.Value,
	node *Relationship,
	omit bool,
	includes map[string]*Resource) error {
	include, err := marshalResource(value, includes)
	if include == nil && omit {
		return nil
	} else if include == nil {
		node.Data = One{}
		return nil
	}

	node.Data = One{Value: include.Ref()}
	return err
}
