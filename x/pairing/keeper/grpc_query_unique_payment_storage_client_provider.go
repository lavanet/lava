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

func (k Keeper) UniquePaymentStorageClientProviderAll(c context.Context, req *types.QueryAllUniquePaymentStorageClientProviderRequest) (*types.QueryAllUniquePaymentStorageClientProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var uniquePaymentStorageClientProviders []types.UniquePaymentStorageClientProvider
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	uniquePaymentStorageClientProviderStore := prefix.NewStore(store, types.KeyPrefix(types.UniquePaymentStorageClientProviderKeyPrefix))

	pageRes, err := query.Paginate(uniquePaymentStorageClientProviderStore, req.Pagination, func(key []byte, value []byte) error {
		var uniquePaymentStorageClientProvider types.UniquePaymentStorageClientProvider
		if err := k.cdc.Unmarshal(value, &uniquePaymentStorageClientProvider); err != nil {
			return err
		}

		uniquePaymentStorageClientProviders = append(uniquePaymentStorageClientProviders, uniquePaymentStorageClientProvider)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllUniquePaymentStorageClientProviderResponse{UniquePaymentStorageClientProvider: uniquePaymentStorageClientProviders, Pagination: pageRes}, nil
}

func (k Keeper) UniquePaymentStorageClientProvider(c context.Context, req *types.QueryGetUniquePaymentStorageClientProviderRequest) (*types.QueryGetUniquePaymentStorageClientProviderResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetUniquePaymentStorageClientProvider(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetUniquePaymentStorageClientProviderResponse{UniquePaymentStorageClientProvider: val}, nil
}
