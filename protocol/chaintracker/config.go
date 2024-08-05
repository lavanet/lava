package chaintracker

import (
	"time"

	"github.com/lavanet/lava/v2/protocol/metrics"
)

const (
	DefualtAssumedBlockMemory      = 20
	DefaultBlockCheckpointDistance = 100
)

type ChainTrackerConfig struct {
	ForkCallback             func(block int64)                                 // a function to be called when a fork is detected
	NewLatestCallback        func(blockFrom int64, blockTo int64, hash string) // a function to be called when a new block is detected
	ConsistencyCallback      func(oldBlock int64, block int64)
	OldBlockCallback         func(latestBlockTime time.Time)
	ServerAddress            string // if not empty will open up a grpc server for that address
	BlocksToSave             uint64
	AverageBlockTime         time.Duration // how often to query latest block
	ServerBlockMemory        uint64
	BlocksCheckpointDistance uint64 // this causes the chainTracker to trigger it's checkpoint every X blocks
	Pmetrics                 *metrics.ProviderMetricsManager
	PollingTimeMultiplier    int
}

func (cnf *ChainTrackerConfig) validate() error {
	if cnf.BlocksToSave == 0 {
		return InvalidConfigErrorBlocksToSave
	}
	if cnf.AverageBlockTime == 0 {
		return InvalidConfigBlockTime
	}

	if cnf.ServerBlockMemory == 0 {
		cnf.ServerBlockMemory = DefualtAssumedBlockMemory
	}
	if cnf.BlocksCheckpointDistance == 0 {
		cnf.BlocksCheckpointDistance = DefaultBlockCheckpointDistance
	}
	// TODO: validate address is in the right format if not empty
	return nil
}
