package scores

import (
	"testing"

	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/stretchr/testify/require"
)

// TestCalcGeoCost tests the CalcGeoCost that is called when the provider doesn't support the required geolocation
func TestCalcGeoCost(t *testing.T) {
	tests := []struct {
		name             string
		reqGeo           planstypes.Geolocation
		providerGeos     []planstypes.Geolocation
		expectedGeo      planstypes.Geolocation
		expectedCostUint int64
	}{
		{
			name:             "happy flow",
			reqGeo:           planstypes.Geolocation(1),
			providerGeos:     []planstypes.Geolocation{planstypes.Geolocation_USE, planstypes.Geolocation_EU},
			expectedGeo:      planstypes.Geolocation_USE,
			expectedCostUint: GEO_LATENCY_MAP[planstypes.Geolocation_USC][planstypes.Geolocation_USE],
		},
		{
			name:             "happy flow - reverse provider geos order",
			reqGeo:           planstypes.Geolocation(1),
			providerGeos:     []planstypes.Geolocation{planstypes.Geolocation_EU, planstypes.Geolocation_USE},
			expectedGeo:      planstypes.Geolocation_USE,
			expectedCostUint: GEO_LATENCY_MAP[planstypes.Geolocation_USC][planstypes.Geolocation_USE],
		},
		{
			name:             "provider not supporting geo requirement",
			reqGeo:           planstypes.Geolocation(1),
			providerGeos:     []planstypes.Geolocation{planstypes.Geolocation_AU, planstypes.Geolocation_AS},
			expectedGeo:      -1,
			expectedCostUint: maxGeoLatency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minGeo, minCost := CalcGeoCost(tt.reqGeo, tt.providerGeos)
			require.Equal(t, tt.expectedGeo, minGeo)
			require.True(t, calculateCostFromLatency(tt.expectedCostUint).Equal(minCost))
		})
	}
}

func TestGeoReqScore(t *testing.T) {
	stakeEntry := epochstoragetypes.StakeEntry{}

	geoReq := GeoReq{}

	tests := []struct {
		name            string
		reqGeo          int32
		providerGeo     int32
		expectedLatency int64
	}{
		{
			name:            "happy flow - provider supports geo",
			reqGeo:          1,
			providerGeo:     1,
			expectedLatency: minGeoLatency,
		},
		{
			name:            "happy flow - provider supports geo and other regions",
			reqGeo:          1,
			providerGeo:     27,
			expectedLatency: minGeoLatency,
		},
		{
			name:            "happy flow - provider supports global",
			reqGeo:          1,
			providerGeo:     int32(planstypes.Geolocation_GL),
			expectedLatency: minGeoLatency,
		},
		{
			name:            "provider doesn't support geo req but is neighbor",
			reqGeo:          1,
			providerGeo:     int32(planstypes.Geolocation_USE),
			expectedLatency: GEO_LATENCY_MAP[planstypes.Geolocation_USC][planstypes.Geolocation_USE],
		},
		{
			name:            "provider doesn't support geo req but isn't neighbor",
			reqGeo:          1,
			providerGeo:     int32(planstypes.Geolocation_AS),
			expectedLatency: GEO_LATENCY_MAP[planstypes.Geolocation_USC][planstypes.Geolocation_AS],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			geoReq.Geo = tt.reqGeo
			stakeEntry.Geolocation = tt.providerGeo
			pairingScore := NewPairingScore(&stakeEntry, types.QualityOfServiceReport{})
			score := geoReq.Score(*pairingScore)
			require.True(t, score.Equal(calculateCostFromLatency(tt.expectedLatency)))
		})
	}
}
