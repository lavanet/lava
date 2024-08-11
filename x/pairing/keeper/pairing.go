package keeper

import (
	"fmt"
	"math"
	"strconv"

	cosmosmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	pairingfilters "github.com/lavanet/lava/v2/x/pairing/keeper/filters"
	pairingscores "github.com/lavanet/lava/v2/x/pairing/keeper/scores"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

func (k Keeper) VerifyPairingData(ctx sdk.Context, chainID string, block uint64) (epoch uint64, providersType spectypes.Spec_ProvidersTypes, errorRet error) {
	// TODO: add support for spec changes
	foundAndActive, _, providersType := k.specKeeper.IsSpecFoundAndActive(ctx, chainID)
	if !foundAndActive {
		return 0, providersType, fmt.Errorf("spec not found and active for chainID given: %s", chainID)
	}
	earliestSavedEpoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
	if block < earliestSavedEpoch {
		return 0, providersType, fmt.Errorf("block %d is earlier than earliest saved block %d", block, earliestSavedEpoch)
	}

	requestedEpochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, block)
	if err != nil {
		return 0, providersType, err
	}
	currentEpochStart := k.epochStorageKeeper.GetEpochStart(ctx)

	if requestedEpochStart > currentEpochStart {
		return 0, providersType, utils.LavaFormatWarning("VerifyPairing requested epoch is too new", fmt.Errorf("cant get epoch start for future block"),
			utils.Attribute{Key: "requested block", Value: block},
			utils.Attribute{Key: "requested epoch", Value: requestedEpochStart},
			utils.Attribute{Key: "current epoch", Value: currentEpochStart},
		)
	}

	blocksToSave, err := k.epochStorageKeeper.BlocksToSave(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return 0, providersType, err
	}

	if requestedEpochStart+blocksToSave < currentEpochStart {
		return 0, providersType, fmt.Errorf("requestedEpochStart %d is earlier current epoch %d by more than BlocksToSave %d", requestedEpochStart, currentEpochStart, blocksToSave)
	}
	return requestedEpochStart, providersType, nil
}

func (k Keeper) GetProjectData(ctx sdk.Context, developerKey sdk.AccAddress, chainID string, blockHeight uint64) (proj projectstypes.Project, errRet error) {
	project, err := k.projectsKeeper.GetProjectForDeveloper(ctx, developerKey.String(), blockHeight)
	if err != nil {
		return projectstypes.Project{}, err
	}

	if !project.Enabled {
		return projectstypes.Project{}, utils.LavaFormatWarning("the developers project is disabled", fmt.Errorf("cannot get project data"),
			utils.Attribute{Key: "project", Value: project.Index},
		)
	}

	return project, nil
}

func (k Keeper) GetPairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress) (providers []epochstoragetypes.StakeEntry, errorRet error) {
	block := uint64(ctx.BlockHeight())
	project, err := k.GetProjectData(ctx, clientAddress, chainID, block)
	if err != nil {
		return nil, err
	}

	strictestPolicy, cluster, err := k.GetProjectStrictestPolicy(ctx, project, chainID, block)
	if err != nil {
		return nil, fmt.Errorf("invalid user for pairing: %s", err.Error())
	}

	providers, _, _, err = k.getPairingForClient(ctx, chainID, block, strictestPolicy, cluster, project.Index, false, true)

	return providers, err
}

// CalculatePairingChance calculates the chance of a provider to be picked in the pairing process for the first pairing slot
func (k Keeper) CalculatePairingChance(ctx sdk.Context, provider string, chainID string, policy *planstypes.Policy, cluster string) (cosmosmath.LegacyDec, error) {
	totalScore := cosmosmath.ZeroUint()
	providerScore := cosmosmath.ZeroUint()

	_, _, scores, err := k.getPairingForClient(ctx, chainID, uint64(ctx.BlockHeight()), policy, cluster, "dummy", true, false)
	if err != nil {
		return cosmosmath.LegacyZeroDec(), err
	}

	for _, score := range scores {
		if score.Provider.Address == provider {
			providerScore = providerScore.Add(score.Score)
		}
		totalScore = totalScore.Add(score.Score)
	}

	if providerScore.IsZero() {
		return cosmosmath.LegacyZeroDec(), utils.LavaFormatError("provider not found in provider scores array", fmt.Errorf("cannot calculate pairing chance"),
			utils.LogAttr("provider", provider),
			utils.LogAttr("chain_id", chainID),
		)
	}

	providerScoreDec := cosmosmath.LegacyNewDecFromInt(cosmosmath.Int(providerScore))
	totalScoreDec := cosmosmath.LegacyNewDecFromInt(cosmosmath.Int(totalScore))

	return providerScoreDec.Quo(totalScoreDec), nil
}

