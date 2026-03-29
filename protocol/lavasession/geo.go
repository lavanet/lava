package lavasession

import (
	planstypes "github.com/lavanet/lava/v5/x/plans/types"
)

const maxGeoLatency int64 = 10000

// CalcGeoLatency computes the best matching geolocation and its latency from a set of provider
// geolocations for a given request geolocation. It was previously sourced from
// x/pairing/keeper/scores but is inlined here to remove the blockchain-module dependency.
func CalcGeoLatency(reqGeo planstypes.Geolocation, providerGeos []planstypes.Geolocation) (planstypes.Geolocation, int64) {
	minGeo := planstypes.Geolocation(-1)
	minLatency := maxGeoLatency
	for _, pGeo := range providerGeos {
		if pGeo == reqGeo {
			minGeo = pGeo
			minLatency = 1
			continue
		}
		if inner, ok := GEO_LATENCY_MAP[reqGeo]; ok {
			if latency, ok := inner[pGeo]; ok {
				if latency < minLatency {
					minGeo = pGeo
					minLatency = latency
				}
			}
		}
	}
	return minGeo, minLatency
}

// GEO_LATENCY_MAP holds approximate inter-region latencies in milliseconds.
// It was previously defined in x/pairing/keeper/scores.
var GEO_LATENCY_MAP = map[planstypes.Geolocation]map[planstypes.Geolocation]int64{
	planstypes.Geolocation_AS: {
		planstypes.Geolocation_AU: 146,
		planstypes.Geolocation_EU: 155,
	},
	planstypes.Geolocation_USE: {
		planstypes.Geolocation_USC: 42,
		planstypes.Geolocation_USW: 68,
		planstypes.Geolocation_EU:  116,
	},
	planstypes.Geolocation_USW: {
		planstypes.Geolocation_USC: 45,
		planstypes.Geolocation_USE: 68,
	},
	planstypes.Geolocation_USC: {
		planstypes.Geolocation_USE: 42,
		planstypes.Geolocation_USW: 45,
		planstypes.Geolocation_EU:  170,
	},
	planstypes.Geolocation_EU: {
		planstypes.Geolocation_USE: 116,
		planstypes.Geolocation_AF:  138,
		planstypes.Geolocation_AS:  155,
		planstypes.Geolocation_USC: 170,
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
