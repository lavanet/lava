package score

import (
	"reflect"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// when adding a new req, update CalcSlots()

const (
	// TODO: temp strategy weight until we implement taking it out of the policy
	UNIFORM_WEIGHT uint64 = 1
)

// Score() calculates a provider's score according to the requirement
// GetName() gets the ScoreReq's name
type ScoreReq interface {
	Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64
}

// object to hold the requirements for a specific slot (map of req names pointing to req object)
type PairingSlot struct {
	Reqs map[reflect.Type]ScoreReq
}

func NewPairingSlot(reqs map[reflect.Type]ScoreReq) *PairingSlot {
	slot := PairingSlot{
		Reqs: reqs,
	}
	return &slot
}

// generate a diff slot that contains the reqs that are in the slot receiver but not in the "other" slot
func (s PairingSlot) Diff(other *PairingSlot) *PairingSlot {
	reqsDiff := make(map[reflect.Type]ScoreReq)
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
	ScoreComponents map[reflect.Type]uint64
}

func NewPairingScore(provider *epochstoragetypes.StakeEntry) *PairingScore {
	score := PairingScore{
		Provider:        provider,
		Score:           1,
		ScoreComponents: map[reflect.Type]uint64{},
	}
	return &score
}

// map: key: ScoreReq name, value: weight in the final pairing score
type ScoreStrategy map[reflect.Type]uint64
