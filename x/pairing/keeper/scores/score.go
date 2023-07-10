// Package scores implements the scoring mechanism used when picking providers in the pairing process.
// The scoring process gets the list of filtered providers and outputs the picked providers to be paired.
//
// General pairing scoring process:
// 	1. Get the pairing requirements and strategy from the policy.
// 	2. Assign slots with requirements.
// 	3. Calculate the score of each provider for each slot (in relation with the strategy and the slot's requirements).
// 	4. Pick a provider for each slot with a pseudo-random weighted choice.
//
// Glossary:
// 	- Pairing requirements:
//		What is it?
// 			Pairing requirements defines the requirements the paired providers must fulfill.
// 	  		For example, a consumer might need a list of providers that will be operate from Africa,
//    		will be able to reply to archive calls, and provide great QoS.
//		Data structure:
//			Each requirement is defined by an object that satisifies the ScoreReq interface.
//
// 	- Pairing slot:
//		What is it?
// 			A Pairing slot represents a part of the pairing list. When a consumer asks to be paired
//  		with a list of providers, he gets a certain number of providers (defined by the policy). Each
// 			slot is a "placeholder" for a provider to be picked which will satisfy part of the pairing requirements.
//			For example, let's say the consumer is supposed to get 6 providers and he needs providers
//			from Asia and Europe. In this scenario, we'll have 6 pairing slots that will have to satisfy
//			the consumer's geo requirements. So there will be 3 slots that will require providers operating
//			from Asia and 3 other slots that will require providers that operate from Europe.
//		Data structure:
//			The PairingSlot object holds a map of requirements. The map keys are the requirement object's type (
//			an object that satisfies the ScoreReq interface) and the map values are the requirement object
//			itself. The slots are assigned with requirements in the CalcSlots() function.
//
// 	- Pairing score:
//		What is it?
// 			The Pairing score defines the score for each provider when picking providers to pair. The score is
//			calculated for all providers for each slot. It depends on the slot requirements since the score of
//			a provider can change with different requirements. For example, say that slot A requires a provider from
//			Europe and slot B requires a provider from Asia. We have 2 possible providers: provider A which operates
//			from Europe and and provider B that operates from Asia. Clearly, provider A's score for the Europe slot
//			will be better than provider B's score for the Europe slot.
//		Data structure:
//			The PairingScore object is created for each provider, holding a pointer to the provider, its final pairing
//			score and the score components (the provider's stake score, geo score, etc.). The ScoreComponents map keys
//			are the requirement object's type and the map values are the score for this requirement.
//
// 	- Pairing score stratgy:
//		What is it?
// 			A Pairing score strategy (or just strategy) defines the weight of each score component in the
// 			final score calculation for each provider. The calculation is done as follows: req1^w1 + req2^w2 + ...
//			For example, say that the score components are stake and geolocation. To make the pairing process
// 			biased towards stake, we could define the strategy: w_stake = 2, w_geo = 1. So the final score
//			will be providerStakeScore^2 + providerGeoScore^1.
//		Data structure:
//			The ScoreStrategy object is a map in which the keys are the requirement object's type and the values
//			are the requirement's weight.
//
//	Adding new requirements:
// 	  To add a new requirement, one should create a new object that satisfies the ScoreReq interface and update the
//	  CalcSlots() function so the new requirement will be assigned to the pairing slots (according to some logic).
//	  Lastly, append the new requirement object to the allReqTypes var in the init() function in this file. Use
//	  StakeReq or GeoReq as examples.

package scores

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

var (
	uniformStrategy ScoreStrategy
	allReqNames     []string
)

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func init() {
	// gather all req names to a list
	allReqNames = append(allReqNames, stakeReqName)

	// init strategy
	uniformStrategy = make(ScoreStrategy)
	for _, reqName := range allReqNames {
		uniformStrategy[reqName] = 1
	}

	if len(allReqNames) != len(uniformStrategy) {
		panic("strategy does not contain all score reqs")
	}
}

// get the overall requirements from the policy and assign slots that'll fulfil them
// TODO: this function should be changed in the future since it only supports stake reqs
func CalcSlots(policy planstypes.Policy, minStake sdk.Int) []*PairingSlot {
	// init slot array (should be as the number of providers to pair)
	slots := make([]*PairingSlot, policy.MaxProvidersToPair)

	// all slots should consider the stake, so we init them with stakeReq
	stakeReq := StakeReq{MinStake: minStake}
	slotReqs := map[string]ScoreReq{stakeReq.GetName(): stakeReq}
	for i := range slots {
		slots[i] = NewPairingSlot()
		slots[i].Reqs = slotReqs
	}

	return slots
}

// group the slots
func GroupSlots(slots []*PairingSlot) []*PairingSlot {
	uniqueSlots := make(map[string]*PairingSlot)
	if len(slots) == 0 {
		utils.LavaFormatError("no slots", sdkerrors.ErrLogic)
		return []*PairingSlot{}
	}

	for i := 0; i < len(slots); i++ {
		slot := slots[i]
		key := slot.GetSlotKey()

		if existingSlot, ok := uniqueSlots[key]; ok {
			// Duplicate slot found
			existingSlot.Count++
		} else {
			// New unique slot
			uniqueSlots[key] = slot
		}
	}

	// Convert map values to slice
	uniqueSlotList := make([]*PairingSlot, 0, len(uniqueSlots))
	for _, slot := range uniqueSlots {
		uniqueSlotList = append(uniqueSlotList, slot)
	}

	return uniqueSlotList
}

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func GetStrategy() ScoreStrategy {
	return uniformStrategy
}

// CalcPairingScore calculates the final pairing score for a pairing slot (with strategy)
// For efficiency purposes, we calculate the score on a diff slot which represents the diff reqs of the current slot
// and the previous slot
func CalcPairingScore(scores []*PairingScore, strategy ScoreStrategy, diffSlot *PairingSlot, minStake sdk.Int) error {
	// calculate the score for each req for each provider
	for _, req := range diffSlot.Reqs {
		reqName := req.GetName()
		weight := strategy[reqName] // not checking if key is found because it's verified in init()

		for _, score := range scores {
			// normalize stake so we won't overflow the score result (uint64)
			newScoreComp := req.Score(*score.Provider, weight)

			// divide by previous score component (if exists) and multiply by new score
			prevReqScoreComp, ok := score.ScoreComponents[reqName]
			if ok {
				if prevReqScoreComp == 0 {
					return utils.LavaFormatError("previous req score is zero", fmt.Errorf("invalid req score"),
						utils.Attribute{Key: "req_name", Value: reqName})
				}
				score.Score /= prevReqScoreComp
			}
			score.Score *= newScoreComp

			// update the score component map
			score.ScoreComponents[reqName] = newScoreComp
		}
	}

	return nil
}

// PickProviders pick a <group-count> providers set with a pseudo-random weighted choice (using the providers' score list and hashData)
func PickProviders(ctx sdk.Context, projectIndex string, scores []*PairingScore, groupCount uint64, chainID string, epochHash []byte, indexToSkipPtr *map[int]bool) (returnedProviders []epochstoragetypes.StakeEntry) {
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
		// we loop the servicers list from the end because the list is sorted, biggest is last,
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
