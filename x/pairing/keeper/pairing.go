package keeper

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/slices"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	pairingfilters "github.com/lavanet/lava/x/pairing/keeper/filters"
	pairingscores "github.com/lavanet/lava/x/pairing/keeper/scores"
	planstypes "github.com/lavanet/lava/x/plans/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func (k Keeper) VerifyPairingData(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, block uint64) (epoch uint64, providersType spectypes.Spec_ProvidersTypes, errorRet error) {
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

func (k Keeper) VerifyClientStake(ctx sdk.Context, chainID string, clientAddress sdk.Address, block uint64, epoch uint64) (clientStakeEntryRet *epochstoragetypes.StakeEntry, errorRet error) {
	verifiedUser := false

	// we get the user stakeEntries at the time of check. for unstaking users, we make sure users can't unstake sooner than blocksToSave so we can charge them if the pairing is valid
	userStakedEntries, found, _ := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, chainID)
	if !found {
		return nil, utils.LavaFormatWarning("no EpochStakeEntries entries at all for this spec", fmt.Errorf("user stake entries not found"),
			utils.Attribute{Key: "chainID", Value: chainID},
			utils.Attribute{Key: "query Epoch", Value: epoch},
			utils.Attribute{Key: "query block", Value: block},
		)
	}
	for i, clientStakeEntry := range userStakedEntries {
		clientAddr, err := sdk.AccAddressFromBech32(clientStakeEntry.Address)
		if err != nil {
			// this should not happen; to avoid panic we simply skip this one (thus
			// freeze the situation so it can be investigated and orderly resolved).
			utils.LavaFormatError("critical: invalid account address inside StakeStorage", err,
				utils.LogAttr("address", clientStakeEntry.Address),
				utils.LogAttr("chainID", clientStakeEntry.Chain),
			)
			continue
		}
		if clientAddr.Equals(clientAddress) {
			if clientStakeEntry.StakeAppliedBlock > block {
				// client is not valid for new pairings yet, or was jailed
				return nil, fmt.Errorf("found staked user %+v, but his stakeAppliedBlock %d, was bigger than checked block: %d", clientStakeEntry, clientStakeEntry.StakeAppliedBlock, block)
			}
			verifiedUser = true
			clientStakeEntryRet = &userStakedEntries[i]
			break
		}
	}
	if !verifiedUser {
		return nil, fmt.Errorf("client: %s isn't staked for spec %s at block %d", clientAddress, chainID, block)
	}
	return clientStakeEntryRet, nil
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
	providers, _, _, err := k.getPairingForClient(ctx, chainID, clientAddress, uint64(ctx.BlockHeight()))
	return providers, err
}

// function used to get a new pairing from provider and client
// first argument has all metadata, second argument is only the addresses
func (k Keeper) getPairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, block uint64) (providers []epochstoragetypes.StakeEntry, allowedCU uint64, projectID string, errorRet error) {
	var strictestPolicy planstypes.Policy

	epoch, providersType, err := k.VerifyPairingData(ctx, chainID, clientAddress, block)
	if err != nil {
		return nil, 0, "", fmt.Errorf("invalid pairing data: %s", err)
	}

	stakeEntries, found, epochHash := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, chainID)
	if !found {
		return nil, 0, "", fmt.Errorf("did not find providers for pairing: epoch:%d, chainID: %s", block, chainID)
	}

	project, err := k.GetProjectData(ctx, clientAddress, chainID, block)
	if err != nil {
		return nil, 0, "", err
	}

	strictestPolicy, err = k.GetProjectStrictestPolicy(ctx, project, chainID)
	if err != nil {
		return nil, 0, "", fmt.Errorf("invalid user for pairing: %s", err.Error())
	}

	if providersType == spectypes.Spec_static {
		return stakeEntries, strictestPolicy.EpochCuLimit, project.Index, nil
	}

	filters := pairingfilters.GetAllFilters()

	stakeEntries, err = pairingfilters.FilterProviders(ctx, filters, stakeEntries, strictestPolicy, epoch)
	if err != nil {
		return nil, 0, "", err
	}

	// create the pairing slots with assigned reqs
	slots := pairingscores.CalcSlots(strictestPolicy)
	if len(slots) >= len(stakeEntries) {
		return stakeEntries, strictestPolicy.EpochCuLimit, project.Index, nil
	}

	// group identical slots (in terms of reqs types)
	slotGroups := pairingscores.GroupSlots(slots)

	// create providerScore array with all possible providers
	providerScores := []*pairingscores.PairingScore{}
	for i := range stakeEntries {
		providerScore := pairingscores.NewPairingScore(&stakeEntries[i])
		providerScores = append(providerScores, providerScore)
	}

	// calculate score (always on the diff in score components of consecutive groups) and pick providers
	prevGroupSlot := pairingscores.NewPairingSlot() // init dummy slot to compare to
	prevGroupSlot.Reqs = map[string]pairingscores.ScoreReq{}

	for idx, group := range slotGroups {
		hashData := pairingscores.PrepareHashData(project.Index, chainID, epochHash, idx)
		diffSlot := group.Subtract(prevGroupSlot)
		err := pairingscores.CalcPairingScore(providerScores, pairingscores.GetStrategy(), diffSlot)
		if err != nil {
			return nil, 0, "", err
		}
		pickedProviders := pairingscores.PickProviders(ctx, providerScores, group.Count, hashData)
		providers = append(providers, pickedProviders...)

		prevGroupSlot = group
	}

	return providers, strictestPolicy.EpochCuLimit, project.Index, err
}

