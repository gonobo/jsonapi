package filter

import (
	"fmt"

	"github.com/gonobo/jsonapi/v1/query"
	"github.com/gonobo/validator"
)

type criteria query.Filter

func (c criteria) validate() error {
	return validator.Validate(
		validator.All(
			validator.Rule(c.Name != "", "token name missing or empty"),
			validator.Rule(c.Condition != "", "token operator missing or empty"),
			validator.Rule(c.Value != "", "token value missing or empty"),
		),
	)
}

func (c criteria) filter() *query.Filter {
	filter := query.Filter(c)
	return &filter
}

func (c criteria) queryParams(name string) map[string]string {
	nameKey := c.queryKeyName(name)
	conditionKey := c.queryKeyCondition(name)
	valueKey := c.queryKeyValue(name)

	return map[string]string{
		nameKey:      c.Name,
		conditionKey: string(c.Condition),
		valueKey:     c.Value,
	}
}

func (c criteria) queryKeyName(name string) string {
	return fmt.Sprintf("filter[%s][name]", name)
}

func (c criteria) queryKeyCondition(name string) string {
	return fmt.Sprintf("filter[%s][condition]", name)
}

func (c criteria) queryKeyValue(name string) string {
	return fmt.Sprintf("filter[%s][value]", name)
}

func (c *criteria) parseQueryParams(name string, params map[string]string) {
	c.Name = params[c.queryKeyName(name)]
	c.Condition = query.FilterCondition(params[c.queryKeyCondition(name)])
	c.Value = params[c.queryKeyValue(name)]
}
