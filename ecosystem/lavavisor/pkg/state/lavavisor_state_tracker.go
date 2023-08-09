package lvstatetracker

import (
	"context"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/utils"
	protocoltypes "github.com/lavanet/lava/x/protocol/types"
)

type LavaVisorStateTracker struct {
	stateQuery *statetracker.StateQuery
	*statetracker.StateTracker
}

func NewLavaVisorStateTracker(ctx context.Context, txFactory tx.Factory, clientCtx client.Context, chainFetcher chaintracker.ChainFetcher) (lvst *LavaVisorStateTracker, err error) {
	stateTrackerBase, err := statetracker.NewStateTracker(ctx, txFactory, clientCtx, chainFetcher)
	if err != nil {
		return nil, err
	}

	lst := &LavaVisorStateTracker{stateQuery: statetracker.NewStateQuery(ctx, clientCtx), StateTracker: stateTrackerBase}
	return lst, nil
}

func (lst *LavaVisorStateTracker) RegisterForVersionUpdates(ctx context.Context, version *protocoltypes.Version, versionValidator statetracker.VersionValidationInf) {
	versionUpdater := statetracker.NewVersionUpdater(lst.stateQuery, lst.StateTracker.GetEventTracker(), version, versionValidator)
	versionUpdaterRaw := lst.StateTracker.RegisterForUpdates(ctx, versionUpdater)
	versionUpdater, ok := versionUpdaterRaw.(*statetracker.VersionUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from LavaVisor's RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: versionUpdaterRaw})
	}
	versionUpdater.RegisterVersionUpdatable()
}

func (lst *LavaVisorStateTracker) GetProtocolVersion(ctx context.Context) (*protocoltypes.Version, error) {
	return lst.stateQuery.GetProtocolVersion(ctx)
}
