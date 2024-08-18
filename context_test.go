package jsonapi_test

import (
	"context"
	"testing"

	"github.com/gonobo/jsonapi/v2"
	"github.com/stretchr/testify/assert"
)

func TestContext(t *testing.T) {
	t.Run("child", func(t *testing.T) {
		ctx := jsonapi.RequestContext{}
		child := ctx.Child()
		assert.True(t, child.Parent() == &ctx)
	})

	t.Run("empty child", func(t *testing.T) {
		ctx := jsonapi.RequestContext{ResourceType: "items", ResourceID: "1", Relationship: "foo", Related: true}
		child := ctx.EmptyChild()
		assert.True(t, child.Parent() == &ctx)
		assert.Equal(t, "", child.ResourceID)
		assert.Equal(t, "", child.ResourceType)
		assert.Equal(t, "", child.Relationship)
		assert.False(t, child.Related)
	})

	t.Run("root", func(t *testing.T) {
		ctx := jsonapi.RequestContext{}
		child := ctx.Child()
		assert.True(t, child.Root() == &ctx)
	})

	t.Run("get context", func(t *testing.T) {
		assert.Panics(t, func() {
			jsonapi.FromContext(context.TODO())
		})
	})

	t.Run("nil context", func(t *testing.T) {
		ctx := jsonapi.WithContext(context.Background(), nil)
		assert.Panics(t, func() {
			jsonapi.FromContext(ctx)
		})
	})

	t.Run("set context", func(t *testing.T) {
		ctx := jsonapi.WithContext(context.Background(), &jsonapi.RequestContext{})
		jsonapictx := jsonapi.FromContext(ctx)
		assert.NotNil(t, jsonapictx)
	})
}
