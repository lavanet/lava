package scores

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

// geo requirement that implements the ScoreReq interface
type GeoReq struct {
	Geo uint64
}

const (
	geoReqName    = "geo-req"
	maxGeoLatency = 10000 // highest geo cost < 300
	minGeoLatency = 1
)

func (gr GeoReq) Init(policy planstypes.Policy) bool {
	return true
}

// Score calculates the geo score of a provider based on preset latency data
// Note: each GeoReq must have exactly a single geolocation (bit)
func (gr GeoReq) Score(provider epochstoragetypes.StakeEntry) sdk.Uint {
	// check if the provider supports the required geolocation
	if gr.Geo&^provider.Geolocation == 0 {
		return calculateCostFromLatency(minGeoLatency)
	}

	providerGeoEnums := types.GetGeolocationsFromUint(int32(provider.Geolocation))
	_, cost := CalcGeoCost(planstypes.Geolocation(gr.Geo), providerGeoEnums)

	return cost
}

func (gr GeoReq) GetName() string {
	return geoReqName
}

// Equal() used to compare slots to determine slot groups
func (gr GeoReq) Equal(other ScoreReq) bool {
	otherGeoReq, ok := other.(GeoReq)
	if !ok {
		return false
	}

	return otherGeoReq == gr
}

// TODO: this function doesn't return the optimal geo reqs for the case
// that there are more required geos than providers to pair
func (gr GeoReq) GetReqForSlot(policy planstypes.Policy, slotIdx int) ScoreReq {
	policyGeoEnums := types.GetGeolocationsFromUint(int32(policy.GeolocationProfile))

	return GeoReq{Geo: uint64(policyGeoEnums[slotIdx%len(policyGeoEnums)])}
}

// a single geolocation and the latency to it (in millieseconds)
type GeoLatency struct {
	geo     planstypes.Geolocation
	latency uint64
}

// CalcGeoCost() finds the minimal latency between the required geo and the provider's supported geolocations
func CalcGeoCost(reqGeo planstypes.Geolocation, providerGeos []planstypes.Geolocation) (minLatencyGeo planstypes.Geolocation, minLatencyCost sdk.Uint) {
	minGeo := planstypes.Geolocation(-1)
	minLatency := uint64(maxGeoLatency)
	for _, pGeo := range providerGeos {
		if inner, ok := GEO_LATENCY_MAP[reqGeo]; ok {
			if latency, ok := inner[pGeo]; ok {
				if latency < minLatency {
					minGeo = pGeo
					minLatency = latency
				}
			}
		}
	}

	return minGeo, calculateCostFromLatency(minLatency)
}

func calculateCostFromLatency(latency uint64) sdk.Uint {
	return sdk.NewUint(maxGeoLatency / latency)
}

// GEO_LATENCY_MAP is a map of lists of GeoLatency that defines the cost of geo mismatch
// for each single geolocation. The map key is a single geolocation and the value is an
// ordered list of neighbors and their latency (ordered by latency)
// latency data from: https://wondernetwork.com/pings (July 2023)
var GEO_LATENCY_MAP = map[planstypes.Geolocation]map[planstypes.Geolocation]uint64{
	planstypes.Geolocation_AS: {
		planstypes.Geolocation_AU: 146,
		planstypes.Geolocation_EU: 155,
	},
	planstypes.Geolocation_USE: {
		planstypes.Geolocation_USC: 42,
		planstypes.Geolocation_USW: 68,
	},
	planstypes.Geolocation_USW: {
		planstypes.Geolocation_USC: 45,
		planstypes.Geolocation_USE: 68,
	},
	planstypes.Geolocation_USC: {
		planstypes.Geolocation_USE: 42,
		planstypes.Geolocation_USW: 45,
	},
	planstypes.Geolocation_EU: {
		planstypes.Geolocation_USE: 116,
		planstypes.Geolocation_AF:  138,
		planstypes.Geolocation_AS:  155,
	},
	planstypes.Geolocation_AF: {
		planstypes.Geolocation_EU:  138,
		planstypes.Geolocation_USE: 203,
		planstypes.Geolocation_AS:  263,
	},
	planstypes.Geolocation_AU: {
		planstypes.Geolocation_AS:  146,
		planstypes.Geolocation_USW: 179,
	},
}
