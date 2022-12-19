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

func (k Keeper) EpochPaymentsAll(c context.Context, req *types.QueryAllEpochPaymentsRequest) (*types.QueryAllEpochPaymentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var epochPaymentss []types.EpochPayments
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	epochPaymentsStore := prefix.NewStore(store, types.KeyPrefix(types.EpochPaymentsKeyPrefix))

	pageRes, err := query.Paginate(epochPaymentsStore, req.Pagination, func(key []byte, value []byte) error {
		var epochPayments types.EpochPayments
		if err := k.cdc.Unmarshal(value, &epochPayments); err != nil {
			return err
		}

		epochPaymentss = append(epochPaymentss, epochPayments)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllEpochPaymentsResponse{EpochPayments: epochPaymentss, Pagination: pageRes}, nil
}

func (k Keeper) EpochPayments(c context.Context, req *types.QueryGetEpochPaymentsRequest) (*types.QueryGetEpochPaymentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetEpochPayments(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetEpochPaymentsResponse{EpochPayments: val}, nil
}
