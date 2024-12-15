package lavaslices

import (
	"math"
	"reflect"
	"testing"
	"time"

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

func TestCalculateVariance(t *testing.T) {
	// Test case 1: Variance of a single data point (undefined)
	data1 := []float64{5.0}
	mean1 := 5.0
	expectedVariance1 := 0.0
	variance1 := Variance(data1, mean1)
	if variance1 != expectedVariance1 {
		t.Errorf("Test case 1: Expected %f, got %f", expectedVariance1, variance1)
	}

	// Test case 2: Variance of multiple identical data points
	data2 := []float64{4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 4.0}
	mean2 := 4.0
	expectedVariance2 := 0.0
	variance2 := Variance(data2, mean2)
	if math.Abs(variance2-expectedVariance2) > 1e-9 {
		t.Errorf("Test case 2: Expected %f, got %f", expectedVariance2, variance2)
	}

	data3 := []float64{4.0, 4.0, 4.0, 4.0, 4.0, 4.0, 5.0, 3.0}
	mean3 := 4.0
	expectedVariance3 := 0.2857 // 2/7
	variance3 := Variance(data3, mean3)
	if math.Abs(variance3-expectedVariance3) > 1e-4 {
		t.Errorf("Test case 3: Expected %f, got %f", expectedVariance3, variance3)
	}

	data4 := []time.Duration{time.Second * 3, time.Second * 1, time.Millisecond, time.Microsecond}
	mean := time.Second // intentionally wrong
	variance := Variance(data4, mean)
	require.Greater(t, variance, time.Duration(0))
}

func TestStability(t *testing.T) {
	// Test case 1: Variance of a single data point (undefined)
	data := []float64{5.0, 5.0, 5.0}
	compare := 5.0
	stability := Stability(data, compare)
	require.Less(t, stability, 0.01)

	data = []float64{100.0, 100.0, 100.0}
	compare = 5.0
	stability = Stability(data, compare)
	require.Greater(t, stability, 1.0)

	data = []float64{10.0, 10.0, 10.0, 20.0, 5.0}
	compare = 10.0
	stability = Stability(data, compare)
	require.Greater(t, stability, 0.0)
	require.Less(t, stability, 0.5)

	data2 := []time.Duration{186 * time.Millisecond, 220 * time.Millisecond, 220 * time.Millisecond, 220 * time.Millisecond, 278 * time.Millisecond, 281 * time.Millisecond, 285 * time.Millisecond, 295 * time.Millisecond, 295 * time.Millisecond, 295 * time.Millisecond, 363 * time.Millisecond}
	compare2 := Median(data2)
	stability = Stability(data2, compare2)
	require.Greater(t, stability, 0.0)

	data2 = []time.Duration{938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 938 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 939 * time.Millisecond, 1876 * time.Millisecond, 1876 * time.Millisecond, 1877 * time.Millisecond, 1878 * time.Millisecond, 1908 * time.Millisecond}
	compare2 = Median(data2)
	stability = Stability(data2, compare2)
	require.Less(t, stability, 0.2)
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

func TestMedianInt(t *testing.T) {
	for _, tt := range []struct {
		name   string
		slice  []int
		median int
	}{
		{"empty", []int{}, 0},
		{"one element", []int{1}, 1},
		{"min is first", []int{1, 2, 3}, 2},
		{"min is middle", []int{2, 1, 3}, 2},
		{"min is last", []int{3, 2, 1}, 2},
		{"min is zero", []int{3, 0, 1}, 1},
		{"min < zero", []int{3, -2, 1}, 1},
		{"min twice", []int{3, 1, 1}, 1},
		{"even length", []int{4, 4, 2, 2}, 3},
		{"even length identical", []int{4, 4, 4, 4}, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			median := Median(tt.slice)
			require.Equal(t, tt.median, median)
		})
	}
}

func TestMedianInt64(t *testing.T) {
	tests := []struct {
		name  string
		slice []int64
		want  int64
	}{
		{"empty slice", []int64{}, 0},
		{"single element", []int64{42}, 42},
		{"odd count", []int64{10, 20, 30}, 20},
		{"even count", []int64{10, 20, 30, 40}, 25},
		{"negative values", []int64{-30, -20, -10}, -20},
		{"mixed values", []int64{-10, 0, 10}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			median := Median(tt.slice)
			require.Equal(t, tt.want, median)
		})
	}
}

func TestMedianFloat64(t *testing.T) {
	tests := []struct {
		name  string
		slice []float64
		want  float64
	}{
		{"empty slice", []float64{}, 0},
		{"single element", []float64{42.0}, 42.0},
		{"odd count", []float64{1.1, 2.2, 3.3}, 2.2},
		{"even count", []float64{1.1, 2.2, 3.3, 4.4}, 2.75},
		{"negative values", []float64{-3.3, -2.2, -1.1}, -2.2},
		{"mixed values", []float64{-1.1, 0, 1.1}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			median := Median(tt.slice)
			require.Equal(t, tt.want, median)
		})
	}
}

func TestMedianUint64(t *testing.T) {
	tests := []struct {
		name  string
		slice []uint64
		want  uint64
	}{
		{"odd count", []uint64{10, 20, 30}, 20},
		{"even count", []uint64{10, 20, 30, 40}, 25},
		{"single element", []uint64{42}, 42},
		{"empty slice", []uint64{}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			median := Median(tt.slice)
			require.Equal(t, tt.want, median)
		})
	}
}

func TestMedianPrecision(t *testing.T) {
	t.Run("test large int64", func(t *testing.T) {
		const largeInt = 9007199254740993
		median := Median([]int64{largeInt, largeInt})
		require.Equal(t, int64(largeInt), median)
	})
}

