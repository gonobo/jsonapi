package query_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gonobo/jsonapi/v1/query"
	"github.com/stretchr/testify/assert"
)

type Evaluator struct {
	Want string
	Got  string
}

func (e *Evaluator) EvaluateFilter(t *query.Filter) error {
	e.Got = e.Got + t.Name
	return nil
}

func (e *Evaluator) EvaluateCustomFilter(value any) error {
	return nil
}

func (e *Evaluator) EvaluateAndFilter(expr *query.AndFilter) error {
	var left, right Evaluator

	leftErr := query.EvaluateFilter(&left, expr.Left)
	rightErr := query.EvaluateFilter(&right, expr.Right)
	e.Got = fmt.Sprintf("%s AND %s", left.Got, right.Got)
	return errors.Join(leftErr, rightErr)
}

func (e *Evaluator) EvaluateOrFilter(expr *query.OrFilter) error {
	var left, right Evaluator

	leftErr := query.EvaluateFilter(&left, expr.Left)
	rightErr := query.EvaluateFilter(&right, expr.Right)
	e.Got = fmt.Sprintf("%s OR %s", left.Got, right.Got)
	return errors.Join(leftErr, rightErr)
}

func (e *Evaluator) EvaluateNotFilter(expr *query.NotFilter) error {
	var value Evaluator

	err := query.EvaluateFilter(&value, expr.Value)
	e.Got = fmt.Sprintf("NOT %s", value.Got)
	return err
}

func (e *Evaluator) EvaluateIdentityFilter() error {
	e.Got = "TRUE"
	return nil
}

func (e Evaluator) Assert(t *testing.T) bool {
	return assert.Equal(t, e.Want, e.Got)
}

type EvaluatorAsserter interface {
	query.FilterEvaluator
	Assert(t *testing.T) bool
}

func TestEvaluate(t *testing.T) {
	type testcase struct {
		evaluator  EvaluatorAsserter
		expression query.FilterExpression
		wantErr    bool
	}

	run := func(t *testing.T, tc testcase) {
		err := query.EvaluateFilter(tc.evaluator, tc.expression)
		if tc.wantErr && assert.Error(t, err) {
			return
		}
		assert.NoError(t, err)
		tc.evaluator.Assert(t)
	}

	t.Run("simple expression", func(t *testing.T) {
		run(t, testcase{
			evaluator: &Evaluator{Want: "p1 AND p2"},
			expression: &query.AndFilter{
				Left:  &query.Filter{Name: "p1"},
				Right: &query.Filter{Name: "p2"},
			},
		})
	})

	t.Run("complex expression", func(t *testing.T) {
		run(t, testcase{
			evaluator: &Evaluator{Want: "p1 AND p2 OR NOT p3"},
			expression: &query.AndFilter{
				Left: &query.Filter{Name: "p1"},
				Right: &query.OrFilter{
					Left: &query.Filter{Name: "p2"},
					Right: &query.NotFilter{
						Value: &query.Filter{Name: "p3"},
					},
				},
			},
		})
	})

	t.Run("identity", func(t *testing.T) {
		run(t, testcase{
			evaluator:  &Evaluator{Want: "TRUE"},
			expression: query.IdentityFilter{},
		})
	})
}
