package jsonapi_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gonobo/jsonapi"
	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	err := jsonapi.NewError(errors.New("new error"), "title")
	assert.Error(t, err)
	assert.Equal(t, "title", err.Title)
	assert.Equal(t, "new error", err.Detail)

	wrapped := fmt.Errorf("wrapped error: %w", err)
	var as jsonapi.Error
	assert.True(t, errors.As(wrapped, &as))
}
