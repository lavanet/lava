package common

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompressData_BelowThreshold(t *testing.T) {
	// Test data smaller than threshold (10KB)
	smallData := []byte("Hello, World!")

	result, wasCompressed, err := CompressData(smallData, CompressionThreshold)

	require.NoError(t, err, "CompressData should not error on small data")
	require.False(t, wasCompressed, "Small data should not be compressed")
	require.Equal(t, smallData, result, "Small data should be returned unchanged")
}

func TestCompressData_AboveThreshold(t *testing.T) {
	// Test data larger than threshold (1 MB = 1024*1024 bytes)
	// Create compressible data (repeated pattern)
	largeData := []byte(strings.Repeat("This is a test string that will compress well. ", 25000))
	require.Greater(t, len(largeData), CompressionThreshold, "Test data should be larger than threshold")

	result, wasCompressed, err := CompressData(largeData, CompressionThreshold)

	require.NoError(t, err, "CompressData should not error on large compressible data")
	require.True(t, wasCompressed, "Large compressible data should be compressed")
	require.Less(t, len(result), len(largeData), "Compressed data should be smaller than original")

	// Verify the compressed data is valid gzip
	reader, err := gzip.NewReader(bytes.NewReader(result))
	require.NoError(t, err, "Compressed data should be valid gzip")
	defer reader.Close()
}

func TestCompressData_IncompressibleData(t *testing.T) {
	// Test data that doesn't compress well (random-like data)
	// Using alternating bytes which gzip won't compress efficiently
	largeIncompressibleData := make([]byte, CompressionThreshold+1000)
	for i := range largeIncompressibleData {
		largeIncompressibleData[i] = byte(i % 256)
	}

	result, wasCompressed, err := CompressData(largeIncompressibleData, CompressionThreshold)

	require.NoError(t, err, "CompressData should not error on incompressible data")

	// The function should return uncompressed data if compression doesn't help
	if !wasCompressed {
		require.Equal(t, largeIncompressibleData, result, "Incompressible data should be returned unchanged")
	} else {
		// If it was compressed, verify compressed size is smaller
		require.Less(t, len(result), len(largeIncompressibleData), "If compressed, result should be smaller")
	}
}

func TestCompressData_CustomThreshold(t *testing.T) {
	// Test with custom threshold
	customThreshold := 100
	data := []byte(strings.Repeat("test", 50)) // 200 bytes

	result, wasCompressed, err := CompressData(data, customThreshold)

	require.NoError(t, err, "CompressData should not error")
	require.True(t, wasCompressed, "Data above custom threshold should be compressed")
	require.Less(t, len(result), len(data), "Compressed data should be smaller")
}

func TestDecompressData_ValidGzip(t *testing.T) {
	// Create valid gzip compressed data
	originalData := []byte(strings.Repeat("This is test data. ", 100))

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write(originalData)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	compressedData := buf.Bytes()

	result, err := DecompressData(compressedData)

	require.NoError(t, err, "DecompressData should not error on valid gzip data")
	require.Equal(t, originalData, result, "Decompressed data should match original")
}

func TestDecompressData_InvalidGzip(t *testing.T) {
	// Test with invalid gzip data
	invalidData := []byte("This is not gzip compressed data")

	result, err := DecompressData(invalidData)

	require.Error(t, err, "DecompressData should error on invalid gzip data")
	require.Nil(t, result, "Result should be nil on error")
}

func TestDecompressData_EmptyData(t *testing.T) {
	// Test with empty data
	emptyData := []byte{}

	result, err := DecompressData(emptyData)

	require.Error(t, err, "DecompressData should error on empty data")
	require.Nil(t, result, "Result should be nil on error")
}

