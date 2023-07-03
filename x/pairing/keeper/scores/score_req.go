package scores

import (
	"fmt"
	"math/big"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	scorestypes "github.com/lavanet/lava/x/pairing/types/scores"
	planstypes "github.com/lavanet/lava/x/plans/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

var (
	uniformStrategy scorestypes.ScoreStrategy
	maxBitInBitMap  uint64
)

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func init() {
	uniformStrategy = make(scorestypes.ScoreStrategy)
	uniformStrategy[scorestypes.STAKE_REQ] = 1
	maxBitInBitMap = 1
}

// get the overall requirements from the policy and assign slots that'll fulfil them
// TODO: this function should be changed in the future since it only supports stake reqs
func CalcSlots(policy planstypes.Policy) []*scorestypes.PairingSlot {
	slots := make([]*scorestypes.PairingSlot, policy.MaxProvidersToPair)
	for i := range slots {
		slots[i] = scorestypes.NewPairingSlot(scorestypes.STAKE_REQ)
	}

	return slots
}

// group the slots and sort them by their sort priority
func GroupAndSortSlots(slots []*scorestypes.PairingSlot) []scorestypes.PairingSlotGroup {
	slotGroups := []scorestypes.PairingSlotGroup{}
	if len(slots) == 0 {
		utils.LavaFormatError("no slots", sdkerrors.ErrLogic)
		return slotGroups
	}

	// create groups for the slots
	initSlotGroup := scorestypes.NewPairingSlotGroup(slots[0].Reqs)
	slotGroups = append(slotGroups, *initSlotGroup)
	for k := range slots {
		foundGroup := false
		for i := range slotGroups {
			reqsDiff := slots[k].Reqs ^ slotGroups[i].Slot.Reqs
			if reqsDiff == 0 {
				slotGroups[i].Count += 1
				foundGroup = true
				break
			}
		}

		if !foundGroup {
			newGroup := scorestypes.NewPairingSlotGroup(slots[k].Reqs)
			slotGroups = append(slotGroups, *newGroup)
		}
	}

	// sort the groups by hamming distance of the Reqs field (in a PairingSlot object)
	sort.Sort(scorestypes.ByHammingDistance(slotGroups))

	return slotGroups
}

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func GetStrategy() scorestypes.ScoreStrategy {
	return uniformStrategy
}

// calculates the final pairing score for all slot groups (with strategy)
// we calculate only the diff between the current and previous slot groups
func CalcPairingScore(scores []*scorestypes.PairingScore, strategy scorestypes.ScoreStrategy, reqsDiff uint64) error {
	// convert the reqsDiff bitmap to ScoreReq objects
	scoreReqs := createScoreReqsFromBitmap(reqsDiff)

	// calculate the score for each req for each provider
	for _, req := range scoreReqs {
		reqBitmapVal := req.GetBitmapValue()
		weight, ok := strategy[reqBitmapVal]
		if !ok {
			return utils.LavaFormatError("req not in strategy", sdkerrors.ErrKeyNotFound,
				utils.Attribute{Key: "req_bitmap_value", Value: reqBitmapVal})
		}

		for _, score := range scores {
			newScoreComp := req.Score(*score.Provider, weight)

			// divide by previous score component (if exists) and multiply by new score
			prevReqScoreComp, ok := score.ScoreComponents[reqBitmapVal]
			if ok {
				if prevReqScoreComp == 0 {
					return utils.LavaFormatError("previous req score is zero", fmt.Errorf("invalid req score"),
						utils.Attribute{Key: "req_bitmap_value", Value: reqBitmapVal})
				}
				score.Score /= prevReqScoreComp
			}
			score.Score *= newScoreComp

			// update the score component map
			score.ScoreComponents[reqBitmapVal] = newScoreComp
		}
	}

	return nil
}

// TODO: add cases when adding new reqs
func createScoreReqsFromBitmap(bitmap uint64) []scorestypes.ScoreReq {
	scoreReqs := []scorestypes.ScoreReq{}

	for i := uint64(0); i < maxBitInBitMap; i++ {
		bit := (bitmap >> i) & 1 // Get the bit value at position i

		switch i {
		case scorestypes.STAKE_REQ:
			if bit == 1 {
				stakeReq := scorestypes.StakeReq{}
				scoreReqs = append(scoreReqs, stakeReq)
			}
		}
	}

	return scoreReqs
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
