package scores

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	scorestypes "github.com/lavanet/lava/x/pairing/types/scores"
	planstypes "github.com/lavanet/lava/x/plans/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

// get the overall requirements from the policy and assign slots that'll fulfil them
// TODO: this function should be changed in the future since it only supports geo and stake reqs
func CalcSlots(policy planstypes.Policy) []*scorestypes.PairingSlot {
	slots := make([]*scorestypes.PairingSlot, policy.MaxProvidersToPair)
	stakeReq := scorestypes.StakeReq{}
	for i := range slots {
		slots[i] = scorestypes.NewPairingSlot()
		slots[i].Reqs[stakeReq.GetName()] = stakeReq
		slots[i].Priority[stakeReq.GetSortPriority()] = stakeReq
	}

	return slots
}

// group the slots and sort them by their sort priority
func GroupAndSortSlots(slots []*scorestypes.PairingSlot) []scorestypes.PairingSlotGroup {
	slotGroups := []scorestypes.PairingSlotGroup{}
	for k := range slots {
		foundGroup := false
		for i := range slotGroups {
			diffSlot := DiffSlot(slots[k], slotGroups[i].Slot)
			if len(diffSlot.Reqs) == 0 {
				slotGroups[i].Count += 1
				foundGroup = true
				break
			}
		}

		if !foundGroup {
			newGroup := scorestypes.NewPairingSlotGroup()
			newGroup.Slot = slots[k]
			newGroup.Count = 1
			slotGroups = append(slotGroups, *newGroup)
		}
	}

	// sort.SliceStable(slotGroups, func(i, j int) bool {
	// 	return slotGroups[i].Slot.Priority[1].Less(slotGroups[j].Slot.Priority[1])
	// })

	return slotGroups
}

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func GetStrategy() scorestypes.ScoreStrategy {
	stakeReq := scorestypes.StakeReq{}
	return scorestypes.ScoreStrategy{stakeReq.GetName(): uint64(1)}
}

// outputs a diff slot - a slot which its requirements are the difference between the previous slot and the current one
func DiffSlot(prev *scorestypes.PairingSlot, current *scorestypes.PairingSlot) *scorestypes.PairingSlot {
	diffSlot := scorestypes.NewPairingSlot()

	for currReqName, currReq := range current.Reqs {
		prevReq, ok := prev.Reqs[currReqName]
		if ok && currReq.Equal(prevReq) {
			continue
		}
		diffSlot.Reqs[currReqName] = currReq
		diffSlot.Priority[currReq.GetSortPriority()] = currReq
	}

	return diffSlot
}

// calculates the final pairing score for all slot groups (with strategy)
// we calculate only the diff between the current and previous slot groups
func CalcPairingScore(scores []*scorestypes.PairingScore, strategy scorestypes.ScoreStrategy, diffSlot *scorestypes.PairingSlot) error {
	for reqName, req := range diffSlot.Reqs {
		weight, ok := strategy[reqName]
		if !ok {
			return utils.LavaFormatError("req not in strategy", sdkerrors.ErrKeyNotFound,
				utils.Attribute{Key: "req_name", Value: reqName})
		}

		for _, score := range scores {
			newScore := req.Score(*score.Provider, weight)
			prevReqScore, ok := score.ScoreComponents[reqName]
			if ok {
				if prevReqScore == 0 {
					return utils.LavaFormatError("previous req score is zero", fmt.Errorf("invalid req score"),
						utils.Attribute{Key: "req_name", Value: reqName})
				}
				score.Score /= prevReqScore
			}
			score.Score *= newScore
			score.ScoreComponents[reqName] = newScore
		}
	}

	return nil
}

// given a list of scores, pick a <group-count> providers with a pseudo-random weighted choice
func PickProviders(ctx sdk.Context, projectIndex string, scores []*scorestypes.PairingScore, groupCount uint64, block uint64, chainID string, epochHash []byte) (returnedProviders []epochstoragetypes.StakeEntry) {
	scoreSum := sdk.NewUint(0)
	hashData := make([]byte, 0)
	for _, providerScore := range scores {
		scoreSum = scoreSum.Add(sdk.NewUint(providerScore.Score))
	}
	if scoreSum.IsZero() {
		// list is empty
		return returnedProviders
	}

	// add the session start block hash to the function to make it as unpredictable as we can
	hashData = append(hashData, epochHash...)
	hashData = append(hashData, chainID...)      // to make this pairing unique per chainID
	hashData = append(hashData, projectIndex...) // to make this pairing unique per consumer

	indexToSkip := make(map[int]bool) // a trick to create a unique set in golang
	for it := 0; it < int(groupCount); it++ {
		hash := tendermintcrypto.Sha256(hashData) // TODO: we use cheaper algo for speed
		bigIntNum := new(big.Int).SetBytes(hash)
		hashAsNumber := sdk.NewUintFromBigInt(bigIntNum)
		modRes := hashAsNumber.Mod(scoreSum)

		newScoreSum := sdk.NewUint(0)
		// we loop the servicers list form the end because the list is sorted, biggest is last,
		// and statistically this will have less iterations

		for idx := len(scores) - 1; idx >= 0; idx-- {
			providerScore := scores[idx]
			if indexToSkip[idx] {
				// this is an index we added
				continue
			}
			newScoreSum = newScoreSum.Add(sdk.NewUint(providerScore.Score))
			if modRes.LT(newScoreSum) {
				// we hit our chosen provider
				returnedProviders = append(returnedProviders, *providerScore.Provider)
				scoreSum = scoreSum.Sub(sdk.NewUint(providerScore.Score)) // we remove this provider from the random pool, so the sum is lower now
				indexToSkip[idx] = true
				break
			}
		}
		if uint64(len(returnedProviders)) >= groupCount {
			return returnedProviders
		}
		if scoreSum.IsZero() {
			break
		}
		hashData = append(hashData, []byte{uint8(it)}...)
	}
	return returnedProviders
}