func TestCompressDecompress_RoundTrip(t *testing.T) {
	// Test full compression and decompression cycle
	testCases := []struct {
		name string
		data []byte
	}{
		{
			name: "Small text",
			data: []byte("Hello, World!"),
		},
		{
			name: "Large compressible text",
			data: []byte(strings.Repeat("Lava Network is awesome! ", 1000)),
		},
		{
			name: "JSON-like data",
			data: []byte(`{"result": "` + strings.Repeat("0x1234567890abcdef", 1000) + `"}`),
		},
		{
			name: "Large binary-like data",
			data: bytes.Repeat([]byte{0x01, 0x02, 0x03, 0x04}, 3000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Compress
			compressed, wasCompressed, err := CompressData(tc.data, CompressionThreshold)
			require.NoError(t, err, "Compression should not error")

			var decompressed []byte
			if wasCompressed {
				// Decompress
				decompressed, err = DecompressData(compressed)
				require.NoError(t, err, "Decompression should not error")
			} else {
				// If not compressed, data should be unchanged
				decompressed = compressed
			}

			// Verify round-trip
			require.Equal(t, tc.data, decompressed, "Round-trip should preserve data")
		})
	}
}

func TestCompressData_LargePayload(t *testing.T) {
	// Test with a large payload similar to actual RPC responses (15MB)
	largePayload := []byte(strings.Repeat("0x", 7500000)) // ~15MB

	compressed, wasCompressed, err := CompressData(largePayload, CompressionThreshold)

	require.NoError(t, err, "CompressData should handle large payloads")
	require.True(t, wasCompressed, "Large payload should be compressed")

	// Verify compression ratio
	compressionRatio := float64(len(compressed)) / float64(len(largePayload))
	t.Logf("Original size: %d bytes, Compressed size: %d bytes, Ratio: %.2f%%",
		len(largePayload), len(compressed), compressionRatio*100)

	// Decompress and verify
	decompressed, err := DecompressData(compressed)
	require.NoError(t, err, "Decompression should not error")
	require.Equal(t, largePayload, decompressed, "Large payload should survive round-trip")
}

func TestCompressData_ExactThreshold(t *testing.T) {
	// Test data exactly at threshold (1 MB)
	// The check is "len(data) <= threshold", so data AT threshold will NOT be compressed
	// Only data LARGER THAN threshold gets compressed
	exactData := make([]byte, CompressionThreshold)
	for i := range exactData {
		exactData[i] = byte('a') // Highly compressible pattern
	}

	result, wasCompressed, err := CompressData(exactData, CompressionThreshold)

	require.NoError(t, err, "CompressData should not error at exact threshold")
	// Data at exact threshold is NOT > threshold, so it will NOT be compressed
	require.False(t, wasCompressed, "Data at exact threshold should NOT be compressed (only > threshold)")
	require.Equal(t, exactData, result, "Uncompressed data should be unchanged")
}

func TestCompressData_OneByteAboveThreshold(t *testing.T) {
	// Test data one byte above threshold
	data := []byte(strings.Repeat("a", CompressionThreshold+1))

	result, wasCompressed, err := CompressData(data, CompressionThreshold)

	require.NoError(t, err, "CompressData should not error")
	require.True(t, wasCompressed, "Data above threshold should be compressed")
	require.Less(t, len(result), len(data), "Compressed data should be smaller")
}

func BenchmarkCompressData_10KB(b *testing.B) {
	data := []byte(strings.Repeat("test data ", 1000)) // ~10KB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = CompressData(data, CompressionThreshold)
	}
}

func BenchmarkCompressData_100KB(b *testing.B) {
	data := []byte(strings.Repeat("test data ", 10000)) // ~100KB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = CompressData(data, CompressionThreshold)
	}
}

func BenchmarkCompressData_1MB(b *testing.B) {
	data := []byte(strings.Repeat("test data ", 100000)) // ~1MB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, _ = CompressData(data, CompressionThreshold)
	}
}

func BenchmarkDecompressData_10KB(b *testing.B) {
	data := []byte(strings.Repeat("test data ", 1000))
	compressed, _, _ := CompressData(data, CompressionThreshold)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = DecompressData(compressed)
	}
}

func BenchmarkDecompressData_100KB(b *testing.B) {
	data := []byte(strings.Repeat("test data ", 10000))
	compressed, _, _ := CompressData(data, CompressionThreshold)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = DecompressData(compressed)
	}
}

func BenchmarkRoundTrip_1MB(b *testing.B) {
	data := []byte(strings.Repeat("test data ", 100000)) // ~1MB
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		compressed, _, _ := CompressData(data, CompressionThreshold)
		_, _ = DecompressData(compressed)
	}
}
