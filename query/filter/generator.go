package filter

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/gonobo/jsonapi/v2/query"
)

func Generate(expr query.FilterExpression, q url.Values) error {
	generator := &urlQueryGenerator{
		counter: new(counter),
		query:   q,
	}

	err := query.EvaluateFilter(generator, expr)
	q.Set(QueryKeyFilterExpression, generator.expr)
	return err
}

type counter int

func (c *counter) increment() int {
	*c++
	return int(*c)
}

type urlQueryGenerator struct {
	counter *counter
	expr    string
	query   url.Values
}

func (g *urlQueryGenerator) EvaluateFilter(e *query.Filter) error {
	ident := fmt.Sprintf("p%02d", g.counter.increment())
	g.expr = ident
	filter(ident).set(g.query, *e)
	return nil
}

func (g *urlQueryGenerator) EvaluateAndFilter(e *query.AndFilter) error {
	left := &urlQueryGenerator{
		counter: g.counter,
		query:   g.query,
	}

	right := &urlQueryGenerator{
		counter: g.counter,
		query:   g.query,
	}

	lefterr := query.EvaluateFilter(left, e.Left)
	righterr := query.EvaluateFilter(right, e.Right)

	g.expr = fmt.Sprintf("%s and %s", left.expr, right.expr)
	return errors.Join(lefterr, righterr)
}

func (g *urlQueryGenerator) EvaluateOrFilter(e *query.OrFilter) error {
	left := &urlQueryGenerator{
		counter: g.counter,
		query:   g.query,
	}

	right := &urlQueryGenerator{
		counter: g.counter,
		query:   g.query,
	}

	lefterr := query.EvaluateFilter(left, e.Left)
	righterr := query.EvaluateFilter(right, e.Right)

	g.expr = fmt.Sprintf("%s or %s", left.expr, right.expr)
	return errors.Join(lefterr, righterr)
}

func (g *urlQueryGenerator) EvaluateNotFilter(e *query.NotFilter) error {
	expr := &urlQueryGenerator{
		counter: new(counter),
		query:   g.query,
	}

	err := query.EvaluateFilter(expr, e.Expression)
	g.expr = fmt.Sprintf("not %s", expr.expr)
	return err
}

func (urlQueryGenerator) EvaluateCustomFilter(e any) error {
	return errors.ErrUnsupported
}

func (urlQueryGenerator) EvaluateIdentityFilter() error {
	return errors.ErrUnsupported
}
