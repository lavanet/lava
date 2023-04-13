package types

import math "math"

func FindUint64Min(values []uint64) uint64 {
	min := uint64(math.MaxUint64)
	for _, v := range values {
		if v < min {
			min = v
		}
	}

	return min
}

func FindUint64Max(values []uint64) uint64 {
	max := uint64(0)
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	return max
}
