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

	_, specContribut := k.specKeeper.GetContributorReward(ctx, req.ChainId)

	storage := k.epochstorageKeeper.GetAllStakeEntriesCurrentForChainId(ctx, req.ChainId)

	totalStake := math.ZeroInt()
	var entry epochstoragetypes.StakeEntry
	found := false
	for _, e := range storage {
		totalStake = totalStake.Add(e.EffectiveStake())
		if e.Address == req.Provider {
			found = true
			entry = e
		}
	}
	if !found {
		return nil, fmt.Errorf("provider not found in stake entries for chain ID: %s", req.ChainId)
	}

	// delegation of the provider
	delegationAmount := entry.Stake
	delegatorPart := sdk.NewDecFromInt(delegationAmount.Amount).QuoInt(totalStake).MulInt64(int64(entry.DelegateCommission)).QuoInt64(100)
	if req.Delegator != "" || req.Amount != "" {
		if req.Amount != "" {
			var err error
			delegationAmount, err = sdk.ParseCoinNormalized(req.Amount)
			if err != nil {
				return nil, err
			}
			entry.DelegateTotal = entry.DelegateTotal.Add(delegationAmount)
		} else if req.Delegator != "" {
			d, found := k.dualstakingKeeper.GetDelegation(ctx, req.Delegator, req.Provider, req.ChainId, uint64(ctx.BlockHeight()))
			if !found {
				return nil, fmt.Errorf("could not find delegator")
			}
			delegationAmount = d.Amount
		}

		delegatorPart = sdk.NewDecFromInt(delegationAmount.Amount).QuoInt(totalStake).MulInt64(int64(100 - entry.DelegateCommission)).QuoInt64(100)
		// calculate existing delegator rewards
		if entry.DelegateLimit.Amount.LT(entry.DelegateTotal.Amount) {
			delegatorPart = delegatorPart.MulInt(entry.DelegateLimit.Amount).QuoInt(entry.DelegateTotal.Amount)
		}
	}
	delegatorPart = delegatorPart.Mul(sdk.OneDec().Sub(specContribut))

	totalSubsRewards := sdk.Coins{}
	subsIndices := k.GetAllSubscriptionsIndices(ctx)
	for _, subIndex := range subsIndices {
		sub, found := k.GetSubscription(ctx, subIndex)
		if found {
			sub.Credit.Amount = sub.Credit.Amount.Quo(sdk.NewIntFromUint64(sub.DurationLeft))
			totalSubsRewards = totalSubsRewards.Add(sub.Credit)
		}
	}

	emissions := k.rewardsKeeper.SpecEmissionParts(ctx)
	var specEmission rewardstypes.SpecEmissionPart
	found = false
	for _, k := range emissions {
		if k.ChainID == req.ChainId {
			specEmission = k
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("spec emission part not found for chain ID: %s", req.ChainId)
	}

	subscriptionRewards := sdk.NewCoins(totalSubsRewards.MulInt(specEmission.Emission.MulInt64(100).RoundInt())...)
	valRewards, comRewards, err := k.rewardsKeeper.CalculateValidatorsAndCommunityParticipationRewards(ctx, subscriptionRewards[0])
	if err != nil {
		return nil, fmt.Errorf("failed to calculate Validators And Community Participation Rewards")
	}

	subscriptionRewards = subscriptionRewards.Sub(valRewards...).Sub(comRewards...)

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
				res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "iprpc", Amount: sdk.NewDecCoinsFromCoins(fund.Fund...).MulDec(delegatorPart)})
			}
		}
	}

	res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "subscriptions", Amount: sdk.NewDecCoinsFromCoins(subscriptionRewards...).MulDec(delegatorPart)})
	res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "boost", Amount: sdk.NewDecCoins(sdk.NewDecCoinFromDec(k.stakingKeeper.BondDenom(ctx), bonusRewards.Mul(delegatorPart)))})

	res.Total = sdk.NewDecCoins()
	for _, k := range res.Info {
		res.Total = res.Total.Add(k.Amount...)
	}

	return &res, nil
}
