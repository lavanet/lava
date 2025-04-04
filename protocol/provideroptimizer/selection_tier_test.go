package provideroptimizer

import (
	"strconv"
	"testing"

	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectionTierInst_AddScore(t *testing.T) {
	st := &SelectionTierInst{scores: []Entry{}}

	st.AddScore("entry1", 5.0)
	st.AddScore("entry2", 3.0)
	st.AddScore("entry3", 7.0)
	st.AddScore("entry4", 1.0)
	st.AddScore("entry5", 8.0)
	st.AddScore("entry6", 4.0)
	st.AddScore("entry7", 0.5)

	expectedScores := []Entry{
		{Address: "entry7", Score: 0.5, Part: 1},
		{Address: "entry4", Score: 1.0, Part: 1},
		{Address: "entry2", Score: 3.0, Part: 1},
		{Address: "entry6", Score: 4.0, Part: 1},
		{Address: "entry1", Score: 5.0, Part: 1},
		{Address: "entry3", Score: 7.0, Part: 1},
		{Address: "entry5", Score: 8.0, Part: 1},
	}

	assert.Equal(t, expectedScores, st.scores)
}

func TestSelectionTierInst_GetTier(t *testing.T) {
	st := NewSelectionTier()

	st.AddScore("entry1", 0.1)
	st.AddScore("entry2", 0.5)
	st.AddScore("entry3", 0.7)
	st.AddScore("entry4", 0.3)
	st.AddScore("entry5", 0.2)
	st.AddScore("entry6", 0.9)

	numTiers := 3
	playbook := []struct {
		tier           int
		minimumEntries int
		expectedTier   []string
		name           string
	}{
		{
			tier:           0,
			minimumEntries: 2,
			expectedTier:   []string{"entry1", "entry5"},
			name:           "tier 0, 2 entries",
		},
		{
			tier:           1,
			minimumEntries: 2,
			expectedTier:   []string{"entry4", "entry2"},
			name:           "tier 1, 2 entries",
		},
		{
			tier:           2,
			minimumEntries: 2,
			expectedTier:   []string{"entry3", "entry6"},
			name:           "tier 2, 2 entries",
		},
		{
			tier:           0,
			minimumEntries: 3,
			expectedTier:   []string{"entry1", "entry5"}, // we can only bring better entries
			name:           "tier 0, 3 entries",
		},
		{
			tier:           1,
			minimumEntries: 4,
			expectedTier:   []string{"entry1", "entry5", "entry4", "entry2"},
			name:           "tier 1, 4 entries",
		},
		{
			tier:           1,
			minimumEntries: 5,
			expectedTier:   []string{"entry1", "entry5", "entry4", "entry2"}, // we can only bring better entries
			name:           "tier 1, 5 entries",
		},
		{
			tier:           2,
			minimumEntries: 4,
			expectedTier:   []string{"entry4", "entry2", "entry3", "entry6"},
			name:           "tier 2, 4 entries",
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			result := st.GetTier(play.tier, numTiers, play.minimumEntries)
			require.Equal(t, len(play.expectedTier), len(result), result)
			for i, entry := range play.expectedTier {
				assert.Equal(t, entry, result[i].Address, "result %v, expected: %v", result, play.expectedTier)
			}
			for i := 1; i < len(result); i++ {
				assert.LessOrEqual(t, result[i-1].Score, result[i].Score)
			}
		})
	}
}

func TestSelectionTierInstGetTierBig(t *testing.T) {
	st := NewSelectionTier()

	for i := 0; i < 25; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.1+0.0001*float64(i))
	}
	for i := 25; i < 50; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.2+0.0001*float64(i))
	}
	for i := 50; i < 75; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.3+0.0001*float64(i))
	}
	for i := 75; i < 100; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.4+0.0001*float64(i))
	}

	numTiers := 4
	playbook := []struct {
		tier            int
		minimumEntries  int
		expectedTierLen int
		name            string
	}{
		{
			tier:            0,
			minimumEntries:  5,
			expectedTierLen: 25,
			name:            "tier 0, 25 entries",
		},
		{
			tier:            1,
			minimumEntries:  5,
			expectedTierLen: 25,
			name:            "tier 1, 25 entries",
		},
		{
			tier:            2,
			minimumEntries:  5,
			expectedTierLen: 25,
			name:            "tier 2, 25 entries",
		},
		{
			tier:            3,
			minimumEntries:  5,
			expectedTierLen: 25,
			name:            "tier 3, 25 entries",
		},
		{
			tier:            0,
			minimumEntries:  26,
			expectedTierLen: 25, // we can't bring entries from lower tiers
			name:            "tier 0, 26 entries",
		},
	}
	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			result := st.GetTier(play.tier, numTiers, play.minimumEntries)
			require.Equal(t, play.expectedTierLen, len(result), result)
			for i := 1; i < len(result); i++ {
				assert.LessOrEqual(t, result[i-1].Score, result[i].Score)
			}
		})
	}
}

