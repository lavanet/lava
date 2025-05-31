package rpcprovider

import (
	"time"

	"github.com/lavanet/lava/v5/protocol/chaintracker"
)

type MockChainTracker struct {
	latestBlock int64
	changeTime  time.Time
}

func (mct *MockChainTracker) GetLatestBlockData(fromBlock int64, toBlock int64, specificBlock int64) (latestBlock int64, requestedHashes []*chaintracker.BlockStore, changeTime time.Time, err error) {
	return mct.latestBlock, nil, mct.changeTime, nil
}

func (mct *MockChainTracker) GetLatestBlockNum() (int64, time.Time) {
	return mct.latestBlock, mct.changeTime
}

func (mct *MockChainTracker) SetLatestBlock(newLatest int64, changeTime time.Time) {
	mct.latestBlock = newLatest
	mct.changeTime = changeTime
}
