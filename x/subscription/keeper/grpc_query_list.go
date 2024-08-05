package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) List(goCtx context.Context, req *types.QueryListRequest) (*types.QueryListResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var allSubsInfo []types.ListInfoStruct
	ctx := sdk.UnwrapSDKContext(goCtx)

	subsIndices := k.GetAllSubscriptionsIndices(ctx)

	for _, consumer := range subsIndices {
		var sub types.Subscription
		if found := k.subsFS.FindEntry(ctx, consumer, uint64(ctx.BlockHeight()), &sub); !found {
			return nil, utils.LavaFormatError("failed to get subscription",
				fmt.Errorf("subscription not found"),
				utils.Attribute{Key: "consumer", Value: consumer},
				utils.Attribute{Key: "block", Value: ctx.BlockHeight()},
			)
		}

		subInfoStruct := types.ListInfoStruct{
			Consumer:            sub.Consumer,
			Plan:                sub.PlanIndex,
			DurationTotal:       sub.DurationTotal,
			DurationLeft:        sub.DurationLeft,
			MonthExpiry:         sub.MonthExpiryTime,
			MonthCuTotal:        sub.MonthCuTotal,
			MonthCuLeft:         sub.MonthCuLeft,
			DurationBought:      sub.DurationBought,
			Cluster:             sub.Cluster,
			AutoRenewalNextPlan: sub.AutoRenewalNextPlan,
			FutureSubscription:  sub.FutureSubscription,
			Credit:              &sub.Credit,
		}

		allSubsInfo = append(allSubsInfo, subInfoStruct)
	}

	return &types.QueryListResponse{SubsInfo: allSubsInfo}, nil
}

func (k Keeper) GetAllSubscriptionsIndices(ctx sdk.Context) []string {
	return k.subsFS.GetAllEntryIndices(ctx)
}
