package lavaprotocol

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/maps"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type BlockToHashesToAgreeingProviders map[int64]map[string]map[string]providerDataContainer // first key is block num, second key is block hash, third key is provider address

func (b BlockToHashesToAgreeingProviders) String() string {
	ret, err := json.Marshal(b)
	if err != nil {
		return ""
	}
	return string(ret)
}

type FinalizationConsensus struct {
	currentEpochBlockToHashesToAgreeingProviders BlockToHashesToAgreeingProviders
	prevEpochBlockToHashesToAgreeingProviders    BlockToHashesToAgreeingProviders
	lock                                         sync.RWMutex
	currentEpoch                                 uint64
	prevLatestBlockByMedian                      uint64 // for caching
	SpecId                                       string
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
		currentEpochBlockToHashesToAgreeingProviders: make(BlockToHashesToAgreeingProviders),
		prevEpochBlockToHashesToAgreeingProviders:    make(BlockToHashesToAgreeingProviders),
	}
}

func GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData int64) int64 {
	finalization_criteria := blockDistanceForFinalizedData
	return latestBlock - finalization_criteria
}

// print the current status
func (fc *FinalizationConsensus) String() string {
	fc.lock.RLock()
	defer fc.lock.RUnlock()
	mapExpectedBlockHeights := fc.getExpectedBlockHeightsOfProviders(10 * time.Millisecond) // it's not super important so we hardcode this
	return fmt.Sprintf("{FinalizationConsensus: {mapExpectedBlockHeights:%v} epoch: %d latestBlockByMedian %d}", mapExpectedBlockHeights, fc.currentEpoch, &fc.prevLatestBlockByMedian)
}

func (fc *FinalizationConsensus) insertProviderToConsensus(latestBlock, blockDistanceForFinalizedData int64, finalizedBlocks map[int64]string, reply *pairingtypes.RelayReply, req *pairingtypes.RelaySession, providerAcc string) {
	newProviderDataContainer := providerDataContainer{
		LatestFinalizedBlock:          GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData),
		LatestBlockTime:               time.Now(),
		FinalizedBlocksHashes:         finalizedBlocks,
		SigBlocks:                     reply.SigBlocks,
		LatestBlock:                   latestBlock,
		BlockDistanceFromFinalization: blockDistanceForFinalizedData,
		RelaySession:                  *req,
	}

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
			utils.LogAttr("currentBlockToHashesToAgreeingProviders", fc.currentEpochBlockToHashesToAgreeingProviders),
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
			relayFinalization, errWrapped := fc.createRelayFinalizationFromProviderDataContainer(providerPrevReply, consumerAddress)
			if errWrapped != nil {
				return nil, errWrapped
			}

			logSuccessUpdate()
			finalizationConflict.RelayFinalization_1 = relayFinalization

			// We now want to block this provider, so we wrap the error with lavasession.BlockProviderError which will be caught later
			errWrapped = sdkerrors.Wrap(lavasession.BlockProviderError, fmt.Sprintf("found same provider conflict on block [%d], provider address [%s]", discrepancyBlock, providerAddress))
			return finalizationConflict, errWrapped
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
		fc.currentEpoch = epoch
	}
}

func (fc *FinalizationConsensus) getExpectedBlockHeightsOfProviders(averageBlockTime_ms time.Duration) map[string]int64 {
	// This function calculates the expected block heights for providers based on their last known activity and the average block time.
	// It accounts for both the current and previous epochs, determining the maximum block height from both and adjusting the expected heights accordingly to not exceed this maximum.
	// The method merges results from both epochs, with the previous epoch's data processed first.

	currentEpochMaxBlockHeight := maps.GetMaxKey(fc.currentEpochBlockToHashesToAgreeingProviders)
	prevEpochMaxBlockHeight := maps.GetMaxKey(fc.prevEpochBlockToHashesToAgreeingProviders)
	highestBlockNumber := utils.Max(currentEpochMaxBlockHeight, prevEpochMaxBlockHeight)

	now := time.Now()
	calcExpectedBlocks := func(mapExpectedBlockHeights map[string]int64, blockToHashesToAgreeingProviders BlockToHashesToAgreeingProviders) map[string]int64 {
		// Since we are looking for the maximum block height, we need to sort the block heights in ascending order.
		// This will ensure that mapExpectedBlockHeights will contain the maximum block height for each provider.
		sortedBlockHeights := maps.StableSortedKeys(blockToHashesToAgreeingProviders)
		for _, blockHeight := range sortedBlockHeights {
			blockHashesToAgreeingProviders := blockToHashesToAgreeingProviders[blockHeight]
			for _, agreeingProviders := range blockHashesToAgreeingProviders {
				for providerAddress, providerDataContainer := range agreeingProviders {
					interpolation := InterpolateBlocks(now, providerDataContainer.LatestBlockTime, averageBlockTime_ms)
					expected := providerDataContainer.LatestFinalizedBlock + interpolation
					// limit the interpolation to the highest seen block height
					if expected > highestBlockNumber {
						expected = highestBlockNumber
					}
					mapExpectedBlockHeights[providerAddress] = expected
				}
			}
		}
		return mapExpectedBlockHeights
	}
	mapExpectedBlockHeights := map[string]int64{}
	// prev must be before current because we overwrite
	mapExpectedBlockHeights = calcExpectedBlocks(mapExpectedBlockHeights, fc.prevEpochBlockToHashesToAgreeingProviders)
	mapExpectedBlockHeights = calcExpectedBlocks(mapExpectedBlockHeights, fc.currentEpochBlockToHashesToAgreeingProviders)
	return mapExpectedBlockHeights
}

