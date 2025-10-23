package utils_test

import (
	"fmt"
	"testing"

	sdkerrors "cosmossdk.io/errors"
	"github.com/lavanet/lava/v5/utils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var TestError = sdkerrors.New("test Error", 123, "error for tests")

// TestErrorTypeChecks verifies that error wrapping preserves error type
func TestErrorTypeChecks(t *testing.T) {
	var err error = TestError
	newErr := utils.LavaFormatError("testing 123", err, utils.Attribute{"attribute", "test"})
	require.True(t, TestError.Is(newErr))
}

// TestExtractErrorStructure validates that the error extraction function
// correctly extracts structured information from different error types
func TestExtractErrorStructure(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected map[string]interface{}
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: nil,
		},
		{
			name: "plain error",
			err:  fmt.Errorf("connection timeout"),
			expected: map[string]interface{}{
				"error":         "connection timeout",
				"error_message": "connection timeout",
			},
		},
		{
			name: "sdkerrors.Error",
			err:  sdkerrors.New("TestError", 1234, "test error description"),
			expected: map[string]interface{}{
				"error_code":    uint32(1234),
				"error":         "test error description",
				"error_message": "test error description",
			},
		},
		{
			name: "gRPC status error with embedded code",
			err:  status.Error(codes.Code(3370), "relayReceiver is disabled"),
			expected: map[string]interface{}{
				"error":         "rpc error: code = Code(3370) desc = relayReceiver is disabled",
				"error_code":    uint32(3370),
				"error_message": "relayReceiver is disabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ExtractErrorStructure(tt.err)

			// Check if both are nil
			if tt.expected == nil && result == nil {
				return
			}

			// Check if one is nil and the other is not
			if (tt.expected == nil) != (result == nil) {
				t.Errorf("expected nil=%v, got nil=%v", tt.expected == nil, result == nil)
				return
			}

			// Check each expected field
			for key, expectedVal := range tt.expected {
				actualVal, exists := result[key]
				if !exists {
					t.Errorf("expected field '%s' not found in result", key)
					continue
				}

				// Compare values based on type
				if fmt.Sprintf("%v", expectedVal) != fmt.Sprintf("%v", actualVal) {
					t.Errorf("field '%s': expected '%v', got '%v'", key, expectedVal, actualVal)
				}
			}

			// Check for unexpected fields
			for key := range result {
				if _, exists := tt.expected[key]; !exists {
					t.Logf("unexpected field '%s' with value '%v' in result", key, result[key])
				}
			}
		})
	}
}

// TestStructuredLogging demonstrates the before/after logging behavior
func TestStructuredLogging(t *testing.T) {
	// Enable JSON format for testing
	originalFormat := utils.JsonFormat
	utils.JsonFormat = true
	defer func() { utils.JsonFormat = originalFormat }()

	t.Run("sdkerrors with attributes", func(t *testing.T) {
		err := sdkerrors.New("DisabledRelayReceiverError Error", 3370, "provider does not pass verification and disabled this interface and spec")

		// This would previously create a concatenated string
		// Now it creates structured fields
		logErr := utils.LavaFormatError(
			"relayReceiver is disabled",
			err,
			utils.LogAttr("relayReceiver", "COSMOSHUBtendermintrpc"),
		)

		if logErr == nil {
			t.Error("expected non-nil error return")
		}

		// The returned error should still work for error handling
		if logErr.Error() == "" {
			t.Error("expected non-empty error message")
		}
	})

	t.Run("gRPC error from provider", func(t *testing.T) {
		// Simulate what provider sends
		grpcErr := status.Error(
			codes.Code(3370),
			"rpc error: code = Code(3370) desc = relayReceiver is disabled ErrMsg: provider does not pass verification and disabled this interface and spec {relayReceiver:COSMOSHUBtendermintrpc}: provider does not pass verification and disabled this interface and spec",
		)

		// Consumer logs this
		logErr := utils.LavaFormatError(
			"could not send relay to provider",
			grpcErr,
			utils.LogAttr("GUID", "7300519992960734468"),
			utils.LogAttr("provider", "lava@1t55ssmcjdz49p9ae8kmgxc06llqn2vnc9942tg"),
		)

		if logErr == nil {
			t.Error("expected non-nil error return")
		}
	})

	t.Run("real consumer log error - unhandled relay receiver", func(t *testing.T) {
		// This is the ACTUAL error format from real CONSUMERS.log
		// Step 1: Provider sends gRPC error with simple message (no "desc =" prefix)
		errorMsg := "got called with unhandled relay receiver: provider does not handle requested api interface and spec"
		grpcErr := status.Error(codes.Code(3369), errorMsg)

		// Step 2: relay_processor wraps it with "failed relay, insufficient results"
		wrappedErr := utils.LavaFormatError("failed relay, insufficient results", grpcErr)

		// Test error extraction on the wrapped error
		structured := utils.ExtractErrorStructure(wrappedErr)
		if structured == nil {
			t.Error("expected non-nil structured error")
		}

		// Verify key fields are extracted from the wrapped error
		if code, exists := structured["error_code"]; !exists {
			t.Error("expected error_code to be extracted from wrapped error")
		} else if fmt.Sprintf("%v", code) != "3369" {
			t.Errorf("expected error_code 3369, got %v", code)
		}

		if desc, exists := structured["error_message"]; !exists {
			t.Error("expected error_message to be extracted")
		} else if desc == "" {
			t.Error("expected non-empty error_message")
		}

		// Step 3: rpcconsumer_server logs the wrapped error with more context
		logErr := utils.LavaFormatError(
			"[-] failed sending init relay",
			wrappedErr,
			utils.LogAttr("APIInterface", "rest"),
			utils.LogAttr("chainID", "LAV1"),
		)

		if logErr == nil {
			t.Error("expected non-nil error return")
		}

		t.Logf("Extracted fields from wrapped error: %v", structured)
	})
}

