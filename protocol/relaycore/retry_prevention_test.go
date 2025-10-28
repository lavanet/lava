package relaycore

import (
	"errors"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib"
	"github.com/stretchr/testify/require"
)

func TestRelayProcessorRetryPrevention(t *testing.T) {
	// Create a mock relay processor to test hasUnsupportedMethodErrors
	// This test verifies that unsupported method errors prevent retries

	t.Run("shouldRetryWithUnsupportedMethodError", func(t *testing.T) {
		// Test the basic retry logic with unsupported method error
		unsupportedErr := chainlib.NewUnsupportedMethodError(errors.New("method not found"), "eth_test")

		result := chainlib.ShouldRetryError(unsupportedErr)
		require.False(t, result, "Should not retry unsupported method errors")
	})

	t.Run("shouldRetryWithNormalError", func(t *testing.T) {
		// Test that normal errors still allow retries
		normalErr := errors.New("network timeout")

		result := chainlib.ShouldRetryError(normalErr)
		require.True(t, result, "Should allow retry for normal errors")
	})
}
