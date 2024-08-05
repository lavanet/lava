package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) UserEntry(goCtx context.Context, req *types.QueryUserEntryRequest) (*types.QueryUserEntryResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// Process the query
	userAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "invalid address")
	}

	epochStart, _, err := k.epochStorageKeeper.GetEpochStartForBlock(ctx, req.Block)
	if err != nil {
		return nil, err
	}

	project, err := k.GetProjectData(ctx, userAddr, req.ChainID, epochStart)
	if err != nil {
		return nil, err
	}

	sub, found := k.subscriptionKeeper.GetSubscription(ctx, project.GetSubscription())
	if !found {
		return nil, fmt.Errorf("could not find subscription with address %s", project.GetSubscription())
	}

	plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, project.GetSubscription(), uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	planPolicy := plan.GetPlanPolicy()
	policies := []*planstypes.Policy{&planPolicy, project.AdminPolicy, project.SubscriptionPolicy}
	// geolocation is a bitmap. common denominator can be calculated with logical AND
	geolocation, err := k.CalculateEffectiveGeolocationFromPolicies(policies)
	if err != nil {
		return nil, err
	}
	allowedCU, allowedCUTotal := k.CalculateEffectiveAllowedCuPerEpochFromPolicies(policies, project.GetUsedCu(), sub.GetMonthCuLeft())
	if !planstypes.VerifyTotalCuUsage(allowedCUTotal, project.GetUsedCu()) {
		allowedCU = 0
	}

	return &types.QueryUserEntryResponse{Consumer: epochstoragetypes.StakeEntry{
		Geolocation: geolocation,
		Address:     req.Address,
		Chain:       req.ChainID,
	}, MaxCU: allowedCU}, nil
}
