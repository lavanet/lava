package lvstatetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	lvchaintracker "github.com/lavanet/lava/ecosystem/lavavisor/pkg/state/chaintracker"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type LavaVisorStateTracker struct {
	stateQuery *StateQuery
	*StateTracker
}

func NewLavaVisorStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher lvchaintracker.ChainFetcher) (lvst *LavaVisorStateTracker, err error) {
	stateTrackerBase, err := NewStateTracker(ctx, txFactory, clientCtx, chainFetcher)
	if err != nil {
		return nil, err
	}

	lst := &LavaVisorStateTracker{stateQuery: NewStateQuery(ctx, clientCtx), StateTracker: stateTrackerBase}
	return lst, nil
}

func (lst *LavaVisorStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, lavavisorPath string, currentBinary string, autoDownload bool, providers ProviderListener) {
	versionUpdater := NewVersionUpdater(lst.stateQuery, lst.eventTracker, version, lavavisorPath, currentBinary, autoDownload, providers)
	versionUpdaterRaw := lst.StateTracker.RegisterForUpdates(ctx, versionUpdater)
	versionUpdater, ok := versionUpdaterRaw.(*VersionUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from LavaVisor's RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: versionUpdaterRaw})
	}
	versionUpdater.RegisterVersionUpdatable()
}

func (lst *LavaVisorStateTracker) GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error) {
	return lst.stateQuery.GetProtocolVersion(ctx)
}
