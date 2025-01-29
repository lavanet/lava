package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) EstimatedPoolsRewards(goCtx context.Context, req *types.QueryEstimatedPoolsRewardsRequest) (*types.QueryEstimatedRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	oldCtx := sdk.UnwrapSDKContext(goCtx)
	ctx, _ := oldCtx.CacheContext() // verify the original ctx is not changed

	res := types.QueryEstimatedRewardsResponse{}

	// we use events to get the detailed info about the rewards (for the provider only use-case)
	// we reset the context's event manager to have a "clean slate"
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// trigger subs
	subs := k.GetAllSubscriptionsIndices(ctx)
	for _, sub := range subs {
		k.advanceMonth(ctx, []byte(sub))
	}

	// get all CU tracker timers (used to keep data on subscription rewards) and distribute all subscription rewards
	gs := k.ExportCuTrackerTimers(ctx)
	for _, timer := range gs.BlockEntries {
		k.RewardAndResetCuTracker(ctx, []byte(timer.Key), timer.Data)
	}

	// distribute bonus and IPRPC rewards
	k.rewardsKeeper.DistributeMonthlyBonusRewards(ctx)

	info, _ := k.getRewardsInfoFromEvents(ctx, "")

	res.Info = info

	// get the last IPRPC rewards distribution block
	rewardsDistributionBlock, err := k.rewardsKeeper.GetLastRewardsBlock(ctx)
	if err != nil {
		utils.LavaFormatError("failed to get last rewards block for pools rewards estimation", err)
	}
	res.RecommendedBlock = rewardsDistributionBlock - 1

	return &res, nil
}