// TestFullErrorFlow simulates the exact error flow from CONSUMERS.log and prints output
func TestFullErrorFlow(t *testing.T) {
	// Test in BOTH formats to show the difference
	originalFormat := utils.JsonFormat
	defer func() { utils.JsonFormat = originalFormat }()

	// First show console format (what you see in logs)
	utils.JsonFormat = false
	t.Log("\n========================================")
	t.Log("CONSOLE FORMAT OUTPUT (current logs)")
	t.Log("========================================")
	testErrorFlow(t)

	// Then show JSON format (what ELK receives)
	utils.JsonFormat = true
	t.Log("\n========================================")
	t.Log("JSON FORMAT OUTPUT (ELK sees this)")
	t.Log("========================================")
	testErrorFlow(t)
}

func testErrorFlow(t *testing.T) {
	// Step 1: Provider sends gRPC error (this happens in provider_listener.go)
	t.Log("\nSTEP 1: Provider sends gRPC error with code 3369")
	providerErrorMsg := "got called with unhandled relay receiver: provider does not handle requested api interface and spec"
	grpcErr := status.Error(codes.Code(3369), providerErrorMsg)
	t.Logf("Provider error: %v\n", grpcErr)

	// Step 2: relay_processor.ProcessingResult() wraps it (relay_processor.go:651)
	t.Log("\nSTEP 2: relay_processor wraps error with 'failed relay, insufficient results'")
	t.Log("Expected JSON output from relay_processor:")
	wrappedErr := utils.LavaFormatError("failed relay, insufficient results", grpcErr)
	t.Log("")

	// Step 3: rpcconsumer_server logs the wrapped error (rpcconsumer_server.go:305)
	t.Log("\nSTEP 3: rpcconsumer_server logs with context (THIS IS WHAT YOU SEE IN CONSUMERS.LOG)")
	_ = utils.LavaFormatError(
		"[-] failed sending init relay",
		wrappedErr,
		utils.LogAttr("APIInterface", "rest"),
		utils.LogAttr("chainID", "LAV1"),
		utils.LogAttr("relayProcessor", "relayProcessor {resultsManager {success 0, nodeErrors:0, specialNodeErrors:0, protocolErrors:1}}"),
	)
	t.Log("")
}

// BenchmarkExtractErrorStructure measures the performance impact of error extraction
func BenchmarkExtractErrorStructure(b *testing.B) {
	err := sdkerrors.New("TestError", 1234, "test error description")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = utils.ExtractErrorStructure(err)
	}
}

// BenchmarkStructuredLogging compares old vs new logging performance
func BenchmarkStructuredLogging(b *testing.B) {
	utils.JsonFormat = true
	err := sdkerrors.New("TestError", 1234, "test error description")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = utils.LavaFormatError(
			"test error",
			err,
			utils.LogAttr("field1", "value1"),
			utils.LogAttr("field2", "value2"),
		)
	}
}
