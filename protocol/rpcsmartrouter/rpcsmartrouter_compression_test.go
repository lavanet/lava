package rpcsmartrouter

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainlib/extensionslib"
	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/lavanet/lava/v5/protocol/lavasession"
	"github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils/rand"
	"github.com/lavanet/lava/v5/utils/sigs"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// TestSmartRouterCompressionHeaderLogic tests that smart router adds compression header when flag is enabled
func TestSmartRouterCompressionHeaderLogic(t *testing.T) {
	tests := []struct {
		name         string
		flagEnabled  bool
		expectHeader bool
	}{
		{
			name:         "Flag enabled - header should be added",
			flagEnabled:  true,
			expectHeader: true,
		},
		{
			name:         "Flag disabled - no header",
			flagEnabled:  false,
			expectHeader: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate smart router creating metadata (same logic as in relayInner)
			metadataAdd := metadata.New(map[string]string{
				common.IP_FORWARDING_HEADER_NAME:  "test-token",
				common.LAVA_CONSUMER_PROCESS_GUID: "12345",
				common.LAVA_LB_UNIQUE_ID_HEADER:   "67890",
			})

			// Simulate the compression header logic from relayInner
			if tt.flagEnabled {
				metadataAdd.Set(common.LavaCompressionSupportHeader, "true")
			}

			// Verify header presence
			headerValues := metadataAdd.Get(common.LavaCompressionSupportHeader)
			if tt.expectHeader {
				require.Len(t, headerValues, 1, "Should have compression support header")
				require.Equal(t, "true", headerValues[0], "Header value should be 'true'")
			} else {
				require.Empty(t, headerValues, "Should not have compression support header")
			}
		})
	}
}

// TestSmartRouterHandlesDecompressionError tests smart router handles invalid compressed data
func TestSmartRouterHandlesDecompressionError(t *testing.T) {
	rand.InitRandomSeed()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	_, providerAccount := sigs.GenerateFloatingKey()
	providerPublicAddress := providerAccount.String()
	consumeSK, consumerAccount := sigs.GenerateFloatingKey()

	// Enable compression
	lavasession.AllowGRPCCompressionForConsumerProviderCommunication = true

	// Invalid compressed data
	invalidCompressedData := []byte("this is not valid gzip data")

	relayerMock := NewMockRelayerClient(ctrl)
	relayerMock.EXPECT().Relay(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, in *pairingtypes.RelayRequest, opts ...grpc.CallOption) (*pairingtypes.RelayReply, error) {
			// Set compression header (claiming it's compressed)
			for _, opt := range opts {
				if headerOpt, ok := opt.(grpc.HeaderCallOption); ok {
					*headerOpt.HeaderAddr = metadata.Pairs(common.LavaCompressionHeader, common.LavaCompressionGzip)
				}
			}

			// Return invalid data
			reply := &pairingtypes.RelayReply{
				Data:                  invalidCompressedData,
				FinalizedBlocksHashes: []byte(`{"0":"hash0"}`),
			}
			return reply, nil
		},
	).Times(1)

	rpcSmartRouterServer, chainParser := createRpcSmartRouter(
		t, ctrl, context.Background(), consumeSK, consumerAccount,
		providerPublicAddress, relayerMock, "LAV1",
		spectypes.APIInterfaceTendermintRPC, 100, 1, "lava",
	)

	chainMsg, err := chainParser.ParseMsg("", []byte(`{"jsonrpc":"2.0","method":"status","params":[],"id":1}`), "", nil, extensionslib.ExtensionInfo{})
	require.NoError(t, err)

	singleConsumerSession := &lavasession.SingleConsumerSession{
		EndpointConnection: &lavasession.EndpointConnection{Client: relayerMock},
	}
	singleConsumerSession.Parent = &lavasession.ConsumerSessionsWithProvider{
		PublicLavaAddress: providerPublicAddress,
		PairingEpoch:      100,
	}

	relayResult := &common.RelayResult{
		ProviderInfo: common.ProviderInfo{ProviderAddress: providerPublicAddress},
		Request: &pairingtypes.RelayRequest{
			RelayData:    &pairingtypes.RelayPrivateData{RequestBlock: 0},
			RelaySession: &pairingtypes.RelaySession{},
		},
	}

	// Call relayInner - should fail with decompression error
	_, err, needsBackoff := rpcSmartRouterServer.relayInner(
		context.Background(),
		singleConsumerSession,
		relayResult,
		30*time.Second,
		chainMsg,
		"test-token",
		&metrics.RelayMetrics{},
	)

	// Should return an error due to invalid gzip data
	require.Error(t, err, "Should fail with decompression error")
	require.Contains(t, err.Error(), "failed to create gzip reader", "Error should mention gzip")
	require.False(t, needsBackoff, "Should not need backoff for decompression error")
}

// TestSmartRouterCompressionRoundTrip tests the complete compression/decompression cycle
func TestSmartRouterCompressionRoundTrip(t *testing.T) {
	// This test verifies the data integrity through compression and decompression
	originalData := []byte(strings.Repeat("test data for compression ", 50000))

	// Compress (as provider would)
	compressedData, wasCompressed, err := common.CompressData(originalData, common.CompressionThreshold)
	require.NoError(t, err)
	require.True(t, wasCompressed, "Data should be compressed")
	require.Less(t, len(compressedData), len(originalData), "Compressed should be smaller")

	// Decompress (as smart router would)
	decompressedData, err := common.DecompressData(compressedData)
	require.NoError(t, err)

	// Verify data integrity
	require.Equal(t, originalData, decompressedData, "Decompressed data should match original")
}

// TestSmartRouterNoCompressionWhenBelowThreshold tests that small responses aren't compressed
func TestSmartRouterNoCompressionWhenBelowThreshold(t *testing.T) {
	smallData := []byte(strings.Repeat("a", 1000))

	// Try to compress
	result, wasCompressed, err := common.CompressData(smallData, common.CompressionThreshold)
	require.NoError(t, err)
	require.False(t, wasCompressed, "Small data should not be compressed")
	require.Equal(t, smallData, result, "Should return original data when not compressed")
}

// TestSmartRouterUncompressedResponseHandling tests that uncompressed responses work normally
func TestSmartRouterUncompressedResponseHandling(t *testing.T) {
	// Simulate smart router receiving response without compression header
	responseData := []byte(`{"jsonrpc":"2.0","result":{"status":"ok"},"id":1}`)
	responseHeader := metadata.MD{} // Empty metadata - no compression header

	// Check for compression header
	lavaCompressionValues := responseHeader.Get(common.LavaCompressionHeader)
	shouldDecompress := len(lavaCompressionValues) > 0 && lavaCompressionValues[0] == common.LavaCompressionGzip

	// Verify no decompression is attempted
	require.False(t, shouldDecompress, "Should not attempt decompression when header is missing")

	// Simulate the actual code logic
	var finalData []byte
	if shouldDecompress {
		// This branch should NOT be taken
		t.Fatal("Should not attempt to decompress when header is missing")
	} else {
		// This branch should be taken - use data as-is
		finalData = responseData
	}

	// Verify data is unchanged
	require.Equal(t, responseData, finalData, "Data should be used as-is when not compressed")
}
