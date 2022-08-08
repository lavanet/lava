package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) FixatedStakeToMaxCuAll(c context.Context, req *types.QueryAllFixatedStakeToMaxCuRequest) (*types.QueryAllFixatedStakeToMaxCuResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var fixatedStakeToMaxCus []types.FixatedStakeToMaxCu
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	fixatedStakeToMaxCuStore := prefix.NewStore(store, types.KeyPrefix(types.FixatedStakeToMaxCuKeyPrefix))

	pageRes, err := query.Paginate(fixatedStakeToMaxCuStore, req.Pagination, func(key []byte, value []byte) error {
		var fixatedStakeToMaxCu types.FixatedStakeToMaxCu
		if err := k.cdc.Unmarshal(value, &fixatedStakeToMaxCu); err != nil {
			return err
		}

		fixatedStakeToMaxCus = append(fixatedStakeToMaxCus, fixatedStakeToMaxCu)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllFixatedStakeToMaxCuResponse{FixatedStakeToMaxCu: fixatedStakeToMaxCus, Pagination: pageRes}, nil
}

func (k Keeper) FixatedStakeToMaxCu(c context.Context, req *types.QueryGetFixatedStakeToMaxCuRequest) (*types.QueryGetFixatedStakeToMaxCuResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetFixatedStakeToMaxCu(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetFixatedStakeToMaxCuResponse{FixatedStakeToMaxCu: val}, nil
}
