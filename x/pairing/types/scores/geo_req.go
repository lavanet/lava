package score

import (
	"fmt"
	"math"
	"sort"

	commontypes "github.com/lavanet/lava/common/types"
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
	GEO_REQ_NAME  = "geo-req"
	maxGeoLatency = 10000 // highest geo cost < 300
	minGeoLatency = 1
)

// Score calculates the geo score of a provider (using the GEO_LATENCY_MAP)
// The score is (maxGeoLatency / minLatency)^geoWeight
// Note: the geo slots are created so that the GeoReq has a single bit geolocation. This function assumes that this is the case
func (gr GeoReq) Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64 {
	if !types.IsGeoEnumSingleBit(int32(gr.Geo)) {
		utils.LavaFormatError("provider geo req is not single bit", fmt.Errorf("invalid geo req"),
			utils.Attribute{Key: "geoReq", Value: gr.Geo},
		)
		return calculateCostFromLatency(maxGeoLatency)
	}

	// check if the provider supports the required geolocation
	if gr.Geo&^provider.Geolocation == 0 {
		cost := calculateCostFromLatency(minGeoLatency)
		return uint64(math.Pow(float64(cost), float64(weight)))
	}

	providerGeoEnums := types.GetGeolocationsFromUint(int32(provider.Geolocation))
	_, cost := GetGeoCost(planstypes.Geolocation(gr.Geo), providerGeoEnums)

	return commontypes.SafePow(cost, weight)
}

func (gr GeoReq) GetName() string {
	return GEO_REQ_NAME
}

// Equal() used to compare slots to determine slot groups
func (gr GeoReq) Equal(other ScoreReq) bool {
	otherGeoReq, ok := other.(GeoReq)
	if !ok {
		return false
	}

	return otherGeoReq.Geo == gr.Geo
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

func (gl GeoLatency) Less(other GeoLatency) bool {
	return gl.latency < other.latency
}

// GetGeoCost() finds the minimal latency between the required geo and the provider's supported geolocations
func GetGeoCost(reqGeo planstypes.Geolocation, providerGeos []planstypes.Geolocation) (minLatencyGeo planstypes.Geolocation, minLatencyCost uint64) {
	geoLatencies := []GeoLatency{}
	latencies := []uint64{}
	for _, pGeo := range providerGeos {
		geoLatency := getGeoLatency(reqGeo, pGeo)
		if geoLatency.latency == 0 {
			continue
		}
		geoLatencies = append(geoLatencies, geoLatency)
		latencies = append(latencies, geoLatency.latency)
	}

	// no geo latencies found -> provider can't support this geo (score = 1 -> max latency)
	if len(geoLatencies) == 0 {
		return -1, calculateCostFromLatency(maxGeoLatency)
	}

	minIndex := commontypes.FindMin(latencies)
	minLatencyGeo = geoLatencies[minIndex].geo
	minLatencyCost = calculateCostFromLatency(geoLatencies[minIndex].latency)
	return minLatencyGeo, minLatencyCost
}

func getGeoLatency(from planstypes.Geolocation, to planstypes.Geolocation) GeoLatency {
	costList := GEO_LATENCY_MAP[from]
	for _, geoLatency := range costList {
		if geoLatency.geo == to {
			return geoLatency
		}
	}

	return GeoLatency{}
}

func calculateCostFromLatency(latency uint64) uint64 {
	return maxGeoLatency / latency
}

// define shortened names for geolocations (for convinience only)
var (
	USC = planstypes.Geolocation_USC
	EU  = planstypes.Geolocation_EU
	USE = planstypes.Geolocation_USE
	USW = planstypes.Geolocation_USW
	AF  = planstypes.Geolocation_AF
	AS  = planstypes.Geolocation_AS
	AU  = planstypes.Geolocation_AU
)

// GEO_LATENCY_MAP is a map of lists of GeoLatency that defines the cost of geo mismatch
// for each single geolocation. The map key is a single geolocation and the value is an
// ordered list of neighbors and their latency (ordered by latency)
// latency data from: https://wondernetwork.com/pings
var GEO_LATENCY_MAP = map[planstypes.Geolocation][]GeoLatency{
	AS: {
		{geo: AU, latency: 146},
		{geo: EU, latency: 155},
	},
	USE: {
		{geo: USC, latency: 42},
		{geo: USW, latency: 68},
	},
	USW: {
		{geo: USC, latency: 45},
		{geo: USE, latency: 68},
	},
	USC: {
		{geo: USE, latency: 42},
		{geo: USW, latency: 45},
	},
	EU: {
		{geo: USE, latency: 116},
		{geo: AF, latency: 138},
		{geo: AS, latency: 155},
	},
	AF: {
		{geo: EU, latency: 138},
		{geo: USE, latency: 203},
		{geo: AS, latency: 263},
	},
	AU: {
		{geo: AS, latency: 146},
		{geo: USW, latency: 179},
	},
}

// verifyGeoScoreForTesting is used to testing purposes only!
// it verifies that the max geo score are for providers that exactly match the geo req
// this function assumes that all the other reqs are equal (for example, stake req)
func VerifyGeoScoreForTesting(providerScores []*PairingScore, slot *PairingSlot, expectedGeoSeen map[uint64]bool) bool {
	if slot == nil || len(providerScores) == 0 {
		return false
	}

	sort.Slice(providerScores, func(i, j int) bool {
		return providerScores[i].Score > providerScores[j].Score
	})

	geoReq, ok := slot.Reqs[GEO_REQ_NAME].(GeoReq)
	if !ok {
		return false
	}

	// verify that the geo is part of the expected geo
	_, ok = expectedGeoSeen[geoReq.Geo]
	if !ok {
		return false
	}
	expectedGeoSeen[geoReq.Geo] = true

	// verify that only providers that match with req geo have max score
	maxScore := providerScores[0].Score
	for _, score := range providerScores {
		if score.Provider.Geolocation == geoReq.Geo {
			if score.Score != maxScore {
				return false
			}
		} else {
			if score.Score == maxScore {
				return false
			} else {
				break
			}
		}
	}

	return true
}
