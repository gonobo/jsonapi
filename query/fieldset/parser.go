package fieldset

import (
	"net/http"
	"strings"

	"github.com/gonobo/jsonapi/v1/query"
)

var DefaultParser = Parser{}

type Parser struct{}

func (p Parser) ParseFieldsetQuery(r *http.Request) ([]query.Fieldset, error) {
	q := r.URL.Query()
	fields := make([]query.Fieldset, 0)

	for k := range q {
		if strings.HasPrefix(k, "fields[") && strings.HasSuffix(k, "]") {
			field := p.extractField(k)
			fields = append(fields, query.Fieldset{Property: field})
		}
	}

	return fields, nil
}

func (Parser) extractField(input string) string {
	start := strings.IndexRune(input, '[')
	end := strings.IndexRune(input, ']')
	field := input[start+1 : end]
	return field
}
