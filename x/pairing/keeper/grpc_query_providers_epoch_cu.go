package keeper

import (
	"context"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProvidersEpochCu(goCtx context.Context, req *types.QueryProvidersEpochCuRequest) (*types.QueryProvidersEpochCuResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	infoMap := map[string]uint64{}

	infos := k.GetAllProviderEpochCuStore(ctx)
	for _, info := range infos {
		if _, ok := infoMap[info.Provider]; !ok {
			infoMap[info.Provider] = info.ProviderEpochCu.ServicedCu
		} else {
			infoMap[info.Provider] += info.ProviderEpochCu.ServicedCu
		}
	}

	info := []types.ProviderCuInfo{}
	for provider, cu := range infoMap {
		info = append(info, types.ProviderCuInfo{Provider: provider, Cu: cu})
	}

	sort.Slice(info, func(i, j int) bool {
		return info[i].Provider < info[j].Provider
	})

	return &types.QueryProvidersEpochCuResponse{Info: info}, nil
}
