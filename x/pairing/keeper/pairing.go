package keeper

import (
	"fmt"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/rpc/core"
)

const INVALID_INDEX = -2

func (k Keeper) verifyPairingData(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, isNew bool, block uint64) (clientStakeEntryRet *epochstoragetypes.StakeEntry, errorRet error) {
	logger := k.Logger(ctx)
	//TODO: add support for spec changes
	foundAndActive, _ := k.specKeeper.IsSpecFoundAndActive(ctx, chainID)
	if !foundAndActive {
		return nil, fmt.Errorf("spec not found and active for chainID given: %s", chainID)
	}
	earliestSavedEpoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
	if earliestSavedEpoch > block {
		return nil, fmt.Errorf("block %d is earlier than earliest saved block %d", block, earliestSavedEpoch)
	}
	verifiedUser := false

	//we get the user stakeEntries at the time of check. for unstaking users, we make sure users can't unstake sooner than blocksToSave so we can charge them if the pairing is valid
	userStakedEntries, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, block, epochstoragetypes.ClientKey, chainID)
	if !found {
		return nil, utils.LavaError(ctx, logger, "client_entries_pairing", map[string]string{"chainID": chainID, "block": strconv.FormatUint(block, 10)}, "no user entries for spec")
	}
	for _, clientStakeEntry := range userStakedEntries {
		clientAddr, err := sdk.AccAddressFromBech32(clientStakeEntry.Address)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved in keeper %s, err: %s", clientStakeEntry.Address, err))
		}
		if clientAddr.Equals(clientAddress) {
			if clientStakeEntry.Deadline > block {
				//client is not valid for new pairings yet, or was jailed
				return nil, fmt.Errorf("found staked user %+v, but his deadline %d, was bigger than checked block: %d", clientStakeEntry, clientStakeEntry.Deadline, block)
			}
			verifiedUser = true
			clientStakeEntryRet = &clientStakeEntry
			break
		}
	}
	if !verifiedUser {
		return nil, fmt.Errorf("client: %s isn't staked for spec %s at block %d", clientAddress, chainID, block)
	}
	return clientStakeEntryRet, nil
}

//function used to get a new pairing from relayer and client
//first argument has all metadata, second argument is only the addresses
func (k Keeper) GetPairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress) (providers []epochstoragetypes.StakeEntry, errorRet error) {
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)
	clientStakeEntry, err := k.verifyPairingData(ctx, chainID, clientAddress, true, currentEpoch)
	if err != nil {
		//user is not valid for pairing
		return nil, fmt.Errorf("invalid user for pairing: %s", err)
	}

	possibleProviders, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, currentEpoch, epochstoragetypes.ProviderKey, chainID)
	if !found {
		return nil, fmt.Errorf("did not find providers for pairing: epoch:%d, chainID: %s", currentEpoch, chainID)
	}
	providers, _, errorRet = k.calculatePairingForClient(ctx, possibleProviders, clientAddress, currentEpoch, chainID, clientStakeEntry.Geolocation)
	return
}

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, providerAddress sdk.AccAddress, block uint64) (isValidPairing bool, isOverlap bool, userStake *epochstoragetypes.StakeEntry, foundIndex int, errorRet error) {

	epochStart, blockInEpoch := k.epochStorageKeeper.GetEpochStartForBlock(ctx, block)
	//TODO: this is by spec ID but spec might change, and we validate a past spec, and all our stuff are by specName, this can be a problem
	userStake, err := k.verifyPairingData(ctx, chainID, clientAddress, false, epochStart)
	if err != nil {
		//user is not valid for pairing
		return false, false, nil, INVALID_INDEX, fmt.Errorf("invalid user for pairing: %s", err)
	}

	providerStakeEntries, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epochStart, epochstoragetypes.ProviderKey, chainID)
	if !found {
		return false, false, nil, INVALID_INDEX, fmt.Errorf("could not get provider epoch stake entries for: %d, %s", epochStart, chainID)
	}

	_, validAddresses, errorRet := k.calculatePairingForClient(ctx, providerStakeEntries, clientAddress, epochStart, chainID, userStake.Geolocation)
	if errorRet != nil {
		return false, false, nil, INVALID_INDEX, errorRet
	}
	for idx, possibleAddr := range validAddresses {
		if possibleAddr.Equals(providerAddress) {
			return true, false, userStake, idx, nil
		}
	}
	//Support overlap
	//if overlap blocks is X then this is an overlap block if the residue, i.e blockInEpoch, is X-1 or lower
	if blockInEpoch < k.EpochBlocksOverlap(ctx) {
		//this is a block that can have overlap
		previousEpochBlock := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, block)
		previousProviderStakeEntries, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, previousEpochBlock, epochstoragetypes.ProviderKey, chainID)
		if !found {
			return false, false, nil, INVALID_INDEX, fmt.Errorf("could not get previous provider epoch stake entries for: %d previous: %d, %s", block, previousEpochBlock, chainID)
		}
		_, validAddressesOverlap, errorRet := k.calculatePairingForClient(ctx, previousProviderStakeEntries, clientAddress, previousEpochBlock, chainID, userStake.Geolocation)
		if errorRet != nil {
			return false, false, nil, INVALID_INDEX, errorRet
		}
		//check overlap addresses from previous session
		for idx, possibleAddr := range validAddressesOverlap {
			if possibleAddr.Equals(providerAddress) {
				return true, true, userStake, idx, nil
			}
		}
	}
	return false, false, userStake, INVALID_INDEX, nil
}

