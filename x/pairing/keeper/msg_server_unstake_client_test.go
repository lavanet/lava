package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestUnstakeClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	_, clientAddr := sigs.GenerateFloatingKey()
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount))))

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
	_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount/10)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	epochsToSave := keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ctx))
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
			_, err := servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: clientAddr.String(), ChainID: tt.chainID})

			if tt.valid {
				require.Nil(t, err)

				for i := 0; i < int(epochsToSave)+1; i++ {
					ctx = testkeeper.AdvanceEpoch(ctx, keepers)
				}

				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, epochstoragetypes.TokenDenom).Amount.Int64()
				require.Equal(t, amount, balance)

			} else {
				require.NotNil(t, err)
			}
		})
	}
}

func TestUnstakeNotStakedClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	_, clientAddr := sigs.GenerateFloatingKey()
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount))))

	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)

	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *epochtypes.DefaultGenesis().EpochDetails)

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
	}{
		{"SameChain", specName},
		{"WrongChain", "not" + specName}, //todo yarom
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: clientAddr.String(), ChainID: tt.chainID})
			require.NotNil(t, err)

			ctx = testkeeper.AdvanceEpoch(ctx, keepers)

			balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, epochstoragetypes.TokenDenom).Amount.Int64()
			require.Equal(t, amount, balance)

		})
	}
}

func TestDoubleUnstakeClient(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	_, clientAddr := sigs.GenerateFloatingKey()
	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount))))

	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)

	keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ctx), *epochtypes.DefaultGenesis().EpochDetails)

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount/2)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)
	testkeeper.AdvanceEpoch(ctx, keepers)

	tests := []struct {
		name    string
		chainID string
		valid   bool
	}{
		{"SameChain", specName, false},
		{"WrongChain", "Not" + specName, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: clientAddr.String(), ChainID: tt.chainID})
			require.Equal(t, spec.Name == tt.chainID, err == nil)
			_, err = servers.PairingServer.UnstakeClient(ctx, &types.MsgUnstakeClient{Creator: clientAddr.String(), ChainID: tt.chainID})
			require.NotNil(t, err)
		})
	}
}
