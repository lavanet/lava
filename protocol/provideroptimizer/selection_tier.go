package provideroptimizer

import (
	"math"

	"github.com/lavanet/lava/v3/utils"
	"github.com/lavanet/lava/v3/utils/lavaslices"
	"github.com/lavanet/lava/v3/utils/rand"
)

type Entry struct {
	Address string
	Score   float64
	Part    float64
}

// selectionTier is a utility to get a tier of addresses based on their scores
type SelectionTier interface {
	AddScore(entry string, score float64)
	GetTier(tier int, numTiers int, minimumEntries int) []Entry
	SelectTierRandomly(numTiers int, tierChances map[int]float64) int
	ShiftTierChance(numTiers int, initialYierChances map[int]float64) map[int]float64
	ScoresCount() int
}

type SelectionTierInst struct {
	scores []Entry
}

func NewSelectionTier() SelectionTier {
	return &SelectionTierInst{scores: []Entry{}}
}

func (st *SelectionTierInst) ScoresCount() int {
	return len(st.scores)
}

func (st *SelectionTierInst) AddScore(entry string, score float64) {
	// add the score to the scores list for the entry while keeping it sorted in ascending order
	// this means that the highest score will be at the front of the list, tier 0 is highest scores
	newEntry := Entry{Address: entry, Score: score, Part: 1}
	// find the correct position to insert the new entry

	for i, existingEntry := range st.scores {
		if existingEntry.Address == entry {
			// overwrite the existing entry
			st.scores[i].Score = score
			return
		}
		if score <= existingEntry.Score {
			st.scores = append(st.scores[:i], append([]Entry{newEntry}, st.scores[i:]...)...)
			return
		}
	}
	// it's not smaller than any existing entry, so add it to the end
	st.scores = append(st.scores, newEntry)
}

func (st *SelectionTierInst) SelectTierRandomly(numTiers int, tierChances map[int]float64) int {
	// select a tier randomly based on the chances given
	// if the chances are not given, select a tier randomly based on the number of tiers
	if len(tierChances) == 0 || len(tierChances) > numTiers {
		utils.LavaFormatError("Invalid tier chances usage", nil, utils.LogAttr("tierChances", tierChances), utils.LogAttr("numTiers", numTiers))
		return rand.Intn(numTiers)
	}
	// calculate the total chance
	chanceForDefaultTiers := st.calcChanceForDefaultTiers(tierChances, numTiers)
	// select a random number between 0 and 1
	randChance := rand.Float64()
	// find the tier that the random chance falls into
	currentChance := 0.0
	for i := 0; i < numTiers; i++ {
		if chance, ok := tierChances[i]; ok {
			currentChance += chance
		} else {
			currentChance += chanceForDefaultTiers
		}
		if randChance < currentChance {
			return i
		}
	}
	// default, should never happen
	return 0
}

func (*SelectionTierInst) calcChanceForDefaultTiers(tierChances map[int]float64, numTiers int) float64 {
	if numTiers <= len(tierChances) {
		return 0
	}
	totalChance := 0.0
	for _, chance := range tierChances {
		totalChance += chance
	}
	// rounding errors can happen
	if totalChance > 1 {
		totalChance = 1
	}
	chanceForDefaultTiers := (1 - totalChance) / float64(numTiers-len(tierChances))
	return chanceForDefaultTiers
}

func (st *SelectionTierInst) averageScoreForTier(tier int, numTiers int) float64 {
	// calculate the average score for the given tier and number of tiers
	start, end, _, _ := getPositionsForTier(tier, numTiers, len(st.scores))
	sum := 0.0
	parts := 0.0
	for i := start; i < end; i++ {
		sum += st.scores[i].Score * st.scores[i].Part
		parts += st.scores[i].Part
	}
	return sum / parts
}

