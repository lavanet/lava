package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	rewardstypes "github.com/lavanet/lava/v4/x/rewards/types"
	"github.com/lavanet/lava/v4/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) EstimatedValidatorRewards(goCtx context.Context, req *types.QueryEstimatedValidatorRewardsRequest) (*types.QueryEstimatedValidatorRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	res := types.QueryEstimatedValidatorRewardsResponse{}

	valAddress, err := sdk.ValAddressFromBech32(req.Validator)
	if err != nil {
		return nil, err
	}

	val, found := k.stakingKeeper.GetValidator(ctx, valAddress)
	if !found {
		return nil, fmt.Errorf("validator not found")
	}

	var delegatorPart math.LegacyDec
	delAddress := sdk.AccAddress(valAddress)
	totalStakedTokens := math.ZeroInt()
	// self delegation
	if req.AmountDelegator == "" {
		del, found := k.stakingKeeper.GetDelegation(ctx, delAddress, valAddress)
		if !found {
			return nil, fmt.Errorf("self delegation not found")
		}
		delegatorPart = del.Shares.Add(val.DelegatorShares.Sub(del.Shares).Mul(val.Commission.Rate)).Quo(val.DelegatorShares)
	} else {
		delAddress, err := sdk.AccAddressFromBech32(req.AmountDelegator)
		// existing delegator
		if err == nil {
			del, found := k.stakingKeeper.GetDelegation(ctx, delAddress, valAddress)
			if !found {
				return nil, fmt.Errorf("delegation not found")
			}
			delegatorPart = del.Shares.Quo(val.DelegatorShares).Mul(math.LegacyOneDec().Sub(val.Commission.Rate))
		} else { // potential delegator
			coins, err := sdk.ParseCoinsNormalized(req.AmountDelegator)
			if err != nil {
				return nil, fmt.Errorf("failed to parse input, input should be a valid lava delegator account or coins")
			}

			totalStakedTokens = coins[0].Amount
			var shares math.LegacyDec
			val, shares = val.AddTokensFromDel(coins[0].Amount)
			delegatorPart = shares.Quo(val.DelegatorShares).Mul(math.LegacyOneDec().Sub(val.Commission.Rate))
		}
	}

	validators := k.stakingKeeper.GetBondedValidatorsByPower(ctx)

	for _, v := range validators {
		totalStakedTokens = totalStakedTokens.Add(v.Tokens)
	}
	delegatorPart = delegatorPart.MulInt(val.Tokens).QuoInt(totalStakedTokens)

	totalSubsRewards := sdk.Coins{}
	subsIndices := k.GetAllSubscriptionsIndices(ctx)
	for _, subIndex := range subsIndices {
		sub, found := k.GetSubscription(ctx, subIndex)
		if found {
			sub.Credit.Amount = sub.Credit.Amount.Quo(math.NewIntFromUint64(sub.DurationLeft))
			totalSubsRewards = totalSubsRewards.Add(sub.Credit)
		}
	}

	valRewardsFromSubscriptions, _, err := k.rewardsKeeper.CalculateValidatorsAndCommunityParticipationRewards(ctx, totalSubsRewards[0])
	if err != nil {
		return nil, fmt.Errorf("failed to calculate Validators And Community Participation Rewards")
	}

	monthsLeft := k.rewardsKeeper.AllocationPoolMonthsLeft(ctx)
	allocationPool := k.rewardsKeeper.TotalPoolTokens(ctx, rewardstypes.ValidatorsRewardsAllocationPoolName)
	blockRewards := sdk.NewDecCoinsFromCoins(allocationPool...).QuoDec(sdk.NewDec(monthsLeft))
	communityTax := k.rewardsKeeper.GetCommunityTax(ctx)
	blockRewards = blockRewards.MulDec(math.LegacyOneDec().Sub(communityTax))

	iprpcReward, found := k.rewardsKeeper.GetIprpcReward(ctx, k.rewardsKeeper.GetIprpcRewardsCurrentId(ctx))
	if found {
		for _, fund := range iprpcReward.SpecFunds {
			coinsForSpec := sdk.NewCoins()
			for _, coin := range fund.Fund {
				validatorCoins, _, err := k.rewardsKeeper.CalculateValidatorsAndCommunityParticipationRewards(ctx, coin)
				if err != nil {
					return nil, fmt.Errorf("failed to calculate Validators And Community Participation Rewards for iprpc")
				}
				coinsForSpec = coinsForSpec.Add(validatorCoins...)
			}

			res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "iprpc_" + fund.Spec, Amount: sdk.NewDecCoinsFromCoins(coinsForSpec...).MulDec(delegatorPart)})
		}
	}

	res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "subscriptions", Amount: sdk.NewDecCoinsFromCoins(valRewardsFromSubscriptions...).MulDec(delegatorPart)})
	res.Info = append(res.Info, &types.EstimatedRewardInfo{Source: "blocks", Amount: blockRewards.MulDec(delegatorPart)})

	res.Total = sdk.NewDecCoins()
	for _, k := range res.Info {
		res.Total = res.Total.Add(k.Amount...)
	}

	return &res, nil
}
