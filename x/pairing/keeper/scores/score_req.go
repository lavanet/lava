package scores

import (
	"fmt"
	"math/big"
	"reflect"

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
	allReqTypes     []reflect.Type

	// req types (syntactic sugar)
	stakeReqType reflect.Type = reflect.TypeOf(scorestypes.StakeReq{})
)

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func init() {
	// gather all req types to a list
	allReqTypes = append(allReqTypes, stakeReqType)

	// init strategy
	uniformStrategy = make(scorestypes.ScoreStrategy)
	for _, reqType := range allReqTypes {
		uniformStrategy[reqType] = 1
	}
}

// get the overall requirements from the policy and assign slots that'll fulfil them
// TODO: this function should be changed in the future since it only supports stake reqs
func CalcSlots(policy planstypes.Policy) []*scorestypes.PairingSlot {
	slots := make([]*scorestypes.PairingSlot, policy.MaxProvidersToPair)
	stakeReq := scorestypes.StakeReq{}
	stakeReqMap := map[reflect.Type]scorestypes.ScoreReq{stakeReqType: stakeReq}
	for i := range slots {
		slots[i] = scorestypes.NewPairingSlot(stakeReqMap)
	}

	return slots
}

// group the slots
func GroupSlots(slots []*scorestypes.PairingSlot) []scorestypes.PairingSlotGroup {
	slotGroups := []scorestypes.PairingSlotGroup{}
	if len(slots) == 0 {
		utils.LavaFormatError("no slots", sdkerrors.ErrLogic)
		return slotGroups
	}

	for k := range slots {
		foundGroup := false
		for i := range slotGroups {
			diff := slots[k].Diff(slotGroups[i].Slot)
			if len(diff.Reqs) == 0 {
				slotGroups[i].Count += 1
				foundGroup = true
				break
			}
		}

		if !foundGroup {
			newGroup := scorestypes.NewPairingSlotGroup(slots[k])
			slotGroups = append(slotGroups, *newGroup)
		}
	}

	return slotGroups
}

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func GetStrategy() scorestypes.ScoreStrategy {
	return uniformStrategy
}

// calculates the final pairing score for all slot groups (with strategy)
// we calculate only the diff between the current and previous slot groups
func CalcPairingScore(scores []*scorestypes.PairingScore, strategy scorestypes.ScoreStrategy, diffSlot *scorestypes.PairingSlot, minStake sdk.Int) error {
	// calculate the score for each req for each provider
	for _, req := range diffSlot.Reqs {
		weight, ok := strategy[reflect.TypeOf(req)]
		if !ok {
			return utils.LavaFormatError("req not in strategy", sdkerrors.ErrKeyNotFound,
				utils.Attribute{Key: "req_name", Value: reflect.TypeOf(req).String()})
		}

		for _, score := range scores {
			// normalize stake so we won't overflow the score result (uint64)
			providerWithNormalizedStake := score.Provider
			providerWithNormalizedStake.Stake.Amount = providerWithNormalizedStake.Stake.Amount.Quo(minStake)
			newScoreComp := req.Score(*providerWithNormalizedStake, weight)

			// divide by previous score component (if exists) and multiply by new score
			prevReqScoreComp, ok := score.ScoreComponents[reflect.TypeOf(req)]
			if ok {
				if prevReqScoreComp == 0 {
					return utils.LavaFormatError("previous req score is zero", fmt.Errorf("invalid req score"),
						utils.Attribute{Key: "req_bitmap_value", Value: reflect.TypeOf(req).String()})
				}
				score.Score /= prevReqScoreComp
			}
			score.Score *= newScoreComp

			// update the score component map
			score.ScoreComponents[reflect.TypeOf(req)] = newScoreComp
		}
	}

	return nil
}

// given a list of scores, pick a <group-count> providers with a pseudo-random weighted choice
func PickProviders(ctx sdk.Context, projectIndex string, scores []*scorestypes.PairingScore, groupCount uint64, block uint64, chainID string, epochHash []byte, indexToSkipPtr *map[int]bool) (returnedProviders []epochstoragetypes.StakeEntry) {
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

	indexToSkip := *indexToSkipPtr
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
