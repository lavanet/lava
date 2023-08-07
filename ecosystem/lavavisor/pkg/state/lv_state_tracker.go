package lvstatetracker

import (
	"context"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	lvchaintracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg/state/chaintracker"
	"github.com/lavanet/lava/utils"
)

const (
	BlocksToSaveLavaChainTracker   = 1 // we only need the latest block
	TendermintConsensusParamsQuery = "consensus_params"
)

type StateTracker struct {
	chainTracker         *lvchaintracker.ChainTracker
	registrationLock     sync.RWMutex
	newLavaBlockUpdaters map[string]Updater
	eventTracker         *EventTracker
}

type Updater interface {
	Update(int64)
	UpdaterKey() string
}

func NewStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher lvchaintracker.ChainFetcher) (ret *StateTracker, err error) {
	// validate chainId
	status, err := clientCtx.Client.Status(ctx)
	if err != nil {
		return nil, err
	}
	if txFactory.ChainID() != status.NodeInfo.Network {
		return nil, utils.LavaFormatError("Chain ID mismatch", nil, utils.Attribute{Key: "--chain-id", Value: txFactory.ChainID()}, utils.Attribute{Key: "Node chainID", Value: status.NodeInfo.Network})
	}

	eventTracker := &EventTracker{clientCtx: clientCtx}
	err = eventTracker.updateBlockResults(0)
	if err != nil {
		return nil, err
	}
	cst := &StateTracker{newLavaBlockUpdaters: map[string]Updater{}, eventTracker: eventTracker}
	resultConsensusParams, err := clientCtx.Client.ConsensusParams(ctx, nil) // nil returns latest
	if err != nil {
		return nil, err
	}
	chainTrackerConfig := lvchaintracker.ChainTrackerConfig{
		NewLatestCallback: cst.newLavaBlock,
		BlocksToSave:      BlocksToSaveLavaChainTracker,
		AverageBlockTime:  time.Duration(resultConsensusParams.ConsensusParams.Block.TimeIotaMs) * time.Millisecond,
		ServerBlockMemory: BlocksToSaveLavaChainTracker,
	}
	cst.chainTracker, err = lvchaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
	return cst, err
}

func (st *StateTracker) newLavaBlock(latestBlock int64, hash string) {
	// go over the registered updaters and trigger update
	st.registrationLock.RLock()
	defer st.registrationLock.RUnlock()
	// first update event tracker
	st.eventTracker.updateBlockResults(latestBlock)
	// after events were updated we can trigger updaters
	for _, updater := range st.newLavaBlockUpdaters {
		updater.Update(latestBlock)
	}
}

func (st *StateTracker) RegisterForUpdates(ctx context.Context, updater Updater) Updater {
	st.registrationLock.Lock()
	defer st.registrationLock.Unlock()
	existingUpdater, ok := st.newLavaBlockUpdaters[updater.UpdaterKey()]
	if !ok {
		st.newLavaBlockUpdaters[updater.UpdaterKey()] = updater
		existingUpdater = updater
	}
	return existingUpdater
}