// Returns the expected latest block to be at based on the current finalization data, and the number of providers we have information for.
// It does the calculation on finalized entries then extrapolates the ending based on blockDistance
func (fc *FinalizationConsensus) GetExpectedBlockHeight(chainParser chainlib.ChainParser) (expectedBlockHeight int64, numOfProviders int) {
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
				utils.LogAttr("prevEpochBlockToHashesToAgreeingProviders", fc.prevEpochBlockToHashesToAgreeingProviders),
				utils.LogAttr("currentBlockToHashesToAgreeingProviders", fc.currentEpochBlockToHashesToAgreeingProviders),
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

func VerifyFinalizationData(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, providerAddr string, consumerAcc sdk.AccAddress, latestSessionBlock, blockDistanceForFinalization, blocksInFinalizationProof int64) (finalizedBlocks map[int64]string, errRet error) {
	relayFinalization := conflicttypes.NewRelayFinalizationFromRelaySessionAndRelayReply(relayRequest.RelaySession, reply, consumerAcc)
	recoveredProviderPubKey, err := sigs.RecoverPubKey(relayFinalization)
	if err != nil {
		return nil, err
	}

	recoveredProviderAddr, err := sdk.AccAddressFromHexUnsafe(recoveredProviderPubKey.Address().String())
	if err != nil {
		return nil, err
	}

	if recoveredProviderAddr.String() != providerAddr {
		return nil, utils.LavaFormatError("provider address mismatch in finalization data ", ProviderFinalizationDataError,
			utils.LogAttr("parsed Address", recoveredProviderAddr.String()),
			utils.LogAttr("expected address", providerAddr),
		)
	}

	finalizedBlocks = map[int64]string{}
	err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return nil, utils.LavaFormatError("failed in unmarshalling finalized blocks data", ProviderFinalizationDataError,
			utils.LogAttr("FinalizedBlocksHashes", string(reply.FinalizedBlocksHashes)),
			utils.LogAttr("errMsg", err.Error()),
		)
	}

	err = verifyFinalizationDataIntegrity(reply, latestSessionBlock, finalizedBlocks, blockDistanceForFinalization, blocksInFinalizationProof, providerAddr)
	if err != nil {
		return nil, err
	}

	providerLatestBlock := reply.LatestBlock
	seenBlock := relayRequest.RelayData.SeenBlock
	requestBlock := relayRequest.RelayData.RequestBlock

	if providerLatestBlock < lavaslices.Min([]int64{seenBlock, requestBlock}) {
		return nil, utils.LavaFormatError("provider response does not meet consistency requirements", ProviderFinalizationDataError,
			utils.LogAttr("ProviderAddress", relayRequest.RelaySession.Provider),
			utils.LogAttr("providerLatestBlock", providerLatestBlock),
			utils.LogAttr("seenBlock", seenBlock),
			utils.LogAttr("requestBlock", requestBlock),
			utils.LogAttr("provider address", providerAddr),
		)
	}
	return finalizedBlocks, errRet
}

func verifyFinalizationDataIntegrity(reply *pairingtypes.RelayReply, latestSessionBlock int64, finalizedBlocks map[int64]string, blockDistanceForFinalization, blocksInFinalizationProof int64, providerAddr string) (err error) {
	latestBlock := reply.LatestBlock
	sorted := make([]int64, len(finalizedBlocks))
	idx := 0
	maxBlockNum := int64(0)

	if int64(len(finalizedBlocks)) != blocksInFinalizationProof {
		return utils.LavaFormatError("Simulation: provider returned incorrect number of finalized blocks", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("Provider", providerAddr),
			utils.LogAttr("blocksInFinalizationProof", blocksInFinalizationProof),
			utils.LogAttr("len(finalizedBlocks)", len(finalizedBlocks)),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	for blockNum := range finalizedBlocks {
		// Check if finalized
		if !spectypes.IsFinalizedBlock(blockNum, latestBlock, blockDistanceForFinalization) {
			return utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability", ProviderFinalizationDataAccountabilityError,
				utils.LogAttr("blockNum", blockNum),
				utils.LogAttr("latestBlock", latestBlock),
				utils.LogAttr("Provider", providerAddr),
				utils.LogAttr("finalizedBlocks", finalizedBlocks),
			)
		}

		// Add to sorted
		sorted[idx] = blockNum

		// Update maxBlockNum
		if blockNum > maxBlockNum {
			maxBlockNum = blockNum
		}
		idx++
	}

	lavaslices.SortStable(sorted)

	// Check for consecutive blocks
	nonConsecutiveIndex, isConsecutive := lavaslices.IsSliceConsecutive(sorted)
	if !isConsecutive {
		return utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("currBlock", sorted[nonConsecutiveIndex]),
			utils.LogAttr("prevBlock", sorted[nonConsecutiveIndex-1]),
			utils.LogAttr("providerAddr", providerAddr),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	// Check that latest finalized block address + 1 points to a non finalized block
	if spectypes.IsFinalizedBlock(maxBlockNum+1, latestBlock, blockDistanceForFinalization) {
		return utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("maxBlockNum", maxBlockNum),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	// New reply should have blocknum >= from block same provider
	if latestSessionBlock > latestBlock {
		return utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("session.LatestBlock", latestSessionBlock),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
		)
	}

	return nil
}
