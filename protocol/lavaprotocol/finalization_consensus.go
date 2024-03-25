package lavaprotocol

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/chainlib"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/lavaslices"
	"github.com/lavanet/lava/utils/maps"
	"github.com/lavanet/lava/utils/sigs"
	conflicttypes "github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

type BlockToHashesToAgreeingProviders map[int64]map[string]map[string]providerDataContainer

func (b BlockToHashesToAgreeingProviders) String() string {
	ret, err := json.Marshal(b)
	if err != nil {
		return ""
	}
	return string(ret)
}

type FinalizationConsensus struct {
	currentBlockToHashesToAgreeingProviders   BlockToHashesToAgreeingProviders
	prevEpochBlockToHashesToAgreeingProviders BlockToHashesToAgreeingProviders
	providerDataContainersMu                  sync.RWMutex
	currentEpoch                              uint64
	prevLatestBlockByMedian                   uint64 // for caching
	specId                                    string
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
		specId:                                  specId,
		currentBlockToHashesToAgreeingProviders: BlockToHashesToAgreeingProviders{},
	}
}

func GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData int64) int64 {
	finalization_criteria := blockDistanceForFinalizedData
	return latestBlock - finalization_criteria
}

// print the current status
func (fc *FinalizationConsensus) String() string {
	fc.providerDataContainersMu.RLock()
	defer fc.providerDataContainersMu.RUnlock()
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
		if _, ok := fc.currentBlockToHashesToAgreeingProviders[blockNum]; !ok {
			fc.currentBlockToHashesToAgreeingProviders[blockNum] = map[string]map[string]providerDataContainer{}
		}

		if _, ok := fc.currentBlockToHashesToAgreeingProviders[blockNum][blockHash]; !ok {
			fc.currentBlockToHashesToAgreeingProviders[blockNum][blockHash] = map[string]providerDataContainer{}
		}

		fc.currentBlockToHashesToAgreeingProviders[blockNum][blockHash][providerAcc] = newProviderDataContainer
	}
}

