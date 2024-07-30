package filter

import (
	"fmt"

	"github.com/gonobo/jsonapi/v1/query"
)

type TransformerMux struct {
	Strict       bool
	Transformers map[string]Transformer
}

func (t TransformerMux) TransformFilterToken(token *query.Filter) (query.FilterExpression, error) {
	transformer, ok := t.Transformers[token.Name]
	if !ok && t.Strict {
		return nil, FilterError(fmt.Sprintf("unknown token '%s'", token.Name))
	}
	return transformer.TransformFilterToken(token)
}
