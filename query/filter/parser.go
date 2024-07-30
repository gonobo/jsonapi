package filter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gonobo/jsonapi/v1/query"
)

const (
	QueryKeyFilterExpression = "q"
)

var (
	ErrFilter = errors.New("filter error")
)

func FilterError(cause string) error {
	return fmt.Errorf("%w: %s", ErrFilter, cause)
}

type Operation string

const (
	OperationAND Operation = "and"
	OperationOR  Operation = "or"
	OperationNOT Operation = "not"
)

type Transformer interface {
	TransformFilterToken(*query.Filter) (query.FilterExpression, error)
}

type TransformerFunc func(*query.Filter) (query.FilterExpression, error)

func (f TransformerFunc) TransformFilterToken(t *query.Filter) (query.FilterExpression, error) {
	return f(t)
}

func ParseQuery(q map[string]string) (query.FilterExpression, error) {
	return ParseQueryWithTransform(q, TransformerFunc(func(t *query.Filter) (query.FilterExpression, error) {
		// passthrough
		return t, nil
	}))
}

func ParseQueryWithTransform(q map[string]string, transformer Transformer) (query.FilterExpression, error) {
	params := filterParams(q)

	// extract filter keys
	filterKeys := params.keys()

	if len(filterKeys) == 0 {
		// no filter keys listed, return identity
		return query.IdentityFilter{}, nil
	}

	// generate expression map from filter keys
	filterExpression := make(map[string]query.FilterExpression)

	for _, key := range filterKeys {
		token := params.criteria(key)

		if err := token.validate(); err != nil {
			cause := fmt.Sprintf("query token: '%s': %s", key, err.Error())
			return nil, FilterError(cause)
		}

		expression, err := transformer.TransformFilterToken(token.filter())

		if err != nil {
			cause := fmt.Sprintf("query token: '%s': %s", key, err.Error())
			return nil, FilterError(cause)
		}

		filterExpression[key] = expression
	}

	// generate expression from query
	return params.evaluate(filterExpression)
}

type filterParams map[string]string

func (f filterParams) criteria(key string) criteria {
	token := criteria{}
	token.parseQueryParams(key, f)
	return token
}

func (f filterParams) keys() []string {
	query := f.query()

	if query == "" {
		return nil
	}

	keys := make([]string, 0)

	for _, key := range strings.Split(query, " ") {
		switch Operation(strings.ToLower(key)) {
		case OperationAND:
			continue
		case OperationOR:
			continue
		case OperationNOT:
			continue
		default:
			keys = append(keys, key)
		}
	}

	return keys
}

func (f filterParams) query() string {
	return f[QueryKeyFilterExpression]
}

func (f filterParams) evaluate(expressions map[string]query.FilterExpression) (query.FilterExpression, error) {
	query := f.query()
	tokens := strings.Split(query, " ")
	builder := astBuilder{}

	root, err := builder.build(tokens)

	if err != nil {
		return nil, FilterError(err.Error())
	}

	return f.walk(root, expressions)
}

func (f filterParams) walk(node *node, expressions map[string]query.FilterExpression) (query.FilterExpression, error) {
	switch node.token.symbol {
	case symbolAND:
		expression := &query.AndFilter{}
		left, lefterr := f.walk(node.left, expressions)
		right, righterr := f.walk(node.right, expressions)
		expression.Left = left
		expression.Right = right
		return expression, errors.Join(lefterr, righterr)
	case symbolOR:
		expression := &query.OrFilter{}
		left, lefterr := f.walk(node.left, expressions)
		right, righterr := f.walk(node.right, expressions)
		expression.Left = left
		expression.Right = right
		return expression, errors.Join(lefterr, righterr)
	case symbolNOT:
		expression := &query.NotFilter{}
		value, err := f.walk(node.left, expressions)
		expression.Value = value
		return expression, err
	case symbolVAR:
		expression := expressions[node.token.value]
		return expression, nil
	}

	return nil, fmt.Errorf("invalid or unknown node: %+v", node)
}

type RequestQueryParser struct {
	Transformer Transformer
}

func NewRequestQueryParser(options ...func(*RequestQueryParser)) RequestQueryParser {
	parser := RequestQueryParser{
		Transformer: TransformerFunc(func(c *query.Filter) (query.FilterExpression, error) {
			return c, nil
		}),
	}

	for _, option := range options {
		option(&parser)
	}

	return parser
}

func (p RequestQueryParser) ParseFilterQuery(r *http.Request) (query.FilterExpression, error) {
	params := make(map[string]string)
	for k, v := range r.URL.Query() {
		params[k] = v[0]
	}

	return ParseQueryWithTransform(params, p.Transformer)
}