// function used to get a new pairing from provider and client
// first argument has all metadata, second argument is only the addresses
// useCache is a boolean argument that is used to determine whether pairing cache should be used
// Note: useCache should only be true for queries! functions that write to the state and use this function should never put useCache=true
func (k Keeper) getPairingForClient(ctx sdk.Context, chainID string, block uint64, policy *planstypes.Policy, cluster string, projectIndex string, calcChance bool, useCache bool) (providers []epochstoragetypes.StakeEntry, allowedCU uint64, providerScores []*pairingscores.PairingScore, errorRet error) {
	epoch, providersType, err := k.VerifyPairingData(ctx, chainID, block)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("invalid pairing data: %s", err)
	}

	if useCache {
		providers, found := k.GetPairingQueryCache(projectIndex, chainID, epoch)
		if found {
			return providers, policy.EpochCuLimit, nil, nil
		}
	}

	stakeEntries := k.epochStorageKeeper.GetAllStakeEntriesForEpochChainId(ctx, epoch, chainID)
	if len(stakeEntries) == 0 {
		return nil, 0, nil, fmt.Errorf("did not find providers for pairing: epoch:%d, chainID: %s", block, chainID)
	}
	epochHash := k.epochStorageKeeper.GetEpochHash(ctx, epoch)

	if providersType == spectypes.Spec_static {
		frozenFilter := pairingfilters.FrozenProvidersFilter{}
		frozenFilter.InitFilter(*policy)
		filterResults := frozenFilter.Filter(ctx, stakeEntries, epoch)
		stakeEntriesFiltered := []epochstoragetypes.StakeEntry{}
		for i := 0; i < len(stakeEntries); i++ {
			if filterResults[i] {
				stakeEntriesFiltered = append(stakeEntriesFiltered, stakeEntries[i])
			}
		}
		if useCache {
			k.SetPairingQueryCache(projectIndex, chainID, epoch, stakeEntriesFiltered)
		}
		return stakeEntriesFiltered, policy.EpochCuLimit, nil, nil
	}

	filters := pairingfilters.GetAllFilters()
	// create the pairing slots with assigned reqs
	slots := pairingscores.CalcSlots(policy)
	// group identical slots (in terms of reqs types)
	slotGroups := pairingscores.GroupSlots(slots)
	// filter relevant providers and add slotFiltering for mix filters
	providerScores, err = pairingfilters.SetupScores(ctx, filters, stakeEntries, policy, epoch, len(slots), cluster, k)
	if err != nil {
		return nil, 0, nil, err
	}

	if len(slots) >= len(providerScores) {
		filteredEntries := []epochstoragetypes.StakeEntry{}
		for _, score := range providerScores {
			filteredEntries = append(filteredEntries, *score.Provider)
		}
		if useCache {
			k.SetPairingQueryCache(projectIndex, chainID, epoch, filteredEntries)
		}
		return filteredEntries, policy.EpochCuLimit, nil, nil
	}

	// calculate score (always on the diff in score components of consecutive groups) and pick providers
	prevGroupSlot := pairingscores.NewPairingSlotGroup(pairingscores.NewPairingSlot(-1)) // init dummy slot to compare to
	for idx, group := range slotGroups {
		hashData := pairingscores.PrepareHashData(projectIndex, chainID, epochHash, idx)
		diffSlot := group.Subtract(prevGroupSlot)
		err := pairingscores.CalcPairingScore(providerScores, pairingscores.GetStrategy(), diffSlot)
		if err != nil {
			return nil, 0, nil, err
		}
		if calcChance {
			break
		}
		pickedProviders := pairingscores.PickProviders(ctx, providerScores, group.Indexes(), hashData)
		providers = append(providers, pickedProviders...)
		prevGroupSlot = group
	}

	if useCache {
		k.SetPairingQueryCache(projectIndex, chainID, epoch, providers)
	}

	return providers, policy.EpochCuLimit, providerScores, nil
}

