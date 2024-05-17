package query

import "fmt"

// FilterCondition defines the parameters of a resource property that should be included
// in the return document.
type FilterCondition string

const (
	Equal            FilterCondition = "eq"          // The resource property must equal the token's value.
	NotEqual         FilterCondition = "neq"         // The resource property must not equal the token's value.
	Contains         FilterCondition = "contains"    // The resource property must contain the token's value.
	LessThan         FilterCondition = "lt"          // The resource property must be less than the token's value.
	LessThanEqual    FilterCondition = "lte"         // The resource property must be less than or equal to the token's value.
	GreaterThan      FilterCondition = "gt"          // The resource property must be greater than the token's value.
	GreaterThanEqual FilterCondition = "gte"         // The resource property must be greater than or equal to the token's value.
	StartsWith       FilterCondition = "starts_with" // The resource property must start with the token's value.
)

// Filter defines a filter request made by JSON:API clients. Multiple filter queries
// are usually combined using logical operators -- such as "and", "or", or "not" -- into
// a single filter expression.
type Filter struct {
	Name      string          // The field name, usually referencing an attribute or relationship on a resource.
	Condition FilterCondition // The operator to apply on both the resource field and the token's value.
	Value     string          // The target value.
}

// String returns a string representation of the filter query.
func (c Filter) String() string {
	return fmt.Sprintf("[%s %s '%s']", c.Name, c.Condition, c.Value)
}

// ApplyFilterEvaluator applies the evaluator to this filter query.
func (c Filter) ApplyFilterEvaluator(e FilterEvaluator) error {
	return e.EvaluateFilter(&c)
}

// FilterExpression defines a logical expression that was requested by a client.
// It can be evaluated by a FilterEvaluator.
type FilterExpression interface {
	fmt.Stringer
	// ApplyFilterEvaluator applies the evaluator to this expression. Implementors
	// should invoke the "EvaluateXXX" method on the evaluator that corresponds to the concrete type.
	ApplyFilterEvaluator(FilterEvaluator) error
}

// FilterEvaluator defines the interface for evaluating a FilterExpression. Implementers can
// extend the evaluation system by defining expression types the invoke EvaluateCustomFilter()
// method on acceptance:
//
//	type MyEvaluator struct {
//		// ...
//	}
//
//	func (e *MyEvaluator) EvaluateCustomFilter(value any) error {
//		if custom, ok := value.(MyCustomExpression); ok {
//			// ...
//		}
//		return nil
//	}
//
//	type MyCustomExpression struct {
//		// ...
//	}
//
//	func (e *MyCustomExpression) ApplyFilterEvaluator(FilterEvaluator) error {
//		return e.EvaluateCustomFilter(e)
//	}
type FilterEvaluator interface {
	// EvaluateFilter evaluates the filter expression using the given evaluator.
	EvaluateFilter(*Filter) error
	// EvaluateAndFilter evaluates the filter expression using the given evaluator.
	EvaluateAndFilter(*AndFilter) error
	// EvaluateOrFilter evaluates the filter expression using the given evaluator.
	EvaluateOrFilter(*OrFilter) error
	// EvaluateNotFilter evaluates the filter expression using the given evaluator.
	EvaluateNotFilter(*NotFilter) error
	// EvaluateIdentityFilter evaluates the filter expression using the given evaluator.
	EvaluateIdentityFilter() error
	// EvaluateCustomFilter evaluates the filter expression using the given evaluator.
	EvaluateCustomFilter(any) error
}

// EvaluateFilter evaluates the filter expression using the given evaluator.
func EvaluateFilter(evaluator FilterEvaluator, expr FilterExpression) error {
	return expr.ApplyFilterEvaluator(evaluator)
}

// AndFilter defines a logical AND expression that was requested by a client.
type AndFilter struct {
	Left  FilterExpression // The left operand of the AND expression.
	Right FilterExpression // The right operand of the AND expression.
}

// String returns a string representation of the AND expression.
func (a AndFilter) String() string {
	return fmt.Sprintf("(%s && %s)", a.Left, a.Right)
}

// ApplyFilterEvaluator applies the evaluator to this expression.
func (a *AndFilter) ApplyFilterEvaluator(e FilterEvaluator) error {
	return e.EvaluateAndFilter(a)
}

// OrFilter defines a logical OR expression that was requested by a client.
type OrFilter struct {
	Left  FilterExpression // The left operand of the OR expression.
	Right FilterExpression // The right operation of the OR expression.
}

// String returns a string representation of the OR expression.
func (o OrFilter) String() string {
	return fmt.Sprintf("(%s || %s)", o.Left, o.Right)
}

// ApplyFilterEvaluator applies the evaluator to this expression.
func (o *OrFilter) ApplyFilterEvaluator(e FilterEvaluator) error {
	return e.EvaluateOrFilter(o)
}

// NotFilter defines a logical NOT expression that was requested by a client.
type NotFilter struct {
	Value FilterExpression
}

// String returns a string representation of the NOT expression.
func (n NotFilter) String() string {
	return fmt.Sprintf("!%s", n.Value)
}

// ApplyFilterEvaluator applies the evaluator to this expression.
func (n *NotFilter) ApplyFilterEvaluator(e FilterEvaluator) error {
	return e.EvaluateNotFilter(n)
}

// IdentityFilter defines an identity filter expression. It always resolves to TRUE.
type IdentityFilter struct{}

// ApplyFilterEvaluator applies the evaluator to this expression.
func (i IdentityFilter) ApplyFilterEvaluator(e FilterEvaluator) error {
	return e.EvaluateIdentityFilter()
}

// String returns a string representation of the identity expression.
func (IdentityFilter) String() string {
	return "TRUE"
}
