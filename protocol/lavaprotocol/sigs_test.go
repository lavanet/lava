package lavaprotocol

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/protocol/lavasession"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/utils/sigs"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

const (
	lavaChainID = "lava"
	chainID     = "LAV1"
	cu          = uint64(10)
	SessionId   = 12345678
	epoch       = 10
	latestBlock = 360
	relayNumber = 1
)

var requestedBlocks = []int64{350, spectypes.LATEST_BLOCK}

type Account struct {
	SK   *btcSecp256k1.PrivateKey
	Addr sdk.AccAddress
}

func TestSigsHappyFlow(t *testing.T) {
	//setup consumer
	var consumer, provider1, provider2 Account
	ctx := context.Background()
	consumer.SK, consumer.Addr = sigs.GenerateFloatingKey()
	provider1.SK, provider1.Addr = sigs.GenerateFloatingKey()
	provider2.SK, provider2.Addr = sigs.GenerateFloatingKey()
	vrf_sk, _, err := utils.GeneratePrivateVRFKey()
	if err != nil {
		utils.LavaFormatFatal("failed getting or creating a VRF key", err)
	}

	fmt.Println("consumer address", consumer.Addr.String())
	log.Println("provider1.Addr.String()", provider1.Addr.String())
	log.Println("provider2.Addr.String()", provider2.Addr.String())

	for _, requestedBlock := range requestedBlocks {
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

		require.True(t, bytes.Equal(sigs.CalculateContentHashForRelayData(relayRequest.RelayData), relayRequest.RelaySession.ContentHash))

		// Consumer Part
		sig, err := sigs.SignRelay(consumer.SK, *relayRequest.RelaySession)
		require.NoError(t, err)
		relayRequest.RelaySession.Sig = sig

		//
		// Provider part:

		// validate consumer's address
		extractedConsumerAddress, err := sigs.ExtractSignerAddress(relayRequest.RelaySession)
		require.NoError(t, err)
		require.Equal(t, consumer.Addr, extractedConsumerAddress)
		require.Equal(t, consumer.Addr.String(), extractedConsumerAddress.String())

		relayRequest.RelayData.RequestBlock = ReplaceRequestedBlock(relayRequest.RelayData.RequestBlock, latestBlock)

		finalizedBlockHashes := map[int]string{
			350: "a", 351: "b", 352: "c",
		}
		jsonStr, err := json.Marshal(finalizedBlockHashes)
		require.NoError(t, err)

		relayReply := &pairingtypes.RelayReply{
			Data:                  []byte{1, 2, 3, 4, 5},
			Nonce:                 125125,
			LatestBlock:           latestBlock,
			FinalizedBlocksHashes: jsonStr,
		}
		reply, err := SignRelayResponse(consumer.Addr, relayRequest, provider1.SK, relayReply, true)
		require.NoError(t, err)
		require.NotNil(t, reply)

		// Back to consumer side after response was signed and returned

		extractedProviderAddress, err := sigs.RecoverPubKeyFromRelayReply(reply, &relayRequest)
		require.NoError(t, err)
		extractedProviderAddressHex, err := sdk.AccAddressFromHex(extractedProviderAddress.Address().String())
		require.NoError(t, err)
		require.Equal(t, provider1.Addr.String(), extractedProviderAddressHex.String())

		vrfRes0, vrfRes1 := utils.CalculateVrfOnRelay(relayRequest.RelayData, reply, vrf_sk, epoch)
		indexesMap := DataReliabilityThresholdToSession([][]byte{vrfRes0, vrfRes1}, []bool{false, true}, 4294967295, 2)
		fmt.Println(indexesMap)

		for _, differentiator := range indexesMap {
			vrf_res, vrf_proof := utils.ProveVrfOnRelay(relayRequest.RelayData, reply, vrf_sk, differentiator, epoch)
			vrf_data := NewVRFData(differentiator, vrf_res, vrf_proof, &relayRequest, reply)
			reliabilityRequest, err := ConstructDataReliabilityRelayRequest(ctx, lavaChainID, vrf_data, consumer.SK, chainID, relayRequest.RelayData, provider2.Addr.String(), epoch, []byte{}, relayNumber)
			require.NoError(t, err)

			// Provider 2 part
			extractedConsumerAddress, err = sigs.ExtractSignerAddress(reliabilityRequest.RelaySession)
			require.NoError(t, err)
			require.Equal(t, consumer.Addr, extractedConsumerAddress)
			require.Equal(t, consumer.Addr.String(), extractedConsumerAddress.String())

			valid := verifyReliabilityAddressSigning(t, consumer.Addr, reliabilityRequest, provider1.Addr.String())
			require.True(t, valid)
		}
	}
}

func verifyReliabilityAddressSigning(t *testing.T, consumer sdk.AccAddress, request *pairingtypes.RelayRequest, provider1PubKey string) (valid bool) {
	queryHash := utils.CalculateQueryHash(*request.RelayData)
	require.True(t, bytes.Equal(queryHash, request.DataReliability.QueryHash))

	valid, err := sigs.ValidateSignerOnVRFData(consumer, *request.DataReliability)
	require.NoError(t, err)
	require.True(t, valid)

	fmt.Println("RecoverProviderPubKeyFromVrfDataAndQuery")
	pubKey, err := sigs.RecoverProviderPubKeyFromVrfDataAndQuery(request)
	require.NoError(t, err)
	providerAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
	fmt.Println("pubKey", providerAddress.String())
	require.NoError(t, err)
	require.Equal(t, provider1PubKey, providerAddress.String())
	return true
}
