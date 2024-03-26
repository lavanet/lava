package keeper

import (
	"context"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k Keeper) ProviderEpochCu(goCtx context.Context, req *types.QueryProviderEpochCuRequest) (*types.QueryProviderEpochCuResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	infoMap := map[string]uint64{}

	_, keys, pecs := k.GetAllProviderEpochCuStore(ctx)
	for i := range pecs {
		provider, _, err := types.DecodeProviderEpochCuKey(keys[i])
		if err != nil {
			continue
		}
		if _, ok := infoMap[provider]; !ok {
			infoMap[provider] = pecs[i].ServicedCu
		} else {
			infoMap[provider] += pecs[i].ServicedCu
		}
	}

	info := []types.ProviderCuInfo{}
	for provider, cu := range infoMap {
		info = append(info, types.ProviderCuInfo{Provider: provider, Cu: cu})
	}

	sort.Slice(info, func(i, j int) bool {
		return info[i].Provider < info[j].Provider
	})

	return &types.QueryProviderEpochCuResponse{Info: info}, nil
}
