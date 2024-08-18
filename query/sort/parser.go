package sort

import (
	"net/http"
	"strings"

	"github.com/gonobo/jsonapi/v2/query"
)

var DefaultParser = RequestQueryParser{}

// RequestQueryParser parses the JSON:API sort query parameters of an http request.
type RequestQueryParser struct{}

// NewRequestQueryParser creates a new RequestQueryParser.
func NewRequestQueryParser(options ...func(*RequestQueryParser)) RequestQueryParser {
	parser := RequestQueryParser{}
	for _, option := range options {
		option(&parser)
	}
	return parser
}

// ParseSortQuery parses the JSON:API sort query parameters of an http request.
func (rq RequestQueryParser) ParseSortQuery(r *http.Request) ([]query.Sort, error) {
	params := make(map[string]string)
	for k, v := range r.URL.Query() {
		params[k] = v[0]
	}
	return ParseQuery(params)
}

// ParseQuery parses the JSON:API sort query parameters into a list of sort criteria.
func ParseQuery(params map[string]string) ([]query.Sort, error) {
	queryParams := queryParams(params)
	criteria := queryParams.sortCriteria()
	return criteria, nil
}

type queryParams map[string]string

func (q queryParams) sortCriteria() []query.Sort {
	criteria := make([]query.Sort, 0)
	tokens := strings.Split(q.sortParams(), ",")
	for _, token := range tokens {
		criteria = append(criteria, q.convertTokenToCriteria(token))
	}
	return criteria
}

func (q queryParams) sortParams() string {
	return q[query.ParamSort]
}

func (q queryParams) convertTokenToCriteria(token string) query.Sort {
	criteria := query.Sort{Descending: false}

	if strings.HasPrefix(token, query.ParamSortAscendingPrefix) {
		criteria.Property = strings.TrimPrefix(token, query.ParamSortAscendingPrefix)
	} else if strings.HasPrefix(token, query.ParamSortDescendingPrefix) {
		criteria.Descending = true
		criteria.Property = strings.TrimPrefix(token, query.ParamSortDescendingPrefix)
	} else {
		criteria.Property = token
	}

	return criteria
}
