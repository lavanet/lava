package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v4/x/epochstorage/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProviderMetaData(c context.Context, req *types.QueryProviderMetaDataRequest) (*types.QueryProviderMetaDataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	var err error
	res := types.QueryProviderMetaDataResponse{}
	if req.Provider == "" {
		res.MetaData, err = k.GetAllMetadata(ctx)
	} else {
		var metadata types.ProviderMetadata
		metadata, err = k.GetMetadata(ctx, req.Provider)
		res.MetaData = append(res.MetaData, metadata)
	}

	return &res, err
}
