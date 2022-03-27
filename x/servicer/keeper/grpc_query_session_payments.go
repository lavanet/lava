package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/lavanet/lava/x/servicer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) SessionPaymentsAll(c context.Context, req *types.QueryAllSessionPaymentsRequest) (*types.QueryAllSessionPaymentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var sessionPaymentss []types.SessionPayments
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(k.storeKey)
	sessionPaymentsStore := prefix.NewStore(store, types.KeyPrefix(types.SessionPaymentsKeyPrefix))

	pageRes, err := query.Paginate(sessionPaymentsStore, req.Pagination, func(key []byte, value []byte) error {
		var sessionPayments types.SessionPayments
		if err := k.cdc.Unmarshal(value, &sessionPayments); err != nil {
			return err
		}

		sessionPaymentss = append(sessionPaymentss, sessionPayments)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllSessionPaymentsResponse{SessionPayments: sessionPaymentss, Pagination: pageRes}, nil
}

func (k Keeper) SessionPayments(c context.Context, req *types.QueryGetSessionPaymentsRequest) (*types.QueryGetSessionPaymentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	val, found := k.GetSessionPayments(
		ctx,
		req.Index,
	)
	if !found {
		return nil, status.Error(codes.InvalidArgument, "not found")
	}

	return &types.QueryGetSessionPaymentsResponse{SessionPayments: val}, nil
}
