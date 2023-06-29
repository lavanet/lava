package score

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
)

// Equal() checks equality between ScoreReq
// Score() calculates a provider's score according to the requirement
// SortPriority() return the sort priority for a ScoreReq type
type ScoreReq interface {
	Equal(other *ScoreReq) bool
	Score(provider epochstoragetypes.StakeEntry) uint64
	GetSortPriority() uint
}

// object to hold the requirements for a specific slot
type PairingSlot struct {
	Reqs []*ScoreReq
}

// object to hold a slot and the number of times it's required
type PairingSlotGroup struct {
	Slot  *PairingSlot
	Count uint64
}

// map: key: ScoreReq, value: weight in the final pairing score
type ScoreStrategy map[ScoreReq]uint64
