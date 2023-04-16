package lavaprotocol

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	"github.com/stretchr/testify/require"
)

const (
	lavaChainID    = "lava"
	chainID        = "LAV1"
	cu             = uint64(10)
	SessionId      = 12345678
	epoch          = 10
	requestedBlock = 350
)

type Account struct {
	SK   *btcSecp256k1.PrivateKey
	Addr sdk.AccAddress
}

func TestSigsHappyFlow(t *testing.T) {
	//setup consumer
	var consumer, provider1, provider2 Account
	consumer.SK, consumer.Addr = sigs.GenerateFloatingKey()
	provider1.SK, provider1.Addr = sigs.GenerateFloatingKey()
	provider2.SK, provider2.Addr = sigs.GenerateFloatingKey()
	fmt.Println("consumer address", consumer.Addr.String())
	log.Println("provider1.Addr.String()", provider1.Addr.String())
	log.Println("provider2.Addr.String()", provider2.Addr.String())

	// Setup all the request parameters
	pd := &pairingtypes.RelayPrivateData{
		ConnectionType: "GET",
		ApiUrl:         "/blocks/latest",
		Data:           []byte{0, 1, 0, 1, 1},
		RequestBlock:   requestedBlock,
		ApiInterface:   "rest",
		Salt:           []byte{1, 2, 3, 4, 5, 6, 7, 8},
	}
	qosReport := lavasession.QoSReport{
		LastQoSReport: &pairingtypes.QualityOfServiceReport{
			Latency:      sdk.NewDec(1),
			Availability: sdk.NewDec(1),
			Sync:         sdk.NewDec(1),
		},
	}
	scs := &lavasession.SingleConsumerSession{
		CuSum:         cu,
		SessionId:     SessionId,
		LatestRelayCu: cu,
		RelayNum:      cu,
		QoSInfo:       qosReport,
	}
	relayRequest := pairingtypes.RelayRequest{
		RelayData:       pd,
		RelaySession:    ConstructRelaySession(lavaChainID, pd, chainID, provider1.Addr.String(), scs, epoch, []byte{}),
		DataReliability: nil,
	}

	// Consumer Part
	sig, err := sigs.SignRelay(consumer.SK, *relayRequest.RelaySession)
	require.NoError(t, err)
	relayRequest.RelaySession.Sig = sig

	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relayRequest.RelaySession)
	require.NoError(t, err)
	require.Equal(t, consumer.Addr, extractedConsumerAddress)
	require.Equal(t, consumer.Addr.String(), extractedConsumerAddress.String())

	// Provider part:
	finalizedBlockHashes := map[int]string{
		350: "a", 351: "b", 352: "c",
	}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)

	relayReply := &pairingtypes.RelayReply{
		Data:                  []byte{1, 2, 3, 4, 5},
		Nonce:                 125125,
		LatestBlock:           requestedBlock + 1,
		FinalizedBlocksHashes: jsonStr,
	}
	reply, err := SignRelayResponse(consumer.Addr, relayRequest, provider1.SK, relayReply, true)
	require.NoError(t, err)
	require.NotNil(t, reply)

	extractedProviderAddress, err := sigs.RecoverPubKeyFromRelayReply(reply, &relayRequest)
	require.NoError(t, err)
	extractedProviderAddressHex, err := sdk.AccAddressFromHex(extractedProviderAddress.Address().String())
	require.NoError(t, err)
	require.Equal(t, provider1.Addr.String(), extractedProviderAddressHex.String())

}
