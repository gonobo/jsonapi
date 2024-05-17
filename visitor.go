package jsonapi

import "errors"

// Visitor can traverse a JSON:API document tree. Use the subvisitors to
// inspect or modify the document in place.
type Visitor struct {
	DocumentVisitor
	LinksVisitor
	MetaVisitor
	ResourceVisitor
	RelationshipVisitor
	ErrorVisitor
}

// DocumentVisitor visits JSON:API document nodes.
type DocumentVisitor interface {
	// VisitDocument visits the document. Return an error to stop visiting.
	VisitDocument(obj *Document) error
}

// LinksVisitor visits JSON:API link nodes.
type LinksVisitor interface {
	// VisitLinks visits the links. Return an error to stop visiting.
	VisitLinks(obj Links) error
	// VisitLink visits a link. Return an error to stop visiting.
	VisitLink(obj *Link) error
}

// MetaVisitor visits JSON:API meta nodes.
type MetaVisitor interface {
	// VisitMeta visits the meta node. Return an error to stop visiting.
	VisitMeta(obj Meta) error
}

// ResourceVisitor visits JSON:API resource nodes.
type ResourceVisitor interface {
	// VisitResource visits the resource node. Return an error to stop visiting.
	VisitResource(obj *Resource) error
}

// RelationshipVisitor visits JSON:API relationship nodes.
type RelationshipVisitor interface {
	// VisitRelationship visits the relationship node. Return an error to stop visiting.
	VisitRelationships(obj RelationshipsNode) error
	// VisitRelationship visits the relationship node. Return an error to stop visiting.
	VisitRelationship(obj *Relationship) error
	// VisitRelationship visits the resource node referenced in a relationship.
	// Return an error to stop visiting.
	VisitRef(obj *Resource) error
}

// ErrorVisitor visits JSON:API error nodes.
type ErrorVisitor interface {
	// VisitError visits the error node. Return an error to stop visiting.
	VisitError(obj *Error) error
}

// VisitorFunc can visit a node of the specified type.
type VisitorFunc[Node any] func(Node) error

// PartialVisitor can visit chosen nodes while ignoring others. For example,
// if you only want to visit the document node and its top links, you can use this to create
// a visitor that only visits the document node. If you want to visit all link nodes, add
// link visitor instead.
type PartialVisitor struct {
	Document      VisitorFunc[*Document]         // Function for visiting document nodes.
	Links         VisitorFunc[Links]             // Function for visiting links nodes.
	Link          VisitorFunc[*Link]             // Function for visiting link nodes.
	Meta          VisitorFunc[Meta]              // Function for visiting meta nodes.
	Resource      VisitorFunc[*Resource]         // Function for visiting resource nodes.
	Relationship  VisitorFunc[*Relationship]     // Function for visiting relationship nodes.
	Relationships VisitorFunc[RelationshipsNode] // Function for visiting relationships nodes.
	Ref           VisitorFunc[*Resource]         // Function for visiting resource nodes referenced in relationships.
	Error         VisitorFunc[*Error]            // Function for visiting error nodes.
}

// Visitor creates a visitor instance that can traverse a document.
func (v *PartialVisitor) Visitor() *Visitor {
	return &Visitor{
		DocumentVisitor:     v,
		LinksVisitor:        v,
		MetaVisitor:         v,
		ResourceVisitor:     v,
		RelationshipVisitor: v,
		ErrorVisitor:        v,
	}
}

// VisitDocument visits the document.
func (v PartialVisitor) VisitDocument(obj *Document) error {
	return visitNode(obj, v.Document, v.Document == nil)
}

// VisitLink visits a link.
func (v PartialVisitor) VisitLink(obj *Link) error {
	return visitNode(obj, v.Link, v.Link == nil)
}

// VisitLinks visits the links.
func (v PartialVisitor) VisitLinks(obj Links) error {
	return visitNode(obj, v.Links, v.Links == nil)
}

// VisitMeta visits the meta node.
func (v PartialVisitor) VisitMeta(obj Meta) error {
	return visitNode(obj, v.Meta, v.Meta == nil)
}