func (st *SelectionTierInst) ShiftTierChance(numTiers int, initialTierChances map[int]float64) map[int]float64 {
	if len(st.scores) == 0 {
		return initialTierChances
	}
	chanceForDefaultTiers := st.calcChanceForDefaultTiers(initialTierChances, numTiers)

	// shift the chances
	shiftedTierChances := make(map[int]float64)
	// shift tier chances based on the difference in the average score of each tier
	scores := make([]float64, numTiers)
	for i := 0; i < numTiers; i++ {
		// scores[i] = 1 / (st.averageScoreForTier(i, numTiers) + 0.0001) // add epsilon to avoid 0
		scores[i] = st.averageScoreForTier(i, numTiers)
	}
	medianScore := lavaslices.Median(scores)
	medianScoreReversed := 1 / (medianScore + 0.0001)
	percentile25Score := lavaslices.Percentile(scores, 0.25)
	percentile25ScoreReversed := 1 / (percentile25Score + 0.0001)

	averageChance := 1 / float64(numTiers)
	for i := 0; i < numTiers; i++ {
		// reverse the score so that higher scores get higher chances
		reversedScore := 1 / (scores[i] + 0.0001)
		// offset the score based on the median and 75th percentile scores, the better they are compared to them the higher the chance
		offsetFactor := 0.5*math.Pow(reversedScore/medianScoreReversed, 2) + 0.5*math.Pow(reversedScore/percentile25ScoreReversed, 2)
		if _, ok := initialTierChances[i]; !ok {
			if chanceForDefaultTiers > 0 {
				shiftedTierChances[i] = chanceForDefaultTiers + averageChance*offsetFactor
			}
		} else {
			if initialTierChances[i] > 0 {
				shiftedTierChances[i] = initialTierChances[i] + averageChance*offsetFactor
			}
		}
	}
	// normalize the chances
	totalChance := 0.0
	for _, chance := range shiftedTierChances {
		totalChance += chance
	}
	for i := 0; i < numTiers; i++ {
		shiftedTierChances[i] /= totalChance
	}
	return shiftedTierChances
}

func (st *SelectionTierInst) GetTier(tier int, numTiers int, minimumEntries int) []Entry {
	// get the tier of scores for the given tier and number of tiers
	entriesLen := len(st.scores)
	if entriesLen < minimumEntries || numTiers == 0 || tier >= numTiers {
		return st.scores
	}

	start, end, fracStart, fracEnd := getPositionsForTier(tier, numTiers, entriesLen)
	if end < minimumEntries {
		// only allow better tiers if there are not enough entries
		return st.scores[:end]
	}
	ret := st.scores[start:end]
	if len(ret) >= minimumEntries {
		// apply the relative parts to the first and last entries
		ret[0].Part = 1 - fracStart
		ret[len(ret)-1].Part = fracEnd
		return ret
	}
	// bring in entries from better tiers if insufficient, give them a handicap to weight
	// end is > minimumEntries, and end - start < minimumEntries
	entriesToTake := minimumEntries - len(ret)
	entriesToTakeStart := start - entriesToTake
	copiedEntries := st.scores[entriesToTakeStart:start]
	entriesToAdd := make([]Entry, len(copiedEntries))
	copy(entriesToAdd, copiedEntries)
	for i := range entriesToAdd {
		entriesToAdd[i].Part = 0.5
	}
	ret = append(entriesToAdd, ret...)
	return ret
}

func getPositionsForTier(tier int, numTiers int, entriesLen int) (start int, end int, fracStart float64, fracEnd float64) {
	rankStart := float64(tier) / float64(numTiers)
	rankEnd := float64(tier+1) / float64(numTiers)
	// Calculate the position based on the rank
	startPositionF := (float64(entriesLen-1) * rankStart)
	endPositionF := (float64(entriesLen-1) * rankEnd)

	positionStart := int(startPositionF)
	positionEnd := int(endPositionF) + 1

	return positionStart, positionEnd, startPositionF - float64(positionStart), float64(positionEnd) - endPositionF
}
