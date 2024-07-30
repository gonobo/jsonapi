package spec

import (
	"errors"
	"fmt"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/validator"
)

const (
	Version11 = "1.1"
)

type Validator struct{}

func (v Validator) ValidateDocument(doc *jsonapi.Document) error {
	var validator documentValidator = baseSpec{}
	version := doc.Jsonapi.Version.Value()
	switch version {
	case Version11:
		validator = spec11{}
	}
	return validator.validate(doc)
}

type documentValidator interface {
	validate(*jsonapi.Document) error
}

type baseSpec struct{}

func (baseSpec) validate(*jsonapi.Document) error {
	return nil
}

type spec11 struct{}

func (s spec11) validate(d *jsonapi.Document) error {
	err := errors.Join(
		baseSpec{}.validate(d),
		s.checkMissingTopLevelMembers(d),
	)
	if err != nil {
		err = fmt.Errorf("spec 1.1: %w", err)
	}
	return err
}

func (spec11) checkMissingTopLevelMembers(d *jsonapi.Document) error {
	// A document must contain at least the following top-level members:
	//	- data
	//	- errors
	//	- meta
	valid := d.Data != nil || len(d.Errors) > 0 || len(d.Meta) > 0
	return validator.Validate(
		validator.Rule(valid, "document must contain at least 'data', 'errors', or 'meta' members"),
	)
}
