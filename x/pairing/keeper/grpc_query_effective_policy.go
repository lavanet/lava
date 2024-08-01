package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	projectstypes "github.com/lavanet/lava/v2/x/projects/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) EffectivePolicy(goCtx context.Context, req *types.QueryEffectivePolicyRequest) (*types.QueryEffectivePolicyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	project, err := k.getProjectByDeveloperOrIndex(ctx, req.Consumer, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}
	strictestPolicy, _, err := k.GetProjectStrictestPolicy(ctx, project, req.SpecID, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, err
	}

	// try getting pending policy changes (applied in next epoch)
	nextEpoch, err := k.epochStorageKeeper.GetNextEpoch(ctx, uint64(ctx.BlockHeight()))
	if err != nil {
		return nil, fmt.Errorf("cannot get pending project policy. errors: %s", err.Error())
	}

	pendingProject, err := k.getProjectByDeveloperOrIndex(ctx, req.Consumer, nextEpoch)
	if err != nil {
		return nil, err
	}

	if pendingProject.Equal(project) {
		return &types.QueryEffectivePolicyResponse{Policy: strictestPolicy}, err
	} else {
		pendingPolicy, _, err := k.GetProjectStrictestPolicy(ctx, pendingProject, req.SpecID, uint64(ctx.BlockHeight()))
		return &types.QueryEffectivePolicyResponse{Policy: strictestPolicy, PendingPolicy: pendingPolicy}, err
	}
}

func (k Keeper) getProjectByDeveloperOrIndex(ctx sdk.Context, idOrDeveloper string, block uint64) (projectstypes.Project, error) {
	project, err := k.projectsKeeper.GetProjectForDeveloper(ctx, idOrDeveloper, block)
	if err != nil {
		origErr := err
		// support giving a project-id
		project, err = k.projectsKeeper.GetProjectForBlock(ctx, idOrDeveloper, block)
		if err != nil {
			return projectstypes.Project{}, fmt.Errorf("failed getting project for key %s errors %s, %s", idOrDeveloper, origErr, err)
		}
	}

	return project, nil
}
