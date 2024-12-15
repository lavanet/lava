package lavaslices

import (
	"math"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

type Number interface {
	constraints.Float | constraints.Integer
}

type ComparableByFunc interface {
	Differentiator() string
}

func Slice[T any](v ...T) []T {
	return v
}

func Concat[T any](s ...[]T) []T {
	v := []T{}
	for _, x := range s {
		v = append(v, x...)
	}
	return v
}

func Min[T constraints.Ordered](s []T) (m T) {
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

func Max[T constraints.Ordered](s []T) (m T) {
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

func Average[T Number](slice []T) T {
	var sum T
	for _, val := range slice {
		sum += val
	}
	return sum / T(len(slice))
}

func Variance[T Number](slice []T, mean T) T {
	if len(slice) < 2 {
		return T(0)
	}
	sumSquaredDiffs := T(0)
	for _, x := range slice {
		diff := x - mean
		diffSq := diff * diff
		sumSquaredDiffs += diffSq
	}
	variance := sumSquaredDiffs / T(len(slice)-1)
	return variance
}

func Median[T Number](slice []T) T {
	slices.Sort(slice)
	data_len := len(slice)
	if data_len == 0 {
		return 0
	} else if data_len%2 == 0 {
		left := slice[data_len/2-1]
		right := slice[data_len/2]
		return (left + right) / 2
	} else {
		return slice[(data_len-1)/2]
	}
}

func Percentile[T Number](slice []T, rank float64) T {
	data_len := len(slice)
	if data_len == 0 || rank < 0.0 || rank > 1.0 {
		return 0
	}
	slices.Sort(slice)

	// Calculate the position based on the rank
	position := int(float64(data_len-1) * rank)

	// Calculate the fractional part

	if data_len%2 == 0 {
		// Interpolate between two middle values
		lower := slice[position]
		upper := slice[position+1]
		return lower + T(float64(upper-lower)*rank)
	} else {
		return slice[position]
	}
}

// the bigger it is the more unstable the values in slice from the given argument "compare"
func Stability[T Number](slice []T, compare T) float64 {
	stabilitySum := 0.0
	for _, x := range slice {
		diff := x - compare
		if x < compare {
			diff = compare - x
		}
		stabilitySum += float64(diff) / float64(compare)
	}
	return stabilitySum / float64(len(slice))
}

func Contains[T comparable](slice []T, elem T) bool {
	for _, e := range slice {
		if e == elem {
			return true
		}
	}
	return false
}

func ContainsPredicate[T comparable](slice []T, predicate func(elem T) bool) bool {
	for _, e := range slice {
		if predicate(e) {
			return true
		}
	}
	return false
}

// Remove removes the first instance (if exists) of elem from the slice, and
// returns the new slice and indication if removal took place.
func Remove[T comparable](slice []T, elem T) ([]T, bool) {
	for i, e := range slice {
		if e == elem {
			last := len(slice) - 1
			if i < last {
				slice[i] = slice[last]
			}
			return slice[0:last], true
		}
	}
	return slice, false
}

func AddUnique[T comparable](slice []T, elem T) []T {
	for _, e := range slice {
		if e == elem {
			return slice
		}
	}
	return append(slice, elem)
}

func IsSubset[T comparable](subset, superset []T) bool {
	subsetMap := make(map[T]bool)
	commonMap := make(map[T]bool)

	for _, elem := range subset {
		subsetMap[elem] = true
	}

	for _, elem := range superset {
		if _, ok := subsetMap[elem]; ok {
			commonMap[elem] = true
		}
	}

	return len(commonMap) == len(subsetMap)
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
	elements := make(map[T]bool)

	for _, arr := range arrays {
		for _, elem := range arr {
			elements[elem] = true
		}
	}

	res := make([]T, len(elements))

	count := 0
	for elem := range elements {
		res[count] = elem
		count++
	}

	return res
}

func UnionByFunc[T ComparableByFunc](arrays ...[]T) []T {
	elements := make(map[string]T)

	for _, arr := range arrays {
		for _, elem := range arr {
			elements[elem.Differentiator()] = elem
		}
	}

	res := make([]T, len(elements))

	count := 0
	for _, elem := range elements {
		res[count] = elem
		count++
	}

	return res
}

func Difference[T comparable](slice1, slice2 []T) []T {
	// This function returns the difference between two slices
	// (i.e., the elements that are in slice1 but not in slice2)

	// Create a map to store elements of the second slice for quick lookup
	elementMap := make(map[T]bool)
	for _, elem := range slice2 {
		elementMap[elem] = true
	}

	// Create a slice to hold the difference
	diff := make([]T, 0)
	for _, elem := range slice1 {
		// If the element in slice1 is not in slice2, add it to the result
		if !elementMap[elem] {
			diff = append(diff, elem)
		}
	}

	return diff
}

func Map[T, V any](slice []T, filter func(T) V) []V {
	values := make([]V, len(slice))
	for i := range slice {
		values[i] = filter(slice[i])
	}
	return values
}

func Filter[T any](slice []T, filter func(T) bool) []T {
	values := make([]T, 0)
	for _, v := range slice {
		if filter(v) {
			values = append(values, v)
		}
	}
	return values
}

func UnorderedEqual[T comparable](slices ...[]T) bool {
	var length int

	if len(slices) > 0 {
		length = len(slices[0])
		for _, s := range slices[1:] {
			if len(s) != length {
				return false
			}
		}

		intersect := Intersection(slices...)
		if len(intersect) != length {
			return false
		}
	}

	return true
}

// splitSliceGeneric splits a slice into smaller slices of at most chunkSize length.
// for example len(arr) == 1400 and chunk size 500 will return [500, 500, 400]
func SplitGenericSliceIntoChunks[T any](arr []T, chunkSize int) [][]T {
	var result [][]T

	// Calculate the number of chunks needed
	numChunks := int(math.Ceil(float64(len(arr)) / float64(chunkSize)))

	// Iterate over the original slice and slice it into chunks
	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize

		// Ensure end doesn't exceed the length of the slice
		if end > len(arr) {
			end = len(arr)
		}

		// Create a chunk with preallocated capacity
		chunk := make([]T, 0, chunkSize)

		// Append elements to the chunk
		for j := start; j < end; j++ {
			chunk = append(chunk, arr[j])
		}

		// Append the chunk to the result slice
		result = append(result, chunk)
	}

	return result
}

func SortStable[T constraints.Ordered](slice []T) {
	slices.SortStableFunc(slice, func(i, j T) bool { return i < j })
}

// This function is used to check if the slice is consecutive.
// It returns the index of the first non-consecutive element or 0 if all elements are consecutive.
func IsSliceConsecutive[T constraints.Integer](slice []T) (int, bool) {
	for index := range slice {
		if index != 0 && slice[index]-1 != slice[index-1] {
			return index, false
		}
	}

	return 0, true
}
