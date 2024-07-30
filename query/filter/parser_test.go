package filter_test

import (
	"errors"
	"testing"

	"github.com/gonobo/jsonapi/v1/query"
	"github.com/gonobo/jsonapi/v1/query/filter"
	"github.com/stretchr/testify/assert"
)

type provider map[string]string

func (p provider) FilterParams() map[string]string {
	return p
}

func TestParseQuery(t *testing.T) {
	type testcase struct {
		provider provider
		want     string
		wantErr  bool
	}

	run := func(t *testing.T, tc testcase) {
		params := tc.provider.FilterParams()
		expr, err := filter.ParseQuery(params)
		if tc.wantErr && assert.Error(t, err) {
			assert.ErrorIs(t, err, filter.ErrFilter)
			return
		}
		assert.NoError(t, err)
		got := expr.String()
		assert.Equal(t, tc.want, got)
	}

	t.Run("no query", func(t *testing.T) {
		run(t, testcase{
			provider: provider{},
			want:     "TRUE",
		})
	})

	t.Run("simple query", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				"q":                     "p1",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
			},
			want: "[value eq '5']",
		})
	})

	t.Run("logical query", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				"q":                     "p1 AND p2",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
			},
			want: "([value eq '5'] && [value eq '2'])",
		})
	})

	t.Run("compound query", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				// (p1 AND p2) OR (NOT p3)
				"q":                     "p1 AND p2 OR NOT p3",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
				"filter[p3][name]":      "value",
				"filter[p3][condition]": "eq",
				"filter[p3][value]":     "4",
			},
			want: "(([value eq '5'] && [value eq '2']) || ![value eq '4'])",
		})
	})

	t.Run("compound query", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				// p1 OR (p2 AND (NOT p3))
				"q":                     "p1 OR p2 AND NOT p3",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
				"filter[p3][name]":      "value",
				"filter[p3][condition]": "eq",
				"filter[p3][value]":     "4",
			},
			want: "([value eq '5'] || ([value eq '2'] && ![value eq '4']))",
		})
	})

	t.Run("compound query", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				// (p1 AND p2) AND p3
				"q":                     "p1 AND p2 AND p3",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
				"filter[p3][name]":      "value",
				"filter[p3][condition]": "eq",
				"filter[p3][value]":     "4",
			},
			want: "(([value eq '5'] && [value eq '2']) && [value eq '4'])",
		})
	})

	t.Run("compound query", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				// (!p1 AND !p2) OR p3
				"q":                     "NOT p1 AND NOT p2 OR p3",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
				"filter[p3][name]":      "value",
				"filter[p3][condition]": "eq",
				"filter[p3][value]":     "4",
			},
			want: "((![value eq '5'] && ![value eq '2']) || [value eq '4'])",
		})
	})

	t.Run("invalid syntax: AND operator", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				"q":                     "AND p2 AND p3",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
				"filter[p3][name]":      "value",
				"filter[p3][condition]": "eq",
				"filter[p3][value]":     "4",
			},
			wantErr: true,
		})
	})

	t.Run("invalid syntax: OR operator", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				"q":                     "OR p2 AND p3",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
				"filter[p3][name]":      "value",
				"filter[p3][condition]": "eq",
				"filter[p3][value]":     "4",
			},
			wantErr: true,
		})
	})

	t.Run("warning: adjacent tokens yield undefined results", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				"q":                     "p2 p3",
				"filter[p2][name]":      "value",
				"filter[p2][condition]": "eq",
				"filter[p2][value]":     "2",
				"filter[p3][name]":      "value",
				"filter[p3][condition]": "eq",
				"filter[p3][value]":     "4",
			},
			want: "[value eq '2']",
		})
	})

	t.Run("invalid syntax: missing tokens", func(t *testing.T) {
		run(t, testcase{
			provider: provider{
				"q": "p1",
			},
			wantErr: true,
		})
	})
}

func TestParseQueryWithTransform(t *testing.T) {
	t.Run("transform error", func(t *testing.T) {
		_, err := filter.ParseQueryWithTransform(
			provider{
				"q":                     "p1",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
			},
			filter.TransformerFunc(func(t *query.Filter) (query.FilterExpression, error) {
				return nil, errors.New("transformer error")
			}),
		)

		assert.Error(t, err)
		assert.ErrorIs(t, err, filter.ErrFilter)
	})

	t.Run("with muxer", func(t *testing.T) {
		got, err := filter.ParseQueryWithTransform(
			provider{
				"q":                     "p1",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
			},
			filter.TransformerMux{Transformers: map[string]filter.Transformer{
				"value": filter.TransformerFunc(func(t *query.Filter) (query.FilterExpression, error) {
					t.Value = "42"
					return t, nil
				}),
			}},
		)
		assert.NoError(t, err)
		assert.Equal(t, "[value eq '42']", got.String())
	})

	t.Run("with strict muxer", func(t *testing.T) {
		_, err := filter.ParseQueryWithTransform(
			provider{
				"q":                     "p1",
				"filter[p1][name]":      "value",
				"filter[p1][condition]": "eq",
				"filter[p1][value]":     "5",
			},
			filter.TransformerMux{
				Strict: true,
				Transformers: map[string]filter.Transformer{
					"foo": filter.TransformerFunc(func(t *query.Filter) (query.FilterExpression, error) {
						t.Value = "42"
						return t, nil
					}),
				}},
		)
		assert.Error(t, err)
	})
}
