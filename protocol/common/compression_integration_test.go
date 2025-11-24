package common

import (
	"bytes"
	"compress/gzip"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// TestCompressionHeadersConsumerToProvider tests that consumer correctly sets compression support header
func TestCompressionHeadersConsumerToProvider(t *testing.T) {
	testCases := []struct {
		name                 string
		compressionEnabled   bool
		expectedHeaderValue  string
		expectedHeaderExists bool
	}{
		{
			name:                 "Compression enabled",
			compressionEnabled:   true,
			expectedHeaderValue:  "true",
			expectedHeaderExists: true,
		},
		{
			name:                 "Compression disabled",
			compressionEnabled:   false,
			expectedHeaderValue:  "",
			expectedHeaderExists: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate consumer creating metadata
			metadataAdd := metadata.New(map[string]string{
				"X-Forwarded-For": "consumer-token",
			})

			if tc.compressionEnabled {
				metadataAdd.Set(LavaCompressionSupportHeader, "true")
			}

			// Verify the header is set correctly
			values := metadataAdd.Get(LavaCompressionSupportHeader)
			if tc.expectedHeaderExists {
				require.Len(t, values, 1, "Should have compression support header")
				require.Equal(t, tc.expectedHeaderValue, values[0], "Header value should match")
			} else {
				require.Len(t, values, 0, "Should not have compression support header")
			}
		})
	}
}

// TestCompressionHeadersProviderToConsumer tests that provider correctly sets compression header on response
func TestCompressionHeadersProviderToConsumer(t *testing.T) {
	testCases := []struct {
		name              string
		dataSize          int
		consumerSupports  bool
		expectCompression bool
		compressionType   string
	}{
		{
			name:              "Large data, consumer supports, should compress",
			dataSize:          CompressionThreshold + 1000,
			consumerSupports:  true,
			expectCompression: true,
			compressionType:   LavaCompressionGzip,
		},
		{
			name:              "Large data, consumer doesn't support, no compression",
			dataSize:          CompressionThreshold + 1000,
			consumerSupports:  false,
			expectCompression: false,
		},
		{
			name:              "Small data, consumer supports, no compression",
			dataSize:          CompressionThreshold - 1000,
			consumerSupports:  true,
			expectCompression: false,
		},
		{
			name:              "Small data, consumer doesn't support, no compression",
			dataSize:          CompressionThreshold - 1000,
			consumerSupports:  false,
			expectCompression: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test data (highly compressible)
			data := []byte(strings.Repeat("a", tc.dataSize))

			// Simulate provider compression logic
			var finalData []byte
			var compressionHeader string

			if tc.consumerSupports {
				compressed, wasCompressed, err := CompressData(data, CompressionThreshold)
				require.NoError(t, err)

				if wasCompressed {
					finalData = compressed
					compressionHeader = LavaCompressionGzip
				} else {
					finalData = data
				}
			} else {
				finalData = data
			}

			// Verify compression outcome
			if tc.expectCompression {
				require.Less(t, len(finalData), len(data), "Data should be compressed")
				require.Equal(t, tc.compressionType, compressionHeader, "Compression header should be set")
			} else {
				require.Equal(t, len(data), len(finalData), "Data should not be compressed")
				require.Empty(t, compressionHeader, "Compression header should not be set")
			}
		})
	}
}

// TestConsumerDecompressionFlow tests the complete consumer decompression flow
func TestConsumerDecompressionFlow(t *testing.T) {
	testCases := []struct {
		name                string
		originalData        []byte
		compressData        bool
		setHeader           bool
		expectDecompression bool
		expectError         bool
	}{
		{
			name:                "Compressed data with header, should decompress",
			originalData:        []byte(strings.Repeat("test data ", 200000)),
			compressData:        true,
			setHeader:           true,
			expectDecompression: true,
			expectError:         false,
		},
		{
			name:                "Uncompressed data, no header, should not decompress",
			originalData:        []byte("small data"),
			compressData:        false,
			setHeader:           false,
			expectDecompression: false,
			expectError:         false,
		},
		{
			name:                "Compressed data but no header, should not attempt decompression",
			originalData:        []byte(strings.Repeat("test data ", 200000)),
			compressData:        true,
			setHeader:           false,
			expectDecompression: false,
			expectError:         false,
		},
		{
			name:                "Header set but data not compressed, should error",
			originalData:        []byte("not actually compressed"),
			compressData:        false,
			setHeader:           true,
			expectDecompression: true,
			expectError:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Prepare response data
			var responseData []byte
			var err error

			if tc.compressData {
				responseData, _, err = CompressData(tc.originalData, CompressionThreshold)
				require.NoError(t, err)
			} else {
				responseData = tc.originalData
			}

			// Simulate provider response headers
			responseHeader := metadata.MD{}
			if tc.setHeader {
				responseHeader.Set(LavaCompressionHeader, LavaCompressionGzip)
			}

			// Simulate consumer decompression logic
			appLevelCompressed := false
			if lavaCompressionValues := responseHeader.Get(LavaCompressionHeader); len(lavaCompressionValues) > 0 {
				appLevelCompressed = lavaCompressionValues[0] == LavaCompressionGzip
			}

			var finalData []byte
			if appLevelCompressed {
				finalData, err = DecompressData(responseData)
				if tc.expectError {
					require.Error(t, err, "Should error on invalid compressed data")
					return
				}
				require.NoError(t, err, "Decompression should not error")
			} else {
				finalData = responseData
			}

			// Verify result
			if tc.expectDecompression && !tc.expectError {
				require.Equal(t, tc.originalData, finalData, "Decompressed data should match original")
			} else if !tc.expectDecompression {
				require.Equal(t, responseData, finalData, "Data should remain unchanged")
			}
		})
	}
}

