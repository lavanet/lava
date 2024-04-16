package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/dualstaking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) DelegatorRewards(goCtx context.Context, req *types.QueryDelegatorRewardsRequest) (res *types.QueryDelegatorRewardsResponse, err error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// if delegator is the operator (to find provider rewards), switch to vault
	delegator := req.Delegator
	delegatorAcc, err := sdk.AccAddressFromBech32(delegator)
	if err != nil {
		return nil, err
	}
	stakeEntry, found, _ := k.epochstorageKeeper.GetStakeEntryByAddressCurrent(ctx, req.ChainId, delegatorAcc)
	if found {
		if delegator == stakeEntry.Operator {
			delegator = stakeEntry.Vault
		}
	}

	var rewards []types.DelegatorRewardInfo
	resProviders, err := k.DelegatorProviders(goCtx, &types.QueryDelegatorProvidersRequest{Delegator: delegator})
	if err != nil {
		return nil, err
	}

	for _, delegation := range resProviders.Delegations {
		if (delegation.ChainID == req.ChainId || req.ChainId == "") &&
			(delegation.Provider == req.Provider || req.Provider == "") {
			ind := types.DelegationKey(delegation.Provider, delegator, delegation.ChainID)
			delegatorReward, found := k.GetDelegatorReward(ctx, ind)
			if found {
				reward := types.DelegatorRewardInfo{
					Provider: delegation.Provider,
					ChainId:  delegation.ChainID,
					Amount:   delegatorReward.Amount,
				}
				rewards = append(rewards, reward)
			}
		}
	}

	return &types.QueryDelegatorRewardsResponse{Rewards: rewards}, nil
}
