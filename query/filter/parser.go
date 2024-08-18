package filter

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/url"
	"strings"

	"github.com/gonobo/jsonapi/v1/query"
	"github.com/gonobo/validator"
)

const (
	QueryKeyFilterExpression = "q"
	OperationAND             = "and"
	OperationOR              = "or"
	OperationNOT             = "not"
)

var (
	ErrEmptyQuery = errors.New("empty query")

	DefaultParser = Parser{
		Transformer: PassthroughTransformer,
	}
)

type Parser struct {
	Transformer Transformer
}

func (Parser) ParseFilter(r *http.Request) (query.FilterExpression, error) {
	return ParseWithTransform(r.URL.Query(), PassthroughTransformer)
}

func Parse(query url.Values) (query.FilterExpression, error) {
	return ParseWithTransform(query, PassthroughTransformer)
}

func ParseWithTransform(query url.Values, transformer Transformer) (query.FilterExpression, error) {
	if len(query) == 0 {
		return nil, ErrEmptyQuery
	}

	// parse query as a go expression
	q := query.Get(QueryKeyFilterExpression)

	// convert logical operators into their go equivalents:
	// "and" -> "&&"
	// "or" -> "||"
	q = strings.ToLower(q)
	q = strings.ReplaceAll(q, OperationAND, "&&")
	q = strings.ReplaceAll(q, OperationOR, "||")
	q = strings.ReplaceAll(q, OperationNOT, "!")

	node, err := parser.ParseExpr(q)
	if err != nil {
		return nil, fmt.Errorf("invalid query '%s': %w", q, err)
	}

	// generate search expression by walking the ast
	expr, err := parseURLQuery(node, query, transformer)
	if err != nil {
		return nil, fmt.Errorf("url query parse failed: %w", err)
	}

	return expr, nil
}

type filter string

func (f filter) set(query url.Values, c query.Filter) url.Values {
	ident := string(f)

	query.Set(fmt.Sprintf("filter[%s][name]", ident), c.Name)
	query.Set(fmt.Sprintf("filter[%s][condition]", ident), c.Condition)
	query.Set(fmt.Sprintf("filter[%s][value]", ident), c.Value)
	return query
}

func (f filter) get(q url.Values) query.Filter {
	return query.Filter{
		Name:      f.name(q),
		Condition: f.condition(q),
		Value:     f.value(q),
	}
}

func (f filter) name(query url.Values) string {
	return query.Get(fmt.Sprintf("filter[%s][name]", string(f)))
}

func (f filter) condition(query url.Values) string {
	return query.Get(fmt.Sprintf("filter[%s][condition]", string(f)))
}

func (f filter) value(query url.Values) string {
	return query.Get(fmt.Sprintf("filter[%s][value]", string(f)))
}

type urlQueryParser struct {
	err         error
	expr        query.FilterExpression
	transformer Transformer
	query       url.Values
}

func parseURLQuery(n ast.Node, query url.Values, transformer Transformer) (query.FilterExpression, error) {
	parser := &urlQueryParser{
		query:       query,
		transformer: transformer,
	}
	ast.Walk(parser, n)
	return parser.expr, parser.err
}

func (u *urlQueryParser) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		// if nil, then we have reached the end of the expression and should end the walk
		// procedure.
		return nil
	}

	switch expr := n.(type) {
	case *ast.BinaryExpr:
		return u.visitBinaryExpr(expr)
	case *ast.UnaryExpr:
		return u.visitUnaryExpr(expr)
	case *ast.Ident:
		return u.visitIdent(expr)
	}

	return nil
}

func (u *urlQueryParser) visitIdent(ident *ast.Ident) ast.Visitor {
	f := filter(ident.Name)
	criteria := f.get(u.query)

	err := validator.Validate(
		validator.All(
			validator.Rule(criteria.Name != "", "filter[%s][name] is required", ident.Name),
			validator.Rule(criteria.Value != "", "filter[%s][value] is required", ident.Name),
			validator.Rule(criteria.Condition != "", "filter[%s][condition] is required", ident.Name),
		),
	)

	if err == nil {
		expr, exprerr := u.transformer.TransformCriteria(&criteria)
		u.expr = expr
		err = exprerr
	}

	u.err = err
	return nil
}

func (u *urlQueryParser) visitUnaryExpr(unaryExpr *ast.UnaryExpr) ast.Visitor {
	parser := u.clone()
	switch unaryExpr.Op {
	case token.NOT:
		expr := &query.NotFilter{}
		parser.Visit(unaryExpr.X)
		expr.Expression = parser.expr
		u.expr = expr
		u.err = parser.err
		return nil
	}

	u.err = fmt.Errorf("unsupported operator '%s'", unaryExpr.Op)
	return nil
}

func (u *urlQueryParser) visitBinaryExpr(binaryExpr *ast.BinaryExpr) ast.Visitor {
	left := u.clone()
	right := u.clone()

	switch binaryExpr.Op {
	case token.LAND:
		expr := &query.AndFilter{}
		left.Visit(binaryExpr.X)
		right.Visit(binaryExpr.Y)
		expr.Left = left.expr
		expr.Right = right.expr
		u.expr = expr
		u.err = errors.Join(left.err, right.err)
		return nil
	case token.LOR:
		expr := &query.OrFilter{}
		left.Visit(binaryExpr.X)
		right.Visit(binaryExpr.Y)
		expr.Left = left.expr
		expr.Right = right.expr
		u.expr = expr
		u.err = errors.Join(left.err, right.err)
		return nil
	}

	u.err = fmt.Errorf("unsupported operator '%s'", binaryExpr.Op)
	return nil
}

func (u urlQueryParser) clone() *urlQueryParser {
	return &urlQueryParser{
		transformer: u.transformer,
		query:       u.query,
	}
}
