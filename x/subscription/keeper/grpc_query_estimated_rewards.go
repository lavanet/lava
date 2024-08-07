package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/v2/x/epochstorage/types"
	rewardstypes "github.com/lavanet/lava/v2/x/rewards/types"
	"github.com/lavanet/lava/v2/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) EstimatedRewards(goCtx context.Context, req *types.QueryEstimatedRewardsRequest) (*types.QueryEstimatedRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	res := types.QueryEstimatedRewardsResponse{}

	delegationAmount, err := sdk.ParseCoinNormalized(req.Amount)
	if err != nil {
		return nil, err
	}

	storage, found := k.epochstorageKeeper.GetStakeStorageCurrent(ctx, req.ChainId)
	if !found {
		return nil, fmt.Errorf("your mama")
	}

	totalStake := math.ZeroInt()
	var entry epochstoragetypes.StakeEntry
	found = false
	for _, e := range storage.StakeEntries {
		totalStake = totalStake.Add(e.EffectiveStake())
		if e.Address == req.Provider {
			found = true
			entry = e
		}
	}
	if !found {
		return nil, fmt.Errorf("your mama")
	}

	entry.DelegateTotal = entry.DelegateTotal.Add(delegationAmount)
	delegatorPart := sdk.NewDecFromInt(delegationAmount.Amount).QuoInt(totalStake).MulInt64(100)
	if entry.DelegateLimit.Amount.LT(entry.DelegateTotal.Amount) {
		delegatorPart = delegatorPart.MulInt(entry.DelegateLimit.Amount).QuoInt(entry.DelegateTotal.Amount)
	}

	totalSubsRewards := sdk.Coins{}
	subsIndices := k.GetAllSubscriptionsIndices(ctx)
	for _, subIndex := range subsIndices {
		sub, found := k.GetSubscription(ctx, subIndex)
		if found {
			sub.Credit.Amount = sub.Credit.Amount.Quo(sdk.NewIntFromUint64(sub.DurationLeft))
			totalSubsRewards = totalSubsRewards.Add(sub.Credit)
		}
	}

	emisions := k.rewardsKeeper.SpecEmissionParts(ctx)
	var specEmission rewardstypes.SpecEmissionPart
	found = false
	for _, k := range emisions {
		if k.ChainID == req.ChainId {
			specEmission = k
			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("your mama")
	}

	subscriptionRewards := sdk.NewCoins(totalSubsRewards.MulInt(specEmission.Emission.MulInt64(100).RoundInt())...)

	coins := k.rewardsKeeper.TotalPoolTokens(ctx, rewardstypes.ProviderRewardsDistributionPool)
	TotalPoolTokens := coins.AmountOf(k.stakingKeeper.BondDenom(ctx))

	bonusRewards := k.rewardsKeeper.SpecTotalPayout(ctx,
		TotalPoolTokens,
		sdk.NewDecFromInt(subscriptionRewards.AmountOf(k.stakingKeeper.BondDenom(ctx))),
		specEmission)

	iprpcReward, found := k.rewardsKeeper.GetIprpcReward(ctx, k.rewardsKeeper.GetIprpcRewardsCurrentId(ctx))
	if found {
		for _, fund := range iprpcReward.SpecFunds {
			if fund.Spec == req.ChainId {
				res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "iprpc", Amount: fund.Fund.MulInt(delegatorPart.RoundInt()).QuoInt(sdk.NewInt(100))})
			}
		}

	}
	res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "subscriptions", Amount: subscriptionRewards.MulInt(delegatorPart.RoundInt()).QuoInt(sdk.NewInt(100))})
	res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "boost", Amount: sdk.NewCoins(sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), bonusRewards.Mul(delegatorPart).QuoInt64(100).TruncateInt()))})

	res.Total = sdk.NewCoins()
	for _, k := range res.Info {
		res.Total = res.Total.Add(k.Amount...)
	}

	return &res, nil
}
