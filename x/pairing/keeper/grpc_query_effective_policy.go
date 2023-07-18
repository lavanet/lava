package keeper

import (
	"context"
	"fmt"

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
		origErr := err
		// support giving a project-id
		project, err = k.projectsKeeper.GetProjectForBlock(ctx, req.Consumer, uint64(ctx.BlockHeight()))
		if err != nil {
			return nil, fmt.Errorf("failed getting project for key %s errors %s, %s", req.Consumer, origErr, err)
		}
	}
	strictestPolicy, err := k.GetProjectStrictestPolicy(ctx, project, req.SpecID)
	return &types.QueryEffectivePolicyResponse{Policy: &strictestPolicy}, err
}
