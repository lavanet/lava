package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/pairing/types"
	subsciptiontypes "github.com/lavanet/lava/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) MonthlyPayout(goCtx context.Context, req *types.QueryMonthlyPayoutRequest) (*types.QueryMonthlyPayoutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	providerAddr, err := sdk.AccAddressFromBech32(req.Provider)
	if err != nil {
		return nil, utils.LavaFormatError("invalid provider address", err,
			utils.Attribute{Key: "provider", Value: req.Provider},
		)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var amount uint64

	subs := k.subscriptionKeeper.GetAllSubscriptionsIndices(ctx)

	for _, sub := range subs {
		trackedCuInds := k.subscriptionKeeper.GetAllSubTrackedCuIndices(ctx, subsciptiontypes.CuTrackerKey(sub, req.Provider, ""))

		for _, trackCU := range trackedCuInds {
			_, _, chainID := subsciptiontypes.DecodeCuTrackerKey(trackCU)

			subObj, found := k.subscriptionKeeper.GetSubscription(ctx, sub)
			if !found {
				return nil, utils.LavaFormatError("cannot get tracked CU", fmt.Errorf("subscription not found"),
					utils.Attribute{Key: "sub", Value: sub},
					utils.Attribute{Key: "provider", Value: req.Provider},
					utils.Attribute{Key: "chain_id", Value: chainID},
				)
			}

			plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, sub, subObj.Block)
			if err != nil {
				return nil, err
			}

			providerCu, found, _ := k.subscriptionKeeper.GetTrackedCu(ctx, sub, req.Provider, chainID, subObj.Block)
			if !found {
				continue
			}
			totalMonthlyReward := k.subscriptionKeeper.CalcTotalMonthlyReward(ctx, plan, providerCu, subObj.MonthCuTotal-subObj.MonthCuLeft)

			// calculate only the provider reward
			providerReward, err := k.dualstakingKeeper.RewardProvidersAndDelegators(ctx, providerAddr, chainID, totalMonthlyReward, subsciptiontypes.ModuleName, true)
			if err != nil {
				return nil, err
			}

			amount += providerReward.Uint64()
		}
	}

	return &types.QueryMonthlyPayoutResponse{Amount: amount}, nil
}
