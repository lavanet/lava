package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
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

	// support legacy
	project, vrfpk_proj, err := k.GetProjectData(ctx, userAddr, req.ChainID, epochStart)
	if err == nil {
		plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, project.GetSubscription())
		if err != nil {
			return nil, err
		}

		planPolicy := plan.GetPlanPolicy()

		// geolocation is a bitmap. common denominator can be calculated with logical AND
		geolocation := project.AdminPolicy.GeolocationProfile & project.SubscriptionPolicy.GeolocationProfile & planPolicy.GeolocationProfile

		allowedCU := commontypes.FindUint64Min([]uint64{
			project.AdminPolicy.GetEpochCuLimit(),
			project.SubscriptionPolicy.GetEpochCuLimit(),
			planPolicy.GetEpochCuLimit(),
		})

		return &types.QueryUserEntryResponse{Consumer: epochstoragetypes.StakeEntry{
			Geolocation: geolocation,
			Address:     req.Address,
			Chain:       req.ChainID,
			Vrfpk:       vrfpk_proj,
		}, MaxCU: allowedCU}, nil
	}

	existingEntry, err := k.epochStorageKeeper.GetStakeEntryForClientEpoch(ctx, req.ChainID, userAddr, epochStart)
	if err != nil {
		return nil, err
	}

	maxCU, err := k.ClientMaxCUProviderForBlock(ctx, req.Block, existingEntry)
	if err != nil {
		return nil, err
	}
	return &types.QueryUserEntryResponse{Consumer: *existingEntry, MaxCU: maxCU}, nil
}
