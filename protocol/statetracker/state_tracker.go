package statetracker

import (
	"context"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v4/protocol/chainlib"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	updaters "github.com/lavanet/lava/v4/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v4/utils"
	specutils "github.com/lavanet/lava/v4/utils/keeper"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

const (
	BlocksToSaveLavaChainTracker   = 1 // we only need the latest block
	TendermintConsensusParamsQuery = "consensus_params"
	MAINNET_SPEC                   = "LAVA"
	TESTNET_SPEC                   = "LAV1"
)

var (
	lavaSpecName = ""
	// TODO: add a governance param change that indicates what spec id belongs to lava.
	LavaSpecOptions = []string{TESTNET_SPEC, MAINNET_SPEC}
)

// ConsumerStateTracker CSTis a class for tracking consumer data from the lava blockchain, such as epoch changes.
// it allows also to query specific data form the blockchain and acts as a single place to send transactions
type StateTracker struct {
	chainTracker         *chaintracker.ChainTracker
	registrationLock     sync.RWMutex
	newLavaBlockUpdaters map[string]Updater
	EventTracker         *updaters.EventTracker
	AverageBlockTime     time.Duration
}

type Updater interface {
	Update(int64)
	Reset(int64)
	UpdaterKey() string
}

type SpecUpdaterInf interface {
	RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error
}

// Either register for spec updates or set spec for offline spec, used in both consumer and provider process
func RegisterForSpecUpdatesOrSetStaticSpec(ctx context.Context, chainParser chainlib.ChainParser, specPath string, rpcEndpoint lavasession.RPCEndpoint, specUpdaterInf SpecUpdaterInf) error {
	if specPath == "" {
		return specUpdaterInf.RegisterForSpecUpdates(ctx, chainParser, rpcEndpoint)
	}

	// offline spec mode.
	parsedOfflineSpec, err := specutils.GetSpecsFromPath(specPath, rpcEndpoint.ChainID, nil, nil)
	if err != nil {
		return utils.LavaFormatError("failed loading offline spec", err, utils.LogAttr("spec_path", specPath), utils.LogAttr("spec_id", rpcEndpoint.ChainID))
	}
	utils.LavaFormatInfo("Loaded offline spec successfully", utils.LogAttr("spec_path", specPath), utils.LogAttr("chain_id", parsedOfflineSpec.Index))
	chainParser.SetSpec(parsedOfflineSpec)

	return nil
}

func GetLavaSpecWithRetry(ctx context.Context, specQueryClient spectypes.QueryClient) (*spectypes.QueryGetSpecResponse, error) {
	var specResponse *spectypes.QueryGetSpecResponse
	var err error
	for i := 0; i < updaters.BlockResultRetry; i++ {
		if lavaSpecName == "" { // spec name is not initialized, try fetching specs.
			for _, specId := range LavaSpecOptions {
				specResponse, err = specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
					ChainID: specId,
				})
				if err != nil {
					continue
				}
				utils.LavaFormatInfo("Lava Spec found on chain", utils.LogAttr("SpecId", specId))
				lavaSpecName = specId
				break
			}
		} else {
			specResponse, err = specQueryClient.Spec(ctx, &spectypes.QueryGetSpecRequest{
				ChainID: lavaSpecName,
			})
		}
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	return specResponse, err
}

func NewStateTracker(ctx context.Context, txFactory tx.Factory, stateQuery *updaters.StateQuery, chainFetcher chaintracker.ChainFetcher, blockNotFoundCallback func(latestBlockTime time.Time)) (ret *StateTracker, err error) {
	// validate chainId
	status, err := stateQuery.Status(ctx)
	if err != nil {
		return nil, utils.LavaFormatError("failed getting status", err)
	}
	if txFactory.ChainID() != status.NodeInfo.Network {
		return nil, utils.LavaFormatError("Chain ID mismatch", nil, utils.Attribute{Key: "--chain-id", Value: txFactory.ChainID()}, utils.Attribute{Key: "Node chainID", Value: status.NodeInfo.Network})
	}

	eventTracker := &updaters.EventTracker{StateQuery: stateQuery}
	for i := 0; i < updaters.BlockResultRetry; i++ {
		err = eventTracker.UpdateBlockResults(0)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond * time.Duration(i+1)) // need this so it doesn't just spam the attempts, and tendermint fails getting block results pretty often
	}
	if err != nil {
		return nil, utils.LavaFormatError("failed getting blockResults after retries", err)
	}
	specQueryClient := stateQuery.GetSpecQueryClient()
	specResponse, err := GetLavaSpecWithRetry(ctx, specQueryClient)
	if err != nil {
		utils.LavaFormatFatal("failed querying lava spec for state tracker", err)
	}
	cst := &StateTracker{newLavaBlockUpdaters: map[string]Updater{}, EventTracker: eventTracker}
	chainTrackerConfig := chaintracker.ChainTrackerConfig{
		NewLatestCallback:     cst.newLavaBlock,
		OldBlockCallback:      blockNotFoundCallback,
		BlocksToSave:          BlocksToSaveLavaChainTracker,
		AverageBlockTime:      time.Duration(specResponse.Spec.AverageBlockTime) * time.Millisecond,
		ServerBlockMemory:     25 + BlocksToSaveLavaChainTracker,
		PollingTimeMultiplier: chaintracker.LavaPollingMultiplierFrequency,
	}
	cst.AverageBlockTime = chainTrackerConfig.AverageBlockTime
	cst.chainTracker, err = chaintracker.NewChainTracker(ctx, chainFetcher, chainTrackerConfig)
	cst.chainTracker.StartAndServe(ctx)
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

func (st *StateTracker) newLavaBlock(blockFrom int64, blockTo int64, hash string) {
	// go over the registered updaters and trigger update
	st.registrationLock.RLock()
	defer st.registrationLock.RUnlock()
	// if we had a huge gap
	if time.Duration(blockTo-blockFrom)*st.AverageBlockTime > time.Hour { // if we are 1H behind
		// in case we have a huge gap we launch a reset on the state of all the updaters. as the state is no longer valid.
		// this can be caused by a huge catch up on blocks after a halt or a sync state on the node. sometimes pruning the blocks the protocol requires.
		// therefore we need to reset the state and fetch all information from the chain
		// first update the event tracker to latest block.
		err := st.EventTracker.UpdateBlockResults(blockTo)
		if err != nil {
			utils.LavaFormatError("failing to fetch latest result after gap", err, utils.LogAttr("blockFrom", blockFrom), utils.LogAttr("blockTo", blockTo))
		}
		// reset will try to reset the updaters. if it fails it will retry every update until it succeeds.
		for _, updater := range st.newLavaBlockUpdaters {
			updater.Reset(blockTo)
		}
		return // return after state has been reset.
	}

	for block := blockFrom + 1; block <= blockTo; block++ {
		// first update event tracker
		err := st.EventTracker.UpdateBlockResults(block)
		if err != nil {
			utils.LavaFormatWarning("calling update without updated events tracker", err)
		}
		// after events were updated we can trigger updaters
		for _, updater := range st.newLavaBlockUpdaters {
			updater.Update(block)
		}
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
func (st *StateTracker) GetEventTracker() *updaters.EventTracker {
	return st.EventTracker
}

func IsLavaNativeSpec(checked string) bool {
	for _, nativeLavaChain := range LavaSpecOptions {
		if checked == nativeLavaChain {
			return true
		}
	}
	return false
}
