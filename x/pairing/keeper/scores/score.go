package scores

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
// To add a new requirement, create an object implementing the ScoreReq interface and add the new requirement in GetAllReqs().

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/rand"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
)

var uniformStrategy ScoreStrategy

// TODO: currently we'll use weight=1 for all reqs. In the future, we'll get it from policy
func init() {
	reqs := GetAllReqs()

	// init strategy
	uniformStrategy = make(ScoreStrategy)
	for _, req := range reqs {
		uniformStrategy[req.GetName()] = 1
	}
}

func GetAllReqs() []ScoreReq {
	return []ScoreReq{
		&StakeReq{},
		&GeoReq{},
		&QosReq{},
	}
}

// get the overall requirements from the policy and assign slots that'll fulfil them
func CalcSlots(policy *planstypes.Policy) []*PairingSlot {
	// init slot array (should be as the number of providers to pair)
	slots := make([]*PairingSlot, policy.MaxProvidersToPair)

	reqs := GetAllReqs()
	for i := range slots {
		reqMap := make(map[string]ScoreReq)
		for _, req := range reqs {
			active := req.Init(*policy)
			if active {
				reqMap[req.GetName()] = req.GetReqForSlot(*policy, i)
			}
		}

		slots[i] = NewPairingSlot(i)
		slots[i].Reqs = reqMap
	}

	return slots
}

