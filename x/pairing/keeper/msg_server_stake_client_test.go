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

func TestNewStakeClient(t *testing.T) {
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
		name  string
		stake sdk.Coin
		valid bool
	}{
		{"MinStake", sdk.NewCoin(spec.MinStakeClient.Denom, spec.MinStakeClient.Amount.Sub(sdk.NewInt(1))), false},
		{"InsufficientFunds", sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount+1)), false},
		{"HappyFlow", sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount/2)), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: tt.stake, Geolocation: 1, Vrfpk: vrfPk.String()})

			ctx = testkeeper.AdvanceEpoch(ctx, keepers)

			if tt.valid {
				require.Nil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, epochstoragetypes.TokenDenom).Amount.Int64()
				require.Equal(t, amount-tt.stake.Amount.Int64(), balance)
			} else {
				require.NotNil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, epochstoragetypes.TokenDenom).Amount.Int64()
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
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount))))

	_, pk, _ := utils.GeneratePrivateVRFKey()
	vrfPk := &utils.VrfPubKey{}
	vrfPk.Unmarshal(pk)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	firstStake := amount / 10
	_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(firstStake)), Geolocation: 1, Vrfpk: vrfPk.String()})
	require.Nil(t, err)
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	tests := []struct {
		name  string
		stake sdk.Coin
		valid bool
	}{
		{"MinStake", sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount/20)), false},
		{"InsufficientFunds", sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount*2)), false},
		{"HappyFlow", sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(amount/2)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAddr.String(), ChainID: spec.Name, Amount: tt.stake, Geolocation: 1, Vrfpk: vrfPk.String()})
			ctx = testkeeper.AdvanceEpoch(ctx, keepers)

			if tt.valid {
				require.Nil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, epochstoragetypes.TokenDenom).Amount.Int64()
				require.Equal(t, amount-tt.stake.Amount.Int64(), balance)
			} else {
				require.NotNil(t, err)
				balance := keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ctx), clientAddr, epochstoragetypes.TokenDenom).Amount.Int64()
				require.Equal(t, amount-firstStake, balance)
			}
		})
	}

}

func TestStakeClientPairingimmediately(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	//init keepers state
	var balance int64 = 10000
	consumer := common.CreateNewAccount(ctx, *keepers, balance)
	provider1 := common.CreateNewAccount(ctx, *keepers, balance)
	provider2 := common.CreateNewAccount(ctx, *keepers, balance)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	stake := balance / 10
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	common.StakeAccount(t, ctx, *keepers, *servers, provider1, spec, stake, true)
	common.StakeAccount(t, ctx, *keepers, *servers, provider2, spec, stake, true)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	common.StakeAccount(t, ctx, *keepers, *servers, consumer, spec, stake, false)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	//check pairing in the same epoch
	clientStakeEntry, err := keepers.Pairing.VerifyPairingData(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	require.Nil(t, err)
	require.Equal(t, clientStakeEntry.Stake.Amount, sdk.NewInt(stake))

	_, err = keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr)
	require.Nil(t, err)

	//try to change stake
	common.StakeAccount(t, ctx, *keepers, *servers, consumer, spec, 2*stake, false)
	clientStakeEntry, err = keepers.Pairing.VerifyPairingData(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	require.Nil(t, err)
	require.Equal(t, clientStakeEntry.Stake.Amount, sdk.NewInt(stake))

	//new stake takes effect
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	clientStakeEntry, err = keepers.Pairing.VerifyPairingData(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr, uint64(sdk.UnwrapSDKContext(ctx).BlockHeight()))
	require.Nil(t, err)
	require.Equal(t, clientStakeEntry.Stake.Amount, sdk.NewInt(2*stake))

}
