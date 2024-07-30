package spec_test

import (
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/gonobo/jsonapi/extra/spec"
	"github.com/stretchr/testify/assert"
)

func TestValidation(t *testing.T) {
	t.Run("using specification validation", func(t *testing.T) {
		doc := jsonapi.Document{}
		assert.Error(t, doc.Validate(spec.Validator{}), "should raise an error on empty document")

		doc.Jsonapi.Version = jsonapi.Version("0.0")
		assert.NoError(t, doc.Validate(spec.Validator{}), "unknown version has no validation")
	})
}
