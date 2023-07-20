package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"golang.org/x/exp/constraints"
)

func FindMin[T constraints.Ordered](s []T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		if m > v {
			m = v
		}
	}
	return m
}

func FindMax[T constraints.Ordered](s []T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		if m < v {
			m = v
		}
	}
	return m
}

func Intersection[T comparable](arrays ...[]T) []T {
	// Create a map to store the elements and their occurrence count
	elements := make(map[T]int)

	// Iterate through each array
	for _, arr := range arrays {
		// Create a map to store the elements of the current array
		arrElements := make(map[T]bool)

		// Populate the map with elements from the current array
		for _, elem := range arr {
			arrElements[elem] = true
		}

		// Increment the occurrence count for each element in the map
		for elem := range arrElements {
			elements[elem]++
		}
	}

	var intersection []T

	// Check the occurrence count of each element
	for elem, count := range elements {
		if count == len(arrays) {
			intersection = append(intersection, elem)
		}
	}

	return intersection
}

// SafePow implements a deterministic power function
// Go's math.Pow() doesn't guarantee determinism when executed on different hardwares
func SafePow(base uint64, exponent uint64) uint64 {
	baseDec := sdk.NewDecWithPrec(int64(base), 0)
	return baseDec.Power(exponent).TruncateInt().Uint64()
}