// Compare finalized block hashes with previous providers
// Looks for discrepancy with current epoch providers
// if no conflicts, insert into consensus and break
// create new consensus group if no consensus matched
// check for discrepancy with old epoch
// checks if there is a consensus mismatch between hashes provided by different providers
func (fc *FinalizationConsensus) UpdateFinalizedHashes(blockDistanceForFinalizedData int64, consumerAddress sdk.AccAddress, providerAddress string, finalizedBlocks map[int64]string, relaySession *pairingtypes.RelaySession, reply *pairingtypes.RelayReply) (finalizationConflict *conflicttypes.FinalizationConflict, err error) {
	logSuccessUpdate := func() {
		if debug {
			utils.LavaFormatDebug("finalization information update successfully",
				utils.LogAttr("specId", fc.specId),
				utils.LogAttr("finalizationData", finalizedBlocks),
				utils.LogAttr("currentBlockToHashesToAgreeingProviders", fc.currentBlockToHashesToAgreeingProviders),
			)
		}
	}

	latestBlock := reply.LatestBlock

	fc.providerDataContainersMu.Lock()
	defer func() {
		fc.insertProviderToConsensus(latestBlock, blockDistanceForFinalizedData, finalizedBlocks, reply, relaySession, providerAddress)
		fc.providerDataContainersMu.Unlock()
	}()

	var blockToHashesToAgreeingProviders BlockToHashesToAgreeingProviders
	foundDiscrepancy, discrepancyBlock := fc.findDiscrepancy(finalizedBlocks, fc.currentBlockToHashesToAgreeingProviders)
	if foundDiscrepancy {
		blockToHashesToAgreeingProviders = fc.currentBlockToHashesToAgreeingProviders
	} else {
		// Could not find discrepancy, let's check with previous epoch
		foundDiscrepancy, discrepancyBlock = fc.findDiscrepancy(finalizedBlocks, fc.prevEpochBlockToHashesToAgreeingProviders)
		if foundDiscrepancy {
			blockToHashesToAgreeingProviders = fc.prevEpochBlockToHashesToAgreeingProviders
		} else {
			// No discrepancy found, log and return nil
			logSuccessUpdate()
			return finalizationConflict, nil
		}
	}

	replyFinalization := conflicttypes.NewRelayFinalizationMetaDataFromRelaySessionAndRelayReply(relaySession, reply, consumerAddress)
	finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}

	var otherBlockHash string
	for blockHash, agreeingProviders := range blockToHashesToAgreeingProviders[discrepancyBlock] {
		if providerPrevReply, ok := agreeingProviders[providerAddress]; ok {
			// Same provider conflict found
			relayFinalization, err := fc.createRelayFinalizationFromProviderDataContainer(providerPrevReply, consumerAddress)
			if err != nil {
				return nil, err
			}

			logSuccessUpdate()
			finalizationConflict.RelayReply1 = relayFinalization
			return finalizationConflict, fmt.Errorf("found same provider [%s] conflict on block %d", providerAddress, discrepancyBlock)
		} else if otherBlockHash == "" {
			// Use some other hash to create proof, in case of no same provider conflict
			otherBlockHash = blockHash
		}
	}

	var otherProviderAddress string
	// Same provider conflict not found, create proof with other provider
	for providerAddress, providerDataContainer := range blockToHashesToAgreeingProviders[discrepancyBlock][otherBlockHash] {
		// Finalization conflict
		relayFinalization, err := fc.createRelayFinalizationFromProviderDataContainer(providerDataContainer, consumerAddress)
		if err != nil {
			return nil, err
		}

		finalizationConflict.RelayReply1 = relayFinalization
		otherProviderAddress = providerAddress
		// TODO: Should we send a conflict proof for all providers?
		break
	}

	logSuccessUpdate()
	return finalizationConflict, fmt.Errorf("found finalization conflict on block %d between provider [%s] and provider [%s] ", discrepancyBlock, providerAddress, otherProviderAddress)
}

// func (fc *FinalizationConsensus) find

