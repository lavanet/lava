package keeper

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/subscription/types"
	timertypes "github.com/lavanet/lava/x/timerstore/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) NextToMonthExpiry(goCtx context.Context, req *types.QueryNextToMonthExpiryRequest) (*types.QueryNextToMonthExpiryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	gs := k.subsTS.Export(ctx)
	if len(gs.TimeEntries) == 0 {
		return &types.QueryNextToMonthExpiryResponse{}, nil
	}

	currentTime := uint64(time.Now().Unix())
	firstTimer := getTimerInfo(gs.TimeEntries[0], currentTime)

	subs := []types.TimerExpiryInfo{}
	for _, timer := range gs.TimeEntries {
		if firstTimer.MonthExpiry < timer.Value {
			break
		}
		subs = append(subs, getTimerInfo(timer, currentTime))
	}

	return &types.QueryNextToMonthExpiryResponse{Subscriptions: subs}, nil
}

func getTimerInfo(timer timertypes.GenesisTimerEntry, currentTime uint64) types.TimerExpiryInfo {
	return types.TimerExpiryInfo{
		Consumer:    timer.Key,
		MonthExpiry: timer.Value,
		TimeLeft:    timer.Value - currentTime,
	}
}
