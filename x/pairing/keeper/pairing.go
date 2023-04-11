package keeper

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	projectstypes "github.com/lavanet/lava/x/projects/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
)

const INVALID_INDEX = -2

func (k Keeper) VerifyPairingData(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, block uint64) (epoch uint64, errorRet error) {
	logger := k.Logger(ctx)
	// TODO: add support for spec changes
	foundAndActive, _ := k.specKeeper.IsSpecFoundAndActive(ctx, chainID)
	if !foundAndActive {
		return 0, fmt.Errorf("spec not found and active for chainID given: %s", chainID)
	}
	earliestSavedEpoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
	if block < earliestSavedEpoch {
		return 0, fmt.Errorf("block %d is earlier than earliest saved block %d", block, earliestSavedEpoch)
	}

	requestedEpochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, block)
	if err != nil {
		return 0, err
	}
	currentEpochStart := k.epochStorageKeeper.GetEpochStart(ctx)

	if requestedEpochStart > currentEpochStart {
		return 0, utils.LavaError(ctx, logger, "verify_pairing_block_sync", map[string]string{"requested block": strconv.FormatUint(block, 10), "requested epoch": strconv.FormatUint(requestedEpochStart, 10), "current epoch": strconv.FormatUint(currentEpochStart, 10)}, "VerifyPairing requested epoch is too new")
	}

	blocksToSave, err := k.epochStorageKeeper.BlocksToSave(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return 0, err
	}

	if requestedEpochStart+blocksToSave < currentEpochStart {
		return 0, fmt.Errorf("requestedEpochStart %d is earlier current epoch %d by more than BlocksToSave %d", requestedEpochStart, currentEpochStart, blocksToSave)
	}
	return requestedEpochStart, nil
}

func (k Keeper) VerifyClientStake(ctx sdk.Context, chainID string, clientAddress sdk.Address, block uint64, epoch uint64) (clientStakeEntryRet *epochstoragetypes.StakeEntry, errorRet error) {
	logger := ctx.Logger()
	verifiedUser := false

	// we get the user stakeEntries at the time of check. for unstaking users, we make sure users can't unstake sooner than blocksToSave so we can charge them if the pairing is valid
	userStakedEntries, found, _ := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, epochstoragetypes.ClientKey, chainID)
	if !found {
		return nil, utils.LavaError(ctx, logger, "client_entries_pairing", map[string]string{"chainID": chainID, "query Epoch": strconv.FormatUint(epoch, 10), "query block": strconv.FormatUint(block, 10), "current epoch": strconv.FormatUint(epoch, 10)}, "no EpochStakeEntries entries at all for this spec")
	}
	for i, clientStakeEntry := range userStakedEntries {
		clientAddr, err := sdk.AccAddressFromBech32(clientStakeEntry.Address)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved in keeper %s, err: %s", clientStakeEntry.Address, err))
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

func (k Keeper) GetProjectData(ctx sdk.Context, developerKey sdk.AccAddress, chainID string, blockHeight uint64) (proj projectstypes.Project, vrfpk string, errRet error) {
	project, vrfpk, err := k.projectsKeeper.GetProjectForDeveloper(ctx, developerKey.String(), blockHeight)
	if err != nil {
		return projectstypes.Project{}, "", err
	}

	if !project.Enabled {
		return projectstypes.Project{}, "", utils.LavaError(ctx, ctx.Logger(), "pairing_project_invalid", map[string]string{"project": project.Index}, "the developers project is disabled")
	}

	return project, vrfpk, nil
}

func (k Keeper) GetPairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress) (providers []epochstoragetypes.StakeEntry, errorRet error) {
	providers, _, _, _, err := k.getPairingForClient(ctx, chainID, clientAddress, uint64(ctx.BlockHeight()))
	return providers, err
}

