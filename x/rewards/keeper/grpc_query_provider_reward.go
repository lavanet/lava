package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProviderReward(goCtx context.Context, req *types.QueryProviderRewardRequest) (*types.QueryProviderRewardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var rewards []types.RewardInfo

	for _, basepay := range k.GetAllBasePay(ctx) {
		index := types.BasePayKeyRecover(basepay.GetIndex())

		if index.Provider != req.Provider {
			continue
		}

		if index.ChainID == req.ChainID || req.ChainID == "" {
			rewards = append(rewards, types.RewardInfo{Provider: index.Provider, ChainId: index.ChainID, Amount: sdk.NewCoin(k.stakingKeeper.BondDenom(ctx), basepay.BasePay.Total)})
		}
	}

	return &types.QueryProviderRewardResponse{Rewards: rewards}, nil
}
