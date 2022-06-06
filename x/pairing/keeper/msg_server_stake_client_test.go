package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestNewStakeClient(t *testing.T) {
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
		name  string
		stake sdk.Coin
		valid bool
	}{
		{"HappyFlow", sdk.NewCoin("stake", sdk.NewInt(amount/2)), true},
		{"MinStake", sdk.NewCoin("stake", sdk.NewInt(10)), false},
		{"InsufficientFunds", sdk.NewCoin("stake", sdk.NewInt(amount+1)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: addstr, ChainID: specName, Amount: tt.stake, Geolocation: 1})
			require.Equal(t, tt.valid, err == nil)
		})
	}

}

func TestAddStakeClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	addstr := "cosmos163f3uxu8tlvtm9dw6pd4hgd3k5tnh7d3w50vee"
	addr, _ := sdk.AccAddressFromBech32(addstr)
	var amount int64 = 10000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), addr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: addstr, ChainID: specName, Amount: sdk.NewCoin("stake", sdk.NewInt(amount/10)), Geolocation: 1})

	tests := []struct {
		name  string
		stake sdk.Coin
		valid bool
	}{
		{"HappyFlow", sdk.NewCoin("stake", sdk.NewInt(amount/2)), true},
		{"MinStake", sdk.NewCoin("stake", sdk.NewInt(amount/20)), false},
		{"InsufficientFunds", sdk.NewCoin("stake", sdk.NewInt(amount*2)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: addstr, ChainID: specName, Amount: tt.stake, Geolocation: 1})
			require.Equal(t, tt.valid, err == nil)
		})
	}

}
