package maps

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

func FindInMap[K comparable, V any](m map[K]V, predicate func(K, V) bool) (K, V, bool) {
	for k, v := range m {
		if predicate(k, v) {
			return k, v, true
		}
	}

	var zeroK K
	var zeroV V
	return zeroK, zeroV, false
}