// TestProviderCompressionDecisionFlow tests the provider's decision logic for compression
func TestProviderCompressionDecisionFlow(t *testing.T) {
	testCases := []struct {
		name                        string
		consumerSupportsCompression bool
		dataSize                    int
		expectCompression           bool
	}{
		{
			name:                        "Consumer supports, large data, should compress",
			consumerSupportsCompression: true,
			dataSize:                    CompressionThreshold + 10000,
			expectCompression:           true,
		},
		{
			name:                        "Consumer doesn't support, large data, should not compress",
			consumerSupportsCompression: false,
			dataSize:                    CompressionThreshold + 10000,
			expectCompression:           false,
		},
		{
			name:                        "Consumer supports, small data, should not compress",
			consumerSupportsCompression: true,
			dataSize:                    CompressionThreshold - 1000,
			expectCompression:           false,
		},
		{
			name:                        "Consumer doesn't support, small data, should not compress",
			consumerSupportsCompression: false,
			dataSize:                    CompressionThreshold - 1000,
			expectCompression:           false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create compressible test data
			data := []byte(strings.Repeat("test", tc.dataSize/4))

			// Simulate incoming context with consumer compression support header
			ctx := context.Background()
			md := metadata.MD{}
			if tc.consumerSupportsCompression {
				md.Set(LavaCompressionSupportHeader, "true")
			}
			ctx = metadata.NewIncomingContext(ctx, md)

			// Simulate provider logic
			incomingMd, _ := metadata.FromIncomingContext(ctx)
			compressionSupport := incomingMd.Get(LavaCompressionSupportHeader)
			consumerSupportsCompression := len(compressionSupport) > 0 && compressionSupport[0] == "true"

			var finalData []byte
			var wasCompressed bool
			var err error

			if consumerSupportsCompression {
				finalData, wasCompressed, err = CompressData(data, CompressionThreshold)
				require.NoError(t, err)
			} else {
				finalData = data
				wasCompressed = false
			}

			// Verify compression decision
			if tc.expectCompression {
				require.True(t, wasCompressed, "Data should have been compressed")
				require.Less(t, len(finalData), len(data), "Compressed data should be smaller")
			} else {
				require.False(t, wasCompressed, "Data should not have been compressed")
				require.Equal(t, len(data), len(finalData), "Data size should be unchanged")
			}
		})
	}
}

