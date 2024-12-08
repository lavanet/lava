package scores

import (
	"math/rand"
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestTotalScore(t *testing.T) {
	templates := []struct {
		name         string
		scores       []*PairingScore
		groupIndexes []int
		equalScores  bool
	}{
		{
			name:         "no mix filters",
			scores:       generateScores(100, -1),
			groupIndexes: []int{1},
		},
		{
			name:         "mix filters but not active",
			scores:       generateScores(100, 1),
			groupIndexes: []int{2},
		},
		{
			name:         "mix filters active",
			scores:       generateScores(100, 1),
			groupIndexes: []int{1},
			equalScores:  true,
		},
		{
			name:         "score with random filter",
			scores:       generateScoresWithRandomFilter(100),
			groupIndexes: []int{1},
		},
	}

	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			totalScore, sloitIndexScores, err := CalculateTotalScoresForGroup(tt.scores, tt.groupIndexes)
			require.NoError(t, err)
			calculatedScore := math.LegacyZeroDec()
			for _, slotIndexScore := range sloitIndexScores {
				calculatedScore = calculatedScore.Add(slotIndexScore)
			}
			require.True(t, totalScore.GTE(calculatedScore))
			if tt.equalScores {
				require.True(t, totalScore.Equal(calculatedScore))
			}
		})
	}
}

func generateScores(count int, slotFilterIndex int) []*PairingScore {
	ret := []*PairingScore{}
	for i := 0; i < count; i++ {
		pairingScore := &PairingScore{
			Provider:        nil,
			Score:           math.LegacyNewDec(100),
			ScoreComponents: map[string]math.LegacyDec{},
		}
		if slotFilterIndex >= 0 {
			pairingScore.SlotFiltering = map[int]struct{}{slotFilterIndex: {}}
		}
		ret = append(ret, pairingScore)
	}
	return ret
}

func generateScoresWithRandomFilter(count int) []*PairingScore {
	ret := []*PairingScore{}
	for i := 0; i < count; i++ {
		pairingScore := &PairingScore{
			Provider:        nil,
			Score:           math.LegacyNewDec(100),
			ScoreComponents: map[string]math.LegacyDec{},
		}
		pairingScore.SlotFiltering = map[int]struct{}{rand.Int(): {}}
		ret = append(ret, pairingScore)
	}
	return ret
}
