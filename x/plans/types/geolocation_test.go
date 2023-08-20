package types_test

import (
	"testing"

	"github.com/lavanet/lava/utils/slices"
	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/stretchr/testify/require"
)

func TestAllGeos(t *testing.T) {
	geos := planstypes.GetAllGeolocations()
	var geosFromMap []int32
	for _, geo := range planstypes.Geolocation_value {
		if geo == int32(planstypes.Geolocation_GL) || geo == int32(planstypes.Geolocation_GLS) {
			continue
		}
		geosFromMap = append(geosFromMap, geo)
	}

	require.True(t, slices.UnorderedEqual(geos, geosFromMap))
}

func TestGetGeoFromUint(t *testing.T) {
	USC := planstypes.Geolocation_USC
	EU := planstypes.Geolocation_EU
	USE := planstypes.Geolocation_USE
	USW := planstypes.Geolocation_USW
	AF := planstypes.Geolocation_AF
	AS := planstypes.Geolocation_AS
	AU := planstypes.Geolocation_AU
	GL := planstypes.Geolocation_GL

	tests := []struct {
		input    int32
		wantList []planstypes.Geolocation
	}{
		{0, []planstypes.Geolocation{}},
		{1, []planstypes.Geolocation{USC}},
		{2, []planstypes.Geolocation{EU}},
		{3, []planstypes.Geolocation{USC, EU}},
		{4, []planstypes.Geolocation{USE}},
		{7, []planstypes.Geolocation{USC, EU, USE}},
		{13, []planstypes.Geolocation{USC, USE, USW}},
		{27, []planstypes.Geolocation{USC, EU, USW, AF}},
		{int32(GL), []planstypes.Geolocation{USC, EU, USE, USW, AF, AS, AU}},
	}

	for _, test := range tests {
		geos := planstypes.GetGeolocationsFromUint(test.input)
		require.Equal(t, len(test.wantList), len(geos))
		for i := 0; i < len(geos); i++ {
			require.Equal(t, test.wantList[i], geos[i])
		}
	}
}