// group the slots
func GroupSlots(slots []*PairingSlot) []*PairingSlotGroup {
	uniqueSlots := []*PairingSlotGroup{}

	if len(slots) == 0 {
		panic("no pairing slots available")
	}

	for k := 0; k < len(slots); k++ {
		isUnique := true

		for i := range uniqueSlots {
			if slots[k].Equal(uniqueSlots[i]) {
				uniqueSlots[i].AddToGroup(slots[k])
				isUnique = false
				break
			}
		}

		if isUnique {
			uniqueSlot := NewPairingSlotGroup(slots[k])
			uniqueSlots = append(uniqueSlots, uniqueSlot)
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
func CalcPairingScore(scores []*PairingScore, strategy ScoreStrategy, diffSlot PairingSlotInf) error {
	// calculate the score for each req for each provider
	keys := []string{}
	requirements := diffSlot.Requirements()
	for key := range requirements {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, score := range scores {
		for _, key := range keys {
			req := requirements[key]
			reqName := req.GetName()
			weight, ok := strategy[reqName]
			if !ok {
				return utils.LavaFormatError("req not found in strategy", fmt.Errorf("cannot calculate pairing score"),
					utils.Attribute{Key: "req", Value: reqName},
				)
			}

			newScoreComp := req.Score(*score)
			if newScoreComp == math.ZeroUint() {
				return utils.LavaFormatError("new score component is zero", fmt.Errorf("cannot calculate pairing score"),
					utils.Attribute{Key: "score component", Value: reqName},
					utils.Attribute{Key: "provider", Value: score.Provider.Address},
				)
			}
			newScoreCompDec := sdk.NewDecFromInt(math.Int(newScoreComp))
			newScoreCompDec = newScoreCompDec.Power(weight)
			newScoreComp = math.Uint(newScoreCompDec.TruncateInt())

			// update the score component map
			score.ScoreComponents[reqName] = newScoreComp
		}

		// calc new score
		newScore := math.OneUint()
		for _, scoreComp := range score.ScoreComponents {
			newScore = newScore.Mul(scoreComp)
		}
		score.Score = newScore
	}

	return nil
}

// PrepareHashData prepares the hash needed in the pseudo-random choice of providers
func PrepareHashData(projectIndex, chainID string, epochHash []byte, idx int) []byte {
	return bytes.Join([][]byte{epochHash, []byte(chainID), []byte(projectIndex), []byte(strconv.Itoa(idx))}, nil)
}

// PickProviders pick a <group-count> providers set with a pseudo-random weighted choice
// (using the providers' score list and hashData)
// selection is done by adding all of the scores in the sorted list and generating a random value
func PickProviders(ctx sdk.Context, scores []*PairingScore, groupIndexes []int, hashData []byte) (returnedProviders []epochstoragetypes.StakeEntry) {
	// skip index of providers already selected
	totalScore, slotIndexScore, err := CalculateTotalScoresForGroup(scores, groupIndexes)
	if err != nil {
		return returnedProviders
	}
	rng := rand.New(hashData)
	for _, groupIndex := range groupIndexes {
		if totalScore.IsZero() {
			// no more possible selections
			return returnedProviders
		}
		// effective score is the total score subtracted by the negative score
		effectiveScore := totalScore.Sub(slotIndexScore[groupIndex])
		if effectiveScore.IsZero() {
			// no possible providers for this slot filter, by setting a slot that doesn't exist we disable mix filters and allow all providers not selected yet
			groupIndex = -1
			effectiveScore = totalScore
		}
		randomValue := uint64(rng.Int63n(effectiveScore.BigInt().Int64())) + 1
		newScoreSum := math.ZeroUint()

		for idx := len(scores) - 1; idx >= 0; idx-- {
			if !scores[idx].IsValidForSelection(groupIndex) {
				// skip index of providers that are filtered or already selected
				continue
			}
			providerScore := scores[idx]
			newScoreSum = newScoreSum.Add(providerScore.Score)
			if randomValue <= newScoreSum.Uint64() {
				// we hit our chosen provider
				// remove this provider from the random pool, so the sum is lower now

				returnedProviders = append(returnedProviders, *providerScore.Provider)
				totalScore, slotIndexScore = RemoveProviderFromSelection(providerScore, groupIndexes, totalScore, slotIndexScore)
				break
			}
		}
	}

	return returnedProviders
}

// this function calculates the total score, and negative score modifiers for each slot.
// the negative score modifiers contain the total stake of providers that are not allowed
// so if a provider is not allowed in slot X, it will be added to the total score but will have slotIndexScore[X]+= providerStake
// and during the selection of slot X we will have the selection effective sum as: totalScore - slotIndexScore[X], and that provider that is not allowed can't be selected
func CalculateTotalScoresForGroup(scores []*PairingScore, groupIndexes []int) (totalScore math.Uint, slotIndexScore map[int]math.Uint, err error) {
	if len(scores) == 0 {
		return math.ZeroUint(), nil, fmt.Errorf("invalid scores length")
	}
	totalScore = math.ZeroUint()
	slotIndexScore = map[int]math.Uint{}
	for _, groupIndex := range groupIndexes {
		slotIndexScore[groupIndex] = math.ZeroUint()
	}
	// all all providers to selection possibilities
	for _, providerScore := range scores {
		totalScore, slotIndexScore = AddProviderToSelection(providerScore, groupIndexes, totalScore, slotIndexScore)
	}

	if totalScore == math.ZeroUint() {
		return math.ZeroUint(), nil, utils.LavaFormatError("score sum is zero", fmt.Errorf("cannot pick providers for pairing"))
	}
	return totalScore, slotIndexScore, nil
}

func AddProviderToSelection(providerScore *PairingScore, groupIndexes []int, totalScore math.Uint, slotIndexScore map[int]math.Uint) (math.Uint, map[int]math.Uint) {
	if providerScore.SkipForSelection {
		return totalScore, slotIndexScore
	}
	totalScore = totalScore.Add(providerScore.Score)
	// whenever a provider is not valid in a specific filter we add it's score to the slotIndexScore so we can subtract it
	for _, groupIndex := range providerScore.InvalidIndexes(groupIndexes) {
		slotIndexScore[groupIndex] = slotIndexScore[groupIndex].Add(providerScore.Score)
	}
	return totalScore, slotIndexScore
}

func RemoveProviderFromSelection(providerScore *PairingScore, groupIndexes []int, totalScore math.Uint, slotIndexScore map[int]math.Uint) (math.Uint, map[int]math.Uint) {
	// remove this provider from the total score
	totalScore = totalScore.Sub(providerScore.Score)
	// remove this provider for the subtraction scores as well
	for _, groupIndex := range providerScore.InvalidIndexes(groupIndexes) {
		slotIndexScore[groupIndex] = slotIndexScore[groupIndex].Sub(providerScore.Score)
	}
	// after modifying the score make sure this provider can't be selected again
	providerScore.SkipForSelection = true
	return totalScore, slotIndexScore
}
