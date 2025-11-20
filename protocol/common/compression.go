package common

import (
	"bytes"
	"compress/gzip"
	"io"

	"github.com/lavanet/lava/v5/utils"
)

const (
	// CompressionThreshold - only compress payloads larger than this (1 MB)
	// Small payloads have compression overhead > bandwidth savings
	CompressionThreshold = 1024 * 1024 * 1 // 1 MB

	// CompressionLevel - balance between speed and compression ratio
	// BestSpeed = 1 (fastest), BestCompression = 9 (smallest)
	// DefaultCompression = 6 (good balance)
	CompressionLevel = gzip.BestSpeed

	// LavaCompressionHeader - custom header to indicate manual compression
	LavaCompressionHeader = "lava-compression"
	LavaCompressionGzip   = "gzip"

	// LavaCompressionSupportHeader - custom header consumer sends to indicate it supports compression
	// This is different from grpc-accept-encoding which is always sent by gRPC
	LavaCompressionSupportHeader = "lava-compression-support"
)

// CompressData compresses data using gzip if it's larger than threshold
// Returns compressed data, whether it was compressed, and any error
func CompressData(data []byte, threshold int) ([]byte, bool, error) {
	if len(data) < threshold {
		// Too small to benefit from compression
		return data, false, nil
	}

	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, CompressionLevel)
	if err != nil {
		return nil, false, utils.LavaFormatError("failed to create gzip writer", err)
	}

	_, err = writer.Write(data)
	if err != nil {
		writer.Close()
		return nil, false, utils.LavaFormatError("failed to write compressed data", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, false, utils.LavaFormatError("failed to close gzip writer", err)
	}

	compressedData := buf.Bytes()

	// Check if compression actually helped (sometimes it doesn't for random data)
	if len(compressedData) >= len(data) {
		// Compression made it bigger, return original
		return data, false, nil
	}

	return compressedData, true, nil
}

// DecompressData decompresses gzip data
// Returns decompressed data and any error
func DecompressData(compressedData []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, utils.LavaFormatError("failed to create gzip reader", err)
	}
	defer reader.Close()

	decompressedData, err := io.ReadAll(reader)
	if err != nil {
		return nil, utils.LavaFormatError("failed to read decompressed data", err)
	}

	return decompressedData, nil
}
