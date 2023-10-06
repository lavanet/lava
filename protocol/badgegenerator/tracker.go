package badgegenerator

import (
	"context"
	"fmt"

	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/protocol/statetracker"
	"github.com/lavanet/lava/utils"
)

type BadgeStateTracker struct {
	stateQuery *statetracker.EpochStateQuery
	*statetracker.StateTracker
}

func NewBadgeStateTracker(ctx context.Context, clientCtx cosmosclient.Context, chainFetcher chaintracker.ChainFetcher, chainId string) (ret *BadgeStateTracker, err error) {
	txFactory := tx.Factory{}
	txFactory = txFactory.WithChainID(chainId)
	stateTrackerBase, err := statetracker.NewStateTracker(ctx, txFactory, clientCtx, chainFetcher)
	if err != nil {
		return nil, err
	}
	sq := statetracker.NewStateQuery(ctx, clientCtx)
	esq := statetracker.NewEpochStateQuery(sq)

	pst := &BadgeStateTracker{StateTracker: stateTrackerBase, stateQuery: esq}
	return pst, nil
}

func (st *BadgeStateTracker) RegisterForEpochUpdates(ctx context.Context, epochUpdatable statetracker.EpochUpdatable) {
	epochUpdater := statetracker.NewEpochUpdater(st.stateQuery)
	epochUpdaterRaw := st.StateTracker.RegisterForUpdates(ctx, epochUpdater)
	epochUpdater, ok := epochUpdaterRaw.(*statetracker.EpochUpdater)
	if !ok {
		err := fmt.Errorf("invalid type")
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", err)
	}

	epochUpdaterWithEmergencyRaw := st.StateTracker.RegisterForEmergencyModeUpdates(ctx, epochUpdater)
	epochUpdater, ok = epochUpdaterWithEmergencyRaw.(*statetracker.EpochUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: epochUpdaterWithEmergencyRaw})
	}
	epochUpdater.RegisterEpochUpdatable(ctx, epochUpdatable)
}

func (st *BadgeStateTracker) RegisterForDowntimeParamsUpdates(ctx context.Context) error {
	downtimeParamsUpdater := statetracker.NewDowntimeParamsUpdater(st.stateQuery, st.EventTracker)
	downtimeParamsUpdaterRaw := st.StateTracker.RegisterForUpdates(ctx, downtimeParamsUpdater)
	downtimeParamsUpdater, ok := downtimeParamsUpdaterRaw.(*statetracker.DowntimeParamsUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: downtimeParamsUpdaterRaw})
	}

	downtimeParamsUpdatable := statetracker.DowntimeParamsUpdatable(st.StateTracker.GetChainTracker())

	err := downtimeParamsUpdater.RegisterDowntimeParamsUpdatable(ctx, &downtimeParamsUpdatable)
	if err != nil {
		utils.LavaFormatFatal("failed to register downtime params updatable", err)
	}

	return err
}

func (st *BadgeStateTracker) RegisterForSpecUpdates(ctx context.Context, specUpdatable statetracker.SpecUpdatable, endpoint lavasession.RPCEndpoint) error {
	// register for spec updates sets spec and updates when a spec has been modified
	specUpdater := statetracker.NewSpecUpdater(endpoint.ChainID, st.stateQuery, st.EventTracker)
	specUpdaterRaw := st.StateTracker.RegisterForUpdates(ctx, specUpdater)
	specUpdater, ok := specUpdaterRaw.(*statetracker.SpecUpdater)
	if !ok {
		utils.LavaFormatFatal("invalid updater type returned from RegisterForUpdates", nil, utils.Attribute{Key: "updater", Value: specUpdaterRaw})
	}
	return specUpdater.RegisterSpecUpdatable(ctx, &specUpdatable, endpoint)
}
