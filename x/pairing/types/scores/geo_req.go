package score

import (
	"math"

	commontypes "github.com/lavanet/lava/common/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
)

// stake requirement that implements the ScoreReq interface
type GeoReq struct {
	Geo uint64
}

const (
	GEO_REQ_NAME = "geo-req"
)

// calculates the stake score of a provider (which is simply the normalized stake)
func (gr GeoReq) Score(provider epochstoragetypes.StakeEntry, weight uint64) uint64 {
	providerGeo := int32(provider.Geolocation)
	missingGeos := providerGeo ^ int32(gr.Geo)

	providerGeoEnums := types.GetGeolocationsFromUint(providerGeo)
	missingGeoEnums := types.GetGeolocationsFromUint(missingGeos)

	costs := []uint64{}
	for _, reqGeo := range missingGeoEnums {
		_, latency := GetGeoCost(reqGeo, providerGeoEnums)
		costs = append(costs, latency)
	}

	score := uint64(0)
	for _, cost := range costs {
		cost = uint64(math.Pow(float64(cost), float64(weight)))
		score += cost
	}

	return score
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

// a single geolocation and the latency to it (in millieseconds)
type GeoLatency struct {
	geo     planstypes.Geolocation
	latency uint64
}

func (gl GeoLatency) Less(other GeoLatency) bool {
	return gl.latency < other.latency
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

// GetGeoCost() finds the minimal latency between the required geo and the provider's supported geolocations
func GetGeoCost(reqGeo planstypes.Geolocation, providerGeos []planstypes.Geolocation) (planstypes.Geolocation, uint64) {
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

	// no geo latencies found -> provider can't support this geo
	// returning latency=0 so he'll never be picked
	if len(geoLatencies) == 0 {
		return -1, 0
	}

	minIndex := commontypes.FindMin(latencies)
	return geoLatencies[minIndex].geo, geoLatencies[minIndex].latency
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
