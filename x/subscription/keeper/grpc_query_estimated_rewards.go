package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/testutil/sample"
	"github.com/lavanet/lava/v3/utils"
	dualstakingtypes "github.com/lavanet/lava/v3/x/dualstaking/types"
	"github.com/lavanet/lava/v3/x/subscription/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) EstimatedProviderRewards(goCtx context.Context, req *types.QueryEstimatedProviderRewardsRequest) (*types.QueryEstimatedRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	res := types.QueryEstimatedRewardsResponse{}

	details := []utils.Attribute{
		utils.LogAttr("block", ctx.BlockHeight()),
		utils.LogAttr("provider", req.Provider),
		utils.LogAttr("delegator_amount", req.AmountDelegator),
	}

	// parse the delegator/delegation optional argument (delegate if needed)
	delegator := req.AmountDelegator
	if req.AmountDelegator != "" {
		delegation, err := sdk.ParseCoinNormalized(req.AmountDelegator)
		if err != nil {
			// arg is not delegation, check if it's an address
			_, err := sdk.AccAddressFromBech32(req.AmountDelegator)
			if err != nil {
				return nil, utils.LavaFormatWarning("cannot estimate rewards for delegator/delegation", fmt.Errorf("delegator/delegation argument is neither a valid coin or a valid bech32 address"), details...)
			}
		} else {
			// arg is delegation, assign arbitrary delegator address and delegate the amount
			delegator = sample.AccAddress()
			err := k.dualstakingKeeper.DelegateFull(ctx, delegator, sample.ValAddress(), req.Provider, delegation, false)
			if err != nil {
				return nil, utils.LavaFormatWarning("cannot estimate rewards, delegation simulation failed", err, details...)
			}
		}
	}

	// get claimable rewards before the rewards distribution
	before, err := k.getClaimableRewards(goCtx, req.Provider, delegator)
	if err != nil {
		return nil, utils.LavaFormatWarning("cannot estimate rewards, cannot get claimable rewards before distribution", err, details...)
	}

	// trigger subs
	subs := k.GetAllSubscriptionsIndices(ctx)
	for _, sub := range subs {
		k.advanceMonth(ctx, []byte(sub))
	}

	// get all CU tracker timers (used to keep data on subscription rewards) and distribute all subscription rewards
	gs := k.ExportCuTrackerTimers(ctx)
	for _, timer := range gs.BlockEntries {
		k.RewardAndResetCuTracker(ctx, []byte(timer.Key), timer.Data)
	}

	// distribute bonus and IPRPC rewards
	k.rewardsKeeper.DistributeMonthlyBonusRewards(ctx)

	// get claimable rewards after the rewards distribution
	after, err := k.getClaimableRewards(goCtx, req.Provider, delegator)
	if err != nil {
		return nil, utils.LavaFormatWarning("cannot estimate rewards, cannot get claimable rewards after distribution", err, details...)
	}

	// calculate the claimable rewards difference
	res.Total = sdk.NewDecCoinsFromCoins(after.Sub(before...)...)

	// get the last IPRPC rewards distribution block
	// note: on error we simply leave RecommendedBlock to be zero since until the
	// upcoming IPRPC rewards will be distributed (from the time this query is merged in main)
	// there will be no LastRewardsBlock set
	rewardsDistributionBlock, after24HoursBlock, err := k.rewardsKeeper.GetLastRewardsBlock(ctx)
	if err == nil {
		// if the query was sent within the first 24 hours of the month,
		// make the recommended block be the rewards distribution block - 1
		if uint64(ctx.BlockHeight()) <= after24HoursBlock {
			res.RecommendedBlock = rewardsDistributionBlock - 1
		}
	}

	return &res, nil
}

// helper function that returns a map of provider->rewards
func (k Keeper) getClaimableRewards(goCtx context.Context, provider string, delegator string) (rewards sdk.Coins, err error) {
	var qRes *dualstakingtypes.QueryDelegatorRewardsResponse
	ctx := sdk.UnwrapSDKContext(goCtx)

	// get delegator rewards
	if delegator == "" {
		// self delegation
		md, err := k.epochstorageKeeper.GetMetadata(ctx, provider)
		if err != nil {
			return nil, err
		}
		qRes, err = k.dualstakingKeeper.DelegatorRewards(goCtx, &dualstakingtypes.QueryDelegatorRewardsRequest{
			Delegator: md.Vault,
			Provider:  md.Provider,
		})
		if err != nil {
			return nil, err
		}
	} else {
		qRes, err = k.dualstakingKeeper.DelegatorRewards(goCtx, &dualstakingtypes.QueryDelegatorRewardsRequest{
			Delegator: delegator,
			Provider:  provider,
		})
		if err != nil {
			return nil, err
		}
	}

	if len(qRes.Rewards) == 0 {
		return sdk.NewCoins(), nil
	}

	return qRes.Rewards[0].Amount, nil
}

func (k Keeper) EstimatedRewards(goCtx context.Context, req *types.QueryEstimatedRewardsRequest) (*types.QueryEstimatedRewardsResponse, error) {
	newReq := types.QueryEstimatedProviderRewardsRequest{Provider: req.Provider, AmountDelegator: req.AmountDelegator}
	return k.EstimatedProviderRewards(goCtx, &newReq)
}
