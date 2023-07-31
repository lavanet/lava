package lvstatetracker

import (
	"context"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

const (
	BlocksToSaveLavaChainTracker   = 1 // we only need the latest block
	TendermintConsensusParamsQuery = "consensus_params"
)

type StateTracker struct {
	registrationLock     sync.RWMutex
	newLavaBlockUpdaters map[string]Updater
	eventTracker         *EventTracker
}

type LavavisorStateTracker struct {
	stateQuery *StateQuery
	*StateTracker
}

type Updater interface {
	Update(int64)
	UpdaterKey() string
}

func NewStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (ret *StateTracker, err error) {
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

// LavaVisor state tracker & version updater
const (
	CallbackKeyForVersionUpdate = "version-update"
)

type VersionStateQuery interface {
	GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error)
}

type VersionUpdater struct {
	lock              sync.RWMutex
	eventTracker      *EventTracker
	versionStateQuery VersionStateQuery
	lastKnownVersion  *protocoltypes.Version
}

func NewVersionUpdater(versionStateQuery VersionStateQuery, eventTracker *EventTracker, version *protocoltypes.Version) *VersionUpdater {
	return &VersionUpdater{versionStateQuery: versionStateQuery, eventTracker: eventTracker, lastKnownVersion: version}
}

func (vu *VersionUpdater) UpdaterKey() string {
	return CallbackKeyForVersionUpdate
}
