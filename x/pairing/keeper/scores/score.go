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
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
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
		panic("uniform strategy does not contain all score reqs")
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
	uniqueSlots := []*PairingSlot{}

	if len(slots) == 0 {
		panic("no pairing slots available")
	}

	uniqueSlots = append(uniqueSlots, slots[0])
	for k := 1; k < len(slots); k++ {
		isUnique := true

		for i := range uniqueSlots {
			if slots[k].Equal(uniqueSlots[i]) {
				uniqueSlots[i].Count += 1
				isUnique = false
				break
			}
		}

		if isUnique {
			uniqueSlots = append(uniqueSlots, slots[k])
		}
	}

	return uniqueSlots
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
			if newScoreComp == 0 {
				err := fmt.Errorf("score component is zero. score component name: %s, provider address: %s", reqName, score.Provider.Address)
				panic(err)
			}
			newScoreComp = commontypes.SafePow(newScoreComp, weight)

			// divide by previous score component (if exists) and multiply by new score
			prevReqScoreComp, ok := score.ScoreComponents[reqName]
			if ok {
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
func PickProviders(ctx sdk.Context, scores []*PairingScore, groupCount int, hashData []byte, chosenProvidersIdx map[int]bool) (returnedProviders []epochstoragetypes.StakeEntry) {
	if len(scores) == 0 {
		return returnedProviders
	}

	var scoreSum uint64
	for idx, providerScore := range scores {
		if chosenProvidersIdx[idx] {
			// skip index of providers already selected
			continue
		}
		scoreSum += providerScore.Score
	}
	if scoreSum == 0 {
		err := fmt.Errorf("score sum is zero. Cannot pick providers for pairing")
		panic(err)
	}

	for it := 0; it < groupCount; it++ {
		hash := tendermintcrypto.Sha256(hashData) // TODO: we use cheaper algo for speed
		bigIntNum := new(big.Int).SetBytes(hash)
		hashAsNumber := sdk.NewUintFromBigInt(bigIntNum)
		scoreSumUint := sdk.NewUint(scoreSum)
		modRes := hashAsNumber.Mod(scoreSumUint).Uint64()

		newScoreSum := uint64(0)

		for idx := len(scores) - 1; idx >= 0; idx-- {
			if chosenProvidersIdx[idx] {
				// skip index of providers already selected
				continue
			}
			providerScore := scores[idx]
			newScoreSum += providerScore.Score
			if modRes < newScoreSum {
				// we hit our chosen provider
				returnedProviders = append(returnedProviders, *providerScore.Provider)
				scoreSum -= providerScore.Score // we remove this provider from the random pool, so the sum is lower now
				chosenProvidersIdx[idx] = true
				break
			}
		}

		if scoreSum == 0 {
			break
		}
		hashData = append(hashData, []byte{uint8(it)}...)
	}
	return returnedProviders
}
