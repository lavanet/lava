package scores

import (
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	scorestypes "github.com/lavanet/lava/x/pairing/types/scores"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

// get the overall requirements from the policy and assign slots that'll fulfil them
func CalcSlots(policy planstypes.Policy) []scorestypes.PairingSlot

// group the slots and sort them by their sort priority
func GroupAndSortSlots(slots []scorestypes.PairingSlot) []scorestypes.PairingSlotGroup

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func GetStrategy() *scorestypes.ScoreStrategy

// outputs a diff slot - a slot which its requirements are the difference between the previous slot and the current one
func DiffSlot(prev *scorestypes.PairingSlot, current *scorestypes.PairingSlot) *scorestypes.PairingSlot

// calculates the final pairing score for all slot groups (with strategy)
// note: we calculate only the diff between the current and previous slots
func CalcPairingScore(provider epochstoragetypes.StakeEntry, ss *scorestypes.ScoreStrategy,
	diffSlot scorestypes.PairingSlot) uint64

// given a list of scores, pick a provider with a pseudo-random weighted choice
func PickProvider(scores []uint64) epochstoragetypes.StakeEntry
