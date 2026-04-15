package relaypolicy

import (
	"fmt"
	"testing"

	"github.com/lavanet/lava/v5/protocol/relaycore"
	"github.com/stretchr/testify/require"
)

func TestDecide_ModeChecks(t *testing.T) {
	policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 2, SendRelayAttempts: 3})

	t.Run("CrossValidation stops", func(t *testing.T) {
		output := policy.Decide(DecisionInput{Selection: relaycore.CrossValidation})
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "CrossValidation", output.Reason)
	})

	t.Run("Stateful stops", func(t *testing.T) {
		output := policy.Decide(DecisionInput{Selection: relaycore.Stateful})
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "Stateful", output.Reason)
	})

	t.Run("Stateless retries by default", func(t *testing.T) {
		output := policy.Decide(DecisionInput{Selection: relaycore.Stateless, Summary: ResultsSummary{NodeErrors: 1}})
		require.Equal(t, Retry, output.Action)
	})
}

func TestDecide_PermanentFailures(t *testing.T) {
	policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 2, SendRelayAttempts: 3})

	t.Run("NonRetryableNodeError stops", func(t *testing.T) {
		output := policy.Decide(DecisionInput{
			Selection: relaycore.Stateless,
			Summary:   ResultsSummary{HasNonRetryableNodeError: true},
		})
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "NonRetryableNodeError", output.Reason)
	})

	t.Run("PermanentProtocolError stops", func(t *testing.T) {
		output := policy.Decide(DecisionInput{
			Selection: relaycore.Stateless,
			Summary:   ResultsSummary{HasPermanentProtocolError: true},
		})
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "PermanentProtocolError", output.Reason)
	})
}

func TestDecide_LimitChecks(t *testing.T) {
	t.Run("MaxRetries stops", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{MaxRetries: 5, RelayRetryLimit: 10, SendRelayAttempts: 3})
		output := policy.Decide(DecisionInput{
			Selection:     relaycore.Stateless,
			AttemptNumber: 5,
			Summary:       ResultsSummary{NodeErrors: 1},
		})
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "MaxRetriesReached", output.Reason)
	})

	t.Run("BatchDisabled stops", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 2, DisableBatchRetry: true, SendRelayAttempts: 3})
		output := policy.Decide(DecisionInput{
			Selection: relaycore.Stateless,
			IsBatch:   true,
			Summary:   ResultsSummary{NodeErrors: 1},
		})
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "BatchDisabled", output.Reason)
	})
}

func TestDecide_ErrorTolerance(t *testing.T) {
	tests := []struct {
		name           string
		relayRetryLimit int
		nodeErrors     int
		expectedAction Action
		expectedReason string
	}{
		{
			name:            "under limit retries",
			relayRetryLimit: 3,
			nodeErrors:      2,
			expectedAction:  Retry,
			expectedReason:  "Default",
		},
		{
			name:            "over limit stops",
			relayRetryLimit: 2,
			nodeErrors:      3,
			expectedAction:  Stop,
			expectedReason:  "ErrorToleranceExceeded",
		},
		{
			name:            "at exact limit retries",
			relayRetryLimit: 2,
			nodeErrors:      2,
			expectedAction:  Retry,
			expectedReason:  "Default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: tc.relayRetryLimit, SendRelayAttempts: 3})
			output := policy.Decide(DecisionInput{
				Selection: relaycore.Stateless,
				Summary:   ResultsSummary{NodeErrors: tc.nodeErrors},
			})
			require.Equal(t, tc.expectedAction, output.Action, "action mismatch")
			require.Equal(t, tc.expectedReason, output.Reason, "reason mismatch")
		})
	}
}

func TestDecide_EpochMismatch(t *testing.T) {
	policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 0, SendRelayAttempts: 3})
	output := policy.Decide(DecisionInput{
		Selection: relaycore.Stateless,
		Summary:   ResultsSummary{HasEpochMismatch: true, SuccessCount: 0, NodeErrors: 5},
	})
	require.Equal(t, Retry, output.Action)
	require.Equal(t, "EpochMismatch", output.Reason)
}

func TestDecide_HashError(t *testing.T) {
	policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 5, SendRelayAttempts: 3})
	output := policy.Decide(DecisionInput{
		Selection: relaycore.Stateless,
		Summary:   ResultsSummary{HashErr: fmt.Errorf("hash failed"), NodeErrors: 1},
	})
	require.Equal(t, Stop, output.Action)
	require.Equal(t, "HashComputationFailed", output.Reason)
}

