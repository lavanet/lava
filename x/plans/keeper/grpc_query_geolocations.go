package keeper

import (
	"context"
	"sort"
	"strconv"

	"github.com/lavanet/lava/x/plans/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type geoInfo struct {
	name string
	val  int32
}

func (k Keeper) Geolocations(goCtx context.Context, req *types.QueryGeolocationsRequest) (*types.QueryGeolocationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var geos []geoInfo
	for geo, geoInt := range types.Geolocation_value {
		geos = append(geos, geoInfo{name: geo, val: geoInt})
	}

	sort.Slice(geos, func(i, j int) bool {
		return geos[i].val < geos[j].val
	})

	var geosStr []string
	for _, info := range geos {
		geosStr = append(geosStr, info.String())
	}

	return &types.QueryGeolocationsResponse{Geos: geosStr}, nil
}

func (gi geoInfo) String() string {
	return gi.name + ": 0x" + strconv.FormatInt(int64(gi.val), 16)
}
