package types

import (
	"unicode"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/exp/constraints"
)

func FindMin[T constraints.Ordered](s []T) T {
	ind := FindIndexOfMin(s)
	if ind == -1 {
		var zero T
		return zero
	}

	return s[ind]
}

func FindIndexOfMin[T constraints.Ordered](s []T) int {
	if len(s) == 0 {
		return -1
	}
	m := s[0]
	mInd := 0
	for i := range s {
		if m > s[i] {
			mInd = i
		}
	}
	return mInd
}

func FindMax[T constraints.Ordered](s []T) T {
	ind := FindIndexOfMax(s)
	if ind == -1 {
		var zero T
		return zero
	}

	return s[ind]
}

func FindIndexOfMax[T constraints.Ordered](s []T) int {
	if len(s) == 0 {
		return -1
	}
	m := s[0]
	mInd := 0
	for i := range s {
		if m < s[i] {
			m = s[i]
			mInd = i
		}
	}
	return mInd
}

func Intersection[T comparable](arrays ...[]T) []T {
	elements := make(map[T]int)

	for _, arr := range arrays {
		arrElements := make(map[T]bool)

		for _, elem := range arr {
			if _, ok := arrElements[elem]; !ok {
				arrElements[elem] = true
				elements[elem]++
			}
		}
	}

	res := make([]T, 0)

	for elem, count := range elements {
		if count == len(arrays) {
			res = append(res, elem)
		}
	}

	return res
}

func Union[T comparable](arrays ...[]T) []T {
	// store seen elements in a map
	arrElements := make(map[T]struct{})
	// Iterate through each array
	for _, arr := range arrays {
		// Create a map to store the elements of the current array

		// Populate the map with elements from the current array
		for _, elem := range arr {
			arrElements[elem] = struct{}{}
		}
	}

	var union []T
	// Check the occurrence count of each element
	for elem := range arrElements {
		union = append(union, elem)
	}

	return union
}

type ComparableByFields interface {
	Differentiator() string
}

func UnionByFields[T ComparableByFields](arrays ...[]T) []T {
	// store seen elements in a map
	arrElements := make(map[string]T)
	// Iterate through each array
	for _, arr := range arrays {
		// Create a map to store the elements of the current array

		// Populate the map with elements from the current array
		for _, elem := range arr {
			arrElements[elem.Differentiator()] = elem
		}
	}

	var union []T
	// Check the occurrence count of each element
	for _, elem := range arrElements {
		union = append(union, elem)
	}

	return union
}

// SafePow implements a deterministic power function
// Go's math.Pow() doesn't guarantee determinism when executed on different hardwares
func SafePow(base, exponent uint64) uint64 {
	baseDec := sdk.NewDecWithPrec(int64(base), 0)
	return baseDec.Power(exponent).TruncateInt().Uint64()
}

// CamelToSnake turns CamelString to camel_string ("snake case")
func CamelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(rune(s[i-1])) || (i+1 < len(s) && unicode.IsLower(rune(s[i+1])))) {
				// Add underscore before the uppercase letter if it's not the first letter or followed by another uppercase letter
				result = append(result, '_')
			}
			// Convert the uppercase letter to lowercase
			r = unicode.ToLower(r)
		}
		result = append(result, r)
	}
	return string(result)
}
