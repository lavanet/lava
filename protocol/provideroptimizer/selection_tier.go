package provideroptimizer

import "github.com/lavanet/lava/v3/utils/rand"

type Entry struct {
	Address string
	Score   float64
	Part    float64
}

type SelectionTier interface {
	AddScore(entry string, score float64)
	GetTier(tier int, numTiers int, minimumEntries int) []Entry
	SelectTierRandomly(numTiers int, tierChances map[int]float64) int
}

type SelectionTierInst struct {
	scores []Entry
}

func NewSelectionTier() SelectionTier {
	return &SelectionTierInst{scores: []Entry{}}
}

func (st *SelectionTierInst) AddScore(entry string, score float64) {
	// add the score to the scores list for the entry while keeping it sorted in descending order
	// this means that the highest score will be at the front of the list, tier 0 is highest scores
	newEntry := Entry{Address: entry, Score: score}
	for i := range st.scores {
		if st.scores[i].Score < score {
			st.scores = append(st.scores[:i], append([]Entry{newEntry}, st.scores[i:]...)...)
			return
		}
	}
	st.scores = append(st.scores, newEntry)
}

func (st *SelectionTierInst) SelectTierRandomly(numTiers int, tierChances map[int]float64) int {
	// select a tier randomly based on the chances given
	// if the chances are not given, select a tier randomly based on the number of tiers
	if len(tierChances) == 0 {
		return rand.Intn(numTiers)
	}
	// calculate the total chance
	totalChance := 0.0
	for _, chance := range tierChances {
		totalChance += chance
	}
	chanceForDefaultTiers := (1 - totalChance) / float64(numTiers-len(tierChances))
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

func (st *SelectionTierInst) GetTier(tier int, numTiers int, minimumEntries int) []Entry {
	// get the tier of scores for the given tier and number of tiers
	entriesLen := len(st.scores)
	if entriesLen < minimumEntries || numTiers == 0 || tier >= numTiers {
		return st.scores
	}

	start, end, fracStart, fracEnd := getPositionsForTier(tier, numTiers, entriesLen)
	if end < minimumEntries {
		return st.scores[:minimumEntries]
	}
	ret := st.scores[start:end]
	if len(ret) >= minimumEntries {
		// apply the relative parts to the first and last entries
		ret[0].Part = 1 - fracStart
		ret[len(ret)-1].Part = fracEnd
		return ret
	}
	// bring in entries from better tiers if insufficient
	// end is > minimumEntries, and end - start < minimumEntries
	entriesToTake := minimumEntries - len(ret)
	entriesToTakeStart := start - entriesToTake
	ret = append(ret, st.scores[entriesToTakeStart:start]...)
	return ret
}

func getPositionsForTier(tier int, numTiers int, entriesLen int) (start int, end int, fracStart float64, fracEnd float64) {
	rankStart := float64(tier+1) / float64(numTiers)
	rankEnd := float64(tier+1) / float64(numTiers)
	// Calculate the position based on the rank
	startPositionF := (float64(entriesLen-1) * rankStart)
	endPositionF := (float64(entriesLen-1) * rankEnd)

	positionStart := int(startPositionF)
	positionEnd := int(endPositionF)
	return positionStart, positionEnd, startPositionF - float64(positionStart), endPositionF - float64(positionEnd)
}
