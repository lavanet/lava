package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestUnstakeClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	addstr := "cosmos163f3uxu8tlvtm9dw6pd4hgd3k5tnh7d3w50vee"
	addr, _ := sdk.AccAddressFromBech32(addstr)
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: addstr, ChainID: specName, Amount: sdk.NewCoin("stake", sdk.NewInt(100)), Geolocation: 1})

	tests := []struct {
		name    string
		chainID string
		valid   bool
	}{
		{"HappyFlow", specName, true},
		{"WrongChain", "Not" + specName, false}, //todo yarom
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: addstr, ChainID: tt.chainID})
			require.Equal(t, err == nil, tt.valid)
		})
	}
}

func TestUnstakeNotStakedClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	addstr := "cosmos163f3uxu8tlvtm9dw6pd4hgd3k5tnh7d3w50vee"
	addr, _ := sdk.AccAddressFromBech32(addstr)
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	tests := []struct {
		name    string
		chainID string
		valid   bool
	}{
		{"HappyFlow", specName, false},
		{"WrongChain", "not" + specName, false}, //todo yarom
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: addstr, ChainID: tt.chainID})
			require.Equal(t, err == nil, tt.valid)
		})
	}
}

func TestDoubleUnstakeClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	addstr := "cosmos163f3uxu8tlvtm9dw6pd4hgd3k5tnh7d3w50vee"
	addr, _ := sdk.AccAddressFromBech32(addstr)
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: addstr, ChainID: specName, Amount: sdk.NewCoin("stake", sdk.NewInt(100)), Geolocation: 1})

	tests := []struct {
		name    string
		chainID string
		valid   bool
	}{
		{"HappyFlow", specName, false},
		{"WrongChain", "Not" + specName, false}, //todo yarom
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: addstr, ChainID: tt.chainID})
			_, err := servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: addstr, ChainID: tt.chainID})
			require.Equal(t, err == nil, tt.valid)
		})
	}
}
