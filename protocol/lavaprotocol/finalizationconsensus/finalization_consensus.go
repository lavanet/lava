package finalizationconsensus

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/protocol/lavasession"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	conflicttypes "github.com/lavanet/lava/v2/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
)

type ChainBlockStatsGetter interface {
	ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData, blocksInFinalizationProof uint32)
}

type (
	BlockToHashesToAgreeingProviders map[int64]map[string]map[string]providerDataContainer // first key is block num, second key is block hash, third key is provider address
	FinalizationConsensusInf         interface {
		UpdateFinalizedHashes(
			blockDistanceForFinalizedData int64,
			consumerAddress sdk.AccAddress,
			providerAddress string,
			finalizedBlocks map[int64]string,
			relaySession *pairingtypes.RelaySession,
			reply *pairingtypes.RelayReply,
		) (finalizationConflict *conflicttypes.FinalizationConflict, err error)
		GetExpectedBlockHeight(chainParser ChainBlockStatsGetter) (expectedBlockHeight int64, numOfProviders int)
		NewEpoch(epoch uint64)
	}
)

type providerLatestBlockTimeAndFinalizedBlockContainer struct {
	LatestBlockTime      time.Time
	LatestFinalizedBlock int64
}

type FinalizationConsensus struct {
	currentEpochBlockToHashesToAgreeingProviders         BlockToHashesToAgreeingProviders
	prevEpochBlockToHashesToAgreeingProviders            BlockToHashesToAgreeingProviders
	lock                                                 sync.RWMutex
	currentEpoch                                         uint64
	prevLatestBlockByMedian                              uint64 // for caching
	SpecId                                               string
	highestRecordedBlockHeight                           int64
	currentEpochLatestProviderBlockTimeAndFinalizedBlock map[string]providerLatestBlockTimeAndFinalizedBlockContainer
	prevEpochLatestProviderBlockTimeAndFinalizedBlock    map[string]providerLatestBlockTimeAndFinalizedBlockContainer
}

type providerDataContainer struct {
	LatestFinalizedBlock          int64
	LatestBlockTime               time.Time
	FinalizedBlocksHashes         map[int64]string
	SigBlocks                     []byte
	LatestBlock                   int64
	BlockDistanceFromFinalization int64
	RelaySession                  pairingtypes.RelaySession
}

func NewFinalizationConsensus(specId string) *FinalizationConsensus {
	return &FinalizationConsensus{
		SpecId: specId,
		currentEpochBlockToHashesToAgreeingProviders:         make(BlockToHashesToAgreeingProviders),
		prevEpochBlockToHashesToAgreeingProviders:            make(BlockToHashesToAgreeingProviders),
		currentEpochLatestProviderBlockTimeAndFinalizedBlock: make(map[string]providerLatestBlockTimeAndFinalizedBlockContainer),
		prevEpochLatestProviderBlockTimeAndFinalizedBlock:    make(map[string]providerLatestBlockTimeAndFinalizedBlockContainer),
	}
}

func GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData int64) int64 {
	finalization_criteria := blockDistanceForFinalizedData
	return latestBlock - finalization_criteria
}

func (fc *FinalizationConsensus) insertProviderToConsensus(latestBlock, blockDistanceForFinalizedData int64, finalizedBlocks map[int64]string, reply *pairingtypes.RelayReply, req *pairingtypes.RelaySession, providerAcc string) {
	latestBlockTime := time.Now()
	newProviderDataContainer := providerDataContainer{
		LatestFinalizedBlock:          GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData),
		LatestBlockTime:               latestBlockTime,
		FinalizedBlocksHashes:         finalizedBlocks,
		SigBlocks:                     reply.SigBlocks,
		LatestBlock:                   latestBlock,
		BlockDistanceFromFinalization: blockDistanceForFinalizedData,
		RelaySession:                  *req,
	}

	previousProviderContainer, foundPrevious := fc.currentEpochLatestProviderBlockTimeAndFinalizedBlock[providerAcc]
	if !foundPrevious || newProviderDataContainer.LatestFinalizedBlock > previousProviderContainer.LatestFinalizedBlock {
		fc.currentEpochLatestProviderBlockTimeAndFinalizedBlock[providerAcc] = providerLatestBlockTimeAndFinalizedBlockContainer{
			LatestFinalizedBlock: newProviderDataContainer.LatestFinalizedBlock,
			LatestBlockTime:      latestBlockTime,
		}
	}

	maxBlockNum := int64(0)

	for blockNum, blockHash := range finalizedBlocks {
		if _, ok := fc.currentEpochBlockToHashesToAgreeingProviders[blockNum]; !ok {
			fc.currentEpochBlockToHashesToAgreeingProviders[blockNum] = map[string]map[string]providerDataContainer{}
		}

		if _, ok := fc.currentEpochBlockToHashesToAgreeingProviders[blockNum][blockHash]; !ok {
			fc.currentEpochBlockToHashesToAgreeingProviders[blockNum][blockHash] = map[string]providerDataContainer{}
		}

		fc.currentEpochBlockToHashesToAgreeingProviders[blockNum][blockHash][providerAcc] = newProviderDataContainer

		utils.LavaFormatTrace("Added provider to block hash consensus",
			utils.LogAttr("blockNum", blockNum),
			utils.LogAttr("blockHash", blockHash),
			utils.LogAttr("provider", providerAcc),
		)

		if blockNum > maxBlockNum {
			maxBlockNum = blockNum
		}
	}

	if maxBlockNum > fc.highestRecordedBlockHeight {
		fc.highestRecordedBlockHeight = maxBlockNum
	}
}