// function used to get a new pairing from provider and client
// first argument has all metadata, second argument is only the addresses
func (k Keeper) getPairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, block uint64) (providers []epochstoragetypes.StakeEntry, vrfk string, allowedCU uint64, legacyStake bool, errorRet error) {
	var geolocation uint64
	var providersToPair uint64
	var projectToPair string

	epoch, err := k.VerifyPairingData(ctx, chainID, clientAddress, block)
	if err != nil {
		return nil, "", 0, false, fmt.Errorf("invalid pairing data: %s", err)
	}

	project, vrfpk_proj, err := k.GetProjectData(ctx, clientAddress, chainID, block)
	if err == nil {
		vrfk = vrfpk_proj
		legacyStake = false
		geolocation, providersToPair, projectToPair, allowedCU, err = k.getProjectStrictestPolicy(ctx, project, chainID)
		if err != nil {
			return nil, "", 0, false, fmt.Errorf("invalid user for pairing: %s", err.Error())
		}
	} else {
		// legacy staked client
		clientStakeEntry, err2 := k.VerifyClientStake(ctx, chainID, clientAddress, block, epoch)
		if err2 != nil {
			// user is not valid for pairing
			return nil, "", 0, false, fmt.Errorf("invalid user for pairing: 1) %s 2) %s", err.Error(), err2.Error())
		}
		geolocation = clientStakeEntry.Geolocation

		servicersToPairCount, err := k.ServicersToPairCount(ctx, block)
		if err != nil {
			return nil, "", 0, false, err
		}

		providersToPair = servicersToPairCount
		projectToPair = clientAddress.String()
		vrfk = clientStakeEntry.Vrfpk

		allowedCU, err = k.ClientMaxCUProviderForBlock(ctx, block, clientStakeEntry)
		if err != nil {
			return nil, "", 0, false, err
		}

		legacyStake = true
	}

	possibleProviders, found, epochHash := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epoch, epochstoragetypes.ProviderKey, chainID)
	if !found {
		return nil, "", 0, false, fmt.Errorf("did not find providers for pairing: epoch:%d, chainID: %s", block, chainID)
	}

	providers, err = k.calculatePairingForClient(ctx, possibleProviders, projectToPair, block, chainID, geolocation, epochHash, providersToPair)

	return providers, vrfk, allowedCU, legacyStake, err
}

func (k Keeper) getProjectStrictestPolicy(ctx sdk.Context, project projectstypes.Project, chainID string) (uint64, uint64, string, uint64, error) {
	plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, project.GetSubscription())
	if err != nil {
		return 0, 0, "", 0, err
	}

	planPolicy := plan.GetPlanPolicy()

	err = project.VerifyProject(chainID, planPolicy)
	if err != nil {
		return 0, 0, "", 0, err
	}

	// geolocation is a bitmap. common denominator can be calculated with logical AND
	geolocation := project.AdminPolicy.GeolocationProfile & project.SubscriptionPolicy.GeolocationProfile & planPolicy.GeolocationProfile

	providersToPair := minMaxProvidersToPair([]uint64{
		project.AdminPolicy.GetMaxProvidersToPair(),
		project.SubscriptionPolicy.GetMaxProvidersToPair(),
		planPolicy.GetMaxProvidersToPair(),
	})

	projectToPair := project.Index
	allowedCU := minCuLimit([]uint64{
		project.AdminPolicy.GetEpochCuLimit(),
		project.SubscriptionPolicy.GetEpochCuLimit(),
		planPolicy.GetEpochCuLimit(),
	})
	err = project.VerifyCuUsage(planPolicy)
	if err != nil {
		return 0, 0, "", 0, err
	}

	return geolocation, providersToPair, projectToPair, allowedCU, nil
}

func minMaxProvidersToPair(values []uint64) uint64 {
	min := uint64(math.MaxUint64)
	for _, v := range values {
		if v < min {
			min = v
		}
	}

	return min
}

func minCuLimit(values []uint64) uint64 {
	min := uint64(math.MaxUint64)
	for _, v := range values {
		// when the CU limits are enforced, 0 indicates unlimited CU. So it's actually larger than non-zero limitations
		if v < min && v != 0 {
			min = v
		}
	}

	// min = initial value -> couldn't find non-zero min value -> unlimited CU
	if min == math.MaxUint64 {
		return 0
	}

	return min
}

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, providerAddress sdk.AccAddress, epoch uint64) (isValidPairing bool, vrfk string, foundIndex int, allowedCU uint64, pairedProviders uint64, legacyStake bool, errorRet error) {
	epoch, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, epoch)
	if err != nil {
		return false, vrfk, INVALID_INDEX, allowedCU, 0, legacyStake, err
	}

	validAddresses, vrfk, allowedCU, legacyStake, err := k.getPairingForClient(ctx, chainID, clientAddress, epoch)
	if err != nil {
		return false, vrfk, INVALID_INDEX, allowedCU, 0, legacyStake, err
	}

	for idx, possibleAddr := range validAddresses {
		providerAccAddr, err := sdk.AccAddressFromBech32(possibleAddr.Address)
		if err != nil {
			panic(fmt.Sprintf("invalid provider address saved in keeper %s, err: %s", providerAccAddr, err))
		}

		if providerAccAddr.Equals(providerAddress) {
			return true, vrfk, idx, allowedCU, uint64(len(validAddresses)), legacyStake, nil
		}
	}

	return false, vrfk, INVALID_INDEX, allowedCU, 0, legacyStake, nil
}

