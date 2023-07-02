package score

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

const MAX_REQ_PRIORITY = 1

// Equal() checks equality between ScoreReq
// Score() calculates a provider's score according to the requirement
// SortPriority() return the sort priority for a ScoreReq type
type ScoreReq interface {
	Equal(other ScoreReq) bool
	Less(other ScoreReq) bool
	Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64
	GetName() string
	GetSortPriority() int
}

// object to hold the requirements for a specific slot (map of req names pointing to req object)
type PairingSlot struct {
	Reqs     map[string]ScoreReq
	Priority map[int]ScoreReq
}

func NewPairingSlot() *PairingSlot {
	slot := PairingSlot{
		Reqs:     map[string]ScoreReq{},
		Priority: map[int]ScoreReq{},
	}
	return &slot
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

// map: key: ScoreReq name, value: weight in the final pairing score
type ScoreStrategy map[string]uint64

// object to hold a slot and the number of times it's required
type PairingSlotGroup struct {
	Slot  *PairingSlot
	Count uint64
}

func NewPairingSlotGroup() *PairingSlotGroup {
	slotGroup := PairingSlotGroup{
		Slot: NewPairingSlot(),
	}
	return &slotGroup
}
