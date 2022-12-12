package keeper

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/rpc/core"
)

const INVALID_INDEX = -2

func (k Keeper) VerifyPairingData(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, block uint64) (clientStakeEntryRet *epochstoragetypes.StakeEntry, errorRet error) {
	logger := k.Logger(ctx)
	//TODO: add support for spec changes
	foundAndActive, _ := k.specKeeper.IsSpecFoundAndActive(ctx, chainID)
	if !foundAndActive {
		return nil, fmt.Errorf("spec not found and active for chainID given: %s", chainID)
	}
	earliestSavedEpoch := k.epochStorageKeeper.GetEarliestEpochStart(ctx)
	if block < earliestSavedEpoch {
		return nil, fmt.Errorf("block %d is earlier than earliest saved block %d", block, earliestSavedEpoch)
	}

	requestedEpochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, block)
	if err != nil {
		return nil, err
	}
	currentEpochStart := k.epochStorageKeeper.GetEpochStart(ctx)

	if requestedEpochStart > currentEpochStart {
		return nil, utils.LavaError(ctx, logger, "verify_pairing_block_sync", map[string]string{"requested block": strconv.FormatUint(block, 10), "requested epoch": strconv.FormatUint(requestedEpochStart, 10), "current epoch": strconv.FormatUint(currentEpochStart, 10)}, "VerifyPairing requested epoch is too new")
	}

	blocksToSave, err := k.epochStorageKeeper.BlocksToSave(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	if requestedEpochStart+blocksToSave < currentEpochStart {
		return nil, fmt.Errorf("requestedEpochStart %d is earlier current epoch %d by more than BlocksToSave %d", requestedEpochStart, currentEpochStart, blocksToSave)
	}
	verifiedUser := false

	//we get the user stakeEntries at the time of check. for unstaking users, we make sure users can't unstake sooner than blocksToSave so we can charge them if the pairing is valid
	userStakedEntries, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, requestedEpochStart, epochstoragetypes.ClientKey, chainID)
	if !found {
		return nil, utils.LavaError(ctx, logger, "client_entries_pairing", map[string]string{"chainID": chainID, "query Epoch": strconv.FormatUint(requestedEpochStart, 10), "query block": strconv.FormatUint(block, 10), "current epoch": strconv.FormatUint(currentEpochStart, 10)}, "no EpochStakeEntries entries at all for this spec")
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

	clientStakeEntry, err := k.VerifyPairingData(ctx, chainID, clientAddress, currentEpoch)
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

func (k Keeper) ValidatePairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, providerAddress sdk.AccAddress, block uint64) (isValidPairing bool, userStake *epochstoragetypes.StakeEntry, foundIndex int, errorRet error) {

	epochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, block)
	if err != nil {
		//could not read epoch start for block
		return false, nil, INVALID_INDEX, fmt.Errorf("epoch start requested: %s", err)
	}
	//TODO: this is by spec ID but spec might change, and we validate a past spec, and all our stuff are by specName, this can be a problem
	userStake, err = k.VerifyPairingData(ctx, chainID, clientAddress, epochStart)
	if err != nil {
		//user is not valid for pairing
		return false, nil, INVALID_INDEX, fmt.Errorf("invalid user for pairing: %s", err)
	}

	providerStakeEntries, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, epochStart, epochstoragetypes.ProviderKey, chainID)
	if !found {
		return false, nil, INVALID_INDEX, fmt.Errorf("could not get provider epoch stake entries for: %d, %s", epochStart, chainID)
	}

	_, validAddresses, errorRet := k.calculatePairingForClient(ctx, providerStakeEntries, clientAddress, epochStart, chainID, userStake.Geolocation)
	if errorRet != nil {
		return false, nil, INVALID_INDEX, errorRet
	}
	for idx, possibleAddr := range validAddresses {
		if possibleAddr.Equals(providerAddress) {
			return true, userStake, idx, nil
		}
	}
	return false, userStake, INVALID_INDEX, nil
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
	servicersToPairCount, err := k.ServicersToPairCount(ctx, epochStartBlock)
	if err != nil {
		return nil, nil, err
	}
	validProviders = k.returnSubsetOfProvidersByStake(ctx, clientAddress, validProviders, servicersToPairCount, epochStartBlock, chainID)

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
func (k Keeper) returnSubsetOfProvidersByStake(ctx sdk.Context, clientAddress sdk.AccAddress, providersMaps []epochstoragetypes.StakeEntry, count uint64, block uint64, chainID string) (returnedProviders []epochstoragetypes.StakeEntry) {
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
	hashData = append(hashData, chainID...)       // to make this pairing unique per chainID
	hashData = append(hashData, clientAddress...) // to make this pairing unique per consumer

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

// Define and initialize averageBlockTime and latestEpochBlockTimeCalculation
var (
	averageBlockTime                float64 = -1
	latestEpochBlockTimeCalculation uint64  = 0 // the latest epoch that an average block time calculation was performed (supposed to make the average block time calculation at most once per epoch)
)

func (k Keeper) calculateNextEpochTime(ctx sdk.Context) (uint64, error) {

	// Get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// In case of a chain fork, the first epoch could be non-zero. So, if the previous epoch getter fails, we assign averageBlockTime = -1 so it'll definitely enter the next if block and calculate the average block time
	_, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		averageBlockTime = -1
	}

	// Check when the last average block time calculation occured. If we're not in the same epoch as latestEpochBlockTimeCalculation, re-calculate the time
	if currentEpoch != latestEpochBlockTimeCalculation || averageBlockTime == -1 {
		err := k.calculateAverageBlockTime(ctx)
		if err != nil {
			return 0, fmt.Errorf("could not calculate average block time, err: %s", err)
		}
		latestEpochBlockTimeCalculation = currentEpoch
	}

	// Get the next epoch from the present reference
	nextEpochStart, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return 0, fmt.Errorf("could not get next epoch start, err: %s", err)
	}

	// Get the defined as overlap blocks
	overlapBlocks := k.EpochBlocksOverlap(ctx)

	// Get number of blocks from the current block to the next epoch
	blocksUntilNewEpoch := nextEpochStart + overlapBlocks - uint64(ctx.BlockHeight())

	// Calculate the time left for the next pairing in seconds (blocks left * avg block time)
	timeLeftToNextEpoch := blocksUntilNewEpoch * uint64(averageBlockTime)

	return timeLeftToNextEpoch, nil
}

