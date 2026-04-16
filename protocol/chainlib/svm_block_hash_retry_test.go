package chainlib

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

func TestIsBlockNotAvailableError_SolanaErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		response []byte
		expected bool
	}{
		{
			name:     "Solana -32004 block not available",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32004,"message":"Block not available for slot 12345"}}`),
			expected: true,
		},
		{
			name:     "Solana -32004 minimal error",
			response: []byte(`{"error":{"code":-32004}}`),
			expected: true,
		},
		{
			name:     "different error code -32001",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32001,"message":"Block cleaned up"}}`),
			expected: false,
		},
		{
			name:     "different error code -32007",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32007,"message":"Slot skipped"}}`),
			expected: false,
		},
		{
			name:     "success response with result",
			response: []byte(`{"jsonrpc":"2.0","id":1,"result":{"blockhash":"abc123","slot":100}}`),
			expected: false,
		},
		{
			name:     "empty response",
			response: []byte{},
			expected: false,
		},
		{
			name:     "nil response",
			response: nil,
			expected: false,
		},
		{
			name:     "invalid JSON",
			response: []byte(`not json`),
			expected: false,
		},
		{
			name:     "no error field",
			response: []byte(`{"jsonrpc":"2.0","id":1}`),
			expected: false,
		},
		{
			name:     "EVM -32004 method not supported (same code different chain)",
			response: []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32004,"message":"Method not supported"}}`),
			expected: true, // Classified via ChainFamilySolana (callers gate by chain); -32004 → BlockNotFound in Solana tier.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBlockNotAvailableError(tt.response)
			require.Equal(t, tt.expected, result)
		})
	}
}

// blockNotAvailableResponse returns a -32004 JSON-RPC error body for a given slot.
func blockNotAvailableResponse(slot int64) []byte {
	return []byte(fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"error":{"code":-32004,"message":"Block not available for slot %d"}}`, slot))
}

// otherErrorResponse returns a non-retryable JSON-RPC error body.
func otherErrorResponse() []byte {
	return []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid request"}}`)
}

func TestFetchBlockHashWithSolanaRetry(t *testing.T) {
	noDelay := time.Duration(0)
	someHash := "abc123blockhash"
	fetchErr := errors.New("fetch failed")

	t.Run("first attempt succeeds - no retry", func(t *testing.T) {
		calls := 0
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			calls++
			require.Equal(t, int64(100), block)
			return someHash, nil, nil
		}
		hash, fetchedBlock, err := FetchBlockHashWithSolanaRetry(context.Background(), 100, noDelay, fetch)
		require.NoError(t, err)
		require.Equal(t, someHash, hash)
		require.Equal(t, int64(100), fetchedBlock)
		require.Equal(t, 1, calls)
	})

	t.Run("phase 1: succeeds on second same-slot attempt (propagation delay)", func(t *testing.T) {
		calls := 0
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			calls++
			require.Equal(t, int64(100), block) // always same slot in phase 1
			if calls < 2 {
				return "", blockNotAvailableResponse(block), fetchErr
			}
			return someHash, nil, nil
		}
		hash, fetchedBlock, err := FetchBlockHashWithSolanaRetry(context.Background(), 100, noDelay, fetch)
		require.NoError(t, err)
		require.Equal(t, someHash, hash)
		require.Equal(t, int64(100), fetchedBlock)
		require.Equal(t, 2, calls)
	})

	t.Run("phase 1 exhausted, phase 2: succeeds on first previous slot", func(t *testing.T) {
		calls := 0
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			calls++
			if block == 100 {
				return "", blockNotAvailableResponse(block), fetchErr
			}
			// block 99 succeeds
			require.Equal(t, int64(99), block)
			return someHash, nil, nil
		}
		hash, fetchedBlock, err := FetchBlockHashWithSolanaRetry(context.Background(), 100, noDelay, fetch)
		require.NoError(t, err)
		require.Equal(t, someHash, hash)
		require.Equal(t, int64(99), fetchedBlock) // previous slot
		require.Equal(t, MaxSameSlotRetries+1+1, calls)
	})

	t.Run("phase 2: skips multiple slots before succeeding", func(t *testing.T) {
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			if block >= 99 { // 100 and 99 are skipped
				return "", blockNotAvailableResponse(block), fetchErr
			}
			return someHash, nil, nil // 98 succeeds
		}
		hash, fetchedBlock, err := FetchBlockHashWithSolanaRetry(context.Background(), 100, noDelay, fetch)
		require.NoError(t, err)
		require.Equal(t, someHash, hash)
		require.Equal(t, int64(98), fetchedBlock)
	})

	t.Run("non-retryable error returns immediately", func(t *testing.T) {
		calls := 0
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			calls++
			return "", otherErrorResponse(), fetchErr
		}
		_, _, err := FetchBlockHashWithSolanaRetry(context.Background(), 100, noDelay, fetch)
		require.Error(t, err)
		require.Equal(t, 1, calls) // no retry on non-retryable error
	})

	t.Run("all retries exhausted returns error", func(t *testing.T) {
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			return "", blockNotAvailableResponse(block), fetchErr
		}
		_, _, err := FetchBlockHashWithSolanaRetry(context.Background(), 100, noDelay, fetch)
		require.Error(t, err)
	})

	t.Run("blockNum near zero stops previous-slot retries before going negative", func(t *testing.T) {
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			require.GreaterOrEqual(t, block, int64(0))
			return "", blockNotAvailableResponse(block), fetchErr
		}
		_, _, err := FetchBlockHashWithSolanaRetry(context.Background(), 1, noDelay, fetch)
		require.Error(t, err)
	})

	t.Run("context cancellation during phase 1 delay", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		calls := 0
		fetch := func(_ context.Context, block int64) (string, []byte, error) {
			calls++
			cancel() // cancel after first call
			return "", blockNotAvailableResponse(block), fetchErr
		}
		_, _, err := FetchBlockHashWithSolanaRetry(ctx, 100, 10*time.Millisecond, fetch)
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, 1, calls)
	})
}

func TestIsSolanaFamily_DefaultCases(t *testing.T) {
	tests := []struct {
		chainID  string
		expected bool
	}{
		{"SOLANA", true},
		{"SOLANAT", true},
		{"KOII", true},
		{"KOIIT", true},
		{"ETH1", false},
		{"COSMOSHUB", false},
		{"BTC", false},
		{"NEAR", false},
		{"solana", false}, // case-sensitive
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.chainID, func(t *testing.T) {
			result := common.IsSolanaFamily(tt.chainID)
			require.Equal(t, tt.expected, result)
		})
	}
}
