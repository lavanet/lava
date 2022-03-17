package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//query the servicers staked for a specific spec
func (k Keeper) StakedServicers(goCtx context.Context, req *types.QueryStakedServicersRequest) (*types.QueryStakedServicersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	specName := types.SpecName{Name: req.SpecName}
	err := specName.ValidateBasic()
	if err != nil {
		return nil, fmt.Errorf("invalid spec name given : %s, %s", specName, err)
	}
	enabled, found := k.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
	if !found {
		return nil, fmt.Errorf("spec name given is not in the spec: %s", specName)
	}
	specStorage, found := k.GetSpecStakeStorage(ctx, req.SpecName)
	if !found {
		return nil, fmt.Errorf("spec name given holds no storage, meaning its empty: %s", specName)
	}
	stakeStorage := specStorage.StakeStorage
	enabledStr := "disabled"
	if enabled {
		enabledStr = "enabled"
	}
	outputStr := fmt.Sprintf("Spec: %s Status: %s Block: %d\nStaked Servicers:\n%s\nUnstaking Servicers:\n%s", specName.Name, enabledStr, ctx.BlockHeight(), stakeStorage.Staked, stakeStorage.Unstaking)
	_ = ctx

	response := types.QueryStakedServicersResponse{StakeStorage: stakeStorage, Output: outputStr}
	return &response, nil
}
