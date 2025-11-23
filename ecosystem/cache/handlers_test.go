package cache

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// TestCompressDecompress tests the basic compress/decompress functionality
func TestCompressDecompress(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Small JSON payload",
			data: []byte(`{"jsonrpc":"2.0","result":{"block":"0x123"},"id":1}`),
		},
		{
			name: "Large JSON payload",
			data: []byte(strings.Repeat(`{"key":"value","data":"test","number":12345},`, 1000)),
		},
		{
			name: "Empty data",
			data: []byte{},
		},
		{
			name: "Single byte",
			data: []byte{0x01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compress - note: common.CompressData doesn't compress small data by default
			// So we pass 0 as threshold to force compression for testing
			compressed, wasCompressed, err := common.CompressData(tt.data, 0)
			require.NoError(t, err)

			// For non-empty data, if it was compressed it should be valid gzip
			if len(tt.data) > 0 && wasCompressed {
				require.NotEmpty(t, compressed)

				// Verify it's valid gzip by checking header
				require.GreaterOrEqual(t, len(compressed), 2)
				require.Equal(t, byte(0x1f), compressed[0]) // gzip magic number
				require.Equal(t, byte(0x8b), compressed[1]) // gzip magic number

				// Decompress
				decompressed, err := common.DecompressData(compressed)
				require.NoError(t, err)

				// Verify roundtrip
				require.Equal(t, tt.data, decompressed)
			}
		})
	}
}

// TestDecompressInvalidData tests decompression error handling
func TestDecompressInvalidData(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "Invalid gzip data",
			data:        []byte("not gzip data"),
			expectError: true,
		},
		{
			name:        "Empty data",
			data:        []byte{},
			expectError: true,
		},
		{
			name:        "Partial gzip header",
			data:        []byte{0x1f},
			expectError: true,
		},
		{
			name:        "Corrupt gzip data",
			data:        []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := common.DecompressData(tt.data)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestFormatCacheValueCompression tests compression logic in formatCacheValue
func TestFormatCacheValueCompression(t *testing.T) {
	// Create large JSON payload (> 1MB threshold)
	largeJSON := []byte(strings.Repeat(`{"key":"value","data":"test","number":12345},`, 50000)) // ~2.5MB

	tests := []struct {
		name               string
		responseData       []byte
		finalized          bool
		expectCompressed   bool
		expectIsCompressed bool
	}{
		{
			name:               "Large payload - finalized",
			responseData:       largeJSON,
			finalized:          true,
			expectCompressed:   true,
			expectIsCompressed: true,
		},
		{
			name:               "Large payload - non-finalized",
			responseData:       largeJSON,
			finalized:          false,
			expectCompressed:   true,
			expectIsCompressed: true,
		},
		{
			name:               "Small payload below threshold - finalized",
			responseData:       []byte(`{"result":"small"}`),
			finalized:          true,
			expectCompressed:   false,
			expectIsCompressed: false,
		},
		{
			name:               "Small payload below threshold - non-finalized",
			responseData:       []byte(`{"result":"small"}`),
			finalized:          false,
			expectCompressed:   false,
			expectIsCompressed: false,
		},
		{
			name:               "Exactly at threshold",
			responseData:       bytes.Repeat([]byte("a"), common.CompressionThreshold),
			finalized:          true,
			expectCompressed:   false, // Not > threshold
			expectIsCompressed: false,
		},
		{
			name:               "One byte over threshold",
			responseData:       bytes.Repeat([]byte("a"), common.CompressionThreshold+1),
			finalized:          true,
			expectCompressed:   true,
			expectIsCompressed: true,
		},
		{
			name:               "Empty data",
			responseData:       []byte{},
			finalized:          true,
			expectCompressed:   false,
			expectIsCompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalSize := len(tt.responseData)
			response := &pairingtypes.RelayReply{
				Data: tt.responseData,
			}

			cacheValue := formatCacheValue(response, []byte("hash"), tt.finalized, nil, 1230)

			// Check IsCompressed flag
			require.Equal(t, tt.expectIsCompressed, cacheValue.IsCompressed, "IsCompressed flag mismatch")

			if tt.expectCompressed {
				// Data should be compressed (smaller and starts with gzip header)
				require.Less(t, len(cacheValue.Response.Data), originalSize, "Compressed data should be smaller")
				if len(cacheValue.Response.Data) >= 2 {
					require.Equal(t, byte(0x1f), cacheValue.Response.Data[0], "Should have gzip header")
					require.Equal(t, byte(0x8b), cacheValue.Response.Data[1], "Should have gzip header")
				}
			} else {
				// Data should be unchanged
				require.Equal(t, originalSize, len(cacheValue.Response.Data), "Uncompressed data size should match")
				if originalSize > 0 {
					require.Equal(t, tt.responseData, cacheValue.Response.Data, "Data should be unchanged")
				}
			}

			// Verify finalized vs non-finalized behavior
			if tt.finalized {
				require.Nil(t, cacheValue.Hash, "Finalized entries should have nil hash")
			} else {
				require.NotNil(t, cacheValue.Hash, "Non-finalized entries should have hash")
			}
		})
	}
}

