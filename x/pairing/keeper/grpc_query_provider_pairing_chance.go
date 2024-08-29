package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v2/x/pairing/types"
	planstypes "github.com/lavanet/lava/v2/x/plans/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Gets a client's provider list in a specific chain. Also returns the start block of the current epoch, time (in seconds) until there's a new pairing, the block that the chain in the request's spec was changed
func (k Keeper) ProviderPairingChance(goCtx context.Context, req *types.QueryProviderPairingChanceRequest) (*types.QueryProviderPairingChanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	providerToCheck, err := k.GetStakeEntry(ctx, req.ChainID, req.Provider)
	if err != nil {
		return nil, err
	}

	// frozen provider could never be picked for pairing
	if providerToCheck.IsFrozen() {
		return &types.QueryProviderPairingChanceResponse{Chance: math.LegacyZeroDec()}, nil
	}

	// construct required geolocations and clusters array to check
	geos := []planstypes.Geolocation{}
	if req.Geolocation == planstypes.Geolocation_value["GL"] {
		geos = planstypes.GetAllGeolocations()
	} else {
		geos = append(geos, planstypes.Geolocation(req.Geolocation))
	}

	clusters := []string{}
	if req.Cluster == "" {
		clusters = k.subscriptionKeeper.GetAllClusters(ctx)
	} else {
		clusters = append(clusters, req.Cluster)
	}

	// calculate the average provider pairing chance
	chance := math.LegacyZeroDec()
	counter := int64(0)
	for _, geo := range geos {
		for _, cluster := range clusters {
			policy := planstypes.Policy{GeolocationProfile: int32(geo), MaxProvidersToPair: 1}
			calculatedChance, err := k.CalculatePairingChance(ctx, providerToCheck.Address, providerToCheck.Chain, &policy, cluster)
			if err != nil {
				return nil, err
			}
			chance = chance.Add(calculatedChance)
			counter++
		}
	}

	if counter != 0 {
		chance = chance.QuoInt64(counter)
	}

	return &types.QueryProviderPairingChanceResponse{Chance: chance}, nil
}
