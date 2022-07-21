package keeper_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/coniks-sys/coniks-go/crypto/vrf"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/utils"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	epochtypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/pairing/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

type Account struct {
	secretKey *btcSecp256k1.PrivateKey
	addr      sdk.AccAddress
	vrfSk     vrf.PrivateKey
	lavaAddr  string
}

type testStructDataReliability struct {
	ctx       context.Context
	keepers   *testkeeper.Keepers
	servers   *testkeeper.Servers
	providers []Account
	clients   []Account
	spec      spectypes.Spec
}

func LoadHexPrivateKey(hexKey string) (secretKey *btcSecp256k1.PrivateKey, addr sdk.AccAddress) {
	privBytes, _ := hexutil.Decode(hexKey)
	secretKey, _ = btcSecp256k1.PrivKeyFromBytes(btcSecp256k1.S256(), privBytes)
	publicBytes := (secp256k1.PubKey)(secretKey.PubKey().SerializeCompressed())
	addr, _ = sdk.AccAddressFromHex(publicBytes.Address().String())
	return
}

// same as setupForPaymentTest
func setupForPaymentTestDataReliability(t *testing.T) testStructDataReliability {
	ts := testStructDataReliability{}
	ts.servers, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// lava@16k6yth6mqup07ld4gc4w906vzkpcj7rh0he9dc
	// cosmos16k6yth6mqup07ld4gc4w906vzkpcj7rhh0wq24
	servicer1Key, servicer1Address := LoadHexPrivateKey("0xad4bc024dc8f1a96bf88f29416f12ef6e302a0ec02b202ad2120d4a743ce2c64")
	// lava@1462tgug3t50p0fdju54j2f2wrw88zy32tfss2f
	// cosmos1462tgug3t50p0fdju54j2f2wrw88zy32n384dy
	servicer2Key, servicer2Address := LoadHexPrivateKey("0x9754025ca8c4f870f5645cdfbe82caab60fd3833b7ef604d99073ac7ee1e1eb4")
	ts.providers = make([]Account, 0)
	ts.providers = append(ts.providers, Account{secretKey: servicer1Key, addr: servicer1Address, lavaAddr: "lava@16k6yth6mqup07ld4gc4w906vzkpcj7rh0he9dc"})
	ts.providers = append(ts.providers, Account{secretKey: servicer2Key, addr: servicer2Address, lavaAddr: "lava@1462tgug3t50p0fdju54j2f2wrw88zy32tfss2f"})

	// lava@1xsq6pfxg3hzmfjk9n3gd44u6vh3c8kjxhrgah9
	// cosmos1xsq6pfxg3hzmfjk9n3gd44u6vh3c8kjx0mlcsg
	user1Key, user1Address := LoadHexPrivateKey("0x32c7e2b96ef903124817d21b50dd4da79f5d019ad5b6032213a0d05081f7a471")
	user1VRF, _ := hexutil.Decode("0xe8fdcbb90db0b2ca13a330153caf9ee85851e84fc808efce694b0657a4261586a9c12bd4a1b1811aec7e8c1ef63ee92d396eaf6486a66dd8376fda29d5bdd0ff")
	ts.clients = make([]Account, 0)
	ts.clients = append(ts.clients, Account{secretKey: user1Key, addr: user1Address, vrfSk: user1VRF, lavaAddr: "lava@1xsq6pfxg3hzmfjk9n3gd44u6vh3c8kjxhrgah9"})

	var balance int64 = 100000
	for _, provider := range ts.providers {
		ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), provider.addr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
	}
	for _, client := range ts.clients {
		ts.keepers.BankKeeper.SetBalance(sdk.UnwrapSDKContext(ts.ctx), client.addr, sdk.NewCoins(sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(balance))))
	}

	ts.keepers.Epochstorage.SetEpochDetails(sdk.UnwrapSDKContext(ts.ctx), *epochtypes.DefaultGenesis().EpochDetails)

	ts.spec = spectypes.Spec{
		Index:   "FTM250",
		Name:    "FTM250",
		Enabled: true,
		Apis: []spectypes.ServiceApi{
			{
				Name:         "eth_getBlockByNumber",
				ComputeUnits: uint64(16),
				Enabled:      true,
				ApiInterfaces: []spectypes.ApiInterface{
					{Interface: "jsonrpc", Type: "get", ExtraComputeUnits: 0},
				},
				BlockParsing: spectypes.BlockParser{
					ParserArg:  []string{"0"},
					ParserFunc: 2,
				},
				Category: &spectypes.SpecCategory{
					Deterministic: true,
					Local:         false,
					Subscription:  false,
					Stateful:      0,
				},
				Parsing: spectypes.Parsing{
					FunctionTag:      "getBlockByNumber",
					FunctionTemplate: "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBlockByNumber\",\"params\":[\"0x%x\", false],\"id\":1}",
					ResultParsing: spectypes.BlockParser{
						ParserArg:  []string{"0", "hash"},
						ParserFunc: 2,
					},
				},
			},
		},
		ReliabilityThreshold:      2147483647,
		ComparesHashes:            true,
		FinalizationCriteria:      0,
		SavedBlocks:               1,
		AverageBlockTime:          1500,
		AllowedBlockLagForQosSync: 5,
	}

	ts.keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ts.ctx), ts.spec)

	var stake int64 = 1000
	for _, client := range ts.clients {
		vrfPk := &utils.VrfPubKey{}
		vrfPk.Unmarshal(client.vrfSk)
		_, err := ts.servers.PairingServer.StakeClient(ts.ctx, &types.MsgStakeClient{Creator: client.addr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Vrfpk: vrfPk.String()})
		require.Nil(t, err)
	}

	for _, provider := range ts.providers {
		endpoints := []epochtypes.Endpoint{}
		endpoints = append(endpoints, epochtypes.Endpoint{IPPORT: "123", UseType: ts.spec.GetApis()[0].ApiInterfaces[0].Interface, Geolocation: 1})
		_, err := ts.servers.PairingServer.StakeProvider(ts.ctx, &types.MsgStakeProvider{Creator: provider.addr.String(), ChainID: ts.spec.Name, Amount: sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.NewInt(stake)), Geolocation: 1, Endpoints: endpoints})
		require.Nil(t, err)
	}
	for i := 0; i < 20; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}
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
	ts := setupForPaymentTestDataReliability(t)
	var err error
	// relayRequestNormal := &types.RelayRequest{
	// 	Provider:        "lava@1462tgug3t50p0fdju54j2f2wrw88zy32tfss2f",
	// 	ApiUrl:          "",
	// 	Data:            hexutil.MustDecode("0x7b226d6574686f64223a226574685f676574426c6f636b42794e756d626572222c22706172616d73223a5b223078633530343366222c66616c73655d2c226964223a312c226a736f6e727063223a22322e30227d"),
	// 	SessionId:       uint64(605394647632969758),
	// 	ChainID:         "FTM250",
	// 	CuSum:           16,
	// 	BlockHeight:     36,
	// 	RelayNum:        1,
	// 	RequestBlock:    12911679,
	// 	DataReliability: nil,
	// 	Sig:             hexutil.MustDecode("0x1c283cce9bfc950582676b4d9a9ef63da42bee42d75fdf9077e2aba3393a81cf3332f79364a96c94807ae0a0052813f8e912dfb49c87423dcdf53f4770b0a1c07e"),
	// }

	// relayResponseNormal := &types.RelayReply{
	// 	Data:                  hexutil.MustDecode("0x7b226a736f6e727063223a22322e30222c226964223a312c22726573756c74223a7b22646966666963756c7479223a22307830222c2265706f6368223a22307835613734222c22657874726144617461223a223078222c226761734c696d6974223a223078666666666666666666666666222c2267617355736564223a223078313864313665222c2268617368223a22307830303030356137343030303030306665633762663166353663363064386663303930336461323964313362386531353163373638633836333062393939623463222c226c6f6773426c6f6f6d223a2230783038323030303030313830303030303030303030303030303830303030303030303030303030303030343030303030303030303032303030303030303031303030303030303030303030303030323030303038303030303030303030313034303038303030303030383030303030303230303030303030303030323030303030303031303030303030303039303430303030303030303038303030303031323034303030303030303030383430303030303030303030303030343030303430303030303030303030303230303030303830303430343030303030303030383030303030303030303030303038303030303030303030303130303430303031303230303030303030303030303030303032323030303030303030303030303030303030303034313030306330303430303830303031303034303030303030303030303630303030303030303030303030303030303030303030303030303030303030303030303030303030303330303030303030303030303030303030343030343030303031303032303030303030313130303030303030303031303230313031303030303030303230303030303031303030303034343030303030303230303030303930303030343030303030303031303030303030303030303430303032303034303030303030343034303030303030303030303430303030303030303032222c226d696e6572223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030222c226d697848617368223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030222c226e6f6e6365223a22307830303030303030303030303030303030222c226e756d626572223a223078633530343366222c22706172656e7448617368223a22307830303030356137343030303030306635626133663564313737623361663532613237623962336265623131636364626264393533313266376665613232633337222c227265636569707473526f6f74223a22307831636364633436653761316665396233326231616338323963383862396462656637363562643937383333316433363936313633613339616534326534636165222c2273686133556e636c6573223a22307831646363346465386465633735643761616238356235363762366363643431616433313234353162393438613734313366306131343266643430643439333437222c2273697a65223a223078613137222c227374617465526f6f74223a22307835616439396136323638326635366632316531313638633939383635383636626361363735396432393762616635346161306165666165626133656465633862222c2274696d657374616d70223a2230783630666439313864222c2274696d657374616d704e616e6f223a22307831363935313539643433633163303835222c22746f74616c446966666963756c7479223a22307830222c227472616e73616374696f6e73223a5b22307835353634323633663264323834306339663065393337363836633661306434306137393263653936663461346535346237656232396462643136343238636337222c22307861623637333162386464393830653061633462333731666332356464383761613334306262663339316339353662316630313438663138316563363731613338222c22307833376331396464343737316664383163643264306633333739653437646334633131666633316637346237373165663633316231363264623065306235313637222c22307834303835626434653839366362663936303238653535353566303062343334373663633431336336636264376265333837383436336163623035393839663464222c22307830373931356466643131626462396137636432626565303530363333393534643261333331326230333363383836373465373537663034623435396566353966225d2c227472616e73616374696f6e73526f6f74223a22307861393936323736666431346632323334316231623138376362643533363935396161616437653633613733363737663531623430663032363134393231636236222c22756e636c6573223a5b5d7d7d"),
	// 	Sig:                   hexutil.MustDecode("0x1c8e7ebb4160a4d180c61808c9705cd9b235325533859467440062921b91e39bc84d187ff2cf6f565d9ae56e684a9c0c63433d2b77a31a73ff6683375ccf5e0e63"),
	// 	Nonce:                 0,
	// 	LatestBlock:           43210396,
	// 	FinalizedBlocksHashes: hexutil.MustDecode("0x7b223433323130333936223a22307830303032313139343030303030363130646664666637306333656530363536663139666265356334633666356534386532366538303836613435386130623036227d"),
	// 	SigBlocks:             hexutil.MustDecode("0x1b1bd66d9c1591619153e5398f158085bf1014531e3de27c2d9e18b204bbffda2a7b5e0ecc2d767e25d7781dfac2d1b80956b4f9f2005b778fe57984bf1826e85a"),
	// }

	// fails with "ERR_relay_payment_addr: invalid provider address in relay msg provider: lava@16k6yth6mqup07ld4gc4w906vzkpcj7rh0he9dc,creator: cosmos16k6yth6mqup07ld4gc4w906vzkpcj7rhh0wq24,"
	// if provider set to ts.providers[0].addr.String() instead of the lava address fails with "ERR_relay_payment_pairing: invalid pairing on proof of relay client: cosmos15w6udxmjmw4g3ra58dk8nelk3cyfpl9emte643,provider: cosmos16k6yth6mqup07ld4gc4w906vzkpcj7rhh0wq24,error: invalid user for pairing: block 20 is earlier than earliest saved block 200,"}
	// cosmos15w6udxmjmw4g3ra58dk8nelk3cyfpl9emte643 isnt the address of the signer. I think the address recovery results to a different one since the Provider has been modifed
	relayRequestVerifiability := &types.RelayRequest{
		// Provider:     "lava@16k6yth6mqup07ld4gc4w906vzkpcj7rh0he9dc",
		Provider:     ts.providers[0].addr.String(),
		ApiUrl:       "",
		Data:         hexutil.MustDecode("0x7b226d6574686f64223a226574685f676574426c6f636b42794e756d626572222c22706172616d73223a5b223078633530343366222c66616c73655d2c226964223a312c226a736f6e727063223a22322e30227d"),
		SessionId:    uint64(0),
		ChainID:      "FTM250",
		CuSum:        32,
		BlockHeight:  36,
		RelayNum:     2,
		RequestBlock: 12911679,
		DataReliability: &types.VRFData{
			Differentiator: false,
			VrfValue:       hexutil.MustDecode("0xc745a2567eaae09e81098f967a671fd77728500f68593c6a9ccabaa2fb7b2e6a"),
			VrfProof:       hexutil.MustDecode("0x56b5a20a969feac1e2aa4b1be5e2dc85bc5f384a79bcfcfb7795a4eb2b2d610199776fd26b7ec647c28f320de0fd24aaf7fc5493eb31d00861fac7a2ef6d8f0ed8eba70ab791e633376f6e489cda8ff96362d8c13377d50ae7782f84bc161d81"),
			ProviderSig:    hexutil.MustDecode("0x1c8e7ebb4160a4d180c61808c9705cd9b235325533859467440062921b91e39bc84d187ff2cf6f565d9ae56e684a9c0c63433d2b77a31a73ff6683375ccf5e0e63"),
			AllDataHash:    hexutil.MustDecode("0x00f8f52f01030cc2aa608ea29766b412f144749a68bd23b4917a9710c3d14d77"),
			QueryHash:      hexutil.MustDecode("0x4e6ab9b21356d9c232e8db4cd1ad73b5b24bab47769fdc20251f74c179b9ca58"),
			Sig:            hexutil.MustDecode("0x1c125e492ed044df19eea0f32769961a064dd8b6963180bd164c3c3dd8bf38f2a7344576666aca3be8fb224d9ec78e7b3598f018f469b094caf3e3dc8c4dc86a96"),
		},
		Sig: hexutil.MustDecode("0x1c9c5c1fba4b965e712f721f69376ce202a44bb7ee2abd45951c69202f94ad2dc54f8ae857ca7d1fc0f1c7a3c88fcae66951e13b8a3bff1a51267518118a42e7b3"),
	}

	// relayResponseVerifiability := &types.RelayReply{
	// 	Data:                  hexutil.MustDecode("0x7b226a736f6e727063223a22322e30222c226964223a312c22726573756c74223a7b22646966666963756c7479223a22307830222c2265706f6368223a22307835613734222c22657874726144617461223a223078222c226761734c696d6974223a223078666666666666666666666666222c2267617355736564223a223078313864313665222c2268617368223a22307830303030356137343030303030306665633762663166353663363064386663303930336461323964313362386531353163373638633836333062393939623463222c226c6f6773426c6f6f6d223a2230783038323030303030313830303030303030303030303030303830303030303030303030303030303030343030303030303030303032303030303030303031303030303030303030303030303030323030303038303030303030303030313034303038303030303030383030303030303230303030303030303030323030303030303031303030303030303039303430303030303030303038303030303031323034303030303030303030383430303030303030303030303030343030303430303030303030303030303230303030303830303430343030303030303030383030303030303030303030303038303030303030303030303130303430303031303230303030303030303030303030303032323030303030303030303030303030303030303034313030306330303430303830303031303034303030303030303030303630303030303030303030303030303030303030303030303030303030303030303030303030303030303330303030303030303030303030303030343030343030303031303032303030303030313130303030303030303031303230313031303030303030303230303030303031303030303034343030303030303230303030303930303030343030303030303031303030303030303030303430303032303034303030303030343034303030303030303030303430303030303030303032222c226d696e6572223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030222c226d697848617368223a22307830303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030222c226e6f6e6365223a22307830303030303030303030303030303030222c226e756d626572223a223078633530343366222c22706172656e7448617368223a22307830303030356137343030303030306635626133663564313737623361663532613237623962336265623131636364626264393533313266376665613232633337222c227265636569707473526f6f74223a22307831636364633436653761316665396233326231616338323963383862396462656637363562643937383333316433363936313633613339616534326534636165222c2273686133556e636c6573223a22307831646363346465386465633735643761616238356235363762366363643431616433313234353162393438613734313366306131343266643430643439333437222c2273697a65223a223078613137222c227374617465526f6f74223a22307835616439396136323638326635366632316531313638633939383635383636626361363735396432393762616635346161306165666165626133656465633862222c2274696d657374616d70223a2230783630666439313864222c2274696d657374616d704e616e6f223a22307831363935313539643433633163303835222c22746f74616c446966666963756c7479223a22307830222c227472616e73616374696f6e73223a5b22307835353634323633663264323834306339663065393337363836633661306434306137393263653936663461346535346237656232396462643136343238636337222c22307861623637333162386464393830653061633462333731666332356464383761613334306262663339316339353662316630313438663138316563363731613338222c22307833376331396464343737316664383163643264306633333739653437646334633131666633316637346237373165663633316231363264623065306235313637222c22307834303835626434653839366362663936303238653535353566303062343334373663633431336336636264376265333837383436336163623035393839663464222c22307830373931356466643131626462396137636432626565303530363333393534643261333331326230333363383836373465373537663034623435396566353966225d2c227472616e73616374696f6e73526f6f74223a22307861393936323736666431346632323334316231623138376362643533363935396161616437653633613733363737663531623430663032363134393231636236222c22756e636c6573223a5b5d7d7d"),
	// 	Sig:                   hexutil.MustDecode("0x1c80e0b1ade7cefbdc5af07bf4dbb71bf62adb363030968d3fb030941febe2dae8239737a45baec12e1f08ad4304ea0051bf6cf34872004710e87291ad69baff8f"),
	// 	Nonce:                 0,
	// 	LatestBlock:           43210396,
	// 	FinalizedBlocksHashes: hexutil.MustDecode("0x7b223433323130333936223a22307830303032313139343030303030363130646664666637306333656530363536663139666265356334633666356534386532366538303836613435386130623036227d"),
	// 	SigBlocks:             hexutil.MustDecode("0x1bd2ae6a1c019858df57960dcc78c3235d8987a6f567ef42954bcd6bfc057b46af36e250af886ca39b2dd64527132c4b414f991a87c3c29bf59c8cedd6fbefe682"),
	// }

	var Relays []*types.RelayRequest
	Relays = append(Relays, relayRequestVerifiability)

	_, err = ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[0].addr.String(), Relays: Relays})
	require.Nil(t, err)

	// ts.keepers.Epochstorage.StakeEntryByAddress(sdk.UnwrapSDKContext(ts.ctx), epochtypes.ClientKey, ts.spec.Index, ts.clients[0].addr)

	// balanceBefore := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].addr, epochstoragetypes.TokenDenom).Amount.Int64()
	// // _, err := ts.servers.PairingServer.RelayPayment(ts.ctx, &types.MsgRelayPayment{Creator: ts.providers[1].addr.String(), Relays: Relays})
	// // require.NotNil(t, err)

	// mint := ts.keepers.Pairing.MintCoinsPerCU(sdk.UnwrapSDKContext(ts.ctx))
	// fmt.Println(mint)
	// // want := mint.MulInt64(int64(ts.spec.GetApis()[0].ComputeUnits * 10))

	// balanceAfter := ts.keepers.BankKeeper.GetBalance(sdk.UnwrapSDKContext(ts.ctx), ts.providers[1].addr, epochstoragetypes.TokenDenom).Amount.Int64()
	// fmt.Println(balanceBefore)
	// fmt.Println(balanceAfter)
	// vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(relayRequest, relayResponse, *ts.vrfSk)

}

