package filter

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gonobo/jsonapi/query"
)

func BuildQuery(expression query.FilterExpression) (map[string]string, error) {
	builder := queryBuilder{
		params: map[string]string{},
		query:  &strings.Builder{},
	}

	err := query.EvaluateFilter(&builder, expression)

	if err == nil {
		builder.params[QueryKeyFilterExpression] = builder.query.String()
	}

	return builder.params, err
}

type queryBuilder struct {
	index  int
	query  *strings.Builder
	params map[string]string
}

func (q *queryBuilder) EvaluateFilter(f *query.Filter) error {
	c := criteria(*f)
	for key, value := range c.queryParams(q.name()) {
		q.params[key] = value
	}
	_, err := q.query.WriteString(q.name())
	return err
}

func (q *queryBuilder) EvaluateAndFilter(e *query.AndFilter) error {
	left := q.clone()
	right := q.clone()

	lefterr, righterr := query.EvaluateFilter(&left, e.Left), query.EvaluateFilter(&right, e.Right)
	_, err := q.query.WriteString(fmt.Sprintf("%s AND %s", left.query.String(), right.query.String()))

	return errors.Join(lefterr, righterr, err)
}

func (q *queryBuilder) EvaluateOrFilter(e *query.OrFilter) error {
	left := q.clone()
	right := q.clone()

	lefterr, righterr := query.EvaluateFilter(&left, e.Left), query.EvaluateFilter(&right, e.Right)
	_, err := q.query.WriteString(fmt.Sprintf("%s OR %s", left.query.String(), right.query.String()))

	return errors.Join(lefterr, righterr, err)
}

func (q *queryBuilder) EvaluateNotFilter(e *query.NotFilter) error {
	value := q.clone()
	valueerr := query.EvaluateFilter(&value, e.Value)
	_, err := q.query.WriteString("NOT %s")

	return errors.Join(valueerr, err)
}

func (q *queryBuilder) EvaluateIdentityFilter() error {
	return errors.New("identity expression not supported for building queries")
}

func (q *queryBuilder) EvaluateCustomFilter(any) error {
	return errors.New("custom expressions not supported by query filter builder")
}

func (q *queryBuilder) clone() queryBuilder {
	q.index++
	return queryBuilder{
		index:  q.index,
		query:  &strings.Builder{},
		params: q.params,
	}
}

func (q queryBuilder) name() string {
	return fmt.Sprintf("p%d", q.index)
}
