package badgegenerator

import (
	"context"
	"fmt"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/lavanet/lava/protocol/chaintracker"
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
	epochUpdater.RegisterEpochUpdatable(ctx, epochUpdatable)
}
