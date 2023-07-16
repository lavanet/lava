package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) EffectivePolicy(goCtx context.Context, req *types.QueryEffectivePolicyRequest) (*types.QueryEffectivePolicyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	project, err := k.projectsKeeper.GetProjectForDeveloper(ctx, req.Consumer, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}
	strictestPolicy, _, err := k.getProjectStrictestPolicy(ctx, project, req.SpecId)
	return &types.QueryEffectivePolicyResponse{Policy: &strictestPolicy}, err
}