func printRelayRequest(request *pairingtypes.RelayRequest) {
	if request.DataReliability != nil {
		fmt.Printf("{Provider: %s, ApiUrl: %s, DataByte: %s, SessionId: %d, ChainID: %s, CuSum: %d, BlockHeight: %d, RelayNum: %d, RequestBlock: %d, DataReliability: {Differentiator: %t, VrfValue: %s, VrfProof: %s, ProviderSig: %s, AllDataHash: %s, QueryHash: %s, Sig: %s} }\n",
			request.Provider, request.ApiUrl, hex.EncodeToString(request.Data),
			request.SessionId, request.ChainID, request.CuSum, request.BlockHeight, request.RelayNum, request.RequestBlock,
			request.DataReliability.Differentiator, hex.EncodeToString(request.DataReliability.VrfValue), hex.EncodeToString(request.DataReliability.VrfProof), hex.EncodeToString(request.DataReliability.ProviderSig),
			hex.EncodeToString(request.DataReliability.AllDataHash), hex.EncodeToString(request.DataReliability.QueryHash), hex.EncodeToString(request.DataReliability.Sig),
		)
	} else {
		fmt.Printf("{Provider: %s, ApiUrl: %s, DataByte: %s, SessionId: %d, ChainID: %s, CuSum: %d, BlockHeight: %d, RelayNum: %d, RequestBlock: %d, DataReliability: %+v }\n",
			request.Provider, request.ApiUrl, hex.EncodeToString(request.Data),
			request.SessionId, request.ChainID, request.CuSum, request.BlockHeight, request.RelayNum, request.RequestBlock,
			request.DataReliability,
		)
	}
}
func printRelayResponse(reply *pairingtypes.RelayReply) {
	fmt.Printf("{Data: %s, Sig: %s, Nonce: %d, LatestBlock: %d, FinalizedBlocksHashes: %s, SigBlocks: %s}\n", hex.EncodeToString(reply.Data), hex.EncodeToString(reply.Sig), reply.Nonce, reply.LatestBlock, hex.EncodeToString(reply.FinalizedBlocksHashes), hex.EncodeToString(reply.SigBlocks))
}