// TestCacheValueToCacheReplyDecompression tests automatic decompression on retrieval
func TestCacheValueToCacheReplyDecompression(t *testing.T) {
	originalData := []byte(strings.Repeat(`{"key":"value","data":"test"},`, 10000)) // Large JSON

	tests := []struct {
		name               string
		isCompressed       bool
		data               []byte
		expectDecompressed bool
	}{
		{
			name:               "Compressed data should be decompressed",
			isCompressed:       true,
			data:               mustCompress(t, originalData),
			expectDecompressed: true,
		},
		{
			name:               "Uncompressed data should remain unchanged",
			isCompressed:       false,
			data:               originalData,
			expectDecompressed: false,
		},
		{
			name:               "Empty compressed data",
			isCompressed:       true,
			data:               []byte{},
			expectDecompressed: false,
		},
		{
			name:               "Empty uncompressed data",
			isCompressed:       false,
			data:               []byte{},
			expectDecompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheValue := CacheValue{
				Response: pairingtypes.RelayReply{
					Data: tt.data,
				},
				IsCompressed: tt.isCompressed,
			}

			cacheReply := cacheValue.ToCacheReply()

			if tt.expectDecompressed && len(tt.data) > 0 {
				// Should be decompressed back to original
				require.Equal(t, originalData, cacheReply.Reply.Data, "Data should be decompressed")
			} else if !tt.isCompressed && len(tt.data) > 0 {
				// Should remain unchanged
				require.Equal(t, tt.data, cacheReply.Reply.Data, "Uncompressed data should be unchanged")
			}
		})
	}
}

// TestCacheValueToCacheReplyDecompressionError tests error handling during decompression
func TestCacheValueToCacheReplyDecompressionError(t *testing.T) {
	// Create a cache value with invalid compressed data
	cacheValue := CacheValue{
		Response: pairingtypes.RelayReply{
			Data: []byte("invalid gzip data"),
		},
		IsCompressed: true, // Flag says it's compressed but it's not
	}

	// Should not panic, should return original data on error
	cacheReply := cacheValue.ToCacheReply()
	require.NotNil(t, cacheReply)
	// On decompression error, original data is returned
	require.Equal(t, []byte("invalid gzip data"), cacheReply.Reply.Data)
}

// TestCacheValueCost tests that Cost() returns compressed size
func TestCacheValueCost(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		isCompressed bool
	}{
		{
			name:         "Small uncompressed data",
			data:         []byte("small data"),
			isCompressed: false,
		},
		{
			name:         "Large compressed data",
			data:         mustCompress(t, bytes.Repeat([]byte("a"), 100000)),
			isCompressed: true,
		},
		{
			name:         "Empty data",
			data:         []byte{},
			isCompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheValue := CacheValue{
				Response: pairingtypes.RelayReply{
					Data: tt.data,
				},
				IsCompressed: tt.isCompressed,
			}

			cost := cacheValue.Cost()
			// Cost should be the actual stored size (compressed if applicable)
			require.Equal(t, int64(len(tt.data)), cost)
		})
	}
}

// TestCompressionRatio verifies that JSON compresses well
func TestCompressionRatio(t *testing.T) {
	// Create realistic JSON payload similar to blockchain responses
	jsonPayload := []byte(strings.Repeat(`{
		"jsonrpc": "2.0",
		"result": {
			"blockNumber": "0x12345",
			"blockHash": "0xabcdef1234567890",
			"transactions": ["0xtx1", "0xtx2", "0xtx3"],
			"timestamp": "0x60a0b0c0",
			"gasUsed": "0x5208",
			"gasLimit": "0x1000000"
		},
		"id": 1
	},`, 1000))

	originalSize := len(jsonPayload)
	compressed, wasCompressed, err := common.CompressData(jsonPayload, 0) // 0 threshold to force compression
	require.NoError(t, err)
	require.True(t, wasCompressed, "Data should have been compressed")

	compressedSize := len(compressed)
	ratio := float64(originalSize-compressedSize) / float64(originalSize) * 100

	t.Logf("Original size: %d bytes", originalSize)
	t.Logf("Compressed size: %d bytes", compressedSize)
	t.Logf("Compression ratio: %.2f%%", ratio)

	// JSON should compress at least 50% (typically 70-90%)
	require.Greater(t, ratio, 50.0, "JSON should compress by at least 50%")
	require.Less(t, compressedSize, originalSize, "Compressed should be smaller than original")
}

// Helper function to compress data for tests
func mustCompress(t *testing.T, data []byte) []byte {
	compressed, _, err := common.CompressData(data, 0) // 0 threshold to force compression
	require.NoError(t, err)
	return compressed
}