func (k Keeper) calculateAverageBlockTime(ctx sdk.Context) (err error) {

	// Get the past reference block for the block time calculation
	prevEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return fmt.Errorf("could not get previous epoch start, err: %s", err)
	}

	// Get the present reference for the block time calculation
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Get the number of blocks from the past reference to the present reference
	if currentEpoch < prevEpoch {
		return fmt.Errorf("previous reference start block height is larger than the present reference start block height")
	}
	epochBlocks := currentEpoch - prevEpoch

	// Define sample step. Determines which timestamps will be taken in the calculation (if epochBlock < 5 -> sampleStep=1)
	sampleStep := uint64(0)
	if epochBlocks < 5 {
		sampleStep = uint64(1)
	} else {
		sampleStep = uint64(float64(epochBlocks) * 0.2)
	}

	// Get the timestamps of all the blocks between prevEpoch and currentEpoch. Then, calculate the differences between adjacent blocks.
	blockTimesList := make([]float64, int(math.Min(float64(epochBlocks), float64(5))))
	loopCounter := 0
	for block := prevEpoch; block < currentEpoch; block = block + sampleStep {

		// core.Block can't get block 0 (results in error from Tendermint code)
		if block == 0 {
			continue
		}

		// Get current block timestamp
		blockInt64 := int64(block)
		currentBlock, err := core.Block(nil, &blockInt64)
		if err != nil {
			return fmt.Errorf("could not get current block header, err: %s", err)
		}
		currentBlockTimestamp := currentBlock.Block.Header.Time.UTC()

		// Get next block timestamp
		nextBlockIndex := blockInt64 + 1
		nextBlock, err := core.Block(nil, &nextBlockIndex)
		if err != nil {
			return fmt.Errorf("could not get next block header, err: %s", err)
		}
		nextBlockTimestamp := nextBlock.Block.Header.Time.UTC()

		// Calculte difference and append to list
		timeDifference := nextBlockTimestamp.Sub(currentBlockTimestamp).Seconds()
		blockTimesList[loopCounter] = timeDifference

		loopCounter++
	}

	// Calculate average block time in seconds (ignore blockTime=0 because it's not possible)
	minBlockTime := blockTimesList[0]
	for _, blockTime := range blockTimesList {
		if blockTime < minBlockTime && blockTime != 0 {
			minBlockTime = blockTime
		}
	}

	// Make sure the time is a non-zero positive
	averageBlockTime = minBlockTime
	if averageBlockTime <= 0 {
		return fmt.Errorf("block creation time is not a non-zero positive number")
	}

	return nil
}
