package keeper_test

import (
	"fmt"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func TestRelayPayment(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)
	unwrapedCtx := sdk.UnwrapSDKContext(ctx)

	//init keepers state
	proSecretKey, err := btcSecp256k1.NewPrivateKey(btcSecp256k1.S256())
	proAddBytes, err := sdk.Bech32ifyAddressBytes("cosmos", proSecretKey.PubKey().SerializeCompressed())
	proAdd, err := sdk.AccAddressFromBech32(proAddBytes)
	clientSecretKey, _ := btcSecp256k1.NewPrivateKey(btcSecp256k1.S256())
	clientAddBytes, _ := sdk.Bech32ifyAddressBytes("cosmos", proSecretKey.PubKey().SerializeCompressed())
	clientAdd, _ := sdk.AccAddressFromBech32(clientAddBytes)

	var amount int64 = 1000
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), proAdd, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))
	keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ctx), clientAdd, sdk.NewCoins(sdk.NewCoin("stake", sdk.NewInt(amount))))

	//file, _ := ioutil.ReadFile("../../../cookbook/spec_add_etherium.json")
	//spec := spectypes.Spec{}
	//_ = json.Unmarshal([]byte(file), &spec)
	//keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	specName := "mockSpec"
	spec := spectypes.Spec{}
	spec.Name = specName
	spec.Index = specName
	spec.Enabled = true
	spec.Apis = append(spec.Apis, spectypes.ServiceApi{Name: specName + "API", ComputeUnits: 100, Enabled: true, ApiInterfaces: nil})
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	servers.PairingServer.StakeClient(ctx, &types.MsgStakeClient{Creator: clientAdd.String(), ChainID: specName, Amount: sdk.NewCoin("stake", sdk.NewInt(100)), Geolocation: 1})
	servers.PairingServer.StakeProvider(ctx, &types.MsgStakeProvider{Creator: proAdd.String(), ChainID: specName, Amount: sdk.NewCoin("stake", sdk.NewInt(100)), Geolocation: 1})

	keepers.Epochstorage.SetEpochDetails(unwrapedCtx, *epochtypes.DefaultGenesis().EpochDetails)

	relayRequest := &types.RelayRequest{
		Provider:        proAdd.String(),
		ApiUrl:          "",
		Data:            []byte(spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         specName,
		CuSum:           spec.Apis[0].ComputeUnits * 10,
		BlockHeight:     unwrapedCtx.BlockHeight(),
		RelayNum:        0,
		RequestBlock:    0,
		DataReliability: nil,
	}

	sig, _ := sigs.SignRelay(clientSecretKey, *relayRequest)

	relayRequest.Sig = sig
	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequest)
	response, err := servers.PairingServer.RelayPayment(ctx, &types.MsgRelayPayment{Creator: proAdd.String(), Relays: Relays})
	fmt.Printf("%v %v\n", response, err)
}
