package utils

import "math"

func SafeUint64ToInt64Convert(val uint64) int64 {
	if val > math.MaxInt64 {
		val = math.MaxInt64
	}
	return int64(val)
}

func Btof(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
