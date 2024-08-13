package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/rewards/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) IprpcProviderRewardEstimation(goCtx context.Context, req *types.QueryIprpcProviderRewardEstimationRequest) (*types.QueryIprpcProviderRewardEstimationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// get current month IPRPC reward
	id := k.GetIprpcRewardsCurrentId(ctx)
	iprpcReward, found := k.GetIprpcReward(ctx, id)
	if !found {
		return nil, fmt.Errorf("current month IPRPC reward does not exist")
	}

	// go over all the IPRPC reward specs and get the provider's relative reward (by CU)
	providerSpecFunds := []types.Specfund{}
	for _, specFund := range iprpcReward.SpecFunds {
		// get all spec basepays and count IPRPC CU
		bps, _ := k.specProvidersBasePay(ctx, specFund.Spec, false)
		providerIprpcCu := uint64(0)
		totalIprpcCu := uint64(0)

		stakeEntry, found := k.epochstorage.GetStakeEntryCurrent(ctx, specFund.Spec, req.Provider)
		if !found {
			continue
		}

		providerBpIndex := types.BasePayWithIndex{Provider: stakeEntry.Address, ChainId: specFund.Spec}
		for _, bp := range bps {
			if bp.Index() == providerBpIndex.Index() {
				providerIprpcCu = bp.BasePay.IprpcCu
			}
			totalIprpcCu += bp.BasePay.IprpcCu
		}

		// get the provider's relative reward by CU
		providerFund, isValid := specFund.Fund.SafeMulInt(sdk.NewIntFromUint64(providerIprpcCu))
		if !isValid {
			continue
		}
		providerFund, isValid = providerFund.SafeQuoInt(sdk.NewIntFromUint64(totalIprpcCu))
		if !isValid {
			continue
		}

		// save the provider's reward
		providerSpecFunds = append(providerSpecFunds, types.Specfund{Spec: specFund.Spec, Fund: providerFund})
	}

	return &types.QueryIprpcProviderRewardEstimationResponse{SpecFunds: providerSpecFunds}, nil
}
