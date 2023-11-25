package lavasession

import (
	"strings"
	"testing"

	planstypes "github.com/lavanet/lava/x/plans/types"
	"github.com/stretchr/testify/require"
)

type printGeos []*Endpoint

func (pg printGeos) String() string {
	stringsArr := []string{}
	for _, endp := range pg {
		stringsArr = append(stringsArr, endp.Geolocation.String())
	}
	return strings.Join(stringsArr, ",")
}

func TestGeoOrdering(t *testing.T) {
	pairingEndpoints := []*Endpoint{
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_EU,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_AF,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_AS,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_AU,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_USE,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_USC,
		},
		{
			NetworkAddress: "",
			Enabled:        true,
			Geolocation:    planstypes.Geolocation_USW,
		},
	}

	playbook := []struct {
		name          string
		currentGeo    planstypes.Geolocation
		expectedOrder []planstypes.Geolocation
	}{
		{
			name:       "USC",
			currentGeo: planstypes.Geolocation_USC,
			expectedOrder: []planstypes.Geolocation{
				planstypes.Geolocation_USC,
				planstypes.Geolocation_USE,
				planstypes.Geolocation_USW,
				planstypes.Geolocation_EU,
				planstypes.Geolocation_AF,
				planstypes.Geolocation_AS,
				planstypes.Geolocation_AU,
			},
		},
		{
			name:       "USW",
			currentGeo: planstypes.Geolocation_USW,
			expectedOrder: []planstypes.Geolocation{
				planstypes.Geolocation_USW,
				planstypes.Geolocation_USC,
				planstypes.Geolocation_USE,
				planstypes.Geolocation_AU,
				planstypes.Geolocation_EU,
				planstypes.Geolocation_AF,
				planstypes.Geolocation_AS,
			},
		},
		{
			name:       "EU",
			currentGeo: planstypes.Geolocation_EU,
			expectedOrder: []planstypes.Geolocation{
				planstypes.Geolocation_EU,
				planstypes.Geolocation_USE,
				planstypes.Geolocation_AF,
				planstypes.Geolocation_AS,
				planstypes.Geolocation_USC,
			},
		},
	}

	for _, play := range playbook {
		t.Run(play.name, func(t *testing.T) {
			SortByGeolocations(pairingEndpoints, play.currentGeo)
			printme := printGeos(pairingEndpoints)
			for idx := range play.expectedOrder {
				require.Equal(t, play.expectedOrder[idx].String(), pairingEndpoints[idx].Geolocation.String(), "different order in index %d %s current Geo: %s", idx, printme, play.currentGeo)
			}
		})
	}

}
