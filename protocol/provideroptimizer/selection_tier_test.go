package provideroptimizer

import (
	"strconv"
	"testing"

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
		{Address: "entry7", Score: 0.5},
		{Address: "entry4", Score: 1.0},
		{Address: "entry2", Score: 3.0},
		{Address: "entry6", Score: 4.0},
		{Address: "entry1", Score: 5.0},
		{Address: "entry3", Score: 7.0},
		{Address: "entry5", Score: 8.0},
	}

	assert.Equal(t, expectedScores, st.scores)
}

func TestSelectionTierInst_SelectTierRandomly(t *testing.T) {
	st := NewSelectionTier()

	numTiers := 5
	tierChances := map[int]float64{
		0: 0.2,
		1: 0.3,
		2: 0.1,
	}

	tier := st.SelectTierRandomly(numTiers, tierChances)

	assert.GreaterOrEqual(t, tier, 0)
	assert.Less(t, tier, numTiers)
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
			expectedTier:   []string{"entry5", "entry4", "entry2"},
			name:           "tier 1, 2 entries",
		},
		{
			tier:           2,
			minimumEntries: 2,
			expectedTier:   []string{"entry2", "entry3", "entry6"},
			name:           "tier 2, 2 entries",
		},
		{
			tier:           0,
			minimumEntries: 3,
			expectedTier:   []string{"entry1", "entry5", "entry4"},
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
			expectedTier:   []string{"entry1", "entry5", "entry4", "entry2", "entry3"},
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
			expectedTierLen: 26,
			name:            "tier 1, 26 entries",
		},
		{
			tier:            2,
			minimumEntries:  5,
			expectedTierLen: 26,
			name:            "tier 2, 26 entries",
		},
		{
			tier:            3,
			minimumEntries:  5,
			expectedTierLen: 26,
			name:            "tier 3, 26 entries",
		},
		{
			tier:            0,
			minimumEntries:  26,
			expectedTierLen: 26,
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
