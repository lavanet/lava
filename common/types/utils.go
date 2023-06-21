package types

import (
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

func RemoveDuplicateFromSlice[T comparable](s []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range s {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
