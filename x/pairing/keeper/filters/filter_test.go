package filters

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"github.com/stretchr/testify/require"
)

type mockFilter struct {
	in string
}

func (mockFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	return []bool{}
}

func (mockFilter) InitFilter(strictestPolicy planstypes.Policy) (bool, []Filter) {
	return true, nil
}

func (mockFilter) IsMix() bool {
	return false
}

func newMockFilter(in string) mockFilter {
	return mockFilter{in: in}
}

func TestMixFilter(t *testing.T) {
	templates := []struct {
		name                 string
		slotCount            int
		mixFilters           []Filter
		expectedFiltersCount int
	}{
		{
			name:                 "1 mix filter - 20",
			slotCount:            20,
			mixFilters:           []Filter{newMockFilter("a")},
			expectedFiltersCount: 10,
		},
		{
			name:                 "1 mix filter - 21",
			slotCount:            21,
			mixFilters:           []Filter{newMockFilter("a")},
			expectedFiltersCount: 10,
		},
		{
			name:                 "2 mix filter - 20",
			slotCount:            20,
			mixFilters:           []Filter{newMockFilter("a"), newMockFilter("b")},
			expectedFiltersCount: 12,
		},
		{
			name:                 "2 mix filter - 21",
			slotCount:            21,
			mixFilters:           []Filter{newMockFilter("a"), newMockFilter("b")},
			expectedFiltersCount: 14,
		},
		{
			name:                 "2 mix filter - 22",
			slotCount:            21,
			mixFilters:           []Filter{newMockFilter("a"), newMockFilter("b")},
			expectedFiltersCount: 14,
		},
		{
			name:                 "no mix filters",
			slotCount:            20,
			mixFilters:           nil,
			expectedFiltersCount: 0,
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			mixFilterIndexes := CalculateMixFilterSlots(tt.mixFilters, tt.slotCount)
			seenIndexes := map[int]struct{}{}
			for _, indexes := range mixFilterIndexes {
				for _, index := range indexes {
					_, ok := seenIndexes[index]
					require.False(t, ok)
					seenIndexes[index] = struct{}{}
				}
			}
			require.Equal(t, tt.expectedFiltersCount, len(seenIndexes))
		})
	}
}

func TestMixFilterBatch(t *testing.T) {
	generateFiltersCount := func(count int) (ret []Filter) {
		for i := 0; i < count; i++ {
			ret = append(ret, newMockFilter(strconv.Itoa(i)))
		}
		return
	}
	templates := []struct {
		name                 string
		slotCount            int
		mixFilters           []Filter
		expectedFiltersCount int
	}{
		{
			name:                 "1 mix filter 10",
			slotCount:            10,
			mixFilters:           []Filter{},
			expectedFiltersCount: 0,
		},
		{
			name:                 "1 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(1),
			expectedFiltersCount: 1,
		},
		{
			name:                 "2 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(2),
			expectedFiltersCount: 1,
		},
		{
			name:                 "9 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(9),
			expectedFiltersCount: 1,
		},
		{
			name:                 "10 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(10),
			expectedFiltersCount: 2,
		},
		{
			name:                 "18 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(18),
			expectedFiltersCount: 2,
		},
		{
			name:                 "19 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(19),
			expectedFiltersCount: 3,
		},
		{
			name:                 "27 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(27),
			expectedFiltersCount: 3,
		},
		{
			name:                 "28 mix filter 10",
			slotCount:            10,
			mixFilters:           generateFiltersCount(28),
			expectedFiltersCount: 4,
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			mixFilterIndexes := CalculateMixFilterSlots(tt.mixFilters, tt.slotCount)
			biggestIndex := -1
			seenIndexes := map[int]int{}
			for _, indexes := range mixFilterIndexes {
				for _, index := range indexes {
					seenIndexes[index]++
					if index > biggestIndex {
						biggestIndex = index
					}
				}
			}
			_, ok := seenIndexes[0]
			require.False(t, ok) // no filters in first batch
			if tt.expectedFiltersCount > 0 {
				require.NotEqual(t, biggestIndex, -1)
			} else {
				require.Equal(t, biggestIndex, -1)
			}
			require.Equal(t, seenIndexes[biggestIndex], tt.expectedFiltersCount)
		})
	}
}
