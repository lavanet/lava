package badgeserver

import (
	"context"
	"fmt"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/v4/protocol/chaintracker"
	"github.com/lavanet/lava/v4/protocol/lavasession"
	"github.com/lavanet/lava/v4/protocol/statetracker"
	"github.com/lavanet/lava/v4/protocol/statetracker/updaters"
	"github.com/lavanet/lava/v4/utils"
)

// adding 3 blocks delay, to update the epoch.
// the reason is for the sdk to wait until all providers
// are synced to the new epoch before SDK gets a new pairing list.
const AddBlockDelayForEpochUpdaterBadgeServer = 2

type BadgeStateTracker struct {
	stateQuery *updaters.EpochStateQuery
	*statetracker.StateTracker
	statetracker.ConsumerEmergencyTrackerInf
}

func NewBadgeStateTracker(ctx context.Context, clientCtx cosmosclient.Context, chainFetcher chaintracker.ChainFetcher, chainId string) (ret *BadgeStateTracker, err error) {
	emergencyTracker, blockNotFoundCallback := statetracker.NewEmergencyTracker(nil)
	txFactory := tx.Factory{}
	txFactory = txFactory.WithChainID(chainId)
	stateQuery := updaters.NewStateQuery(ctx, updaters.NewStateQueryAccessInst(clientCtx))
	stateTrackerBase, err := statetracker.NewStateTracker(ctx, txFactory, stateQuery, chainFetcher, blockNotFoundCallback)
	if err != nil {
		return nil, err
	}
	epochStateTracker := updaters.NewEpochStateQuery(stateQuery)

	badgeStateTracker := &BadgeStateTracker{
		StateTracker:                stateTrackerBase,
		stateQuery:                  epochStateTracker,
		ConsumerEmergencyTrackerInf: emergencyTracker,
	}

	badgeStateTracker.RegisterForEpochUpdates(ctx, emergencyTracker)
	err = badgeStateTracker.RegisterForDowntimeParamsUpdates(ctx, emergencyTracker)
	return badgeStateTracker, err
}

func (st *BadgeStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable updaters.EpochUpdatable) {
	epochUpdater := updaters.NewEpochUpdater(st.stateQuery)
	epochUpdaterRaw := st.StateTracker.RegisterForUpdates(ctx, epochUpdater)
	epochUpdater, ok := epochUpdaterRaw.(*updaters.EpochUpdater)
	if !ok {
		err := fmt.Errorf("invalid type")
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", err)
	}

	epochUpdater.RegisterEpochUpdatable(ctx, epochUpdatable, AddBlockDelayForEpochUpdaterBadgeServer)
}

func (st *BadgeStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable updaters.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	// register for spec updates sets spec and updates when a spec has been modified
	specUpdater := updaters.NewSpecUpdater(endpoint.ChainID, st.stateQuery, st.EventTracker)
	specUpdaterRaw := st.StateTracker.RegisterForUpdates(ctx, specUpdater)
	specUpdater, ok := specUpdaterRaw.(*updaters.SpecUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: specUpdaterRaw})
	}
	return specUpdater.RegisterSpecUpdatable(ctx, &specUpdatable, endpoint)
}

func (st *BadgeStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context, downtimeParamsUpdatable updaters.DowntimeParamsUpdatable) error {
	// register for downtimeParams updates sets downtimeParams and updates when downtimeParams has been changed
	downtimeParamsUpdater := updaters.NewDowntimeParamsUpdater(st.stateQuery, st.EventTracker)
	downtimeParamsUpdaterRaw := st.StateTracker.RegisterForUpdates(ctx, downtimeParamsUpdater)
	downtimeParamsUpdater, ok := downtimeParamsUpdaterRaw.(*updaters.DowntimeParamsUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: downtimeParamsUpdaterRaw})
	}

	return downtimeParamsUpdater.RegisterDowntimeParamsUpdatable(ctx, &downtimeParamsUpdatable)
}