func (k Keeper) calculatePairingForClient(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, developerAddress string, epochStartBlock uint64, chainID string, geolocation uint64, epochHash []byte, providersToPair uint64) (validProviders []epochstoragetypes.StakeEntry, err error) {
	if epochStartBlock > uint64(ctx.BlockHeight()) {
		k.Logger(ctx).Error("\ninvalid session start\n")
		panic(fmt.Sprintf("invalid session start saved in keeper %d, current block was %d", epochStartBlock, uint64(ctx.BlockHeight())))
	}

	spec, found := k.specKeeper.GetSpec(ctx, chainID)
	if !found {
		return nil, fmt.Errorf("spec not found or not enabled")
	}

	validProviders = k.getGeolocationProviders(ctx, providers, geolocation)

	if spec.ProvidersTypes == spectypes.Spec_dynamic {
		// calculates a hash and randomly chooses the providers

		validProviders = k.returnSubsetOfProvidersByStake(ctx, developerAddress, validProviders, providersToPair, epochStartBlock, chainID, epochHash)
	} else {
		validProviders = k.returnSubsetOfProvidersByHighestStake(ctx, validProviders, providersToPair)
	}

	return validProviders, nil
}

func (k Keeper) getGeolocationProviders(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, geolocation uint64) []epochstoragetypes.StakeEntry {
	validProviders := []epochstoragetypes.StakeEntry{}
	// create a list of valid providers (stakeAppliedBlock reached)
	for _, stakeEntry := range providers {
		if stakeEntry.StakeAppliedBlock > uint64(ctx.BlockHeight()) {
			// provider stakeAppliedBlock wasn't reached yet
			continue
		}
		geolocationSupported := stakeEntry.Geolocation & geolocation
		if geolocationSupported == 0 && geolocation != 0 {
			// no match in geolocation bitmap (geolocation = 0 --> support all regions)
			continue
		}
		validProviders = append(validProviders, stakeEntry)
	}
	return validProviders
}

// this function randomly chooses count providers by weight
func (k Keeper) returnSubsetOfProvidersByStake(ctx sdk.Context, clientAddress string, providersMaps []epochstoragetypes.StakeEntry, count uint64, block uint64, chainID string, epochHash []byte) (returnedProviders []epochstoragetypes.StakeEntry) {
	stakeSum := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(0))
	hashData := make([]byte, 0)
	for _, stakedProvider := range providersMaps {
		stakeSum = stakeSum.Add(stakedProvider.Stake)
	}
	if stakeSum.IsZero() {
		// list is empty
		return
	}

	// add the session start block hash to the function to make it as unpredictable as we can
	hashData = append(hashData, epochHash...)
	hashData = append(hashData, chainID...)       // to make this pairing unique per chainID
	hashData = append(hashData, clientAddress...) // to make this pairing unique per consumer

	indexToSkip := make(map[int]bool) // a trick to create a unique set in golang
	for it := 0; it < int(count); it++ {
		hash := tendermintcrypto.Sha256(hashData) // TODO: we use cheaper algo for speed
		bigIntNum := new(big.Int).SetBytes(hash)
		hashAsNumber := sdk.NewIntFromBigInt(bigIntNum)
		modRes := hashAsNumber.Mod(stakeSum.Amount)

		newStakeSum := sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(0))
		// we loop the servicers list form the end because the list is sorted, biggest is last,
		// and statistically this will have less iterations

		for idx := len(providersMaps) - 1; idx >= 0; idx-- {
			stakedProvider := providersMaps[idx]
			if indexToSkip[idx] {
				// this is an index we added
				continue
			}
			newStakeSum = newStakeSum.Add(stakedProvider.Stake)
			if modRes.LT(newStakeSum.Amount) {
				// we hit our chosen provider
				returnedProviders = append(returnedProviders, stakedProvider)
				stakeSum = stakeSum.Sub(stakedProvider.Stake) // we remove this provider from the random pool, so the sum is lower now
				indexToSkip[idx] = true
				break
			}
		}
		if uint64(len(returnedProviders)) >= count {
			return returnedProviders
		}
		if stakeSum.IsZero() {
			break
		}
		hashData = append(hashData, []byte{uint8(it)}...)
	}
	return returnedProviders
}

func (k Keeper) returnSubsetOfProvidersByHighestStake(ctx sdk.Context, providersEntries []epochstoragetypes.StakeEntry, count uint64) (returnedProviders []epochstoragetypes.StakeEntry) {
	if uint64(len(providersEntries)) <= count {
		return providersEntries
	} else {
		return providersEntries[0:count]
	}
}
