package finalizationconsensus

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
)

type ChainBlockStatsGetter interface {
	ChainBlockStats() (allowedBlockLagForQosSync int64, averageBlockTime time.Duration, blockDistanceForFinalizedData, blocksInFinalizationProof uint32)
}

type FinalizationConsensusInf interface {
	UpdateLatestBlock(
		blockDistanceForFinalizedData int64,
		providerAddress string,
		reply *pairingtypes.RelayReply,
	) error
	GetExpectedBlockHeight(chainParser ChainBlockStatsGetter) (expectedBlockHeight int64, numOfProviders int)
	NewEpoch(epoch uint64)
}

type LatestBlockInfo struct {
	LatestBlockTime      time.Time // the time of the latest block (non-finalized)
	LatestFinalizedBlock int64     // the latest finalized block height
}

type FinalizationConsensus struct {
	lock                    sync.RWMutex
	currentEpoch            uint64
	prevLatestBlockByMedian uint64 // for caching
	SpecId                  string
	currentEpochInfo        map[string]LatestBlockInfo
	prevEpochInfo           map[string]LatestBlockInfo
}

func NewFinalizationConsensus(specId string) *FinalizationConsensus {
	return &FinalizationConsensus{
		SpecId:           specId,
		currentEpochInfo: make(map[string]LatestBlockInfo),
		prevEpochInfo:    make(map[string]LatestBlockInfo),
	}
}

func GetLatestFinalizedBlock(latestBlock, blockDistanceForFinalizedData int64) int64 {
	finalization_criteria := blockDistanceForFinalizedData
	return latestBlock - finalization_criteria
}

// Update the latest block info for a provider: latest block time and latest finalized block height
func (fc *FinalizationConsensus) UpdateLatestBlock(finalizationDistance int64, providerAddress string, reply *pairingtypes.RelayReply) error {
	latestBlockTime := time.Now()

	prevBlockInfo, foundPrevious := fc.getLatestBlockInfo(providerAddress)
	if !foundPrevious || reply.LatestBlock > prevBlockInfo.LatestFinalizedBlock {
		fc.setLatestBlockInfo(providerAddress, LatestBlockInfo{
			LatestFinalizedBlock: GetLatestFinalizedBlock(reply.LatestBlock, finalizationDistance),
			LatestBlockTime:      latestBlockTime,
		})
	}

	return nil
}

func (fc *FinalizationConsensus) getLatestBlockInfo(providerAddress string) (LatestBlockInfo, bool) {
	fc.lock.RLock()
	defer fc.lock.RUnlock()
	blockInfo, found := fc.currentEpochInfo[providerAddress]
	return blockInfo, found
}

func (fc *FinalizationConsensus) setLatestBlockInfo(providerAddress string, blockInfo LatestBlockInfo) {
	fc.lock.Lock()
	defer fc.lock.Unlock()
	fc.currentEpochInfo[providerAddress] = blockInfo
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
		fc.prevEpochInfo = fc.currentEpochInfo
		fc.currentEpochInfo = make(map[string]LatestBlockInfo)

		fc.currentEpoch = epoch
	}
}

func (fc *FinalizationConsensus) getExpectedBlockHeightsOfProviders(averageBlockTime_ms time.Duration) map[string]int64 {
	// This function calculates the expected block heights for providers based on their last known activity and the average block time.
	// It accounts for both the current and previous epochs, determining the maximum block height from both and adjusting the expected heights accordingly to not exceed this maximum.
	// The method merges results from both epochs, with the previous epoch's data processed first.

	now := time.Now()
	mapExpectedBlockHeights := map[string]int64{}
	for providerAddress, latestProviderContainer := range fc.prevEpochInfo {
		interpolation := InterpolateBlocks(now, latestProviderContainer.LatestBlockTime, averageBlockTime_ms)
		expected := latestProviderContainer.LatestFinalizedBlock + interpolation
		mapExpectedBlockHeights[providerAddress] = expected
	}

	for providerAddress, latestProviderContainer := range fc.currentEpochInfo {
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
		atomic.StoreUint64(&fc.prevLatestBlockByMedian, uint64(providersMedianOfLatestBlock))
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
