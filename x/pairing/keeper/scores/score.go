// Package scores implements the scoring mechanism used for picking providers in the pairing process.
//
// The pairing process involves the following steps:
// 1. Collect pairing requirements and strategy from the policy.
// 2. Generate pairing slots with requirements (one slot per provider).
// 3. Compute the pairing score of each provider with respect to each slot.
// 4. Pick a provider for each slot with a pseudo-random weighted choice.
//
// Pairing requirements describe the policy-imposed requirements for paired providers. Examples include
// geolocation constraints, ability to service archive requests, and expectations regarding QoS ranking
// of selected providers. Pairing requirements must satisfy the ScoreReq interface, whose methods can
// identify the ScoreReq (by name), and compute a score for a provider with respect to that requirement.
//
// A pairing slot represents a single provider slot in the pairing list (The number of slots for pairing
// is defined by the policy). Each pairing slot holds a set of pairing requirements (a pairing slot may
// repeat). For example, a policy may state that the pairing list has 6 slots, and providers should be
// located in Asia and Europe. This can be satisfied with a pairing list that has 3 (identical) slots
// that require providers in Asia and 3 (identical) slots that require providers in Europe.
//
// A pairing score describes the suitability of a provider for a pairing slot (under a given strategy).
// The score depends on the slot's requirements: for example, given a slot which requires geolocation in
// Asia, a provider in Asia will generally get higher score than one in Europe. The score is calculated for
// each <provider, slot> combination.
//
// A pairing score strategy defines the weight of each score requirement in the final score calculation
// for a <provider, slot> combination. For example, given a slot with several requirements, then the
// overall pairing score would be calculated as score1^w1 + score2^w2 + ... (where score1 is the score
// of the provider with respect to the first requirement, score2 with respect to the second requirement
// and so on).
//
//
// To add a new requirement, one should create a new object that satisfies the ScoreReq interface and update the
// CalcSlots() function so the new requirement will be assigned to the pairing slots (according to some logic).
// Lastly, append the new requirement object to the allReqTypes var in the init() function in this file. Use
// StakeReq or GeoReq as examples.

package scores

import (
	"fmt"
	"math"
	"math/big"
	"sort"

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
			newScoreComp := req.Score(*score.Provider)
			newScoreComp = uint64(math.Pow(float64(newScoreComp), float64(weight)))

			// divide by previous score component (if exists) and multiply by new score
			prevReqScoreComp, ok := score.ScoreComponents[reqName]
			if ok {
				if prevReqScoreComp == 0 {
					utils.LavaFormatFatal("previous req score is zero", fmt.Errorf("invalid req score"),
						utils.Attribute{Key: "req_name", Value: reqName},
						utils.Attribute{Key: "provider", Value: score.Provider.Address},
						utils.Attribute{Key: "stake", Value: score.Provider.Stake.Amount},
						utils.Attribute{Key: "chain_id", Value: score.Provider.Chain},
						utils.Attribute{Key: "geolocation", Value: score.Provider.Geolocation},
					)
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

// PrepareHashData prepares the hash needed in the pseudo-random choice of providers
func PrepareHashData(projectIndex string, chainID string, epochHash []byte) []byte {
	hashData := []byte{}

	// add the session start block hash to the function to make it as unpredictable as we can
	hashData = append(hashData, epochHash...)
	hashData = append(hashData, chainID...)      // to make this pairing unique per chainID
	hashData = append(hashData, projectIndex...) // to make this pairing unique per consumer

	return hashData
}

// PickProviders pick a <group-count> providers set with a pseudo-random weighted choice (using the providers' score list and hashData)
func PickProviders(ctx sdk.Context, scores []*PairingScore, groupCount uint64, hashData []byte, indexToSkipPtr *map[int]bool) (returnedProviders []epochstoragetypes.StakeEntry) {
	scoreSum := sdk.NewUint(0)
	for _, providerScore := range scores {
		scoreSum = scoreSum.Add(sdk.NewUint(providerScore.Score))
	}
	if scoreSum.IsZero() {
		// list is empty
		return returnedProviders
	}

	// sort the list by score (larger scores last). If there are equal scores, sort by provider address (alphabetically)
	sort.SliceStable(scores, func(i, j int) bool {
		// First, compare the Score field
		if scores[i].Score != scores[j].Score {
			return scores[i].Score < scores[j].Score
		}

		// If scores are equal, compare Provider.addr.string field alphabetically
		return scores[i].Provider.Address < scores[j].Provider.Address
	})

	indexToSkip := *indexToSkipPtr
	for it := 0; it < int(groupCount); it++ {
		hash := tendermintcrypto.Sha256(hashData) // TODO: we use cheaper algo for speed
		bigIntNum := new(big.Int).SetBytes(hash)
		hashAsNumber := sdk.NewUintFromBigInt(bigIntNum)
		modRes := hashAsNumber.Mod(scoreSum)

		newScoreSum := sdk.NewUint(0)
		// we loop the scores list from the end because the list is sorted, biggest is last,
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