func TestSelectionTierInstShiftTierChance(t *testing.T) {
	st := NewSelectionTier()
	numTiers := 4
	for i := 0; i < 25; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.1)
	}
	for i := 25; i < 50; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.1)
	}
	for i := 50; i < 75; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.1)
	}
	for i := 75; i < 100; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.1)
	}
	selectionTierChances := st.ShiftTierChance(numTiers, nil)
	require.Equal(t, numTiers, len(selectionTierChances))
	require.Equal(t, selectionTierChances[0], selectionTierChances[1])

	selectionTierChances = st.ShiftTierChance(numTiers, map[int]float64{0: 0.5, 1: 0.5})
	require.Equal(t, 0.0, selectionTierChances[len(selectionTierChances)-1])
	require.Equal(t, 0.5, selectionTierChances[0])

	selectionTierChances = st.ShiftTierChance(numTiers, map[int]float64{0: 0.5, len(selectionTierChances) - 1: 0.1})
	require.Less(t, selectionTierChances[0], 0.5)
	require.Greater(t, selectionTierChances[0], 0.25)
	require.Greater(t, selectionTierChances[len(selectionTierChances)-1], 0.1)

	st = NewSelectionTier()
	for i := 0; i < 25; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.1)
	}
	for i := 25; i < 50; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.2)
	}
	for i := 50; i < 75; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.3)
	}
	for i := 75; i < 100; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.4)
	}
	selectionTierChances = st.ShiftTierChance(numTiers, nil)
	require.Equal(t, numTiers, len(selectionTierChances))
	require.Greater(t, selectionTierChances[0], selectionTierChances[1])
	require.Greater(t, selectionTierChances[1]*3, selectionTierChances[0]) // make sure the adjustment is not that strong
	require.Greater(t, selectionTierChances[1], selectionTierChances[2])

	st = NewSelectionTier()
	for i := 0; i < 25; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.01)
	}
	for i := 25; i < 50; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 1.2)
	}
	for i := 50; i < 75; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 1.3)
	}
	for i := 75; i < 100; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 1.4)
	}
	selectionTierChances = st.ShiftTierChance(numTiers, nil)
	require.Equal(t, numTiers, len(selectionTierChances))
	require.Greater(t, selectionTierChances[0], 0.9)

	st = NewSelectionTier()

	for i := 25; i < 50; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.5)
	}
	for i := 0; i < 25; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.1)
	}
	for i := 50; i < 75; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.5)
	}
	for i := 75; i < 100; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.5)
	}
	selectionTierChances = st.ShiftTierChance(numTiers, nil)
	require.Equal(t, numTiers, len(selectionTierChances))
	require.Greater(t, selectionTierChances[0], selectionTierChances[1]*2.5) // make sure the adjustment is strong enough
	require.Greater(t, selectionTierChances[1]*10, selectionTierChances[0])  // but not too much

	selectionTierChances = st.ShiftTierChance(numTiers, map[int]float64{0: 0.5})
	require.Equal(t, numTiers, len(selectionTierChances))
	require.Greater(t, selectionTierChances[0], 0.5)                                            // make sure the adjustment increases the base chance
	require.Less(t, selectionTierChances[1], (1-0.5)/float64(numTiers-1), selectionTierChances) // and reduces it for lesser tiers
}

func TestSelectionTierInstShiftTierChance_MaintainTopTierAdvantage(t *testing.T) {
	st := NewSelectionTier()
	numTiers := 4
	for i := 0; i < 3; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.195)
	}
	for i := 3; i < 10; i++ {
		st.AddScore("entry"+strconv.Itoa(i), 0.399)
	}

	selectionTierChances := st.ShiftTierChance(numTiers, map[int]float64{0: 0.75, numTiers - 1: 0})
	require.InDelta(t, 0.75, selectionTierChances[0], 0.25)
}

