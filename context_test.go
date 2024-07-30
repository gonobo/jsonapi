package jsonapi_test

import (
	"context"
	"testing"

	"github.com/gonobo/jsonapi/v1"
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
		ctx, ok := jsonapi.Context(context.TODO())
		assert.Nil(t, ctx)
		assert.False(t, ok)
	})

	t.Run("set context", func(t *testing.T) {
		ctx := jsonapi.ContextWithValue(context.Background(), &jsonapi.RequestContext{})
		jsonapictx, ok := jsonapi.Context(ctx)
		assert.True(t, ok)
		assert.NotNil(t, jsonapictx)
	})
}
