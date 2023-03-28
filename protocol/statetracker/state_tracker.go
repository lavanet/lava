package statetracker

import (
	"context"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
)

const (
	BlocksToSaveLavaChainTracker   = 1 // we only need the latest block
	TendermintConsensusParamsQuery = "consensus_params"
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type StateTracker struct {
	chainTracker         *chaintracker.ChainTracker
	registrationLock     sync.RWMutex
	newLavaBlockUpdaters map[string]Updater
}

type Updater interface {
	Update(int64)
	UpdaterKey() string
}

func NewStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (ret *StateTracker, err error) {
	cst := &StateTracker{newLavaBlockUpdaters: map[string]Updater{}}
	resultConsensusParams, err := clientCtx.Client.ConsensusParams(ctx, nil) // nil returns latest
	if err != nil {
		return nil, err
	}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		NewLatestCallback: cst.newLavaBlock,
		BlocksToSave:      BlocksToSaveLavaChainTracker,
		AverageBlockTime:  time.Duration(resultConsensusParams.ConsensusParams.Block.TimeIotaMs) * time.Millisecond,
		ServerBlockMemory: BlocksToSaveLavaChainTracker,
	}
	cst.chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
	return cst, err
}

func (cst *StateTracker) newLavaBlock(latestBlock int64) {
	// go over the registered updaters and trigger update
	cst.registrationLock.RLock()
	defer cst.registrationLock.RUnlock()
	for _, updater := range cst.newLavaBlockUpdaters {
		updater.Update(latestBlock)
	}
}

func (cst *StateTracker) RegisterForUpdates(ctx context.Context, updater Updater) Updater {
	cst.registrationLock.Lock()
	defer cst.registrationLock.Unlock()
	existingUpdater, ok := cst.newLavaBlockUpdaters[updater.UpdaterKey()]
	if !ok {
		cst.newLavaBlockUpdaters[updater.UpdaterKey()] = updater
		existingUpdater = updater
	}
	return existingUpdater
}
