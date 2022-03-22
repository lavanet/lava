package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/user/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) StakedUsers(goCtx context.Context, req *types.QueryStakedUsersRequest) (*types.QueryStakedUsersResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	specName := types.SpecName{Name: req.SpecName}
	err := specName.ValidateBasic()
	if err != nil {
		return nil, fmt.Errorf("invalid spec name given : %s, %s", specName, err)
	}
	enabled, found, _ := k.specKeeper.IsSpecFoundAndActive(ctx, specName.Name)
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
	unstaking_Users := make([]string, 0)
	unstakingUsersAllSpecs := k.GetAllUnstakingUsersAllSpecs(ctx)
	for _, unstakingUser := range unstakingUsersAllSpecs {
		if unstakingUser.SpecStakeStorage.Index == specName.Name {
			unstaking_Users = append(unstaking_Users, unstakingUser.Unstaking.String())
		}
	}

	outputStr := fmt.Sprintf("Staked Users Query Output:\nSpec: %s Status: %s Block: %d\nStaked Users:\n%s\nUnstaking Users:\n%s\n--------------------------------------\n", specName.Name, enabledStr, ctx.BlockHeight(), stakeStorage.StakedUsers, unstaking_Users)

	response := types.QueryStakedUsersResponse{StakeStorage: stakeStorage, Output: outputStr}

	return &response, nil
}