// TestEndToEndCompressionFlow tests the complete flow from consumer to provider and back
func TestEndToEndCompressionFlow(t *testing.T) {
	// Original data to send (large and compressible)
	originalData := []byte(strings.Repeat("This is test RPC response data that should compress well. ", 30000))
	require.Greater(t, len(originalData), CompressionThreshold, "Test data should be larger than threshold")

	t.Run("Full compression flow with support", func(t *testing.T) {
		// Step 1: Consumer prepares request with compression support header
		consumerMd := metadata.New(map[string]string{
			"X-Forwarded-For":            "token",
			LavaCompressionSupportHeader: "true",
		})
		_ = metadata.NewOutgoingContext(context.Background(), consumerMd)

		// Step 2: Provider receives request and checks compression support
		providerCtx := metadata.NewIncomingContext(context.Background(), consumerMd)
		incomingMd, _ := metadata.FromIncomingContext(providerCtx)
		compressionSupport := incomingMd.Get(LavaCompressionSupportHeader)
		consumerSupportsCompression := len(compressionSupport) > 0 && compressionSupport[0] == "true"
		require.True(t, consumerSupportsCompression, "Provider should detect consumer compression support")

		// Step 3: Provider compresses response
		compressedData, wasCompressed, err := CompressData(originalData, CompressionThreshold)
		require.NoError(t, err)
		require.True(t, wasCompressed, "Data should be compressed")
		require.Less(t, len(compressedData), len(originalData), "Compressed data should be smaller")

		compressionRatio := float64(len(compressedData)) / float64(len(originalData))
		t.Logf("Compression ratio: %.2f%% (original: %d bytes, compressed: %d bytes)",
			compressionRatio*100, len(originalData), len(compressedData))

		// Step 4: Provider sets compression header
		providerResponseMd := metadata.MD{}
		providerResponseMd.Set(LavaCompressionHeader, LavaCompressionGzip)

		// Step 5: Consumer receives response and checks for compression header
		appLevelCompressed := false
		if lavaCompressionValues := providerResponseMd.Get(LavaCompressionHeader); len(lavaCompressionValues) > 0 {
			appLevelCompressed = lavaCompressionValues[0] == LavaCompressionGzip
		}
		require.True(t, appLevelCompressed, "Consumer should detect compressed response")

		// Step 6: Consumer decompresses response
		decompressedData, err := DecompressData(compressedData)
		require.NoError(t, err)
		require.Equal(t, originalData, decompressedData, "Decompressed data should match original")
	})

	t.Run("No compression when consumer doesn't support", func(t *testing.T) {
		// Step 1: Consumer prepares request WITHOUT compression support header
		consumerMd := metadata.New(map[string]string{
			"X-Forwarded-For": "token",
			// Note: No LavaCompressionSupportHeader
		})
		_ = metadata.NewOutgoingContext(context.Background(), consumerMd)

		// Step 2: Provider receives request and checks compression support
		providerCtx := metadata.NewIncomingContext(context.Background(), consumerMd)
		incomingMd, _ := metadata.FromIncomingContext(providerCtx)
		compressionSupport := incomingMd.Get(LavaCompressionSupportHeader)
		consumerSupportsCompression := len(compressionSupport) > 0 && compressionSupport[0] == "true"
		require.False(t, consumerSupportsCompression, "Provider should detect consumer doesn't support compression")

		// Step 3: Provider does NOT compress response
		responseData := originalData

		// Step 4: Provider does NOT set compression header
		providerResponseMd := metadata.MD{}
		// No compression header

		// Step 5: Consumer receives response and checks for compression header
		appLevelCompressed := false
		if lavaCompressionValues := providerResponseMd.Get(LavaCompressionHeader); len(lavaCompressionValues) > 0 {
			appLevelCompressed = lavaCompressionValues[0] == LavaCompressionGzip
		}
		require.False(t, appLevelCompressed, "Consumer should detect uncompressed response")

		// Step 6: Consumer uses data as-is
		require.Equal(t, originalData, responseData, "Data should be unchanged")
	})
}

// TestCompressionWithRealGzipData tests with actual gzip compressed data
func TestCompressionWithRealGzipData(t *testing.T) {
	originalData := []byte(strings.Repeat("Real RPC response data ", 100000))

	// Manually create gzip compressed data (simulating provider)
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write(originalData)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)
	manuallyCompressed := buf.Bytes()

	// Decompress using our function (simulating consumer)
	decompressed, err := DecompressData(manuallyCompressed)
	require.NoError(t, err)
	require.Equal(t, originalData, decompressed, "Should correctly decompress real gzip data")
}

// TestCompressionWithIncompressibleData tests behavior with data that doesn't compress well
func TestCompressionWithIncompressibleData(t *testing.T) {
	// Create incompressible data (random-like)
	incompressibleData := make([]byte, CompressionThreshold+10000)
	for i := range incompressibleData {
		incompressibleData[i] = byte(i % 256)
	}

	// Attempt compression
	result, wasCompressed, err := CompressData(incompressibleData, CompressionThreshold)
	require.NoError(t, err)

	// Should return original data if compression doesn't help
	if !wasCompressed {
		require.Equal(t, incompressibleData, result, "Should return original if compression doesn't help")
		t.Log("Compression correctly skipped for incompressible data")
	} else {
		// If it was compressed, verify it's smaller
		require.Less(t, len(result), len(incompressibleData), "If compressed, should be smaller")
		t.Logf("Incompressible data was compressed: %d -> %d bytes", len(incompressibleData), len(result))
	}
}
