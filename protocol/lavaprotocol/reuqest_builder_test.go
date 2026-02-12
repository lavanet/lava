package lavaprotocol

import (
	"context"
	"testing"

	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/qos"
	"github.com/lavanet/lava/v5/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestSignAndExtract(t *testing.T) {
	ctx := context.Background()
	sk, address := sigs.GenerateFloatingKey()
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
		Name:  "x-cosmos-block-height:",
		Value: "55",
	}
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, 10, "tendermintrpc", metadataValue, "test", nil)
	require.Equal(t, relayRequestData.Metadata, metadataValue)
	relay, err := ConstructRelayRequest(ctx, sk, "lava", specId, relayRequestData, "lava@stubProviderAddress", singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)

	// check signature
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.NoError(t, err)
	require.Equal(t, extractedConsumerAddress, address)
}

func TestConstructRelayRequestWithSkipSigning(t *testing.T) {
	ctx := context.Background()
	sk, address := sigs.GenerateFloatingKey()
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
	relayRequestData := NewRelayData(ctx, "GET", "stub_url", []byte("stub_data"), 0, 10, "tendermintrpc", metadataValue, "test", nil)

	// Test with SkipRelaySigning enabled
	originalSkipRelaySigning := SkipRelaySigning
	SkipRelaySigning = true
	defer func() { SkipRelaySigning = originalSkipRelaySigning }()

	relay, err := ConstructRelayRequest(ctx, sk, "lava", specId, relayRequestData, "lava@stubProviderAddress", singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)
	require.NotNil(t, relay)
	require.NotNil(t, relay.RelaySession)
	// When signing is skipped, signature should be nil
	require.Nil(t, relay.RelaySession.Sig, "Signature should be nil when SkipRelaySigning is enabled")
	// Verify that we cannot extract address from unsigned session
	_, err = sigs.ExtractSignerAddress(relay.RelaySession)
	require.Error(t, err, "Should not be able to extract signer address from unsigned session")

	// Test with SkipRelaySigning disabled (normal behavior)
	SkipRelaySigning = false
	relay, err = ConstructRelayRequest(ctx, sk, "lava", specId, relayRequestData, "lava@stubProviderAddress", singleConsumerSession, epoch, unresponsiveProviderStub())
	require.NoError(t, err)
	require.NotNil(t, relay)
	require.NotNil(t, relay.RelaySession)
	// When signing is enabled, signature should be present
	require.NotNil(t, relay.RelaySession.Sig, "Signature should be present when SkipRelaySigning is disabled")
	// Verify that we can extract address from signed session
	extractedConsumerAddress, err := sigs.ExtractSignerAddress(relay.RelaySession)
	require.NoError(t, err)
	require.Equal(t, extractedConsumerAddress, address)
}
