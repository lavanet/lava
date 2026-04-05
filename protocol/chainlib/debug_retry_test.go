package chainlib

import (
	"errors"
	"testing"

	"github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSpecificErrorFromUser(t *testing.T) {
	// Test the specific error pattern from the user
	errorMsg := "rpc error: code = Unknown desc = unsupported method 'Default-/cosmos/base/tendermint/v1beta1/blocks1/2': Not Implemented"

	// Test with actual error
	t.Run("Error object detection", func(t *testing.T) {
		err := errors.New(errorMsg)
		result := IsUnsupportedMethodError(err)
		require.True(t, result, "Should detect unsupported method error from error object")
	})

	// Test with gRPC Unknown status
	t.Run("gRPC Unknown status", func(t *testing.T) {
		grpcErr := status.Error(codes.Unknown, "unsupported method 'Default-/cosmos/base/tendermint/v1beta1/blocks1/2': Not Implemented")
		result := IsUnsupportedMethodError(grpcErr)
		require.True(t, result, "Should detect unsupported method from gRPC Unknown status")
	})

	// Test ShouldRetryErrorWithContext — this is a gRPC error ("rpc error: code = Unknown desc = ..."),
	// so the gRPC transport must be specified for the registry to detect "Not Implemented".
	t.Run("ShouldRetryError with user error", func(t *testing.T) {
		err := errors.New(errorMsg)
		result := ShouldRetryErrorWithContext(err, -1, common.TransportGRPC)
		require.False(t, result, "Should NOT retry with unsupported method error")
	})
}
