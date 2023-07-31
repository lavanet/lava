package slices

import (
	"golang.org/x/exp/constraints"
)

type Number interface {
	constraints.Float | constraints.Integer
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

func Filter[T, V any](slice []T, filter func(T) V) []V {
	values := make([]V, len(slice))
	for i := range slice {
		values[i] = filter(slice[i])
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
