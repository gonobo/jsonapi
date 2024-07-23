package sort

import (
	"slices"

	"github.com/gonobo/jsonapi/query"
)

// Comparer determines the order of two items based on their attributes.
type Comparer[T any] interface {
	// Compare returns a negative number if the attribute of a is less than b,
	// a positive number if the attribute of a is greater than b, or zero if
	// the values are equivalent.
	Compare(a, b T, attribute string) int
}

// Slice orders items according to the specified sorting parameters.
func Slice[T any](items []T, comparer Comparer[T], params []query.Sort) {
	for _, param := range params {
		slices.SortFunc(items, func(a, b T) int {
			if param.Descending {
				return comparer.Compare(b, a, param.Property)
			}
			return comparer.Compare(a, b, param.Property)
		})
	}
}