func TestPercentile(t *testing.T) {
	// test it equals median
	for _, tt := range []struct {
		name   string
		slice  []int
		median int
	}{
		{"empty", []int{}, 0},
		{"one element", []int{1}, 1},
		{"min is first", []int{1, 2, 3}, 2},
		{"min is middle", []int{2, 1, 3}, 2},
		{"min is last", []int{3, 2, 1}, 2},
		{"min is zero", []int{3, 0, 1}, 1},
		{"min < zero", []int{3, -2, 1}, 1},
		{"min twice", []int{3, 1, 1}, 1},
		{"even length", []int{4, 4, 2, 2}, 3},
		{"even length identical", []int{4, 4, 4, 4}, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			percentile := Percentile(tt.slice, 0.5)
			median := Median(tt.slice)
			require.Equal(t, tt.median, median)
			require.Equal(t, tt.median, percentile)
		})
	}

	for _, tt2 := range []struct {
		name       string
		slice      []time.Duration
		percentile time.Duration
		rank       float64
	}{
		{"rank empty", []time.Duration{}, 0, 0.3},
		{"rank one element", []time.Duration{1 * time.Millisecond}, 1 * time.Millisecond, 0.3},
		{"rank min is first", []time.Duration{0 * time.Millisecond, 1 * time.Millisecond, 2 * time.Millisecond}, 0, 0.33},
		{"rank min is middle", []time.Duration{2 * time.Millisecond, 1 * time.Millisecond, 3 * time.Millisecond}, 1 * time.Millisecond, 0.33},
		{"rank min is last", []time.Duration{3 * time.Millisecond, 2 * time.Millisecond, 1 * time.Millisecond}, 1 * time.Millisecond, 0.33},
		{"rank min is zero", []time.Duration{3 * time.Millisecond, 0 * time.Millisecond, 1 * time.Millisecond}, 0, 0.33},
		{"rank min < zero", []time.Duration{3 * time.Millisecond, -2 * time.Millisecond, 1 * time.Millisecond}, -2 * time.Millisecond, 0.33},
		{"rank min twice", []time.Duration{3 * time.Millisecond, 1 * time.Millisecond, 1 * time.Millisecond}, 1 * time.Millisecond, 0.33},
		{"rank even length", []time.Duration{4 * time.Millisecond, 4 * time.Millisecond, 2 * time.Millisecond, 2 * time.Millisecond}, 2 * time.Millisecond, 0.33},
		{"rank even length", []time.Duration{1 * time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond, 4 * time.Millisecond}, 1330 * time.Microsecond, 0.33},
		{"rank even length identical", []time.Duration{4 * time.Millisecond, 4 * time.Millisecond, 4 * time.Millisecond, 4 * time.Millisecond}, 4 * time.Millisecond, 0.33},
	} {
		t.Run(tt2.name, func(t *testing.T) {
			percentile := Percentile(tt2.slice, tt2.rank)
			require.Equal(t, tt2.percentile, percentile)
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

func TestMap(t *testing.T) {
	mapFunc := func(_ int) int { return 10 }
	require.Equal(t, []int{}, Map([]int{}, mapFunc))
	require.Equal(t, []int{10}, Map([]int{1}, mapFunc))
	require.Equal(t, []int{10, 10, 10}, Map([]int{1, 2, 3}, mapFunc))
}

func TestFilter(t *testing.T) {
	filter := func(v int) bool { return v%2 == 0 }
	require.Equal(t, []int{}, Filter([]int{}, filter))
	require.Equal(t, []int{}, Filter([]int{1}, filter))
	require.Equal(t, []int{2, 4}, Filter([]int{1, 2, 3, 4}, filter))
}

func TestSliceSplitter(t *testing.T) {
	// Sample usage
	originalSliceSize := 1400
	testSlice := make([]int, originalSliceSize) // Assuming this is your original array
	for i := 0; i < originalSliceSize; i++ {
		testSlice[i] = i
	}
	testSizes := []int{500, 333, 200, 20, 1}
	for _, i := range testSizes {
		originalSizeCopy := originalSliceSize
		retSlice := SplitGenericSliceIntoChunks(testSlice, i)
		for _, k := range retSlice {
			if originalSizeCopy < i {
				require.Len(t, k, originalSizeCopy)
			} else {
				require.Len(t, k, i)
			}
			originalSizeCopy -= i
		}
	}
}

func TestDifference(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []int
		slice2   []int
		expected []int
	}{
		{
			name:     "Basic difference",
			slice1:   []int{1, 2, 3, 4},
			slice2:   []int{3, 4, 5, 6},
			expected: []int{1, 2},
		},
		{
			name:     "No difference",
			slice1:   []int{1, 2, 3},
			slice2:   []int{1, 2, 3},
			expected: []int{},
		},
		{
			name:     "All elements different",
			slice1:   []int{1, 2, 3},
			slice2:   []int{4, 5, 6},
			expected: []int{1, 2, 3},
		},
		{
			name:     "Empty first slice",
			slice1:   []int{},
			slice2:   []int{1, 2, 3},
			expected: []int{},
		},
		{
			name:     "Empty second slice",
			slice1:   []int{1, 2, 3},
			slice2:   []int{},
			expected: []int{1, 2, 3},
		},
		{
			name:     "Mixed elements",
			slice1:   []int{1, 2, 2, 3, 4},
			slice2:   []int{2, 4},
			expected: []int{1, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Difference(tt.slice1, tt.slice2)
			require.True(t, reflect.DeepEqual(result, tt.expected))
		})
	}
}
