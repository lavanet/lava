package maps

import (
	"github.com/lavanet/lava/v4/utils/lavaslices"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
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

func StableSortedKeys[T constraints.Ordered, V any](m map[T]V) []T {
	keys := maps.Keys(m)
	lavaslices.SortStable(keys)
	return keys
}

func GetMaxKey[T constraints.Ordered, V any](m map[T]V) T {
	var maxKey T
	for k := range m {
		if k > maxKey {
			maxKey = k
		}
	}
	return maxKey
}

func KeysSlice[T comparable, V any](in map[T]V) []T {
	keys := []T{}
	for k := range in {
		keys = append(keys, k)
	}
	return keys
}

func ValuesSlice[T comparable, V any](in map[T]V) []V {
	values := []V{}
	for _, v := range in {
		values = append(values, v)
	}
	return values
}
