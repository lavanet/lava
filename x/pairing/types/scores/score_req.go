package score

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// How to add new reqs:
// 1. add a hex const for all possible req scenarios
// 2. update createScoreReqsFromBitmap()
// 3. update CalcSlots()
// 4. update init() (in x/pairing/keeper/scores/score_req.go)

const (
	// TODO: temp strategy weight until we implement taking it out of the policy
	UNIFORM_WEIGHT uint64 = 1
)

// Score() calculates a provider's score according to the requirement
// GetName() gets the ScoreReq's name
type ScoreReq interface {
	Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64
	GetName() string
}

// object to hold the requirements for a specific slot (map of req names pointing to req object)
type PairingSlot struct {
	Reqs map[string]ScoreReq
}

func NewPairingSlot(reqs map[string]ScoreReq) *PairingSlot {
	slot := PairingSlot{
		Reqs: reqs,
	}
	return &slot
}

// generate a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (s PairingSlot) Diff(other *PairingSlot) *PairingSlot {
	reqsDiff := make(map[string]ScoreReq)
	for key := range s.Reqs {
		if _, found := other.Reqs[key]; !found {
			reqsDiff[key] = s.Reqs[key]
		}
	}

	return NewPairingSlot(reqsDiff)
}

// object to hold a slot and the number of times it's required
type PairingSlotGroup struct {
	Slot  *PairingSlot
	Count uint64
}

func NewPairingSlotGroup(slot *PairingSlot) *PairingSlotGroup {
	slotGroup := PairingSlotGroup{
		Slot:  slot,
		Count: 1,
	}
	return &slotGroup
}

// object to hold a provider's score and the requirements that construct the score (map[scoreReqName] -> score from that req)
type PairingScore struct {
	Provider        *epochstoragetypes.StakeEntry
	Score           uint64
	ScoreComponents map[string]uint64
}

func NewPairingScore(provider *epochstoragetypes.StakeEntry) *PairingScore {
	score := PairingScore{
		Provider:        provider,
		Score:           1,
		ScoreComponents: map[string]uint64{},
	}
	return &score
}

// map: key: ScoreReq bitmap value, value: weight in the final pairing score
type ScoreStrategy map[string]uint64
