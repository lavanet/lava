package scores

import (
	"fmt"
	"math"

	"github.com/lavanet/lava/utils"
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

// Score calculates the geo score of a provider based on preset latency data
// Note: each GeoReq must have exactly a single geolocation (bit)
func (gr GeoReq) Score(provider epochstoragetypes.StakeEntry) uint64 {
	if !types.IsGeoEnumSingleBit(int32(gr.Geo)) {
		utils.LavaFormatError("critical: provider geo req is not single bit", fmt.Errorf("invalid geo req"),
			utils.Attribute{Key: "geoReq", Value: gr.Geo},
		)
		return calculateCostFromLatency(maxGeoLatency)
	}

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

func GetGeoReqsForSlots(policy planstypes.Policy) []GeoReq {
	geoReqsForSlots := []GeoReq{}

	policyGeoEnums := types.GetGeolocationsFromUint(int32(policy.GeolocationProfile))
	switch {
	// TODO: implement the case below
	// case len(policyGeoEnums) > int(policy.MaxProvidersToPair):
	default:
		for i := 0; i < int(policy.MaxProvidersToPair); i++ {
			geoReq := GeoReq{Geo: uint64(policyGeoEnums[i%len(policyGeoEnums)])}
			geoReqsForSlots = append(geoReqsForSlots, geoReq)
		}
	}

	return geoReqsForSlots
}

// a single geolocation and the latency to it (in millieseconds)
type GeoLatency struct {
	geo     planstypes.Geolocation
	latency uint64
}

// CalcGeoCost() finds the minimal latency between the required geo and the provider's supported geolocations
func CalcGeoCost(reqGeo planstypes.Geolocation, providerGeos []planstypes.Geolocation) (minLatencyGeo planstypes.Geolocation, minLatencyCost uint64) {
	var minGeo planstypes.Geolocation
	minLatency := uint64(math.MaxUint64)
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
	if minLatency == math.MaxUint64 {
		return -1, calculateCostFromLatency(maxGeoLatency)
	}

	return minGeo, calculateCostFromLatency(minLatency)
}

func calculateCostFromLatency(latency uint64) uint64 {
	return maxGeoLatency / latency
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
