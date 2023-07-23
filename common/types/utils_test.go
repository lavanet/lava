package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindMin(t *testing.T) {
	for _, tt := range []struct {
		name  string
		slice []int
		min   int
	}{
		{"empty", []int{}, 0},
		{"one element", []int{1}, 1},
		{"min is first", []int{1, 2, 3}, 1},
		{"min is middle", []int{2, 1, 3}, 1},
		{"min is last", []int{3, 2, 1}, 1},
		{"min is zero", []int{3, 0, 1}, 0},
		{"min < zero", []int{3, -2, 1}, -2},
		{"min twice", []int{3, 1, 1}, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			min := FindMin(tt.slice)
			require.Equal(t, tt.min, min)
		})
	}
}

func TestFindMax(t *testing.T) {
	for _, tt := range []struct {
		name  string
		slice []int
		max   int
	}{
		{"empty", []int{}, 0},
		{"one element", []int{1}, 1},
		{"max is first", []int{3, 2, 1}, 3},
		{"max is middle", []int{2, 1, 3}, 3},
		{"max is last", []int{1, 2, 3}, 3},
		{"max is zero", []int{-3, 0, -1}, 0},
		{"max < zero", []int{-3, -2, -5}, -2},
		{"max twice", []int{1, 3, 3}, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			max := FindMax(tt.slice)
			require.Equal(t, tt.max, max)
		})
	}
}

func TestIntersection(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slices [][]int
		result []int
	}{
		{"zero slices", [][]int{}, []int{}},
		{"one slice", [][]int{{1, 2}}, []int{1, 2}},
		{"two slices, one empty", [][]int{{1, 2}, {}}, []int{}},
		{"two slices, non empty", [][]int{{1, 2, 3}, {1, 3}}, []int{1, 3}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := Intersection(tt.slices...)
			require.Subset(t, tt.result, res)
			require.Subset(t, res, tt.result)
		})
	}
}
