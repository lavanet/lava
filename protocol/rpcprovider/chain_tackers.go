package rpcprovider

import (
	"sync"

	"github.com/lavanet/lava/v2/protocol/chaintracker"
	"github.com/lavanet/lava/v2/utils"
)

type ChainTrackers struct {
	stateTrackersPerChain sync.Map
}

func (ct *ChainTrackers) GetTrackerPerChain(specID string) (chainTracker *chaintracker.ChainTracker, found bool) {
	chainTrackerInf, found := ct.stateTrackersPerChain.Load(specID)
	if !found {
		return nil, found
	}
	var ok bool
	chainTracker, ok = chainTrackerInf.(*chaintracker.ChainTracker)
	if !ok {
		utils.LavaFormatFatal("invalid usage of syncmap, could not cast result into a chaintracker", nil)
	}
	return chainTracker, true
}

func (ct *ChainTrackers) SetTrackerForChain(specId string, chainTracker *chaintracker.ChainTracker) {
	ct.stateTrackersPerChain.Store(specId, chainTracker)
}

func (ct *ChainTrackers) GetLatestBlockNumForSpec(specID string) int64 {
	chainTracker, found := ct.GetTrackerPerChain(specID)
	if !found {
		return 0
	}
	latestBlock, _ := chainTracker.GetLatestBlockNum()
	return latestBlock
}
