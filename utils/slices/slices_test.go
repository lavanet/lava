package slices

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlice(t *testing.T) {
	for _, tt := range []struct {
		name   string
		input  []int
		output []int
	}{
		{"empty", []int{}, []int{}},
		{"one element", []int{1}, []int{1}},
		{"many elements", []int{1, 2, 3}, []int{1, 2, 3}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := Slice(tt.input...)
			require.Equal(t, tt.output, output)
		})
	}
}

func TestConcat(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slices [][]int
		concat []int
	}{
		{"one empty slice", [][]int{{}}, []int{}},
		{"two empty slices", [][]int{{}, {}}, []int{}},
		{"empty slice with slice", [][]int{{}, {1, 2}, {}}, []int{1, 2}},
		{"slice with empty slice", [][]int{{1, 2}, {}}, []int{1, 2}},
		{"regular slices", [][]int{{1, 2}, {3, 4}, {5, 6}}, []int{1, 2, 3, 4, 5, 6}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			concat := Concat(tt.slices...)
			require.Equal(t, tt.concat, concat)
		})
	}
}

func TestMin(t *testing.T) {
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
			min := Min(tt.slice)
			require.Equal(t, tt.min, min)
		})
	}
}

func TestMax(t *testing.T) {
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
			max := Max(tt.slice)
			require.Equal(t, tt.max, max)
		})
	}
}

func TestAverage(t *testing.T) {
	for _, tt := range []struct {
		name    string
		slice   []int
		average int
	}{
		{"one element", []int{1}, 1},
		{"two element", []int{1, 3}, 2},
		{"six elements", []int{1, 2, 3, 5, 6, 7}, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			average := Average(tt.slice)
			require.Equal(t, tt.average, average)
		})
	}
}

func TestContains(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slice  []int
		elem   int
		result bool
	}{
		{"empty slice", []int{}, 1, false},
		{"one elem not found", []int{1}, 2, false},
		{"one elem found", []int{1}, 1, true},
		{"elem found twice", []int{1, 1, 2}, 1, true},
		{"elem found last", []int{1, 2}, 2, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := Contains(tt.slice, tt.elem)
			require.Equal(t, tt.result, res)
		})
	}
}

func TestRemove(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slice  []int
		elem   int
		result []int
		found  bool
	}{
		{"empty slice", []int{}, 1, []int{}, false},
		{"elem not found", []int{1, 2, 3}, 4, []int{1, 2, 3}, false},
		{"elem found", []int{1, 2, 3, 4}, 2, []int{1, 4, 3}, true},
		{"elem found (only)", []int{1}, 1, []int{}, true},
		{"elem found first", []int{1, 2, 3}, 1, []int{3, 2}, true},
		{"elem found last", []int{1, 2, 3}, 3, []int{1, 2}, true},
		{"elem found twice", []int{1, 2, 3, 2, 3}, 3, []int{1, 2, 3, 2}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res, found := Remove(tt.slice, tt.elem)
			require.Equal(t, tt.result, res)
			require.Equal(t, tt.found, found)
		})
	}
}

func TestIsSubset(t *testing.T) {
	for _, tt := range []struct {
		name     string
		subset   []int
		superset []int
		result   bool
	}{
		{"empty subset/superset", []int{}, []int{}, true},
		{"empty subset", []int{}, []int{1, 2}, true},
		{"empty superset", []int{1, 2}, []int{}, false},
		{"is subset", []int{1, 2}, []int{1, 3, 2}, true},
		{"is not subset", []int{1, 2, 3}, []int{1, 3}, false},
		{"subset duplicates", []int{1, 2, 2}, []int{1, 2, 3}, true},
		{"superset duplicates", []int{1, 2}, []int{1, 2, 2, 3}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := IsSubset(tt.subset, tt.superset)
			require.Equal(t, tt.result, res)
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

func TestUnion(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slices [][]int
		result []int
	}{
		{"zero slices", [][]int{}, []int{}},
		{"one slice", [][]int{{1, 2}}, []int{1, 2}},
		{"two slices, one empty", [][]int{{1, 2}, {}}, []int{1, 2}},
		{"two slices, non empty", [][]int{{1, 2, 3}, {1, 4}}, []int{1, 2, 3, 4}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := Union(tt.slices...)
			require.Subset(t, tt.result, res)
			require.Subset(t, res, tt.result)
		})
	}
}

// simple struct with Differentiator for TestUnionByFunc
type testUnion struct {
	s string
}

func (x testUnion) Differentiator() string {
	return x.s[0:1]
}

func TestUnionByFunc(t *testing.T) {
	tu := []testUnion{
		{s: "after"},
		{s: "aleph"},
		{s: "about"},
		{s: "below"},
		{s: "bring"},
		{s: "chain"},
		{s: "chase"},
		{s: "chill"},
	}
	for _, tt := range []struct {
		name   string
		slices [][]testUnion
		result []testUnion
	}{
		{"zero slices", [][]testUnion{}, []testUnion{}},
		{"one slice", [][]testUnion{Slice(tu[1], tu[2])}, Slice(tu[2])},
		{"two slices, one empty", [][]testUnion{Slice(tu[1], tu[2]), {}}, Slice(tu[2])},
		{"two slices, non empty", [][]testUnion{Slice(tu[1], tu[2], tu[3]), Slice(tu[1], tu[4])}, Slice(tu[1], tu[4])},
		{"test differentiator same", [][]testUnion{Slice(tu[1], tu[3], tu[6]), Slice(tu[2], tu[4], tu[7])}, Slice(tu[2], tu[4], tu[7])},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := UnionByFunc(tt.slices...)
			require.Subset(t, tt.result, res)
			require.Subset(t, res, tt.result)
		})
	}
}

func TestUnorderedEqual(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slices [][]int
		result bool
	}{
		{"no slices", [][]int{}, true},
		{"one slice empty", [][]int{{}}, true},
		{"one slice, non empty", [][]int{{1, 2}}, true},
		{"two slices, both empty", [][]int{{}, {}}, true},
		{"two slices, one empty", [][]int{{}, {1, 2}}, false},
		{"two slices, non empty", [][]int{{1, 2, 3}, {3, 2, 1}}, true},
		{"two slices, different", [][]int{{1, 2, 4}, {3, 2, 1}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := UnorderedEqual(tt.slices...)
			require.Equal(t, tt.result, res)
		})
	}
}

func TestFilterField(t *testing.T) {
	filter := func(_ int) int { return 10 }
	require.Equal(t, FilterField([]int{}, filter), []int{})
	require.Equal(t, FilterField([]int{1}, filter), []int{10})
	require.Equal(t, FilterField([]int{1, 2, 3}, filter), []int{10, 10, 10})
}

func TestFilter(t *testing.T) {
	filter := func(num int) bool { return num == 3 }
	require.Equal(t, Filter([]int{}, filter), []int{})
	require.Equal(t, Filter([]int{1, 2}, filter), []int{})
	require.Equal(t, Filter([]int{1, 2, 3}, filter), []int{3})
	require.Equal(t, Filter([]int{1, 2, 3, 3}, filter), []int{3, 3})
}