func (k Keeper) GetProjectStrictestPolicy(ctx sdk.Context, project projectstypes.Project, chainID string) (planstypes.Policy, error) {
	plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, project.GetSubscription())
	if err != nil {
		return planstypes.Policy{}, err
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
		return planstypes.Policy{}, fmt.Errorf("chain ID not allowed in all policies, or collections specified and have no intersection %#v", policies)
	}
	geolocation := k.CalculateEffectiveGeolocationFromPolicies(policies)

	providersToPair, err := k.CalculateEffectiveProvidersToPairFromPolicies(policies)
	if err != nil {
		return planstypes.Policy{}, err
	}

	sub, found := k.subscriptionKeeper.GetSubscription(ctx, project.GetSubscription())
	if !found {
		return planstypes.Policy{}, fmt.Errorf("could not find subscription with address %s", project.GetSubscription())
	}
	allowedCUEpoch, allowedCUTotal := k.CalculateEffectiveAllowedCuPerEpochFromPolicies(policies, project.GetUsedCu(), sub.GetMonthCuLeft())

	selectedProvidersMode, selectedProvidersList := k.CalculateEffectiveSelectedProviders(policies)

	strictestPolicy := planstypes.Policy{
		GeolocationProfile:    geolocation,
		MaxProvidersToPair:    providersToPair,
		SelectedProvidersMode: selectedProvidersMode,
		SelectedProviders:     selectedProvidersList,
		ChainPolicies:         []planstypes.ChainPolicy{chainPolicy},
		EpochCuLimit:          allowedCUEpoch,
		TotalCuLimit:          allowedCUTotal,
	}

	return strictestPolicy, nil
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

	effectiveMode := slices.Max(selectedProvidersModeList)
	effectiveSelectedProviders := slices.Intersection(selectedProvidersList...)

	return effectiveMode, effectiveSelectedProviders
}

func (k Keeper) CalculateEffectiveGeolocationFromPolicies(policies []*planstypes.Policy) uint64 {
	geolocation := uint64(math.MaxUint64)

	// geolocation is a bitmap. common denominator can be calculated with logical AND
	for _, policy := range policies {
		if policy != nil {
			geolocation &= policy.GetGeolocationProfile()
		}
	}

	return geolocation
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

func (k Keeper) CalculateEffectiveAllowedCuPerEpochFromPolicies(policies []*planstypes.Policy, cuUsedInProject uint64, cuLeftInSubscription uint64) (allowedCUThisEpoch uint64, allowedCUTotal uint64) {
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

	effectiveTotalCuOfProject := slices.Min(policyTotalCuLimit)
	cuLeftInProject := effectiveTotalCuOfProject - cuUsedInProject

	effectiveEpochCuOfProject := slices.Min(policyEpochCuLimit)

	slice := []uint64{effectiveEpochCuOfProject, cuLeftInProject, cuLeftInSubscription}
	return slices.Min(slice), effectiveTotalCuOfProject
}

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, providerAddress sdk.AccAddress, epoch uint64) (isValidPairing bool, allowedCU uint64, pairedProviders uint64, projectID string, errorRet error) {
	epoch, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, epoch)
	if err != nil {
		return false, allowedCU, 0, "", err
	}

	validAddresses, allowedCU, projectID, err := k.getPairingForClient(ctx, chainID, clientAddress, epoch)
	if err != nil {
		return false, allowedCU, 0, "", err
	}

	for _, possibleAddr := range validAddresses {
		providerAccAddr, err := sdk.AccAddressFromBech32(possibleAddr.Address)
		if err != nil {
			// panic:ok: provider address saved on chain must be valid
			utils.LavaFormatPanic("critical: invalid provider address for payment", err,
				utils.Attribute{Key: "chainID", Value: chainID},
				utils.Attribute{Key: "client", Value: clientAddress},
				utils.Attribute{Key: "provider", Value: providerAccAddr},
				utils.Attribute{Key: "epochBlock", Value: epoch},
			)
		}

		if providerAccAddr.Equals(providerAddress) {
			return true, allowedCU, uint64(len(validAddresses)), projectID, nil
		}
	}

	return false, allowedCU, 0, projectID, nil
}