func (fc *FinalizationConsensus) findDiscrepancy(finalizedBlocksA map[int64]string, consensus BlockToHashesToAgreeingProviders) (foundDiscrepancy bool, discrepancyBlock int64) {
	for blockNum, blockHash := range finalizedBlocksA {
		blockHashes, ok := consensus[blockNum]
		if !ok {
			continue
		}

		// We found the matching block, now we need to check if the hash exists
		if _, ok := blockHashes[blockHash]; !ok {
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
	}, nil
}

func (fc *FinalizationConsensus) NewEpoch(epoch uint64) {
	fc.providerDataContainersMu.Lock()
	defer fc.providerDataContainersMu.Unlock()

	if fc.currentEpoch < epoch {
		if debug {
			utils.LavaFormatDebug("finalization information epoch changed", utils.Attribute{Key: "specId", Value: fc.specId}, utils.Attribute{Key: "epoch", Value: epoch})
		}
		// means it's time to refresh the epoch
		fc.prevEpochBlockToHashesToAgreeingProviders = fc.currentBlockToHashesToAgreeingProviders
		fc.currentBlockToHashesToAgreeingProviders = BlockToHashesToAgreeingProviders{}
		fc.currentEpoch = epoch
	}
}

func (fc *FinalizationConsensus) getExpectedBlockHeightsOfProviders(averageBlockTime_ms time.Duration) map[string]int64 {
	currentEpochMaxBlockHeight := maps.GetMaxKey(fc.currentBlockToHashesToAgreeingProviders)
	prevEpochMaxBlockHeight := maps.GetMaxKey(fc.prevEpochBlockToHashesToAgreeingProviders)
	highestBlockNumber := utils.Max(currentEpochMaxBlockHeight, prevEpochMaxBlockHeight)

	now := time.Now()
	calcExpectedBlocks := func(mapExpectedBlockHeights map[string]int64, blockToHashesToAgreeingProviders BlockToHashesToAgreeingProviders) map[string]int64 {
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
	mapExpectedBlockHeights = calcExpectedBlocks(mapExpectedBlockHeights, fc.currentBlockToHashesToAgreeingProviders)
	return mapExpectedBlockHeights
}

// Returns the expected latest block to be at based on the current finalization data, and the number of providers we have information for.
// It does the calculation on finalized entries then extrapolates the ending based on blockDistance
func (fc *FinalizationConsensus) GetExpectedBlockHeight(chainParser chainlib.ChainParser) (expectedBlockHeight int64, numOfProviders int) {
	fc.providerDataContainersMu.RLock()
	defer fc.providerDataContainersMu.RUnlock()

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
	if debug {
		utils.LavaFormatDebug("finalization information",
			utils.LogAttr("specId", fc.specId),
			utils.LogAttr("mapExpectedBlockHeights", mapExpectedBlockHeights),
			utils.LogAttr("medianOfExpectedBlocks", medianOfExpectedBlocks),
			utils.LogAttr("latestBlock", fc.prevLatestBlockByMedian),
			utils.LogAttr("providersMedianOfLatestBlock", providersMedianOfLatestBlock),
		)
	}

	if medianOfExpectedBlocks > 0 && uint64(providersMedianOfLatestBlock) > fc.prevLatestBlockByMedian {
		if uint64(providersMedianOfLatestBlock) > fc.prevLatestBlockByMedian+1000 && fc.prevLatestBlockByMedian > 0 {
			utils.LavaFormatError("uncontinuous jump in finalization data", nil,
				utils.LogAttr("specId", fc.specId),
				utils.LogAttr("prevEpochBlockToHashesToAgreeingProviders", fc.prevEpochBlockToHashesToAgreeingProviders),
				utils.LogAttr("currentBlockToHashesToAgreeingProviders", fc.currentBlockToHashesToAgreeingProviders),
				utils.LogAttr("latestBlock", fc.prevLatestBlockByMedian),
				utils.LogAttr("providersMedianOfLatestBlock", providersMedianOfLatestBlock),
			)
		}
		fc.prevLatestBlockByMedian = uint64(providersMedianOfLatestBlock) // we can only set conflict to "reported".
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

func VerifyFinalizationData(reply *pairingtypes.RelayReply, relayRequest *pairingtypes.RelayRequest, providerAddr string, consumerAcc sdk.AccAddress, latestSessionBlock, blockDistanceForFinalization int64) (finalizedBlocks map[int64]string, finalizationConflict *conflicttypes.FinalizationConflict, errRet error) {
	// TODO: Should block provider and report

	relayFinalization := conflicttypes.NewRelayFinalizationMetaDataFromRelaySessionAndRelayReply(relayRequest.RelaySession, reply, consumerAcc)
	recoveredProviderPubKey, err := sigs.RecoverPubKey(relayFinalization)
	if err != nil {
		return nil, nil, err
	}

	recoveredProviderAddr, err := sdk.AccAddressFromHexUnsafe(recoveredProviderPubKey.Address().String())
	if err != nil {
		return nil, nil, err
	}

	if recoveredProviderAddr.String() != providerAddr {
		return nil, nil, utils.LavaFormatError("provider address mismatch in finalization data ", ProviderFinalizationDataError,
			utils.LogAttr("parsed Address", recoveredProviderAddr.String()),
			utils.LogAttr("expected address", providerAddr),
		)
	}

	finalizedBlocks = map[int64]string{}
	err = json.Unmarshal(reply.FinalizedBlocksHashes, &finalizedBlocks)
	if err != nil {
		return nil, nil, utils.LavaFormatError("failed in unmarshalling finalized blocks data", ProviderFinalizationDataError,
			utils.LogAttr("FinalizedBlocksHashes", string(reply.FinalizedBlocksHashes)),
			utils.LogAttr("errMsg", err.Error()),
		)
	}

	finalizationConflict, err = verifyFinalizationDataIntegrity(relayRequest.RelaySession, reply, latestSessionBlock, finalizedBlocks, blockDistanceForFinalization, consumerAcc, providerAddr)
	if err != nil {
		return nil, finalizationConflict, err
	}

	providerLatestBlock := reply.LatestBlock
	seenBlock := relayRequest.RelayData.SeenBlock
	requestBlock := relayRequest.RelayData.RequestBlock

	if providerLatestBlock < lavaslices.Min([]int64{seenBlock, requestBlock}) {
		return nil, nil, utils.LavaFormatError("provider response does not meet consistency requirements", ProviderFinalizationDataError,
			utils.LogAttr("ProviderAddress", relayRequest.RelaySession.Provider),
			utils.LogAttr("providerLatestBlock", providerLatestBlock),
			utils.LogAttr("seenBlock", seenBlock),
			utils.LogAttr("requestBlock", requestBlock),
			utils.LogAttr("provider address", providerAddr),
		)
	}
	return finalizedBlocks, finalizationConflict, errRet
}

func verifyFinalizationDataIntegrity(relaySession *pairingtypes.RelaySession, reply *pairingtypes.RelayReply, latestSessionBlock int64, finalizedBlocks map[int64]string, blockDistanceForFinalization int64, consumerAddr sdk.AccAddress, providerAddr string) (finalizationConflict *conflicttypes.FinalizationConflict, err error) {
	latestBlock := reply.LatestBlock
	sorted := make([]int64, len(finalizedBlocks))
	idx := 0
	maxBlockNum := int64(0)
	// TODO: compare finalizedBlocks len vs chain parser len to validate (get from same place as blockDistanceForFinalization arrives)

	replyFinalization := conflicttypes.NewRelayFinalizationMetaDataFromRelaySessionAndRelayReply(relaySession, reply, consumerAddr)

	for blockNum := range finalizedBlocks {
		// Check if finalized
		if !spectypes.IsFinalizedBlock(blockNum, latestBlock, blockDistanceForFinalization) {
			finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
			return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non finalized block reply for reliability", ProviderFinalizationDataAccountabilityError,
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

		// TODO: check block hash length and format
	}

	lavaslices.SortStable(sorted)

	// Check for consecutive blocks
	nonConsecutiveIndex, isConsecutive := lavaslices.IsSliceConsecutive(sorted)
	if !isConsecutive {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
		return finalizationConflict, utils.LavaFormatError("Simulation: provider returned non consecutive finalized blocks reply", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("currBlock", sorted[nonConsecutiveIndex]),
			utils.LogAttr("prevBlock", sorted[nonConsecutiveIndex-1]),
			utils.LogAttr("providerAddr", providerAddr),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	// Check that latest finalized block address + 1 points to a non finalized block
	if spectypes.IsFinalizedBlock(maxBlockNum+1, latestBlock, blockDistanceForFinalization) {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
		return finalizationConflict, utils.LavaFormatError("Simulation: provider returned finalized hashes for an older latest block", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("maxBlockNum", maxBlockNum),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
			utils.LogAttr("finalizedBlocks", finalizedBlocks),
		)
	}

	// New reply should have blocknum >= from block same provider
	if latestSessionBlock > latestBlock {
		finalizationConflict = &conflicttypes.FinalizationConflict{RelayReply0: &replyFinalization}
		return finalizationConflict, utils.LavaFormatError("Simulation: Provider supplied an older latest block than it has previously", ProviderFinalizationDataAccountabilityError,
			utils.LogAttr("session.LatestBlock", latestSessionBlock),
			utils.LogAttr("latestBlock", latestBlock),
			utils.LogAttr("Provider", providerAddr),
		)
	}

	return finalizationConflict, nil
}
