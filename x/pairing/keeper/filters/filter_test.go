package filters

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/stretchr/testify/require"
)

type mockFilter struct {
	in string
}

func (mockFilter) Filter(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, currentEpoch uint64) []bool {
	return []bool{}
}

func (mockFilter) InitFilter(strictestPolicy planstypes.Policy) bool {
	return true
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
