package keeper

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	fixationtypes "github.com/lavanet/lava/x/fixationstore/types"
	"github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) NextToMonthExpiry(goCtx context.Context, req *types.QueryNextToMonthExpiryRequest) (*types.QueryNextToMonthExpiryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	subAddrs, expiries := k.subsTS.GetFrontTimers(ctx, fixationtypes.BlockTime)
	if len(subAddrs) == 0 {
		return &types.QueryNextToMonthExpiryResponse{}, nil
	}

	currentTime := uint64(time.Now().Unix())

	subs := []types.TimerExpiryInfo{}
	for i := range subAddrs {
		addr := string(subAddrs[i])
		subs = append(subs, createTimerInfo(addr, expiries[i], currentTime))
	}

	return &types.QueryNextToMonthExpiryResponse{Subscriptions: subs}, nil
}

func createTimerInfo(sub string, expiry uint64, currentTime uint64) types.TimerExpiryInfo {
	return types.TimerExpiryInfo{
		Consumer:    sub,
		MonthExpiry: expiry,
		TimeLeft:    expiry - currentTime,
	}
}
