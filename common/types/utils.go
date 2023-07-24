package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/exp/constraints"
)

func FindMin[T constraints.Ordered](s []T) (m T) {
	if len(s) > 0 {
		m = s[0]
		for _, v := range s[1:] {
			if m > v {
				m = v
			}
		}
	}
	return m
}

func FindMax[T constraints.Ordered](s []T) (m T) {
	if len(s) > 0 {
		m = s[0]
		for _, v := range s[1:] {
			if m < v {
				m = v
			}
		}
	}
	return m
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

// SafePow implements a deterministic power function
// Go's math.Pow() doesn't guarantee determinism when executed on different hardwares
func SafePow(base uint64, exponent uint64) uint64 {
	baseDec := sdk.NewDecWithPrec(int64(base), 0)
	return baseDec.Power(exponent).TruncateInt().Uint64()
}