func (k Keeper) calculatePairingForClient(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, clientAddress sdk.AccAddress, epochStartBlock uint64, chainID string, geolocation uint64) (validProviders []epochstoragetypes.StakeEntry, addrList []sdk.AccAddress, err error) {
	if epochStartBlock > uint64(ctx.BlockHeight()) {
		k.Logger(ctx).Error("\ninvalid session start\n")
		panic(fmt.Sprintf("invalid session start saved in keeper %d, current block was %d", epochStartBlock, uint64(ctx.BlockHeight())))
	}

	//create a list of valid providers (deadline reached)
	for _, stakeEntry := range providers {
		if stakeEntry.Deadline > uint64(ctx.BlockHeight()) {
			//provider deadline wasn't reached yet
			continue
		}
		geolocationSupported := stakeEntry.Geolocation & geolocation
		if geolocationSupported == 0 {
			//no match in geolocation bitmap
			continue
		}
		validProviders = append(validProviders, stakeEntry)
	}

	//calculates a hash and randomly chooses the providers
	validProviders = k.returnSubsetOfProvidersByStake(ctx, validProviders, k.ServicersToPairCount(ctx), epochStartBlock, chainID)

	for _, stakeEntry := range validProviders {
		providerAddress := stakeEntry.Address
		providerAccAddr, err := sdk.AccAddressFromBech32(providerAddress)
		if err != nil {
			panic(fmt.Sprintf("invalid provider address saved in keeper %s, err: %s", providerAddress, err))
		}
		addrList = append(addrList, providerAccAddr)
	}
	return validProviders, addrList, nil
}

//this function randomly chooses count providers by weight
func (k Keeper) returnSubsetOfProvidersByStake(ctx sdk.Context, providersMaps []epochstoragetypes.StakeEntry, count uint64, block uint64, chainID string) (returnedProviders []epochstoragetypes.StakeEntry) {
	var stakeSum sdk.Coin = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(0))
	hashData := make([]byte, 0)
	for _, stakedProvider := range providersMaps {
		stakeSum = stakeSum.Add(stakedProvider.Stake)
	}
	if stakeSum.IsZero() {
		//list is empty
		return
	}

	//add the session start block hash to the function to make it as unpredictable as we can
	block_height := int64(block)
	epochStartBlock, err := core.Block(nil, &block_height)
	if err != nil {
		k.Logger(ctx).Error("Failed To Get block from tendermint core")
	}
	sessionBlockHash := epochStartBlock.Block.Hash()
	hashData = append(hashData, sessionBlockHash...)
	hashData = append(hashData, chainID...) // to make this pairing unique per chainID

	indexToSkip := make(map[int]bool) // a trick to create a unique set in golang
	for it := 0; it < int(count); it++ {
		hash := tendermintcrypto.Sha256(hashData) // TODO: we use cheaper algo for speed
		bigIntNum := new(big.Int).SetBytes(hash)
		hashAsNumber := sdk.NewIntFromBigInt(bigIntNum)
		modRes := hashAsNumber.Mod(stakeSum.Amount)

		var newStakeSum = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(0))
		//we loop the servicers list form the end because the list is sorted, biggest is last,
		// and statistically this will have less iterations

		for idx := len(providersMaps) - 1; idx >= 0; idx-- {
			stakedProvider := providersMaps[idx]
			if indexToSkip[idx] {
				//this is an index we added
				continue
			}
			newStakeSum = newStakeSum.Add(stakedProvider.Stake)
			if modRes.LT(newStakeSum.Amount) {
				//we hit our chosen provider
				returnedProviders = append(returnedProviders, stakedProvider)
				stakeSum = stakeSum.Sub(stakedProvider.Stake) //we remove this provider from the random pool, so the sum is lower now
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
