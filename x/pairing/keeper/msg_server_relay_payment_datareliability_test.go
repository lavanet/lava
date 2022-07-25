package keeper_test

import (
	"context"
	"fmt"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/coniks-sys/coniks-go/crypto/vrf"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type Account struct {
	secretKey *btcSecp256k1.PrivateKey
	addr      sdk.AccAddress
	vrfSk     vrf.PrivateKey
	vrfPk     vrf.PublicKey
}

type testStructDataReliability struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	providers []Account
	clients   []Account
	spec      spectypes.Spec
}

// same as setupForPaymentTest
func setupForPaymentTestDataReliability(t *testing.T) testStructDataReliability {
	ts := testStructDataReliability{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	ts.providers = make([]Account, 0)
	for i := 0; i < 2; i++ {
		sk, addr := sigs.GenerateFloatingKey()
		ts.providers = append(ts.providers, Account{secretKey: sk, addr: addr})
	}

	ts.clients = make([]Account, 0)
	for i := 0; i < 1; i++ {
		sk, addr := sigs.GenerateFloatingKey()
		vrfSk, vrfPk, _ := utils.GeneratePrivateVRFKey()
		ts.clients = append(ts.clients, Account{secretKey: sk, addr: addr, vrfSk: vrfSk, vrfPk: vrfPk})
	}

	var balance int64 = 100000
	for _, provider := range ts.providers {
		ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), provider.addr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
	}
	for _, client := range ts.clients {
		ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), client.addr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
	}

	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochtypes.DefaultGenesis().EpochDetails)

	ts.spec = common.CreateMockSpec()
	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	var stake int64 = 1000
	for _, client := range ts.clients {
		vrfPk := &utils.VrfPubKey{}
		vrfPk.Unmarshal(client.vrfPk)
		_, err := ts.servers.PairingServer.StakeClient(ts.ctx, &types.MsgStakeClient{Creator: client.addr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Vrfpk: vrfPk.String()})
		require.Nil(t, err)
	}

	for _, provider := range ts.providers {
		endpoints := []epochtypes.Endpoint{}
		endpoints = append(endpoints, epochtypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
		_, err := ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: provider.addr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints})
		require.Nil(t, err)
	}
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	return ts
}

// flow
// since the vrf is random set one manually then just cache the keys and the request

// create 2 providers
// create 1 client
// from client send RelayRequest0 to provider0
// provider0 replies with RelayReply0
// client runs utils.CalculateVrfOnRelay with the RelayRequest0, RelayReply0 and vrf private key (created when staking client)
// client makes a RelayRequest1 with the DataReliability not nil to another provider1 (change RelayNum to 1)
// provider1 receives RelayRequest1 then replies with the data requested (RelayReply1) to client normally
// provider1 does a DataReliability test on the provider0
// provider1 adds RelayRequest1 to his Relays array
// TEST SCOPE STARTS HERE
// corrupt parts or none of relayRequest with datareliability
// provider1 sends a RelayPayment
// check results
// POSSIBLE SCENARIOS
// One of them is corrupted/missing
// type VRFData struct {
// 	Differentiator bool   `protobuf:"varint,1,opt,name=differentiator,proto3" json:"differentiator,omitempty"`
// 	VrfValue       []byte `protobuf:"bytes,2,opt,name=vrf_value,json=vrfValue,proto3" json:"vrf_value,omitempty"`
// 	VrfProof       []byte `protobuf:"bytes,3,opt,name=vrf_proof,json=vrfProof,proto3" json:"vrf_proof,omitempty"`
// 	ProviderSig    []byte `protobuf:"bytes,4,opt,name=provider_sig,json=providerSig,proto3" json:"provider_sig,omitempty"`
// 	AllDataHash    []byte `protobuf:"bytes,5,opt,name=allDataHash,proto3" json:"allDataHash,omitempty"`
// 	QueryHash      []byte `protobuf:"bytes,6,opt,name=queryHash,proto3" json:"queryHash,omitempty"`
// 	Sig            []byte `protobuf:"bytes,7,opt,name=sig,proto3" json:"sig,omitempty"`
// }
// provider attached the vrf to another request double spend