// Compare finalized block hashes with previous providers
// Looks for discrepancy with current epoch providers
// if no conflicts, insert into consensus and break
// create new consensus group if no consensus matched
// check for discrepancy with old epoch
// checks if there is a consensus mismatch between hashes provided by different providers
func (fc *FinalizationConsensus) UpdateFinalizedHashes(blockDistanceForFinalizedData int64, consumerAddress sdk.AccAddress, providerAddress string, finalizedBlocks map[int64]string, relaySession *pairingtypes.RelaySession, reply *pairingtypes.RelayReply) (finalizationConflict *conflicttypes.FinalizationConflict, err error) {
	fc.lock.Lock()
	defer func() {
		fc.insertProviderToConsensus(reply.LatestBlock, blockDistanceForFinalizedData, finalizedBlocks, reply, relaySession, providerAddress)
		fc.lock.Unlock()
	}()

	logSuccessUpdate := func() {
		utils.LavaFormatTrace("finalization information update successfully",
			utils.LogAttr("specId", fc.SpecId),
			utils.LogAttr("finalizationData", finalizedBlocks),
		)
	}

	var blockToHashesToAgreeingProviders BlockToHashesToAgreeingProviders
	foundDiscrepancy, discrepancyBlock := fc.findDiscrepancy(finalizedBlocks, fc.currentEpochBlockToHashesToAgreeingProviders)
	if foundDiscrepancy {
		utils.LavaFormatTrace("found discrepancy for provider",
			utils.LogAttr("specId", fc.SpecId),
			utils.LogAttr("currentEpoch", fc.currentEpoch),
			utils.LogAttr("provider", providerAddress),
			utils.LogAttr("discrepancyBlock", discrepancyBlock),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
		blockToHashesToAgreeingProviders = fc.currentEpochBlockToHashesToAgreeingProviders
	} else {
		// Could not find discrepancy, let's check with previous epoch
		foundDiscrepancy, discrepancyBlock = fc.findDiscrepancy(finalizedBlocks, fc.prevEpochBlockToHashesToAgreeingProviders)
		if foundDiscrepancy {
			utils.LavaFormatTrace("found discrepancy for provider with previous epoch",
				utils.LogAttr("specId", fc.SpecId),
				utils.LogAttr("currentEpoch", fc.currentEpoch),
				utils.LogAttr("provider", providerAddress),
				utils.LogAttr("discrepancyBlock", discrepancyBlock),
				utils.LogAttr("finalizedBlocks", finalizedBlocks),
			)
			blockToHashesToAgreeingProviders = fc.prevEpochBlockToHashesToAgreeingProviders
		} else {
			// No discrepancy found, log and return nil
			logSuccessUpdate()
			return finalizationConflict, nil
		}
	}

	// Found discrepancy, create finalization conflict
	replyFinalization := conflicttypes.NewRelayFinalizationFromRelaySessionAndRelayReply(relaySession, reply, consumerAddress)
	finalizationConflict = &conflicttypes.FinalizationConflict{RelayFinalization_0: &replyFinalization}

	var otherBlockHash string
	for blockHash, agreeingProviders := range blockToHashesToAgreeingProviders[discrepancyBlock] {
		if providerPrevReply, ok := agreeingProviders[providerAddress]; ok {
			// Same provider conflict found - Report and block provider
			utils.LavaFormatTrace("found same provider conflict, creating relay finalization from provider data container",
				utils.LogAttr("provider", providerAddress),
				utils.LogAttr("discrepancyBlock", discrepancyBlock),
				utils.LogAttr("blockHash", blockHash),
			)
			relayFinalization, err := fc.createRelayFinalizationFromProviderDataContainer(providerPrevReply, consumerAddress)
			if err != nil {
				return nil, err
			}

			logSuccessUpdate()
			finalizationConflict.RelayFinalization_1 = relayFinalization

			// We now want to block this provider, so we wrap the error with lavasession.BlockProviderError which will be caught later
			err = sdkerrors.Wrap(errors.Join(lavasession.BlockProviderError, lavasession.SessionOutOfSyncError),
				fmt.Sprintf("found same provider conflict on block [%d], provider address [%s]", discrepancyBlock, providerAddress))
			return finalizationConflict, err
		} else if otherBlockHash == "" {
			// Use some other hash to create proof, in case of no same provider conflict
			otherBlockHash = blockHash
		}
	}

	var otherProviderAddress string
	// Same provider conflict not found, create proof with other provider
	utils.LavaFormatTrace("did not find any same provider conflict, looking for other providers to create conflict proof", utils.LogAttr("provider", providerAddress))
	for currentOtherProviderAddress, providerDataContainer := range blockToHashesToAgreeingProviders[discrepancyBlock][otherBlockHash] {
		// Finalization conflict
		utils.LavaFormatTrace("creating finalization conflict with another provider",
			utils.LogAttr("provider", providerAddress),
			utils.LogAttr("otherProvider", currentOtherProviderAddress),
			utils.LogAttr("discrepancyBlock", discrepancyBlock),
			utils.LogAttr("blockHash", otherBlockHash),
		)
		relayFinalization, err := fc.createRelayFinalizationFromProviderDataContainer(providerDataContainer, consumerAddress)
		if err != nil {
			return nil, err
		}

		finalizationConflict.RelayFinalization_1 = relayFinalization
		otherProviderAddress = currentOtherProviderAddress
		break
	}

	logSuccessUpdate()
	// We don't block the provider here, as we found a conflict with another provider, and we don't know which one is correct
	return finalizationConflict, fmt.Errorf("found finalization conflict on block %d between provider [%s] and provider [%s] ", discrepancyBlock, providerAddress, otherProviderAddress)
}

func (fc *FinalizationConsensus) findDiscrepancy(finalizedBlocks map[int64]string, consensus BlockToHashesToAgreeingProviders) (foundDiscrepancy bool, discrepancyBlock int64) {
	for blockNum, blockHash := range finalizedBlocks {
		blockHashes, ok := consensus[blockNum]
		if !ok {
			continue
		}

		// We found the matching block, now we need to check if the hash exists
		if _, ok := blockHashes[blockHash]; !ok {
			utils.LavaFormatTrace("found discrepancy in finalized blocks",
				utils.LogAttr("blockNumber", blockNum),
				utils.LogAttr("blockHash", blockHash),
				utils.LogAttr("consensusBlockHashes", blockHashes),
			)
			return true, blockNum
		}
	}

	return false, 0
}

func (fc *FinalizationConsensus) createRelayFinalizationFromProviderDataContainer(dataContainer providerDataContainer, consumerAddr sdk.AccAddress) (*conflicttypes.RelayFinalization, error) {
	finalizedBlocksHashesBytes, err := json.Marshal(dataContainer.FinalizedBlocksHashes)
	if err != nil {
		utils.LavaFormatError("error in marshaling finalized blocks hashes", err)
		return nil, err
	}

	return &conflicttypes.RelayFinalization{
		FinalizedBlocksHashes: finalizedBlocksHashesBytes,
		LatestBlock:           dataContainer.LatestBlock,
		ConsumerAddress:       consumerAddr.String(),
		RelaySession:          &dataContainer.RelaySession,
		SigBlocks:             dataContainer.SigBlocks,
	}, nil
}

func (fc *FinalizationConsensus) NewEpoch(epoch uint64) {
	fc.lock.Lock()
	defer fc.lock.Unlock()

	if fc.currentEpoch < epoch {
		utils.LavaFormatTrace("finalization information epoch changed",
			utils.LogAttr("specId", fc.SpecId),
			utils.LogAttr("epoch", epoch),
		)

		// means it's time to refresh the epoch
		fc.prevEpochBlockToHashesToAgreeingProviders = fc.currentEpochBlockToHashesToAgreeingProviders
		fc.currentEpochBlockToHashesToAgreeingProviders = BlockToHashesToAgreeingProviders{}

		fc.prevEpochLatestProviderBlockTimeAndFinalizedBlock = fc.currentEpochLatestProviderBlockTimeAndFinalizedBlock
		fc.currentEpochLatestProviderBlockTimeAndFinalizedBlock = make(map[string]providerLatestBlockTimeAndFinalizedBlockContainer)

		fc.currentEpoch = epoch
	}
}

func (fc *FinalizationConsensus) getExpectedBlockHeightsOfProviders(averageBlockTime_ms time.Duration) map[string]int64 {
	// This function calculates the expected block heights for providers based on their last known activity and the average block time.
	// It accounts for both the current and previous epochs, determining the maximum block height from both and adjusting the expected heights accordingly to not exceed this maximum.
	// The method merges results from both epochs, with the previous epoch's data processed first.

	now := time.Now()
	mapExpectedBlockHeights := map[string]int64{}
	for providerAddress, latestProviderContainer := range fc.prevEpochLatestProviderBlockTimeAndFinalizedBlock {
		interpolation := InterpolateBlocks(now, latestProviderContainer.LatestBlockTime, averageBlockTime_ms)
		expected := latestProviderContainer.LatestFinalizedBlock + interpolation
		mapExpectedBlockHeights[providerAddress] = expected
	}

	for providerAddress, latestProviderContainer := range fc.currentEpochLatestProviderBlockTimeAndFinalizedBlock {
		interpolation := InterpolateBlocks(now, latestProviderContainer.LatestBlockTime, averageBlockTime_ms)
		expected := latestProviderContainer.LatestFinalizedBlock + interpolation
		mapExpectedBlockHeights[providerAddress] = expected
	}

	return mapExpectedBlockHeights
}

// Returns the expected latest block to be at based on the current finalization data, and the number of providers we have information for.
// It does the calculation on finalized entries then extrapolates the ending based on blockDistance
func (fc *FinalizationConsensus) GetExpectedBlockHeight(chainParser ChainBlockStatsGetter) (expectedBlockHeight int64, numOfProviders int) {
	fc.lock.RLock()
	defer fc.lock.RUnlock()

	allowedBlockLagForQosSync, averageBlockTime_ms, blockDistanceForFinalizedData, _ := chainParser.ChainBlockStats()
	mapExpectedBlockHeights := fc.getExpectedBlockHeightsOfProviders(averageBlockTime_ms)
	median := func(dataMap map[string]int64) int64 {
		data := make([]int64, len(dataMap))
		i := 0
		for _, latestBlock := range dataMap {
			data[i] = latestBlock
			i++
		}
		return lavaslices.Median(data)
	}

	medianOfExpectedBlocks := median(mapExpectedBlockHeights)
	providersMedianOfLatestBlock := medianOfExpectedBlocks + int64(blockDistanceForFinalizedData)

	utils.LavaFormatTrace("finalization information",
		utils.LogAttr("specId", fc.SpecId),
		utils.LogAttr("mapExpectedBlockHeights", mapExpectedBlockHeights),
		utils.LogAttr("medianOfExpectedBlocks", medianOfExpectedBlocks),
		utils.LogAttr("latestBlock", fc.prevLatestBlockByMedian),
		utils.LogAttr("providersMedianOfLatestBlock", providersMedianOfLatestBlock),
	)

	if medianOfExpectedBlocks > 0 && uint64(providersMedianOfLatestBlock) > fc.prevLatestBlockByMedian {
		if uint64(providersMedianOfLatestBlock) > fc.prevLatestBlockByMedian+1000 && fc.prevLatestBlockByMedian > 0 {
			utils.LavaFormatError("uncontinuous jump in finalization data", nil,
				utils.LogAttr("specId", fc.SpecId),
				utils.LogAttr("latestBlock", fc.prevLatestBlockByMedian),
				utils.LogAttr("providersMedianOfLatestBlock", providersMedianOfLatestBlock),
			)
		}
		atomic.StoreUint64(&fc.prevLatestBlockByMedian, uint64(providersMedianOfLatestBlock)) // we can only set conflict to "reported".
	}

	// median of all latest blocks after interpolation minus allowedBlockLagForQosSync is the lowest block in the finalization proof
	// then we move forward blockDistanceForFinalizedData to get the expected latest block
	return providersMedianOfLatestBlock - allowedBlockLagForQosSync, len(mapExpectedBlockHeights)
}

func InterpolateBlocks(timeNow, latestBlockTime time.Time, averageBlockTime time.Duration) int64 {
	if timeNow.Before(latestBlockTime) {
		return 0
	}
	return int64(timeNow.Sub(latestBlockTime) / averageBlockTime) // interpolation
}
