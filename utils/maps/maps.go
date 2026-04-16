package maps

import (
	"cmp"
	"maps"

	"github.com/lavanet/lava/v5/utils/lavaslices"
)

func FindLargestIntValueInMap[K comparable](myMap map[K]int) (K, int) {
	var maxVal int
	var maxKey K
	firstIteration := true

	for key, val := range myMap {
		if firstIteration || val > maxVal {
			maxVal = val
			maxKey = key
			firstIteration = false
		}
	}

	return maxKey, maxVal
}

func StableSortedKeys[T cmp.Ordered, V any](m map[T]V) []T {
	keys := KeysSlice(m)
	lavaslices.SortStable(keys)
	return keys
}

func GetMaxKey[T cmp.Ordered, V any](m map[T]V) T {
	var maxKey T
	for k := range m {
		if k > maxKey {
			maxKey = k
		}
	}
	return maxKey
}

func KeysSlice[T comparable, V any](in map[T]V) []T {
	return slicesFromIter(maps.Keys(in))
}

func ValuesSlice[T comparable, V any](in map[T]V) []V {
	return slicesFromIter(maps.Values(in))
}

// slicesFromIter collects an iter.Seq into a slice.
func slicesFromIter[T any](seq func(yield func(T) bool)) []T {
	var s []T
	for v := range seq {
		s = append(s, v)
	}
	return s
}