func (k Keeper) GetProjectStrictestPolicy(ctx sdk.Context, project projectstypes.Project, chainID string, block uint64) (*planstypes.Policy, string, error) {
	plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, project.GetSubscription(), block)
	if err != nil {
		return nil, "", err
	}

	planPolicy := plan.GetPlanPolicy()
	policies := []*planstypes.Policy{&planPolicy}
	if project.SubscriptionPolicy != nil {
		policies = append(policies, project.SubscriptionPolicy)
	}
	if project.AdminPolicy != nil {
		policies = append(policies, project.AdminPolicy)
	}
	chainPolicy, allowed := planstypes.GetStrictestChainPolicyForSpec(chainID, policies)
	if !allowed {
		return nil, "", fmt.Errorf("chain ID not allowed in all policies, or collections specified and have no intersection %#v", policies)
	}
	geolocation, err := k.CalculateEffectiveGeolocationFromPolicies(policies)
	if err != nil {
		return nil, "", err
	}

	providersToPair, err := k.CalculateEffectiveProvidersToPairFromPolicies(policies)
	if err != nil {
		return nil, "", err
	}

	sub, found := k.subscriptionKeeper.GetSubscription(ctx, project.GetSubscription())
	if !found {
		return nil, "", fmt.Errorf("could not find subscription with address %s", project.GetSubscription())
	}
	allowedCUEpoch, allowedCUTotal := k.CalculateEffectiveAllowedCuPerEpochFromPolicies(policies, project.GetUsedCu(), sub.GetMonthCuLeft())

	selectedProvidersMode, selectedProvidersList := k.CalculateEffectiveSelectedProviders(policies)

	strictestPolicy := &planstypes.Policy{
		GeolocationProfile:    geolocation,
		MaxProvidersToPair:    providersToPair,
		SelectedProvidersMode: selectedProvidersMode,
		SelectedProviders:     selectedProvidersList,
		ChainPolicies:         []planstypes.ChainPolicy{chainPolicy},
		EpochCuLimit:          allowedCUEpoch,
		TotalCuLimit:          allowedCUTotal,
	}

	return strictestPolicy, sub.Cluster, nil
}

func (k Keeper) CalculateEffectiveSelectedProviders(policies []*planstypes.Policy) (planstypes.SELECTED_PROVIDERS_MODE, []string) {
	selectedProvidersModeList := []planstypes.SELECTED_PROVIDERS_MODE{}
	selectedProvidersList := [][]string{}
	for _, p := range policies {
		selectedProvidersModeList = append(selectedProvidersModeList, p.SelectedProvidersMode)
		if p.SelectedProvidersMode == planstypes.SELECTED_PROVIDERS_MODE_EXCLUSIVE || p.SelectedProvidersMode == planstypes.SELECTED_PROVIDERS_MODE_MIXED {
			if len(p.SelectedProviders) != 0 {
				selectedProvidersList = append(selectedProvidersList, p.SelectedProviders)
			}
		}
	}

	effectiveMode := lavaslices.Max(selectedProvidersModeList)
	effectiveSelectedProviders := lavaslices.Intersection(selectedProvidersList...)

	return effectiveMode, effectiveSelectedProviders
}

func (k Keeper) CalculateEffectiveGeolocationFromPolicies(policies []*planstypes.Policy) (int32, error) {
	geolocation := int32(math.MaxInt32)

	// geolocation is a bitmap. common denominator can be calculated with logical AND
	for _, policy := range policies {
		if policy != nil {
			geo := policy.GetGeolocationProfile()
			if geo == planstypes.Geolocation_value["GLS"] {
				return planstypes.Geolocation_value["GL"], nil
			}
			geolocation &= geo
		}
	}

	if geolocation == 0 {
		return 0, utils.LavaFormatWarning("invalid strictest geolocation", fmt.Errorf("strictest geo = 0"))
	}

	return geolocation, nil
}