func TestSelectionTierInst_SelectTierRandomly(t *testing.T) {
	st := NewSelectionTier()
	rand.InitRandomSeed()
	// UndoForConnectionChange TEST PR:
	// we only use 4 tiers, adjust the test to use 4 tiers
	numTiers := 4
	counter := map[int]int{}
	for i := 0; i < 10000; i++ {
		// these a are close to the chances used today 0.75 12.5 and 12.5 - last is always 0
		tier := st.SelectTierRandomly(numTiers, map[int]float64{0: 0.8, 1: 0.1, 2: 0.1, 3: 0})
		counter[tier]++
		assert.GreaterOrEqual(t, tier, 0)
		assert.Less(t, tier, numTiers)
	}

	require.Zero(t, counter[4])
	for i := 1; i < 3; i++ {
		require.Greater(t, counter[i], 100)
	}
	require.Greater(t, counter[0], 7000)
	// chance for last tier is 0
	require.Zero(t, counter[3])
}

// UndoForConnectionChange TEST PR: the function does not support nil for tierChances - comment this out
// We only test the use case that we have - this code was written 7 month ago by Omer
// func TestSelectionTierInst_SelectTierRandomly_Default(t *testing.T) {
// 	st := NewSelectionTier()
// 	rand.InitRandomSeed()
// 	// adjusting to 4 tiers - the only case we have
// 	//still not sure about this error - it does not fail locally
// 	numTiers := 4
// 	counter := map[int]int{}
// 	for i := 0; i < 10000; i++ {
// 		tier := st.SelectTierRandomly(numTiers, st.ShiftTierChance(numTiers, nil))
// 		counter[tier]++
// 		assert.GreaterOrEqual(t, tier, 0)
// 		assert.Less(t, tier, numTiers)
// 	}

// 	expectedDistribution := 10000 / numTiers
// 	for _, count := range counter {
// 		assert.InDelta(t, expectedDistribution, count, 300)
// 	}
// }

// TestTierParts tests that when getting a tier, the sum of the parts of the entries
// in each tier is equal to the expected value of entries/numTiers
// note, it's assumed that the number of entries is greater or equal to the number of tiers
func TestTierParts(t *testing.T) {
	templete := []struct {
		name       string
		numTiers   int
		entriesLen int
		expected   map[int][]float64 // expected parts for each tier
	}{
		{"3 tiers 6 entries", 3, 6, map[int][]float64{
			0: {1.0, 1.0},
			1: {1.0, 1.0},
			2: {1.0, 1.0},
		}},
		{"3 tiers 3 entries", 3, 3, map[int][]float64{
			0: {1.0},
			1: {1.0},
			2: {1.0},
		}},
		{"3 tiers 5 entries", 3, 5, map[int][]float64{
			0: {1.0, 2.0 / 3.0},
			1: {1.0 / 3.0, 1.0, 1.0 / 3.0},
			2: {2.0 / 3.0, 1.0},
		}},
		{"3 tiers 4 entries", 3, 4, map[int][]float64{
			0: {1.0, 1.0 / 3.0},
			1: {2.0 / 3.0, 2.0 / 3.0},
			2: {1.0 / 3.0, 1.0},
		}},
		{"4 tiers 11 entries", 4, 11, map[int][]float64{
			0: {1.0, 1.0, 0.75},
			1: {0.25, 1.0, 1.0, 0.5},
			2: {0.5, 1.0, 1.0, 0.25},
			3: {0.75, 1.0, 1.0},
		}},
		{"4 tiers 10 entries", 4, 10, map[int][]float64{
			0: {1.0, 1.0, 0.5},
			1: {0.5, 1.0, 1.0},
			2: {1.0, 1.0, 0.5},
			3: {0.5, 1.0, 1.0},
		}},
	}

	for _, play := range templete {
		for tier := 0; tier < play.numTiers; tier++ {
			st := NewSelectionTier()
			// add entries to the selection tier
			for i := 0; i < play.entriesLen; i++ {
				st.AddScore("entry"+strconv.Itoa(i), 0.1)
			}

			// get tier parts
			partsSum := 0.0
			parts := []float64{}
			entries := st.GetTier(tier, play.numTiers, 1)
			for _, entry := range entries {
				partsSum += entry.Part
				parts = append(parts, entry.Part)
			}

			for i := range parts {
				require.InDelta(t, play.expected[tier][i], parts[i], 0.001,
					"tier: %d, entriesLen: %d, numTiers: %d, index: %d", tier, play.entriesLen, play.numTiers, i)
			}
			assert.InDelta(t, float64(play.entriesLen)/float64(play.numTiers), partsSum, 0.01,
				"tier: %d, entriesLen: %d, numTiers: %d", tier, play.entriesLen, play.numTiers)
		}
	}
}
