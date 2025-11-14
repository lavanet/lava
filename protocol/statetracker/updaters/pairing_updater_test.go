package updaters

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/utils/rand"
	epochstoragetypes "github.com/lavanet/lava/v5/x/epochstorage/types"
	"github.com/stretchr/testify/require"
)

type matcher struct {
	expected map[uint64]*lavasession.ConsumerSessionsWithProvider
}

func (m matcher) Matches(arg interface{}) bool {
	actual, ok := arg.(map[uint64]*lavasession.ConsumerSessionsWithProvider)
	if !ok {
		return false
	}
	if len(actual) != len(m.expected) {
		return false
	}
	for k, v := range m.expected {
		if actual[k].StaticProvider != v.StaticProvider {
			return false
		}
	}
	return true
}

func (m matcher) String() string {
	return ""
}

func TestPairingUpdater(t *testing.T) {
	rand.InitRandomSeed()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	specID := "test-spec"
	apiInterface := "test-inf"
	initialPairingList := []epochstoragetypes.StakeEntry{
		{
			Address: "initial2",
			Endpoints: []epochstoragetypes.Endpoint{
				{
					IPPORT:        "1234567",
					Geolocation:   1,
					Addons:        []string{},
					ApiInterfaces: []string{"banana"},
					Extensions:    []string{},
				},
			},
			Geolocation: 1,
			Chain:       specID,
		},
		{
			Address: "initial0",
			Endpoints: []epochstoragetypes.Endpoint{
				{
					IPPORT:        "123",
					Geolocation:   0,
					Addons:        []string{},
					ApiInterfaces: []string{apiInterface},
					Extensions:    []string{},
				},
			},
			Geolocation: 0,
			Chain:       specID,
		},
		{
			Address: "initial1",
			Endpoints: []epochstoragetypes.Endpoint{
				{
					IPPORT:        "1234",
					Geolocation:   0,
					Addons:        []string{},
					ApiInterfaces: []string{apiInterface},
					Extensions:    []string{},
				},
			},
			Geolocation: 0,
			Chain:       specID,
		},
	}
	// Create a new mock object
	stateQuery := NewMockConsumerStateQueryInf(ctrl)
	stateQuery.EXPECT().GetPairing(gomock.Any(), gomock.Any(), gomock.Any()).Return(initialPairingList, uint64(0), uint64(0), nil).AnyTimes()
	stateQuery.EXPECT().GetMaxCUForUser(gomock.Any(), gomock.Any(), gomock.Any()).Return(uint64(999999999), nil).AnyTimes()

	t.Run("UpdateStaticProviders", func(t *testing.T) {
		pu := NewPairingUpdater(stateQuery, specID)
		staticProviders := []*lavasession.RPCStaticProviderEndpoint{
			{
				ChainID:      specID,
				ApiInterface: apiInterface,
				Geolocation:  0,
				NodeUrls: []common.NodeUrl{
					{
						Url: "0123",
					},
				},
				Name: "TestProvider1",
			},
			{
				ChainID:      "banana",
				ApiInterface: apiInterface,
				Geolocation:  0,
				NodeUrls: []common.NodeUrl{
					{
						Url: "01234",
					},
				},
				Name: "TestProvider2",
			},
			{
				ChainID:      "specID",
				ApiInterface: "wrong",
				Geolocation:  0,
				NodeUrls: []common.NodeUrl{
					{
						Url: "01235",
					},
				},
				Name: "TestProvider3",
			},
		}

		pu.updateStaticProviders(staticProviders)

		// only one of the specs is relevant
		require.Len(t, pu.staticProviders, 1)

		staticProviders = []*lavasession.RPCStaticProviderEndpoint{
			{
				ChainID:      specID,
				ApiInterface: apiInterface,
				Geolocation:  0,
				NodeUrls: []common.NodeUrl{
					{
						Url: "01236",
					},
				},
				Name: "TestProvider4",
			},
			{
				ChainID:      "banana",
				ApiInterface: apiInterface,
				Geolocation:  0,
				NodeUrls: []common.NodeUrl{
					{
						Url: "01237",
					},
				},
				Name: "TestProvider5",
			},
		}

		pu.updateStaticProviders(staticProviders)

		// can only update them once
		require.Len(t, pu.staticProviders, 1)
	})

	t.Run("RegisterPairing", func(t *testing.T) {
		pu := NewPairingUpdater(stateQuery, specID)
		consumerSessionManager := NewMockConsumerSessionManagerInf(ctrl)
		consumerSessionManager.EXPECT().RPCEndpoint().Return(lavasession.RPCEndpoint{
			ChainID:      specID,
			ApiInterface: apiInterface,
			Geolocation:  0,
		}).AnyTimes()

		staticProviders := []*lavasession.RPCStaticProviderEndpoint{
			{
				ChainID:      specID,
				ApiInterface: apiInterface,
				Geolocation:  0,
				NodeUrls: []common.NodeUrl{
					{
						Url: "00123",
					},
				},
				Name: "TestStaticProvider",
			},
		}
		pairingMatcher := matcher{
			expected: map[uint64]*lavasession.ConsumerSessionsWithProvider{
				1: {},
				2: {},
				3: {StaticProvider: true},
			},
		}
		consumerSessionManager.EXPECT().UpdateAllProviders(gomock.Any(), pairingMatcher, gomock.Any()).Times(1).Return(nil)
		err := pu.RegisterPairing(context.Background(), consumerSessionManager, staticProviders, nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(pu.consumerSessionManagersMap) != 1 {
			t.Errorf("Expected 1 consumer session manager, got %d", len(pu.consumerSessionManagersMap))
		}

		consumerSessionManager.EXPECT().UpdateAllProviders(gomock.Any(), pairingMatcher, gomock.Any()).Times(1).Return(nil)
		pu.Update(20)
	})
}
