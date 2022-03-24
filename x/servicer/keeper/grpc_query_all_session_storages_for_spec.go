package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) AllSessionStoragesForSpec(goCtx context.Context, req *types.QueryAllSessionStoragesForSpecRequest) (*types.QueryAllSessionStoragesForSpecResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	earliestSessionStart, found := k.GetEarliestSessionStart(ctx)
	if !found {
		return nil, fmt.Errorf("keeper didn't find GetEarliestSessionStart")
	}

	currentSessionStart, found := k.GetCurrentSessionStart(ctx)
	if !found {
		return nil, fmt.Errorf("keeper didn't find GetCurrentSessionStart")
	}
	returnedStorages := []types.SessionStorageForSpec{}
	for sessionBlock := earliestSessionStart.Block.Num; sessionBlock <= currentSessionStart.Block.Num; {
		sessionBlockObj := types.BlockNum{Num: sessionBlock}
		stakeStorage, _, err := k.GetSpecStakeStorageInSessionStorageForSpec(ctx, sessionBlockObj, req.SpecName)
		sessionBlocks, _ := k.GetSessionBlocksAndOverlapForBlock(ctx, sessionBlockObj)
		if err != nil {
			//this condition can happen if the spec stake storage was empty due to no servicers
			//so just skip this entry
			sessionBlock += sessionBlocks
			// fmt.Printf("error getting GetSpecStakeStorageInSessionStorageForSpec %s spec: %s, current block: %d", sessionBlockObj, req.SpecName, ctx.BlockHeight())
			// fmt.Printf("sessionBlock: %s, sessionBlocks: %d", sessionBlock, sessionBlocks)
			continue
			// return nil, fmt.Errorf("error getting GetSpecStakeStorageInSessionStorageForSpec %s spec: %s, current block: %d", sessionBlockObj, req.SpecName, ctx.BlockHeight())
		}
		returnedStorages = append(returnedStorages, types.SessionStorageForSpec{Index: strconv.FormatUint(sessionBlock, 10), StakeStorage: stakeStorage})
		sessionBlock += sessionBlocks
	}

	return &types.QueryAllSessionStoragesForSpecResponse{Storages: returnedStorages}, nil
}
