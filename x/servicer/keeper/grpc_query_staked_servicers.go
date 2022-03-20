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
	unstaking_servicers := make([]string, 0)
	unstakingServicersAllSpecs := k.GetAllUnstakingServicersAllSpecs(ctx)
	for _, unstakingServicer := range unstakingServicersAllSpecs {
		if unstakingServicer.SpecStakeStorage.Index == specName.Name {
			unstaking_servicers = append(unstaking_servicers, unstakingServicer.Unstaking.String())
		}
	}

	outputStr := fmt.Sprintf("Staked Servicers Query Output:\nSpec: %s Status: %s Block: %d\nStaked Servicers:\n%s\nUnstaking Servicers:\n%s\n--------------------------------------\n", specName.Name, enabledStr, ctx.BlockHeight(), stakeStorage.Staked, unstaking_servicers)

	response := types.QueryStakedServicersResponse{StakeStorage: stakeStorage, Output: outputStr}
	return &response, nil
}
