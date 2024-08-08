package scores

import (
	"cosmossdk.io/math"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

const (
	// TODO: temp strategy weight until we implement taking it out of the policy
	UNIFORM_WEIGHT uint64 = 1
)

// PairingScore holds a provider's score with respect to a set of requirements (ScoreReq), indexed by their unique name.
type PairingScore struct {
	Provider            *epochstoragetypes.StakeEntry
	Score               math.Uint
	ScoreComponents     map[string]math.Uint
	SkipForSelection    bool
	SlotFiltering       map[int]struct{} // slot indexes here are skipped
	QosExcellenceReport pairingtypes.QualityOfServiceReport
}

func (ps *PairingScore) IsValidForSelection(slotIndex int) bool {
	if ps.SkipForSelection {
		return false
	}
	_, ok := ps.SlotFiltering[slotIndex]
	// invalid if provider has mix filtering for this slot index
	return !ok
}

func (ps *PairingScore) InvalidIndexes(possibleIndexes []int) []int {
	invalidIndexes := []int{}
	for _, slotIndex := range possibleIndexes {
		if _, ok := ps.SlotFiltering[slotIndex]; ok {
			invalidIndexes = append(invalidIndexes, slotIndex)
		}
	}
	return invalidIndexes
}

func NewPairingScore(stakeEntry *epochstoragetypes.StakeEntry, qos pairingtypes.QualityOfServiceReport) *PairingScore {
	score := PairingScore{
		Provider:            stakeEntry,
		Score:               math.OneUint(),
		ScoreComponents:     map[string]math.Uint{},
		SkipForSelection:    false,
		QosExcellenceReport: qos,
	}
	return &score
}

// map: key: ScoreReq name, value: weight in the final pairing score
type ScoreStrategy map[string]uint64