func TestRelayPaymentDataReliability(t *testing.T) {
	ts := setupForPaymentTestDataReliability(t) //reset the keepers state before each state

	cuSum := ts.spec.Apis[0].ComputeUnits * 10

	relayRequest := &types.RelayRequest{
		Provider:        ts.providers[0].addr.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: nil,
	}

	requestSig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequest)
	relayRequest.Sig = requestSig
	require.Nil(t, err)

	var Provider0Relays []*types.RelayRequest
	Provider0Relays = append(Provider0Relays, relayRequest)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].addr.String(), Relays: Provider0Relays})
	require.Nil(t, err)

	relayReply := &types.RelayReply{}
	replySig, err := sigs.SignRelayResponse(ts.providers[0].secretKey, relayReply, relayRequest)
	relayReply.Sig = replySig
	require.Nil(t, err)

	vrf_res, vrf_proof := utils.ProveVrfOnRelay(relayRequest, relayReply, ts.clients[0].vrfSk, false)

	dataReliability := &types.VRFData{
		Differentiator: false,
		VrfValue:       vrf_res,
		VrfProof:       vrf_proof,
		ProviderSig:    relayReply.Sig,
		AllDataHash:    sigs.AllDataHash(relayReply, relayRequest),
		QueryHash:      utils.CalculateQueryHash(*relayRequest),
		Sig:            nil,
	}

	relayRequestReliability := &types.RelayRequest{
		Provider:        ts.providers[1].addr.String(),
		ApiUrl:          "",
		Data:            []byte(ts.spec.Apis[0].Name),
		SessionId:       uint64(1),
		ChainID:         ts.spec.Name,
		CuSum:           cuSum,
		BlockHeight:     sdk.UnwrapSDKContext(ts.ctx).BlockHeight(),
		RelayNum:        0,
		RequestBlock:    -1,
		DataReliability: dataReliability,
	}

	dataReliabilitySig, err := sigs.SignVRFData(ts.clients[0].secretKey, relayRequestReliability.DataReliability)
	require.Nil(t, err)
	relayRequestReliability.DataReliability.Sig = dataReliabilitySig

	relayRequestReliabilitySig, err := sigs.SignRelay(ts.clients[0].secretKey, *relayRequestReliability)
	require.Nil(t, err)
	relayRequestReliability.Sig = relayRequestReliabilitySig

	var Provider1Relays []*types.RelayRequest
	Provider1Relays = append(Provider1Relays, relayRequestReliability)

	balanceBeforeProvider1 := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].addr, epochstoragetypes.TokenDenom).Amount.Int64()
	stakeClientBefore, found, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clients[0].addr)
	require.Equal(t, true, found)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[1].addr.String(), Relays: Provider1Relays})
	require.Nil(t, err)

	mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
	want := mint.MulInt64(int64(cuSum)).TruncateInt64()
	want = want * 105 //multiply by 1.05
	want = want / 100

	balanceAfterProvider1 := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].addr, epochstoragetypes.TokenDenom).Amount.Int64()
	require.Equal(t, balanceBeforeProvider1+want, balanceAfterProvider1)

	// burn := ts.keepers.Pairing.BurnCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx)).MulInt64(int64(cuSum))
	stakeClientAfter, _, _ := ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clients[0].addr)

	// require.Equal(t, stakeClientBefore.Stake.Amount.Int64()-burn.TruncateInt64(), stakeClientAfter.Stake.Amount.Int64())

	fmt.Println(balanceBeforeProvider1, balanceAfterProvider1)
	fmt.Println(stakeClientBefore.Stake.Amount.Int64(), stakeClientAfter.Stake.Amount.Int64())
}
