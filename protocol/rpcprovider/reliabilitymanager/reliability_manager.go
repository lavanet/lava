package reliabilitymanager

import "github.com/lavanet/lava/protocol/chaintracker"

type ReliabilityManager struct {
	chainTracker *chaintracker.ChainTracker
}

func (rm *ReliabilityManager) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, err error) {
	return rm.chainTracker.GetLatestBlockData(fromBlock, toBlock, specificBlock)
}

func (rm *ReliabilityManager) GetLatestBlockNum() int64 {
	return rm.chainTracker.GetLatestBlockNum()
}

func NewReliabilityManager(chainTracker *chaintracker.ChainTracker) *ReliabilityManager {
	rm := &ReliabilityManager{}
	rm.chainTracker = chainTracker
	return rm
}
