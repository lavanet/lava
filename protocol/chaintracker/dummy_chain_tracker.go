package chaintracker

import (
	"context"
	"math"
	"time"
)

const DummyChainTrackerLatestBlock = int64(math.MaxInt64)

type DummyChainTracker struct{}

// RegisterForBlockTimeUpdates registers an updatable to receive block time updates
func (dct *DummyChainTracker) RegisterForBlockTimeUpdates(updatable blockTimeUpdatable) {}

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
func (dct *DummyChainTracker) AddBlockGap(newData time.Duration, blocks uint64) {}

func (dct *DummyChainTracker) IsDummy() bool {
	return true
}