func (k Keeper) CalculateEffectiveProvidersToPairFromPolicies(policies []*planstypes.Policy) (uint64, error) {
	providersToPair := uint64(math.MaxUint64)

	for _, policy := range policies {
		val := policy.GetMaxProvidersToPair()
		if policy != nil && val < providersToPair {
			providersToPair = val
		}
	}

	if providersToPair == uint64(math.MaxUint64) {
		return 0, fmt.Errorf("could not calculate providersToPair value: all policies are nil")
	}

	return providersToPair, nil
}

func (k Keeper) CalculateEffectiveAllowedCuPerEpochFromPolicies(policies []*planstypes.Policy, cuUsedInProject, cuLeftInSubscription uint64) (allowedCUThisEpoch, allowedCUTotal uint64) {
	var policyEpochCuLimit []uint64
	var policyTotalCuLimit []uint64
	for _, policy := range policies {
		if policy != nil {
			if policy.EpochCuLimit != 0 {
				policyEpochCuLimit = append(policyEpochCuLimit, policy.GetEpochCuLimit())
			}
			if policy.TotalCuLimit != 0 {
				policyTotalCuLimit = append(policyTotalCuLimit, policy.GetTotalCuLimit())
			}
		}
	}

	effectiveTotalCuOfProject := lavaslices.Min(policyTotalCuLimit)
	cuLeftInProject := effectiveTotalCuOfProject - cuUsedInProject

	effectiveEpochCuOfProject := lavaslices.Min(policyEpochCuLimit)

	slice := []uint64{effectiveEpochCuOfProject, cuLeftInProject, cuLeftInSubscription}
	return lavaslices.Min(slice), effectiveTotalCuOfProject
}

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, chainID string, providerAddress sdk.AccAddress, reqEpoch uint64, project projectstypes.Project) (isValidPairing bool, allowedCU uint64, pairedProviders []epochstoragetypes.StakeEntry, errorRet error) {
	epoch, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, reqEpoch)
	if err != nil {
		return false, allowedCU, []epochstoragetypes.StakeEntry{}, err
	}
	if epoch != reqEpoch {
		return false, allowedCU, []epochstoragetypes.StakeEntry{}, utils.LavaFormatError("requested block is not an epoch start", nil, utils.Attribute{Key: "epoch", Value: epoch}, utils.Attribute{Key: "requested", Value: reqEpoch})
	}

	validAddresses, allowedCU, ok := k.GetPairingRelayCache(ctx, project.Index, chainID, epoch)
	if !ok {
		_, err := sdk.AccAddressFromBech32(project.Subscription)
		if err != nil {
			return false, allowedCU, []epochstoragetypes.StakeEntry{}, err
		}

		strictestPolicy, cluster, err := k.GetProjectStrictestPolicy(ctx, project, chainID, reqEpoch)
		if err != nil {
			return false, allowedCU, []epochstoragetypes.StakeEntry{}, fmt.Errorf("invalid user for pairing: %s", err.Error())
		}

		validAddresses, allowedCU, _, err = k.getPairingForClient(ctx, chainID, epoch, strictestPolicy, cluster, project.Index, false, false)
		if err != nil {
			return false, allowedCU, []epochstoragetypes.StakeEntry{}, err
		}
	}

	for _, possibleAddr := range validAddresses {
		providerAccAddr, err := sdk.AccAddressFromBech32(possibleAddr.Address)
		if err != nil {
			// panic:ok: provider address saved on chain must be valid
			utils.LavaFormatPanic("critical: invalid provider address for payment", err,
				utils.Attribute{Key: "chainID", Value: chainID},
				utils.Attribute{Key: "client", Value: project.Subscription},
				utils.Attribute{Key: "provider", Value: providerAccAddr.String()},
				utils.Attribute{Key: "epochBlock", Value: strconv.FormatUint(epoch, 10)},
			)
		}

		if providerAccAddr.Equals(providerAddress) {
			if !ok {
				// if we had a cache miss, set cache
				k.SetPairingRelayCache(ctx, project.Index, chainID, epoch, validAddresses, allowedCU)
			}
			return true, allowedCU, validAddresses, nil
		}
	}

	return false, allowedCU, []epochstoragetypes.StakeEntry{}, nil
}
