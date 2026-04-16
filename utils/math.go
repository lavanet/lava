package utils

import (
	"cmp"
)

func Min[T cmp.Ordered](x, y T) T {
	if x < y {
		return x
	}
	return y
}

func Max[T cmp.Ordered](x, y T) T {
	if x > y {
		return x
	}
	return y
}