func TestOnSendRelayResult(t *testing.T) {
	t.Run("success resets counters", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{SendRelayAttempts: 3, EnableCircuitBreaker: true, CircuitBreakerThreshold: 2})
		policy.OnSendRelayResult(fmt.Errorf("err"), false)
		result := policy.OnSendRelayResult(nil, false)
		require.Equal(t, SendSuccess, result)
		require.Equal(t, 0, policy.GetConsecutiveBatchErrors())
	})

	t.Run("batch errors stop after threshold", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{SendRelayAttempts: 2})
		require.Equal(t, SendRetry, policy.OnSendRelayResult(fmt.Errorf("err1"), false))
		require.Equal(t, SendRetry, policy.OnSendRelayResult(fmt.Errorf("err2"), false))
		require.Equal(t, SendStop, policy.OnSendRelayResult(fmt.Errorf("err3"), false))
	})

	t.Run("circuit breaker trips on pairing errors", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{SendRelayAttempts: 10, EnableCircuitBreaker: true, CircuitBreakerThreshold: 2})
		require.Equal(t, SendRetry, policy.OnSendRelayResult(fmt.Errorf("pairing"), true))
		require.Equal(t, SendStop, policy.OnSendRelayResult(fmt.Errorf("pairing"), true))
	})

	t.Run("non-pairing error resets pairing counter", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{SendRelayAttempts: 10, EnableCircuitBreaker: true, CircuitBreakerThreshold: 2})
		policy.OnSendRelayResult(fmt.Errorf("pairing"), true)
		policy.OnSendRelayResult(fmt.Errorf("other"), false) // resets pairing counter
		result := policy.OnSendRelayResult(fmt.Errorf("pairing"), true)
		require.Equal(t, SendRetry, result) // only 1 consecutive, threshold is 2
	})
}

func TestDecide_IsTickerHedge(t *testing.T) {
	t.Run("ticker hedge skips error tolerance", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 2, SendRelayAttempts: 3})
		// 3 errors exceeds limit of 2 — gotResults path would stop
		input := DecisionInput{
			Selection: relaycore.Stateless,
			Summary:   ResultsSummary{NodeErrors: 3},
		}
		output := policy.Decide(input)
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "ErrorToleranceExceeded", output.Reason)

		// Same input but as ticker hedge — should still retry
		input.IsTickerHedge = true
		output = policy.Decide(input)
		require.Equal(t, Retry, output.Action)
		require.Equal(t, "Default", output.Reason)
	})

	t.Run("ticker hedge skips hash error", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 5, SendRelayAttempts: 3})
		input := DecisionInput{
			Selection: relaycore.Stateless,
			Summary:   ResultsSummary{HashErr: fmt.Errorf("hash failed"), NodeErrors: 1},
		}
		output := policy.Decide(input)
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "HashComputationFailed", output.Reason)

		input.IsTickerHedge = true
		output = policy.Decide(input)
		require.Equal(t, Retry, output.Action)
	})

	t.Run("ticker hedge still respects mode and limit checks", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{MaxRetries: 5, RelayRetryLimit: 2, SendRelayAttempts: 3})
		// MaxRetries should still stop ticker hedges
		input := DecisionInput{
			Selection:     relaycore.Stateless,
			AttemptNumber: 5,
			IsTickerHedge: true,
			Summary:       ResultsSummary{NodeErrors: 1},
		}
		output := policy.Decide(input)
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "MaxRetriesReached", output.Reason)

		// Non-retryable node error should still stop ticker hedges
		input2 := DecisionInput{
			Selection:     relaycore.Stateless,
			IsTickerHedge: true,
			Summary:       ResultsSummary{HasNonRetryableNodeError: true},
		}
		output = policy.Decide(input2)
		require.Equal(t, Stop, output.Action)
		require.Equal(t, "NonRetryableNodeError", output.Reason)
	})
}

func TestDecide_ArchiveMutation(t *testing.T) {
	t.Run("first retry adds archive", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 5, SendRelayAttempts: 3})
		archiveStatus := &relaycore.ArchiveStatus{}
		output := policy.Decide(DecisionInput{
			Selection:     relaycore.Stateless,
			AttemptNumber: 1,
			Summary:       ResultsSummary{NodeErrors: 1},
			ArchiveStatus: archiveStatus,
			NodeErrors:    1,
		})
		require.Equal(t, Retry, output.Action)
		require.Equal(t, AddArchive, output.Mutation.ArchiveAction)
		require.False(t, output.Mutation.CacheHashes)
	})

	t.Run("no archive status returns no mutation", func(t *testing.T) {
		policy := NewPolicy(PolicyConfig{MaxRetries: 10, RelayRetryLimit: 5, SendRelayAttempts: 3})
		output := policy.Decide(DecisionInput{
			Selection:     relaycore.Stateless,
			AttemptNumber: 1,
			Summary:       ResultsSummary{NodeErrors: 1},
			ArchiveStatus: nil,
		})
		require.Equal(t, Retry, output.Action)
		require.Equal(t, NoChange, output.Mutation.ArchiveAction)
	})
}
