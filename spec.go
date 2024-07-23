package jsonapi

import (
	"errors"
	"fmt"

	"github.com/gonobo/validator"
)

type Specification interface {
	validate(*Document) error
}

type SpecificationVersion string

const (
	Version1_1 SpecificationVersion = "1.1"
)

func ValidateSpec(doc *Document) error {
	var spec Specification = no_spec{}
	switch SpecificationVersion(doc.Jsonapi.Version.Value()) {
	case Version1_1:
		spec = spec1_1{}
	}
	return spec.validate(doc)
}

type no_spec struct{}

func (no_spec) validate(d *Document) error {
	return nil
}

type spec1_1 struct{}

func (s spec1_1) validate(d *Document) error {
	err := errors.Join(
		s.checkMissingTopLevelMembers(d),
	)
	if err != nil {
		err = fmt.Errorf("spec 1.1: %w", err)
	}
	return err
}

func (spec1_1) checkMissingTopLevelMembers(d *Document) error {
	// A document must contain at least the following top-level members:
	//	- data
	//	- errors
	//	- meta
	valid := d.Data != nil || len(d.Errors) > 0 || len(d.Meta) > 0
	return validator.Validate(
		validator.Rule(valid, "document must contain at least 'data', 'errors', or 'meta' members"),
	)
}
