package keeper_test

import (
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/client/cli"
	"github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// Test that the optional moniker argument in StakeProvider doesn't break anything
func TestStakeProviderWithMoniker(t *testing.T) {
	// Create teststruct ts
	ts := &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochstoragetypes.DefaultGenesis().EpochDetails)
	// Create a mock spec
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// define tests (valid indicates whether the test should succeed)
	tests := []struct {
		name         string
		moniker      string
		validStake   bool
		validMoniker bool
	}{
		{"NormalMoniker", "exampleMoniker", true, true},
		{"WeirdCharsMoniker", "ビッグファームへようこそ", true, true},
		{"OversizedMoniker", "aReallyReallyReallyReallyReallyReallyReallyLongMoniker", true, false}, // validMoniker = false because moniker should be < 50 characters -> the original moniker won't be equal to the assigned one
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Advance epoch
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			// Stake provider with moniker
			sk, address := sigs.GenerateFloatingKey()
			ts.providers = append(ts.providers, &common.Account{SK: sk, Addr: address})
			err := ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), address, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
			require.Nil(t, err)
			endpoints := []epochstoragetypes.Endpoint{}
			endpoints = append(endpoints, epochstoragetypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
			_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: address.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints, Moniker: tt.moniker})
			require.Nil(t, err)

			// Advance epoch to apply the stake
			ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

			// Get the stake entry and check the provider is staked
			stakeEntry, foundProvider, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), address)
			require.Equal(t, tt.validStake, foundProvider)

			// Check the assigned moniker
			if tt.validMoniker {
				require.Equal(t, tt.moniker, stakeEntry.Moniker)
			} else {
				require.NotEqual(t, tt.moniker, stakeEntry.Moniker)
			}
		})
	}
}

func TestModifyStakeProviderWithMoniker(t *testing.T) {
	// Create teststruct ts
	ts := &testStruct{
		providers: make([]*common.Account, 0),
		clients:   make([]*common.Account, 0),
	}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)
	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochstoragetypes.DefaultGenesis().EpochDetails)
	// Create a mock spec
	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	// Advance epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	moniker := "exampleMoniker"

	// Stake provider with moniker
	sk, address := sigs.GenerateFloatingKey()
	ts.providers = append(ts.providers, &common.Account{SK: sk, Addr: address})
	err := ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), address, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
	require.Nil(t, err)
	endpoints := []epochstoragetypes.Endpoint{}
	endpoints = append(endpoints, epochstoragetypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
	_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: address.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake/2)), Geolocation: 1, Endpoints: endpoints, Moniker: moniker})
	require.Nil(t, err)

	// Advance epoch to apply the stake
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider, _ := ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), address)
	require.True(t, foundProvider)
	require.Equal(t, moniker, stakeEntry.Moniker)

	// modify moniker
	moniker = "anotherExampleMoniker"
	_, err = ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: address.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints, Moniker: moniker})
	require.Nil(t, err)

	// Advance epoch to apply the stake
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// Get the stake entry and check the provider is staked
	stakeEntry, foundProvider, _ = ts.keepers.Epochstorage.GetStakeEntryByAddressCurrent(sdk.UnwrapSDKContext(ts.ctx), ts.spec.GetIndex(), address)
	require.True(t, foundProvider)

	require.Equal(t, moniker, stakeEntry.Moniker)
}

