package keeper

import (
	"context"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/utils"
	"github.com/lavanet/lava/v2/x/pairing/types"
	subsciption "github.com/lavanet/lava/v2/x/subscription/keeper"
	subsciptiontypes "github.com/lavanet/lava/v2/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SubscriptionMonthlyPayout(goCtx context.Context, req *types.QuerySubscriptionMonthlyPayoutRequest) (*types.QuerySubscriptionMonthlyPayoutResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	var total uint64
	detailsMap := make(map[string][]*types.ProviderPayout)
	var details []*types.ChainIDPayout

	sub := req.Consumer
	subObj, found := k.subscriptionKeeper.GetSubscription(ctx, sub)
	if !found {
		return nil, utils.LavaFormatError("cannot get tracked CU", fmt.Errorf("subscription not found"),
			utils.Attribute{Key: "sub", Value: sub},
		)
	}

	plan, err := k.subscriptionKeeper.GetPlanFromSubscription(ctx, sub, subObj.Block)
	if err != nil {
		return nil, err
	}

	trackedCuInds := k.subscriptionKeeper.GetAllSubTrackedCuIndices(ctx, req.Consumer)

	for _, trackCU := range trackedCuInds {
		_, provider, chainID := subsciptiontypes.DecodeCuTrackerKey(trackCU)

		providerCu, found, _ := k.subscriptionKeeper.GetTrackedCu(ctx, sub, provider, chainID, subObj.Block)
		if !found {
			continue
		}
		totalTokenAmount := plan.Price.Amount
		totalCuTracked := subObj.MonthCuTotal - subObj.MonthCuLeft
		// Sanity check - totalCuTracked > 0
		if totalCuTracked <= 0 {
			return nil, utils.LavaFormatWarning("totalCuTracked is zero or negative", fmt.Errorf("critical: Attempt to divide by zero or negative number"),
				utils.LogAttr("subObj.MonthCuTotal", subObj.MonthCuTotal),
				utils.LogAttr("subObj.MonthCuLeft", subObj.MonthCuLeft),
				utils.LogAttr("totalCuTracked", totalCuTracked),
			)
		}

		if plan.Price.Amount.Quo(sdk.NewIntFromUint64(totalCuTracked)).GT(sdk.NewIntFromUint64(subsciption.LIMIT_TOKEN_PER_CU)) {
			totalTokenAmount = sdk.NewIntFromUint64(subsciption.LIMIT_TOKEN_PER_CU * totalCuTracked)
		}

		totalMonthlyReward := k.subscriptionKeeper.CalcTotalMonthlyReward(ctx, totalTokenAmount, providerCu, totalCuTracked)

		total += totalMonthlyReward.Uint64()
		if _, ok := detailsMap[chainID]; ok {
			payouts := detailsMap[chainID]
			payouts = append(payouts, &types.ProviderPayout{Provider: provider, Amount: totalMonthlyReward.Uint64()})
			detailsMap[chainID] = payouts
		} else {
			detailsMap[chainID] = []*types.ProviderPayout{{Provider: provider, Amount: totalMonthlyReward.Uint64()}}
		}
	}

	var chains []string
	for chainID := range detailsMap {
		chains = append(chains, chainID)
	}

	sort.Strings(chains)

	for _, chainID := range chains {
		details = append(details, &types.ChainIDPayout{ChainId: chainID, Payouts: detailsMap[chainID]})
	}

	return &types.QuerySubscriptionMonthlyPayoutResponse{Total: total, Details: details}, nil
}
