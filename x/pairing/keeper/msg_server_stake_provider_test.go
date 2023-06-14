package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils/sigs"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
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
