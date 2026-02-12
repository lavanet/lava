package relaycore

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/lavaslices"
)

// providerBlockObservation holds the latest block data reported by a provider.
type providerBlockObservation struct {
	LatestBlock int64
	LastUpdated time.Time
}

// LatestBlockEstimator approximates the network latest block based on provider observations.
// It mirrors the behavior of the removed finalization consensus estimator, but without relying on DR-specific data.
// Consumers (classic and smart router) feed the estimator with every successful relay response so archive extensions
// and "latest" requests can continue to auto-detect heights even after DR-specific components are removed.
type LatestBlockEstimator struct {
	mu           sync.RWMutex
	observations map[string]providerBlockObservation
	prevMedian   atomic.Uint64
}

func NewLatestBlockEstimator() *LatestBlockEstimator {
	return &LatestBlockEstimator{
		observations: make(map[string]providerBlockObservation),
	}
}

func (lbe *LatestBlockEstimator) Record(providerAddress string, latestBlock int64) {
	if providerAddress == "" || latestBlock <= 0 {
		return
	}

	lbe.mu.Lock()
	defer lbe.mu.Unlock()

	// Cleanup old observations (older than 1 hour) to prevent memory leak
	const maxObservationAge = time.Hour
	now := time.Now()
	for addr, obs := range lbe.observations {
		if now.Sub(obs.LastUpdated) > maxObservationAge {
			delete(lbe.observations, addr)
		}
	}

	// Add or update the observation for this provider
	lbe.observations[providerAddress] = providerBlockObservation{
		LatestBlock: latestBlock,
		LastUpdated: now,
	}
}

// Estimate expected block height based on time elapsed.
// Returns the expected block height as well as the number of provider observations included.
func (lbe *LatestBlockEstimator) Estimate(chainParser chainlib.ChainParser) (int64, int) {
	if lbe == nil || chainParser == nil {
		return 0, 0
	}

	allowedBlockLagForQosSync, averageBlockTime, blockDistanceForFinalizedData, _ := chainParser.ChainBlockStats()

	lbe.mu.RLock()
	providerObservations := make([]providerBlockObservation, 0, len(lbe.observations))
	for _, observation := range lbe.observations {
		providerObservations = append(providerObservations, observation)
	}
	lbe.mu.RUnlock()

	if len(providerObservations) == 0 {
		return 0, 0
	}

	now := time.Now()
	expectedBlocks := make([]int64, 0, len(providerObservations))
	for _, observation := range providerObservations {
		if observation.LatestBlock <= 0 {
			continue
		}

		latestFinalizedBlock := observation.LatestBlock - int64(blockDistanceForFinalizedData)
		if latestFinalizedBlock < 0 {
			latestFinalizedBlock = 0
		}

		interpolation := interpolateBlocks(now, observation.LastUpdated, averageBlockTime)
		expectedBlocks = append(expectedBlocks, latestFinalizedBlock+interpolation)
	}

	if len(expectedBlocks) == 0 {
		return 0, 0
	}

	medianOfExpectedBlocks := lavaslices.Median(expectedBlocks)
	providersMedianOfLatestBlock := medianOfExpectedBlocks + int64(blockDistanceForFinalizedData)

	utils.LavaFormatTrace("latest block estimator",
		utils.LogAttr("medianOfExpectedBlocks", medianOfExpectedBlocks),
		utils.LogAttr("providersMedianOfLatestBlock", providersMedianOfLatestBlock),
		utils.LogAttr("numProviders", len(expectedBlocks)),
	)

	if medianOfExpectedBlocks > 0 {
		prevLatest := lbe.prevMedian.Load()
		if uint64(providersMedianOfLatestBlock) > prevLatest {
			if prevLatest > 0 && uint64(providersMedianOfLatestBlock) > prevLatest+1000 {
				utils.LavaFormatError("uncontinuous jump in latest block estimate", nil,
					utils.LogAttr("previousMedian", prevLatest),
					utils.LogAttr("currentMedian", providersMedianOfLatestBlock),
				)
			}
			lbe.prevMedian.Store(uint64(providersMedianOfLatestBlock))
		}
	}

	expectedBlockHeight := providersMedianOfLatestBlock - allowedBlockLagForQosSync
	if expectedBlockHeight < 0 {
		expectedBlockHeight = 0
	}

	return expectedBlockHeight, len(expectedBlocks)
}

func interpolateBlocks(now, latest time.Time, averageBlockTime time.Duration) int64 {
	if averageBlockTime <= 0 || now.Before(latest) {
		return 0
	}

	return int64(now.Sub(latest) / averageBlockTime)
}
