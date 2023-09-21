package statetracker

import (
	"context"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/utils"
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
	EventTracker         *EventTracker
}

type Updater interface {
	Update(int64)
	UpdaterKey() string
}

func NewStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (ret *StateTracker, err error) {
	// validate chainId
	status, err := clientCtx.Client.Status(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("failed getting status", err)
	}
	if txFactory.ChainID() != status.NodeInfo.Network {
		return nil, utils.LavaFormatError("Chain ID mismatch", nil, utils.Attribute{Key: "--chain-id", Value: txFactory.ChainID()}, utils.Attribute{Key: "Node chainID", Value: status.NodeInfo.Network})
	}

	eventTracker := &EventTracker{clientCtx: clientCtx}
	err = eventTracker.updateBlockResults(0)
	if err != nil {
		return nil, utils.LavaFormatError("failed getting blockResults", err)
	}

	// TODO: fix average block time.
	averageBlockTime := 1
	if utils.ExtendedLogLevel == "production" {
		averageBlockTime = 30
	}

	cst := &StateTracker{newLavaBlockUpdaters: map[string]Updater{}, EventTracker: eventTracker}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		NewLatestCallback: cst.newLavaBlock,
		BlocksToSave:      BlocksToSaveLavaChainTracker,
		AverageBlockTime:  time.Duration(averageBlockTime) * time.Second,
		ServerBlockMemory: BlocksToSaveLavaChainTracker,
	}
	cst.chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
	return cst, err
}

func (st *StateTracker) newLavaBlock(latestBlock int64, hash string) {
	// go over the registered updaters and trigger update
	st.registrationLock.RLock()
	defer st.registrationLock.RUnlock()
	// first update event tracker
	st.EventTracker.updateBlockResults(latestBlock)
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

// For lavavisor access
func (s *StateTracker) GetEventTracker() *EventTracker {
	return s.EventTracker
}
