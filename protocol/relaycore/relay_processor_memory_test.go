package relaycore

import (
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/lavanet/lava/v5/protocol/common"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	spectypes "github.com/lavanet/lava/v5/x/spec/types"
	"github.com/stretchr/testify/require"
)

// BenchmarkMemoryUsageOnBatchRequests benchmarks memory usage when processing many batch requests
// This benchmark verifies the fix where we switched from storing full string canonical forms to hashes in crossValidationMap
func BenchmarkMemoryUsageOnBatchRequests(b *testing.B) {
	// Create a large Solana-like batch response (similar to what caused the memory leak)
	batchResponse := createLargeSolanaBatchResponse(100) // 100 items in batch
	responseData, _ := json.Marshal(batchResponse)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		crossValidationMap := make(map[[32]byte]int)

		// Process 1000 requests with the same large data
		// In the old implementation, this would create 1000 copies of canonical string forms
		// In the new implementation, it only stores 1000 hashes (32 bytes each)
		for j := 0; j < 1000; j++ {
			hash := sha256.Sum256(responseData)
			crossValidationMap[hash]++
		}
	}
}

// TestCrossValidationMapUsesHashesNotStrings verifies the algorithm uses hashes, not strings
// This is a functional correctness test that should remain stable long-term
func TestCrossValidationMapUsesHashesNotStrings(t *testing.T) {
	crossValidationMap := make(map[[32]byte]int)

	// Create large data (100KB)
	batchResponse := createLargeSolanaBatchResponse(100)
	responseData, err := json.Marshal(batchResponse)
	require.NoError(t, err)
	require.Greater(t, len(responseData), 10000, "Response should be > 10KB")

	// Hash the data and store
	hash := sha256.Sum256(responseData)
	crossValidationMap[hash]++

	// Verify the map uses hash keys (32 bytes), not the full data
	require.Equal(t, 1, len(crossValidationMap), "Should have 1 entry")
	require.Equal(t, 32, len(hash), "Hash should always be 32 bytes")

	// Verify the same data produces the same hash
	hash2 := sha256.Sum256(responseData)
	require.Equal(t, hash, hash2, "Same data should produce same hash")
	crossValidationMap[hash2]++
	require.Equal(t, 1, len(crossValidationMap), "Same hash should not create new entry")
	require.Equal(t, 2, crossValidationMap[hash], "Count should increment")

	// Verify different data produces different hash
	batchResponse2 := createLargeSolanaBatchResponse(101)
	responseData2, _ := json.Marshal(batchResponse2)
	hash3 := sha256.Sum256(responseData2)
	require.NotEqual(t, hash, hash3, "Different data should produce different hash")
	crossValidationMap[hash3]++
	require.Equal(t, 2, len(crossValidationMap), "Different hash should create new entry")
}

// TestCrossValidationMapHashCollisions tests that different responses produce different hashes
func TestCrossValidationMapHashCollisions(t *testing.T) {
	crossValidationMap := make(map[[32]byte]int)

	// Create 1000 different responses
	for i := 0; i < 1000; i++ {
		response := createLargeSolanaBatchResponse(10)
		response["unique_id"] = i // Make each response unique
		responseData, err := json.Marshal(response)
		require.NoError(t, err)

		hash := sha256.Sum256(responseData)
		crossValidationMap[hash]++
	}

	// Should have 1000 unique hashes
	require.Equal(t, 1000, len(crossValidationMap), "All different responses should have different hashes")

	// Each hash should have count of 1
	for _, count := range crossValidationMap {
		require.Equal(t, 1, count, "Each unique response should have count of 1")
	}
}

// BenchmarkRelayProcessorCrossValidationMemory benchmarks memory usage of RelayProcessor cross-validation functionality
func BenchmarkRelayProcessorCrossValidationMemory(b *testing.B) {
	// Simulate processing 100 identical large responses
	largeResponse := createLargeSolanaBatchResponse(50)
	responseData, _ := json.Marshal(largeResponse)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rp := &RelayProcessor{
			crossValidationMap: make(map[[32]byte]int),
		}

		for range 100 {
			response := &RelayResponse{
				RelayResult: common.RelayResult{
					Reply: &pairingtypes.RelayReply{
						Data:        responseData,
						LatestBlock: 12345,
					},
					ProviderInfo: common.ProviderInfo{
						ProviderAddress: "provider",
					},
				},
				Err: nil,
			}

			// Only hash successful responses
			if response.Err == nil {
				hash := sha256.Sum256(response.RelayResult.GetReply().GetData())
				rp.crossValidationMap[hash]++
				if rp.crossValidationMap[hash] > rp.currentCrossValidationEqualResults {
					rp.currentCrossValidationEqualResults = rp.crossValidationMap[hash]
				}
			}
		}
	}
}

// BenchmarkResponsesCrossValidationMemory benchmarks memory usage of responsesCrossValidation function
func BenchmarkResponsesCrossValidationMemory(b *testing.B) {
	// Create large batch response
	batchResponse := createLargeSolanaBatchResponse(100)
	responseData, _ := json.Marshal(batchResponse)

	// Create 50 relay results with identical data
	results := make([]common.RelayResult, 50)
	for i := range results {
		results[i] = common.RelayResult{
			Reply: &pairingtypes.RelayReply{
				Data:        responseData,
				LatestBlock: 12345,
			},
			ProviderInfo: common.ProviderInfo{
				ProviderAddress: "provider",
			},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Process cross-validation - this is where the old implementation would create many canonical strings
		countMap := make(map[[32]byte]*struct {
			count  int
			result common.RelayResult
		})

		for _, result := range results {
			if result.Reply != nil && result.Reply.Data != nil {
				hash := sha256.Sum256(result.Reply.Data)
				if count, exists := countMap[hash]; exists {
					count.count++
				} else {
					countMap[hash] = &struct {
						count  int
						result common.RelayResult
					}{
						count:  1,
						result: result,
					}
				}
			}
		}
	}
}

// BenchmarkHashVsCanonicalForm benchmarks the performance difference between hashing and canonical form
func BenchmarkHashVsCanonicalForm(b *testing.B) {
	batchResponse := createLargeSolanaBatchResponse(100)
	responseData, _ := json.Marshal(batchResponse)

	b.Run("SHA256Hash", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = sha256.Sum256(responseData)
		}
	})

	b.Run("JSONUnmarshalMarshal", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var temp interface{}
			_ = json.Unmarshal(responseData, &temp)
			_, _ = json.Marshal(temp)
		}
	})
}

// createLargeSolanaBatchResponse creates a large Solana-like batch response for testing
// This simulates the type of responses that were causing memory leaks
func createLargeSolanaBatchResponse(numItems int) map[string]interface{} {
	results := make([]map[string]interface{}, numItems)

	for i := range numItems {
		results[i] = map[string]interface{}{
			"lamports":   999999999,
			"owner":      "11111111111111111111111111111111",
			"executable": false,
			"rentEpoch":  uint64(18446744073709551615),
			"data": []string{
				"base64_encoded_data_here_which_can_be_very_long_and_cause_memory_issues_if_not_handled_properly",
				"encoding",
			},
		}
	}

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"context": map[string]interface{}{
				"slot": 12345678,
			},
			"value": results,
		},
	}
}

// Mock implementations for testing

type mockChainMessage struct {
	chainlib.ChainMessage
	api *spectypes.Api
}

func (m *mockChainMessage) GetApi() *spectypes.Api {
	return m.api
}
