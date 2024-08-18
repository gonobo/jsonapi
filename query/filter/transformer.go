package filter

import (
	"fmt"

	"github.com/gonobo/jsonapi/v1/query"
)

// Transformer can transform [query.Filter] criteria into another [query.FilterExpression]. This is useful
// when decoding input criteria into more complex filters.
type Transformer interface {
	// TransformCriteria transforms the given criteria into another expression.
	// When the criteria should not be transformed, it returns the original criteria.
	// If an error occurs during transformation, it returns the error.
	TransformCriteria(*query.Filter) (query.FilterExpression, error)
}

// TransformerFunc is a function that implements the [Transformer] interface.
//
// Example:
//
//	var uppercase TransformerFunc = func(c *query.Filter) (query.FilterExpression, error) {
//	  c.Value = strings.ToUpper(c.Value.(string))
//	  return c, nil
//	}
type TransformerFunc func(*query.Filter) (query.FilterExpression, error)

// TransformCriteria implements the [Transformer] interface.
func (f TransformerFunc) TransformCriteria(c *query.Filter) (query.FilterExpression, error) {
	return f(c)
}

// PassthroughTransformer is a transformer that passes the criteria through without
// any transformation.
var PassthroughTransformer Transformer = TransformerFunc(func(c *query.Filter) (query.FilterExpression, error) {
	return c, nil
})

// TransformerMux combines several [Transformer] instances into a single one. Each transformer
// is mapped to a unique key, which will be invoked when the key matches the [Criteria] property
// value. If no transformer is found for the given property, the [PassthroughTransformer] is used.
//
// Example:
//
//	mux := TransformerMux{
//	  "name": TransformerFunc(func(c *query.Filter) (query.FilterExpression, error) {
//	    c.Value = strings.ToUpper(c.Value.(string))
//	    return c, nil
//	  }),
//	}
//	mux.TransformCriteria(&Criteria{Property: "name", Value: "john"}) // returns the transformed criteria
//	mux.TransformCriteria(&Criteria{Property: "age", Value: 30}) // returns the original criteria
type TransformerMux map[string]Transformer

// TransformCriteria implements the [Transformer] interface.
func (m TransformerMux) TransformCriteria(c *query.Filter) (query.FilterExpression, error) {
	t, ok := m[c.Name]
	if !ok {
		return PassthroughTransformer.TransformCriteria(c)
	}
	return t.TransformCriteria(c)
}

// StrictTransformerMux is like [TransformerMux] but it returns an error if no transformer is found for a given property.
type StrictTransformerMux struct {
	mux TransformerMux
}

// Strict returns a [StrictTransformerMux] that returns an error if no transformer is found for a given property.
func (m TransformerMux) Strict() StrictTransformerMux {
	return StrictTransformerMux{mux: m}
}

// TransformCriteria implements the [Transformer] interface.
func (m StrictTransformerMux) TransformCriteria(c *query.Filter) (query.FilterExpression, error) {
	t, ok := m.mux[c.Name]
	if !ok {
		return nil, fmt.Errorf("unknown filter: %s", c.Name)
	}
	return t.TransformCriteria(c)
}
