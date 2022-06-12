package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestNewStakeClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	_, clientAddr := sigs.GenerateFloatingKey()
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))

	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *epochtypes.DefaultGenesis().EpochDetails)

	tests := []struct {
		name  string
		stake sdk.Coin
		valid bool
	}{
		{"MinStake", sdk.NewCoin("stake", sdk.NewInt(10)), false},
		{"InsufficientFunds", sdk.NewCoin("stake", sdk.NewInt(amount+1)), false},
		{"HappyFlow", sdk.NewCoin("stake", sdk.NewInt(amount/2)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: tt.stake, Geolocation: 1, Vrfpk: vrfPk.String()})

			ctx = testkeeper.AdvanceEpoch(ctx, keepers)

			if tt.valid {
				require.Nil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, "stake").Amount.Int64()
				require.Equal(t, amount-tt.stake.Amount.Int64(), balance)
			} else {
				require.NotNil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, "stake").Amount.Int64()
				require.Equal(t, amount, balance)
			}
		})
	}

}

func TestAddStakeClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	_, clientAddr := sigs.GenerateFloatingKey()
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))

	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *epochtypes.DefaultGenesis().EpochDetails)
	firstStake := amount / 10
	_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: sdk.NewCoin("stake", sdk.NewInt(firstStake)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	tests := []struct {
		name  string
		stake sdk.Coin
		valid bool
	}{
		{"MinStake", sdk.NewCoin("stake", sdk.NewInt(amount/20)), false},
		{"InsufficientFunds", sdk.NewCoin("stake", sdk.NewInt(amount*2)), false},
		{"HappyFlow", sdk.NewCoin("stake", sdk.NewInt(amount/2)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: tt.stake, Geolocation: 1, Vrfpk: vrfPk.String()})
			ctx = testkeeper.AdvanceEpoch(ctx, keepers)

			if tt.valid {
				require.Nil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, "stake").Amount.Int64()
				require.Equal(t, amount-tt.stake.Amount.Int64(), balance)
			} else {
				require.NotNil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, "stake").Amount.Int64()
				require.Equal(t, amount-firstStake, balance)
			}
		})
	}

}
