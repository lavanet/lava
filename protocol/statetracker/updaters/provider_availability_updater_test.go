package updaters

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v3/protocol/common"
	"github.com/lavanet/lava/v3/protocol/lavasession"
	"github.com/lavanet/lava/v3/utils/rand"
	epochstoragetypes "github.com/lavanet/lava/v3/x/epochstorage/types"
)

func TestFrozenAvailability(t *testing.T) {
	rand.InitRandomSeed()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	specID := "test-spec"
	apiInterface := "test-inf"
	initialPairingList := []epochstoragetypes.StakeEntry{
		{
			Address: "initial1",
			Endpoints: []epochstoragetypes.Endpoint{
				{
					IPPORT:        "1234567",
					Geolocation:   1,
					Addons:        []string{},
					ApiInterfaces: []string{"banana"},
					Extensions:    []string{},
				},
			},
			StakeAppliedBlock: 1000,
		},
	}
	// Create a new mock object
	stateQuery := NewMockProviderPairingStatusStateQueryInf(ctrl)
	stateQuery.EXPECT().Providers(gomock.Any(), gomock.Any(), gomock.Any()).Return(initialPairingList, uint64(0), uint64(0), nil).AnyTimes()
	staticProviders := &lavasession.RPCProviderEndpoint{
		NetworkAddress: lavasession.NetworkAddressData{
			Address:    "mockaddress",
			KeyPem:     "",
			CertPem:    "",
			DisableTLS: false,
		},
		ChainID:      specID,
		ApiInterface: apiInterface,
		Geolocation:  1,
		NodeUrls: []common.NodeUrl{
			{
				Url:               "mockurl",
				InternalPath:      "",
				AuthConfig:        common.AuthConfig{},
				IpForwarding:      false,
				Timeout:           0,
				Addons:            nil,
				SkipVerifications: []string{},
			},
		},
	}
	clientCtx := client.Context{}
	pau := NewProviderAvailabilityUpdater(stateQuery, staticProviders, clientCtx, nil)
	pau.runProviderAvailabilityUpdate(1)

}