// VisitResource visits the resource node.
func (v PartialVisitor) VisitResource(obj *Resource) error {
	return visitNode(obj, v.Resource, v.Resource == nil)
}

// VisitRelationship visits the relationship node.
func (v PartialVisitor) VisitRelationship(obj *Relationship) error {
	return visitNode(obj, v.Relationship, v.Relationship == nil)
}

// VisitRelationships visits the relationships node.
func (v PartialVisitor) VisitRelationships(obj RelationshipsNode) error {
	return visitNode(obj, v.Relationships, v.Relationships == nil)
}

// VisitRef visits the resource node referenced in a relationship.
func (v PartialVisitor) VisitRef(obj *Resource) error {
	return visitNode(obj, v.Ref, v.Ref == nil)
}

// VisitError visits the error node.
func (v PartialVisitor) VisitError(obj *Error) error {
	return visitNode(obj, v.Error, v.Error == nil)
}

func visitNode[Node any](node Node, f VisitorFunc[Node], isNil bool) error {
	if isNil {
		return nil
	}
	return f(node)
}

func applyDocumentVisitor(d *Document, v *Visitor) error {
	errs := make([]error, 0)
	errs = append(errs, v.VisitDocument(d))

	if d.Links != nil {
		errs = append(errs, applyLinksVisitor(d.Links, v))
	}

	if d.Meta != nil {
		errs = append(errs, applyMetaVisitor(d.Meta, v))
	}

	if d.Data != nil {
		for _, item := range d.Data.Items() {
			errs = append(errs, applyResourceVisitor(item, v))
		}
	}

	for _, item := range d.Errors {
		errs = append(errs, applyErrorVisitor(item, v))
	}

	for _, item := range d.Included {
		errs = append(errs, applyResourceVisitor(item, v))
	}

	return errors.Join(errs...)
}

func applyErrorVisitor(e *Error, v *Visitor) error {
	errs := make([]error, 0)
	errs = append(errs, v.VisitError(e))

	if e.Meta != nil {
		errs = append(errs, applyMetaVisitor(e.Meta, v))
	}

	return errors.Join(errs...)
}

func applyLinksVisitor(l Links, v *Visitor) error {
	errs := make([]error, 0)
	errs = append(errs, v.VisitLinks(l))
	for _, link := range l {
		errs = append(errs, v.VisitLink(link))
	}
	return errors.Join(errs...)
}

func applyMetaVisitor(m Meta, v *Visitor) error {
	return v.VisitMeta(m)
}

func applyResourceVisitor(r *Resource, v *Visitor) error {
	errs := make([]error, 0)
	errs = append(errs, v.VisitResource(r))

	if r.Links != nil {
		errs = append(errs, applyLinksVisitor(r.Links, v))
	}

	if r.Meta != nil {
		errs = append(errs, applyMetaVisitor(r.Meta, v))
	}

	if r.Relationships != nil {
		errs = append(errs, applyRelationshipsVisitor(r.Relationships, v))
	}

	return errors.Join(errs...)
}

func applyRelationshipsVisitor(r RelationshipsNode, v *Visitor) error {
	errs := make([]error, 0)
	errs = append(errs, v.VisitRelationships(r))

	for _, relationship := range r {
		errs = append(errs, applyRelationshipVisitor(relationship, v))
	}

	return errors.Join(errs...)
}

func applyRelationshipVisitor(r *Relationship, v *Visitor) error {
	errs := make([]error, 0)
	errs = append(errs, v.VisitRelationship(r))

	if r.Links != nil {
		errs = append(errs, applyLinksVisitor(r.Links, v))
	}

	if r.Meta != nil {
		errs = append(errs, applyMetaVisitor(r.Meta, v))
	}

	if r.Data != nil {
		for _, item := range r.Data.Items() {
			errs = append(errs, applyRefVisitor(item, v))
		}
	}

	return errors.Join(errs...)
}

func applyRefVisitor(r *Resource, v *Visitor) error {
	errs := make([]error, 0)
	errs = append(errs, v.VisitRef(r))

	if r.Meta != nil {
		errs = append(errs, applyMetaVisitor(r.Meta, v))
	}

	return errors.Join(errs...)
}
