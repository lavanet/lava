package chaintracker

import (
	"context"
	"math"
	"time"
)

const DummyChainTrackerLatestBlock = int64(math.MaxInt64)

type DummyChainTracker struct{}

// GetLatestBlockData returns block hashes for the specified block range and a specific block
func (dct *DummyChainTracker) GetLatestBlockData(fromBlock, toBlock, specificBlock int64) (latestBlock int64, requestedHashes []*BlockStore, changeTime time.Time, err error) {
	return DummyChainTrackerLatestBlock, []*BlockStore{}, time.Now(), nil
}

// RegisterForBlockTimeUpdates registers an updatable to receive block time updates
func (dct *DummyChainTracker) RegisterForBlockTimeUpdates(updatable blockTimeUpdatable) {
	return
}

// GetLatestBlockNum returns the current latest block number and the time it was last changed
func (dct *DummyChainTracker) GetLatestBlockNum() (int64, time.Time) {
	return DummyChainTrackerLatestBlock, time.Now()
}

// GetAtomicLatestBlockNum returns the current latest block number atomically
func (dct *DummyChainTracker) GetAtomicLatestBlockNum() int64 {
	return DummyChainTrackerLatestBlock
}

// StartAndServe starts the chain tracker and serves gRPC if configured
func (dct *DummyChainTracker) StartAndServe(ctx context.Context) error {
	return nil
}

// AddBlockGap adds a new block gap measurement
func (dct *DummyChainTracker) AddBlockGap(newData time.Duration, blocks uint64) {
	return
}

func (dct *DummyChainTracker) IsDummy() bool {
	return true
}
