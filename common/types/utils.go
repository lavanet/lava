package types

import (
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

// SafePow implements a deterministic power function
// Go's math.Pow() doesn't guarantee determinism when executed on different hardwares
func SafePow(base uint64, exponent uint64) uint64 {
	baseDec := sdk.NewDecWithPrec(int64(base), 0)
	return baseDec.Power(exponent).TruncateInt().Uint64()
}
