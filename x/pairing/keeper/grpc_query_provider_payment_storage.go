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

func (k Keeper) ProviderPaymentStorageAll(c context.Context, req *types.QueryAllProviderPaymentStorageRequest) (*types.QueryAllProviderPaymentStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var providerPaymentStorages []types.ProviderPaymentStorage
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	providerPaymentStorageStore := prefix.NewStore(store, types.KeyPrefix(types.ProviderPaymentStorageKeyPrefix))

	pageRes, err := query.Paginate(providerPaymentStorageStore, req.Pagination, func(key []byte, value []byte) error {
		var providerPaymentStorage types.ProviderPaymentStorage
		if err := k.cdc.Unmarshal(value, &providerPaymentStorage); err != nil {
			return err
		}

		providerPaymentStorages = append(providerPaymentStorages, providerPaymentStorage)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllProviderPaymentStorageResponse{ProviderPaymentStorage: providerPaymentStorages, Pagination: pageRes}, nil
}

func (k Keeper) ProviderPaymentStorage(c context.Context, req *types.QueryGetProviderPaymentStorageRequest) (*types.QueryGetProviderPaymentStorageResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetProviderPaymentStorage(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.NotFound, "not found")
	}

	return &types.QueryGetProviderPaymentStorageResponse{ProviderPaymentStorage: val}, nil
}
