package score

import (
	"math/bits"

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

	// reqs bitmap
	STAKE_REQ uint64 = 0x1
)

// Equal() checks equality between ScoreReq
// Score() calculates a provider's score according to the requirement
// GetName() gets the ScoreReq's name
// SortPriority() return the sort priority for a ScoreReq type
type ScoreReq interface {
	Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64
	GetBitmapValue() uint64
}

// object to hold the requirements for a specific slot (map of req names pointing to req object)
type PairingSlot struct {
	Reqs uint64
}

func NewPairingSlot(reqs uint64) *PairingSlot {
	slot := PairingSlot{
		Reqs: reqs,
	}
	return &slot
}

// object to hold a slot and the number of times it's required
type PairingSlotGroup struct {
	Slot  *PairingSlot
	Count uint64
}

func NewPairingSlotGroup(reqs uint64) *PairingSlotGroup {
	slotGroup := PairingSlotGroup{
		Slot:  NewPairingSlot(reqs),
		Count: 1,
	}
	return &slotGroup
}

// type that satisfies the sort interface. The groups will be sorted by hamming distance
type ByHammingDistance []PairingSlotGroup

func (s ByHammingDistance) Len() int {
	return len(s)
}

func (s ByHammingDistance) Less(i, j int) bool {
	hammingDistanceI := hammingDistance(s[i].Slot.Reqs, s[j].Slot.Reqs)
	hammingDistanceJ := hammingDistance(s[j].Slot.Reqs, s[(j+1)%len(s)].Slot.Reqs)

	return hammingDistanceI < hammingDistanceJ
}

func (s ByHammingDistance) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Calculate the Hamming distance between two bitmaps
func hammingDistance(a, b uint64) int {
	xor := a ^ b
	return bits.OnesCount64(xor)
}

// object to hold a provider's score and the requirements that construct the score (map[scoreReqName] -> score from that req)
type PairingScore struct {
	Provider        *epochstoragetypes.StakeEntry
	Score           uint64
	ScoreComponents map[uint64]uint64
}

func NewPairingScore(provider *epochstoragetypes.StakeEntry) *PairingScore {
	score := PairingScore{
		Provider:        provider,
		Score:           1,
		ScoreComponents: map[uint64]uint64{},
	}
	return &score
}

// map: key: ScoreReq bitmap value, value: weight in the final pairing score
type ScoreStrategy map[uint64]uint64