func TestCmdStakeProviderGeoConfigAndEnum(t *testing.T) {
	currentGlobalGeo := types.GetCurrentGlobalGeolocation()

	testCases := []struct {
		name        string
		endpoints   []string
		geolocation string
		validConfig bool
		validGeo    bool
	}{
		// single uint geolocation config tests
		{
			name:        "Single uint geolocation - happy flow",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,1"},
			geolocation: "1",
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Single uint geolocation - endpoint geo not equal to geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,2"},
			geolocation: "1",
			validConfig: true,
			validGeo:    false,
		},
		{
			name:        "Single uint geolocation - endpoint geo not equal to geo (geo includes endpoint geo)",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,1"},
			geolocation: "3",
			validConfig: true,
			validGeo:    false,
		},
		{
			name:        "Single uint geolocation - endpoint has geo of multiple regions",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,3"},
			geolocation: "3",
			validConfig: false,
		},
		{
			name:        "Single uint geolocation - bad endpoint geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,20555"},
			geolocation: "1",
			validConfig: false,
		},

		// single string geolocation config tests
		{
			name:        "Single string geolocation - happy flow",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,EU"},
			geolocation: "EU",
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Single string geolocation - endpoint geo not equal to geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,AS"},
			geolocation: "EU",
			validConfig: true,
			validGeo:    false,
		},
		{
			name:        "Single string geolocation - endpoint geo not equal to geo (geo includes endpoint geo)",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,EU"},
			geolocation: "EU,USC",
			validConfig: true,
			validGeo:    false,
		},
		{
			name:        "Single string geolocation - endpoint has geo of multiple regions",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,EU,USC"},
			geolocation: "3",
			validConfig: false,
		},
		{
			name:        "Single string geolocation - bad geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,EU"},
			geolocation: "BLABLA",
			validConfig: false,
		},
		{
			name:        "Single string geolocation - bad geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,BLABLA"},
			geolocation: "EU",
			validConfig: false,
		},

		// multiple uint geolocation config tests
		{
			name:        "Multiple uint geolocations - happy flow",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,1", "127.0.0.1:3352,jsonrpc,2"},
			geolocation: "3",
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Multiple uint geolocations - endpoint geo not equal to geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,1", "127.0.0.1:3352,jsonrpc,4"},
			geolocation: "2",
			validConfig: true,
			validGeo:    false,
		},
		{
			name:        "Multiple uint geolocations - one endpoint has multi-region geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,1", "127.0.0.1:3352,jsonrpc,3"},
			geolocation: "2",
			validConfig: false,
		},

		// multiple string geolocation config tests
		{
			name:        "Multiple string geolocations - happy flow",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,AS", "127.0.0.1:3352,jsonrpc,EU"},
			geolocation: "EU,AS",
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Multiple string geolocations - endpoint geo not equal to geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,EU", "127.0.0.1:3352,jsonrpc,USC"},
			geolocation: "EU,AS",
			validConfig: true,
			validGeo:    false,
		},

		// global config tests
		{
			name:        "Global uint geolocation - happy flow",
			endpoints:   []string{"127.0.0.1:3352,jsonrpc,65535"},
			geolocation: strconv.FormatUint(currentGlobalGeo, 10),
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Global uint geolocation - happy flow 2 - global in one endpoint",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,2", "127.0.0.1:3352,jsonrpc,65535"},
			geolocation: "65535",
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Global uint geolocation - endpoint geo not match geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,2", "127.0.0.1:3352,jsonrpc,65535"},
			geolocation: "7",
			validConfig: true,
			validGeo:    false,
		},
		{
			name:        "Global string geolocation - happy flow",
			endpoints:   []string{"127.0.0.1:3352,jsonrpc,GL"},
			geolocation: "GL",
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Global string geolocation - happy flow 2 - global in one endpoint",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,EU", "127.0.0.1:3352,jsonrpc,GL"},
			geolocation: "GL",
			validConfig: true,
			validGeo:    true,
		},
		{
			name:        "Global string geolocation - endpoint geo not match geo",
			endpoints:   []string{"127.0.0.1:3351,jsonrpc,EU", "127.0.0.1:3352,jsonrpc,GL"},
			geolocation: "EU,AS,USC",
			validConfig: true,
			validGeo:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endp, geo, err := cli.HandleEndpointsAndGeolocationArgs(tc.endpoints, tc.geolocation)
			if tc.validConfig {
				require.Nil(t, err)
				err = types.ValidateGeoFields(endp, geo)
				if tc.validGeo {
					require.Nil(t, err)
				} else {
					require.NotNil(t, err)
				}
			} else {
				require.NotNil(t, err)
			}
		})
	}
}
