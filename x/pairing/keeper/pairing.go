package keeper

import (
	"fmt"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/rpc/core"
)

func (k Keeper) verifyPairingData(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, isNew bool, block uint64) (clientStakeEntryRet *epochstoragetypes.StakeEntry, errorRet error) {
	logger := k.Logger(ctx)
	//TODO: add support for spec changes
	foundAndActive, _, _ := k.specKeeper.IsSpecFoundAndActive(ctx, chainID)
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
				return nil, fmt.Errorf("found staked user %s, but his deadline %d, was bigger than checked block: %d", clientStakeEntry, clientStakeEntry.Deadline, block)
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
	_, err := k.verifyPairingData(ctx, chainID, clientAddress, true, currentEpoch)
	if err != nil {
		//user is not valid for pairing
		return nil, fmt.Errorf("invalid user for pairing: %s", err)
	}

	possibleProviders, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, currentEpoch, epochstoragetypes.ProviderKey, chainID)
	if !found {
		return nil, fmt.Errorf("did not find providers for pairing: epoch:%d, chainID: %s", currentEpoch, chainID)
	}
	providers, _, errorRet = k.calculatePairingForClient(ctx, possibleProviders, clientAddress, currentEpoch)
	return
}

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, servicerAddress sdk.AccAddress, block uint64) (isValidPairing bool, isOverlap bool, userStake *epochstoragetypes.StakeEntry, errorRet error) {
	//TODO: this is by spec ID but spec might change, and we validate a past spec, and all our stuff are by specName, this can be a problem
	userStake, err := k.verifyPairingData(ctx, chainID, clientAddress, false, block)
	if err != nil {
		//user is not valid for pairing
		return false, false, nil, fmt.Errorf("invalid user for pairing: %s", err)
	}

	stakeStorage, previousOverlappingStakeStorage, err := k.GetSpecStakeStorageInSessionStorageForSpec(ctx, block, spec.Name)
	if err != nil {
		return false, false, nil, err
	}
	sessionStart, overlappingBlock, err := k.GetSessionStartForBlock(ctx, block)
	if err != nil {
		return false, false, nil, err
	}
	_, validAddresses, errorRet := k.calculatePairingForClient(ctx, stakeStorage, clientAddress, *sessionStart)
	if errorRet != nil {
		return false, false, nil, errorRet
	}
	for _, possibleAddr := range validAddresses {
		if possibleAddr.Equals(servicerAddress) {
			return true, false, userStake, nil
		}
	}
	if previousOverlappingStakeStorage != nil {
		if overlappingBlock == nil {
			panic("no overlapping block but has overlapping stakeStorage")
		}
		_, validAddressesOverlap, errorRet := k.calculatePairingForClient(ctx, previousOverlappingStakeStorage, clientAddress, *overlappingBlock)
		if errorRet != nil {
			return false, false, nil, errorRet
		}
		//check overlap addresses from previous session
		for _, possibleAddr := range validAddressesOverlap {
			if possibleAddr.Equals(servicerAddress) {
				return true, true, userStake, nil
			}
		}
	}
	return false, false, userStake, nil
}

func (k Keeper) calculatePairingForClient(ctx sdk.Context, providers []epochstoragetypes.StakeEntry, clientAddress sdk.AccAddress, sessionStartBlock types.BlockNum) (validServicers []types.StakeMap, addrList []sdk.AccAddress, err error) {
	if sessionStartBlock.Num > uint64(ctx.BlockHeight()) {
		k.Logger(ctx).Error("\ninvalid session start\n")
		panic(fmt.Sprintf("invalid session start saved in keeper %d, current block was %d", sessionStartBlock.Num, uint64(ctx.BlockHeight())))
	}
	stakedServicers := stakedStorage.Staked
	//create a list of valid servicers (deadline reached)
	for _, stakeMap := range stakedServicers {
		if stakeMap.Deadline.Num > uint64(ctx.BlockHeight()) {
			//servicer deadline wasn't reached yet
			continue
		}
		validServicers = append(validServicers, stakeMap)
	}

	//calculates a hash and randomly chooses the servicers
	validServicers = k.returnSubsetOfServicersByStake(ctx, validServicers, k.ServicersToPairCount(ctx), sessionStartBlock.Num)

	for _, stakeMap := range validServicers {
		servicerAddress := stakeMap.Index
		servicerAccAddr, err := sdk.AccAddressFromBech32(servicerAddress)
		if err != nil {
			panic(fmt.Sprintf("invalid servicer address saved in keeper %s, err: %s", servicerAddress, err))
		}
		addrList = append(addrList, servicerAccAddr)
	}
	return validServicers, addrList, nil
}

//this function randomly chooses count servicers by weight
func (k Keeper) returnSubsetOfServicersByStake(ctx sdk.Context, servicersMaps []types.StakeMap, count uint64, block uint64) (returnedServicers []types.StakeMap) {
	var stakeSum uint64 = 0
	hashData := make([]byte, 0)
	for _, stakedServicer := range servicersMaps {
		stakeSum += stakedServicer.Stake.Amount.Uint64()
	}
	if stakeSum == 0 {
		//list is empty
		return
	}

	//add the session start block hash to the function to make it as unpredictable as we can
	block_height := int64(block)
	sessionStartBlock, err := core.Block(nil, &block_height)
	if err != nil {
		k.Logger(ctx).Error("Failed To Get block from tendermint core")
	}
	sessionBlockHash := sessionStartBlock.Block.Hash()
	// k.Logger(ctx).Error(fmt.Sprintf("Block Hash!!!: %s", sessionBlockHash))
	hashData = append(hashData, sessionBlockHash...)

	//TODO: sort servicers by stake (done only once), so we statisticly go over the list less
	indexToSkip := make(map[int]bool) // a trick to create a unique set in golang
	for it := 0; it < int(count); it++ {
		hash := tendermintcrypto.Sha256(hashData) // TODO: we use cheaper algo for speed
		bigIntNum := new(big.Int).SetBytes(hash)
		hashAsNumber := sdk.NewIntFromBigInt(bigIntNum)
		modRes := hashAsNumber.ModRaw(int64(stakeSum)).Uint64()
		var newStakeSum uint64 = 0
		for idx, stakedServicer := range servicersMaps {
			if indexToSkip[idx] {
				//this is an index we added
				continue
			}
			newStakeSum += stakedServicer.Stake.Amount.Uint64()
			if modRes < newStakeSum {
				//we hit our chosen servicer
				returnedServicers = append(returnedServicers, stakedServicer)
				stakeSum -= stakedServicer.Stake.Amount.Uint64() //we remove this servicer from the random pool, so the sum is lower now
				indexToSkip[idx] = true
				break
			}
		}
		if uint64(len(returnedServicers)) >= count {
			return returnedServicers
		}
		hashData = append(hashData, []byte{uint8(it)}...)
	}
	return returnedServicers
}
