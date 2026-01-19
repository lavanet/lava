package lavaprotocol

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/qos"
	"github.com/lavanet/lava/v5/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

func unresponsiveProviderStub() []*pairingtypes.ReportedProvider {
	return []*pairingtypes.ReportedProvider{{Address: "stub"}}
}

func TestSignAndExtractResponse(t *testing.T) {
	ctx := context.Background()
	// consumer
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	// provider
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	specId := "LAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:              20,
		LatestRelayCu:      10, // set by GetSessions cuNeededForSession
		QoSManager:         qos.NewQoSManager(),
		SessionId:          123,
		Parent:             nil,
		RelayNum:           1,
		LatestBlock:        epoch,
		EndpointConnection: nil,
		BlockListed:        false, // if session lost sync we blacklist it.
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "x-cosmos-block-height",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, 55, "tendermintrpc", metadataValue, "test", nil)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)

	// check signature
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.NoError(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.HashMsg(relay.RelayData.GetContentHashData())))
	finalizedBlockHashes := map[int64]interface{}{123: "AAA"}
	reply := &pairingtypes.RelayReply{}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = 123
	reply, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply)
	require.NoError(t, err)
	err = VerifyRelayReply(ctx, reply, relay, provider_address.String())
	require.NoError(t, err)
}

func TestSignAndExtractResponseLatest(t *testing.T) {
	ctx := context.Background()
	// consumer
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	// provider
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	testSpecId := "BLAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:              20,
		LatestRelayCu:      10, // set by GetSessions cuNeededForSession
		QoSManager:         qos.NewQoSManager(),
		SessionId:          123,
		Parent:             nil,
		RelayNum:           1,
		LatestBlock:        epoch,
		EndpointConnection: nil,
		BlockListed:        false, // if session lost sync we blacklist it.
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "banana",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, spectypes.LATEST_BLOCK, "tendermintrpc", metadataValue, "test", nil)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := ConstructRelayRequest(ctx, consumer_sk, "lava", testSpecId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)

	// provider checks
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.NoError(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)
	require.True(t, bytes.Equal(relay.RelaySession.ContentHash, sigs.HashMsg(relay.RelayData.GetContentHashData())))
	latestBlock := int64(123)
	// provider handling the response
	finalizedBlockHashes := map[int64]interface{}{latestBlock: "AAA"}
	reply := &pairingtypes.RelayReply{}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = latestBlock
	reply, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply)
	require.NoError(t, err)
	err = VerifyRelayReply(ctx, reply, relay, provider_address.String())
	require.NoError(t, err)
}

func TestSignRelayResponseWithSkipSigning(t *testing.T) {
	ctx := context.Background()
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	specId := "LAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:              20,
		LatestRelayCu:      10,
		QoSManager:         qos.NewQoSManager(),
		SessionId:          123,
		Parent:             nil,
		RelayNum:           1,
		LatestBlock:        epoch,
		EndpointConnection: nil,
		BlockListed:        false,
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "x-cosmos-block-height",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, 55, "tendermintrpc", metadataValue, "test", nil)
	relay, err := ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)

	finalizedBlockHashes := map[int64]interface{}{123: "AAA"}
	reply := &pairingtypes.RelayReply{}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = 123

	// Test with SkipRelaySigning enabled
	originalSkipRelaySigning := SkipRelaySigning
	SkipRelaySigning = true
	defer func() { SkipRelaySigning = originalSkipRelaySigning }()

	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.NoError(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)

	reply, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply)
	require.NoError(t, err)
	require.NotNil(t, reply)
	// When signing is skipped, signature should be nil
	require.Nil(t, reply.Sig, "Signature should be nil when SkipRelaySigning is enabled")

	// Test with SkipRelaySigning disabled (normal behavior)
	SkipRelaySigning = false
	reply2 := &pairingtypes.RelayReply{}
	reply2.FinalizedBlocksHashes = jsonStr
	reply2.LatestBlock = 123

	reply2, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply2)
	require.NoError(t, err)
	require.NotNil(t, reply2)
	// When signing is enabled, signature should be present
	require.NotNil(t, reply2.Sig, "Signature should be present when SkipRelaySigning is disabled")
}

