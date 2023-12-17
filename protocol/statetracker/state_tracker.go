package statetracker

import (
	"context"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/utils"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	BlocksToSaveLavaChainTracker   = 1 // we only need the latest block
	TendermintConsensusParamsQuery = "consensus_params"
	BlockResultRetry               = 20
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type StateTracker struct {
	chainTracker         *chaintracker.ChainTracker
	registrationLock     sync.RWMutex
	newLavaBlockUpdaters map[string]Updater
	EventTracker         *EventTracker
	AverageBlockTime     time.Duration
}

type Updater interface {
	Update(int64)
	UpdaterKey() string
}

func NewStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher, blockNotFoundCallback func(latestBlockTime time.Time)) (ret *StateTracker, err error) {
	// validate chainId
	status, err := clientCtx.Client.Status(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("failed getting status", err)
	}
	if txFactory.ChainID() != status.NodeInfo.Network {
		return nil, utils.LavaFormatError("Chain ID mismatch", nil, utils.Attribute{Key: "--chain-id", Value: txFactory.ChainID()}, utils.Attribute{Key: "Node chainID", Value: status.NodeInfo.Network})
	}

	eventTracker := &EventTracker{clientCtx: clientCtx}
	for i := 0; i < BlockResultRetry; i++ {
		err = eventTracker.updateBlockResults(0)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond * time.Duration(i+1)) // need this so it doesn't just spam the attempts, and tendermint fails getting block results pretty often
	}
	if err != nil {
		return nil, utils.LavaFormatError("failed getting blockResults after retries", err)
	}
	specQueryClient := spectypes.NewQueryClient(clientCtx)
	var specResponse *spectypes.QueryGetSpecResponse
	for i := 0; i < BlockResultRetry; i++ {
		specResponse, err = specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
			ChainID: "LAV1",
		})
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		utils.LavaFormatFatal("failed querying lava spec for state tracker", err)
	}
	cst := &StateTracker{newLavaBlockUpdaters: map[string]Updater{}, EventTracker: eventTracker}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		NewLatestCallback: cst.newLavaBlock,
		OldBlockCallback:  blockNotFoundCallback,
		BlocksToSave:      BlocksToSaveLavaChainTracker,
		AverageBlockTime:  time.Duration(specResponse.Spec.AverageBlockTime) * time.Millisecond,
		ServerBlockMemory: 25 + BlocksToSaveLavaChainTracker,
	}
	cst.AverageBlockTime = chainTrackerConfig.AverageBlockTime
	cst.chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
	cst.chainTracker.RegisterForBlockTimeUpdates(cst) // registering for block time updates.
	return cst, err
}

func (st *StateTracker) UpdateBlockTime(blockTime time.Duration) {
	st.registrationLock.Lock()
	defer st.registrationLock.Unlock()
	st.AverageBlockTime = blockTime
}

func (st *StateTracker) GetAverageBlockTime() time.Duration {
	st.registrationLock.RLock()
	defer st.registrationLock.RUnlock()
	return st.AverageBlockTime
}

func (st *StateTracker) newLavaBlock(latestBlock int64, hash string) {
	// go over the registered updaters and trigger update
	st.registrationLock.RLock()
	defer st.registrationLock.RUnlock()
	// first update event tracker
	err := st.EventTracker.updateBlockResults(latestBlock)
	if err != nil {
		utils.LavaFormatWarning("calling update without updated events tracker", err)
	}
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
func (st *StateTracker) GetEventTracker() *EventTracker {
	return st.EventTracker
}
