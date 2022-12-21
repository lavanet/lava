package keeper

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	tendermintcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/rpc/core"
)

const INVALID_INDEX = -2

func (k Keeper) VerifyPairingData(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress, block uint64) (clientStakeEntryRet *epochstoragetypes.StakeEntry, errorRet error) {
	logger := k.Logger(ctx)
	// TODO: add support for spec changes
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

	// we get the user stakeEntries at the time of check. for unstaking users, we make sure users can't unstake sooner than blocksToSave so we can charge them if the pairing is valid
	userStakedEntries, found := k.epochStorageKeeper.GetEpochStakeEntries(ctx, requestedEpochStart, epochstoragetypes.ClientKey, chainID)
	if !found {
		return nil, utils.LavaError(ctx, logger, "client_entries_pairing", map[string]string{"chainID": chainID, "query Epoch": strconv.FormatUint(requestedEpochStart, 10), "query block": strconv.FormatUint(block, 10), "current epoch": strconv.FormatUint(currentEpochStart, 10)}, "no EpochStakeEntries entries at all for this spec")
	}
	for i, clientStakeEntry := range userStakedEntries {
		clientAddr, err := sdk.AccAddressFromBech32(clientStakeEntry.Address)
		if err != nil {
			panic(fmt.Sprintf("invalid user address saved in keeper %s, err: %s", clientStakeEntry.Address, err))
		}
		if clientAddr.Equals(clientAddress) {
			if clientStakeEntry.Deadline > block {
				// client is not valid for new pairings yet, or was jailed
				return nil, fmt.Errorf("found staked user %+v, but his deadline %d, was bigger than checked block: %d", clientStakeEntry, clientStakeEntry.Deadline, block)
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

// function used to get a new pairing from relayer and client
// first argument has all metadata, second argument is only the addresses
func (k Keeper) GetPairingForClient(ctx sdk.Context, chainID string, clientAddress sdk.AccAddress) (providers []epochstoragetypes.StakeEntry, errorRet error) {
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	clientStakeEntry, err := k.VerifyPairingData(ctx, chainID, clientAddress, currentEpoch)
	if err != nil {
		// user is not valid for pairing
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
		// could not read epoch start for block
		return false, nil, INVALID_INDEX, fmt.Errorf("epoch start requested: %s", err)
	}
	// TODO: this is by spec ID but spec might change, and we validate a past spec, and all our stuff are by specName, this can be a problem
	userStake, err = k.VerifyPairingData(ctx, chainID, clientAddress, epochStart)
	if err != nil {
		// user is not valid for pairing
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

	// create a list of valid providers (deadline reached)
	for _, stakeEntry := range providers {
		if stakeEntry.Deadline > uint64(ctx.BlockHeight()) {
			// provider deadline wasn't reached yet
			continue
		}
		geolocationSupported := stakeEntry.Geolocation & geolocation
		if geolocationSupported == 0 {
			// no match in geolocation bitmap
			continue
		}
		validProviders = append(validProviders, stakeEntry)
	}

	// calculates a hash and randomly chooses the providers
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

// this function randomly chooses count providers by weight
func (k Keeper) returnSubsetOfProvidersByStake(ctx sdk.Context, clientAddress sdk.AccAddress, providersMaps []epochstoragetypes.StakeEntry, count uint64, block uint64, chainID string) (returnedProviders []epochstoragetypes.StakeEntry) {
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

const (
	EpochBlocksDivider = 5 // determines how many blocks from the previous epoch will be included in the average block time calculation
	MinSampleStep      = 1 // the minimal sample step when calculating the average block time
)

// Function to calculate how much time (in seconds) is left until the next epoch
func (k Keeper) calculateNextEpochTime(ctx sdk.Context) (uint64, error) {
	// Get current epoch
	currentEpoch := k.epochStorageKeeper.GetEpochStart(ctx)

	// Calculate the average block time (i.e., how much time it takes to create a new block, in average)
	averageBlockTime, err := k.calculateAverageBlockTime(ctx, currentEpoch)
	if err != nil {
		return 0, fmt.Errorf("could not calculate average block time, err: %s", err)
	}

	// Get the next epoch
	nextEpochStart, err := k.epochStorageKeeper.GetNextEpoch(ctx, currentEpoch)
	if err != nil {
		return 0, fmt.Errorf("could not get next epoch start, err: %s", err)
	}

	// Get epochBlocksOverlap
	overlapBlocks := k.EpochBlocksOverlap(ctx)

	// Get number of blocks from the current block to the next epoch
	blocksUntilNewEpoch := nextEpochStart + overlapBlocks - uint64(ctx.BlockHeight())

	// Calculate the time left for the next pairing in seconds (blocks left * avg block time)
	timeLeftToNextEpoch := blocksUntilNewEpoch * uint64(averageBlockTime)

	return timeLeftToNextEpoch, nil
}

// Function to calculate the average block time (i.e., how much time it takes to create a new block, in average)
func (k Keeper) calculateAverageBlockTime(ctx sdk.Context, epoch uint64) (float64, error) {
	// Get epochBlocks (the number of blocks in an epoch)
	epochBlocks, err := k.epochStorageKeeper.EpochBlocks(ctx, epoch)
	if err != nil {
		return 0, fmt.Errorf("could not get epochBlocks, err: %s", err)
	}

	// Define sample step. Determines which timestamps will be taken in the average block time calculation.
	//    if epochBlock < EpochBlocksDivider -> sampleStep = MinSampleStep.
	//    else sampleStep will be epochBlocks/EpochBlocksDivider
	if MinSampleStep > epochBlocks {
		return 0, fmt.Errorf("invalid MinSampleStep value since it's larger than epochBlocks. MinSampleStep: %v, epochBlocks: %v", MinSampleStep, epochBlocks)
	}
	sampleStepInt64 := int64(MinSampleStep)
	if epochBlocks > EpochBlocksDivider {
		sampleStepInt64 = int64(epochBlocks) / EpochBlocksDivider
	}
	sampleStep := uint64(sampleStepInt64)

	// Get a list of the previous epoch's blocks timestamp and height
	prevEpochTimestampAndHeightList, err := k.getPreviousEpochTimestampsByHeight(ctx, epoch, sampleStep)
	if err != nil {
		return 0, fmt.Errorf("couldn't get prevEpochTimestampAndHeightList. err: %v", err)
	}

	// Calculate the average block time from prevEpochTimestampAndHeightList
	averageBlockTime, err := calculateAverageBlockTimeFromList(ctx, prevEpochTimestampAndHeightList, sampleStep)
	if err != nil {
		return 0, fmt.Errorf("couldn't get calculate average block time. err: %v", err)
	}

	return averageBlockTime, nil
}

type blockHeightAndTime struct {
	blockHeight uint64
	blockTime   time.Time
}

// Function to get a list of the timestamps of the blocks in the previous epoch of the input (so it'll be deterministic)
func (k Keeper) getPreviousEpochTimestampsByHeight(ctx sdk.Context, epoch uint64, sampleStep uint64) ([]blockHeightAndTime, error) {
	// Check if a previous epoch exists (on the first epoch or after a chain fork, there is no previous epoch. In this case, return an error since this function only calculates the average block time on epochs that fully passed).
	prevEpoch, err := k.epochStorageKeeper.GetPreviousEpochStartForBlock(ctx, epoch)
	if err != nil {
		return nil, fmt.Errorf("can't get previous epoch for current epoch. epoch: %d", epoch)
	}

	// Get previous epoch timestamps, in sampleStep steps
	prevEpochTimestampAndHeightList := []blockHeightAndTime{}
	for block := prevEpoch; block < epoch; block += sampleStep {
		// Get current block's height and timestamp
		blockInt64 := int64(block)
		blockCore, err := core.Block(nil, &blockInt64)
		if err != nil {
			return nil, fmt.Errorf("could not get current block header, err: %s", err)
		}
		currentBlockTimestamp := blockCore.Block.Header.Time.UTC()
		currentBlockHeight := uint64(blockCore.Block.Height)
		blockHeightAndTimeStruct := blockHeightAndTime{blockHeight: currentBlockHeight, blockTime: currentBlockTimestamp}

		// Append the timestamp to the timestamp list
		prevEpochTimestampAndHeightList = append(prevEpochTimestampAndHeightList, blockHeightAndTimeStruct)
	}

	return prevEpochTimestampAndHeightList, nil
}

func calculateAverageBlockTimeFromList(ctx sdk.Context, blockHeightAndTimeList []blockHeightAndTime, sampleStep uint64) (float64, error) {
	if blockHeightAndTimeList == nil {
		return 0, fmt.Errorf("empty blockHeightAndTimeList")
	}

	averageBlockTime := math.MaxFloat64
	for i := 1; i < len(blockHeightAndTimeList); i++ {
		// Calculate time difference (the average block time will be the minimum difference)
		currentAverageBlockTime := blockHeightAndTimeList[i].blockTime.Sub(blockHeightAndTimeList[i-1].blockTime).Seconds() / float64(sampleStep)
		if currentAverageBlockTime <= 0 {
			return 0, fmt.Errorf("calculated average block time is less than or equal to zero. block %v timestamp: %s, block %v timestamp: %s", blockHeightAndTimeList[i].blockHeight, blockHeightAndTimeList[i].blockTime.String(), blockHeightAndTimeList[i-1].blockHeight, blockHeightAndTimeList[i-1].blockTime.String())
		}
		if averageBlockTime > currentAverageBlockTime {
			averageBlockTime = currentAverageBlockTime
		}
	}

	return averageBlockTime, nil
}

func (k Keeper) GetEpochBlockDivider() uint64 {
	return EpochBlocksDivider
}

func (k Keeper) GetMinSampleTime() uint64 {
	return MinSampleStep
}