func TestVerifyRelayReplyWithSkipSigning(t *testing.T) {
	ctx := context.Background()
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	specId := "LAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:              20,
		LatestRelayCu:      10,
		QoSManager:         qos.NewQoSManager(),
		SessionId:          123,
		Parent:             nil,
		RelayNum:           1,
		LatestBlock:        epoch,
		EndpointConnection: nil,
		BlockListed:        false,
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "x-cosmos-block-height",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, 55, "tendermintrpc", metadataValue, "test", nil)
	relay, err := ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)

	finalizedBlockHashes := map[int64]interface{}{123: "AAA"}
	reply := &pairingtypes.RelayReply{}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = 123

	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.NoError(t, err)
	require.Equal(t, extractedConsumerAddress, consumer_address)

	// Test with SkipRelaySigning enabled
	originalSkipRelaySigning := SkipRelaySigning
	SkipRelaySigning = true
	defer func() { SkipRelaySigning = originalSkipRelaySigning }()

	// Sign response with skip signing enabled (no signature)
	reply, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply)
	require.NoError(t, err)
	require.Nil(t, reply.Sig, "Signature should be nil when SkipRelaySigning is enabled")

	// Verification should pass when SkipRelaySigning is enabled (skips verification)
	err = VerifyRelayReply(ctx, reply, relay, provider_address.String())
	require.NoError(t, err, "Verification should pass when SkipRelaySigning is enabled")

	// Test with SkipRelaySigning disabled (normal behavior)
	SkipRelaySigning = false
	reply2 := &pairingtypes.RelayReply{}
	reply2.FinalizedBlocksHashes = jsonStr
	reply2.LatestBlock = 123

	// Sign response normally
	reply2, err = SignRelayResponse(extractedConsumerAddress, *relay, provider_sk, reply2)
	require.NoError(t, err)
	require.NotNil(t, reply2.Sig, "Signature should be present when SkipRelaySigning is disabled")

	// Verification should pass with valid signature
	err = VerifyRelayReply(ctx, reply2, relay, provider_address.String())
	require.NoError(t, err, "Verification should pass with valid signature")

	// Test verification failure with wrong address
	_, wrong_address := sigs.GenerateFloatingKey()
	err = VerifyRelayReply(ctx, reply2, relay, wrong_address.String())
	require.Error(t, err, "Verification should fail with wrong address")
}

func TestSignAndVerifyWithSkipSigningEndToEnd(t *testing.T) {
	ctx := context.Background()
	consumer_sk, consumer_address := sigs.GenerateFloatingKey()
	provider_sk, provider_address := sigs.GenerateFloatingKey()
	specId := "LAV1"
	epoch := int64(100)
	singleConsumerSession := &lavasession.SingleConsumerSession{
		CuSum:              20,
		LatestRelayCu:      10,
		QoSManager:         qos.NewQoSManager(),
		SessionId:          123,
		Parent:             nil,
		RelayNum:           1,
		LatestBlock:        epoch,
		EndpointConnection: nil,
		BlockListed:        false,
	}
	metadataValue := make([]pairingtypes.Metadata, 1)
	metadataValue[0] = pairingtypes.Metadata{
		Name:  "x-cosmos-block-height",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, 55, "tendermintrpc", metadataValue, "test", nil)

	originalSkipRelaySigning := SkipRelaySigning
	defer func() { SkipRelaySigning = originalSkipRelaySigning }()

	// Test end-to-end with SkipRelaySigning enabled
	SkipRelaySigning = true

	// Construct request (should skip signing)
	relay, err := ConstructRelayRequest(ctx, consumer_sk, "lava", specId, relayRequestData, provider_address.String(), singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)
	require.Nil(t, relay.RelaySession.Sig, "Request signature should be nil")

	// Sign response (should skip signing)
	finalizedBlockHashes := map[int64]interface{}{123: "AAA"}
	reply := &pairingtypes.RelayReply{}
	jsonStr, err := json.Marshal(finalizedBlockHashes)
	require.NoError(t, err)
	reply.FinalizedBlocksHashes = jsonStr
	reply.LatestBlock = 123

	// We can't extract address from unsigned request, so use the original address
	reply, err = SignRelayResponse(consumer_address, *relay, provider_sk, reply)
	require.NoError(t, err)
	require.Nil(t, reply.Sig, "Response signature should be nil")

	// Verify reply (should skip verification)
	err = VerifyRelayReply(ctx, reply, relay, provider_address.String())
	require.NoError(t, err, "Verification should pass when SkipRelaySigning is enabled")
}
