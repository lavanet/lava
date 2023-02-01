package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
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

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount/10)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	epochsToSave, err := keepers.Epochstorage.EpochsToSave(sdk.UnwrapSDKContext(ctx), uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	require.NoError(t, err)
	tests := []struct {
		name    string
		chainID string
		valid   bool
	}{
		{"HappyFlow", spec.Index, true},
		{"WrongChain", "Not" + spec.Index, false},
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

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	tests := []struct {
		name    string
		chainID string
	}{
		{"SameChain", spec.Index},
		{"WrongChain", "not" + spec.Index},
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

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount/2)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)
	testkeeper.AdvanceEpoch(ctx, keepers)

	tests := []struct {
		name    string
		chainID string
		valid   bool
	}{
		{"SameChain", spec.Index, false},
		{"WrongChain", "Not" + spec.Index, false},
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
