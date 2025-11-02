package metrics

import (
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/v5/protocol/lavaprotocol/protocolerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStructuredErrorExtraction tests that GetUniqueGuidResponseForError
// properly extracts structured error information using ExtractErrorStructure
func TestStructuredErrorExtraction(t *testing.T) {
	// Set ReturnMaskedErrors to "false" to enable error details
	originalValue := ReturnMaskedErrors
	ReturnMaskedErrors = "false"
	defer func() { ReturnMaskedErrors = originalValue }()

	plog, err := NewRPCConsumerLogs(nil, nil, nil, nil)
	require.NoError(t, err)

	testCases := []struct {
		name               string
		err                error
		expectedGUID       string
		expectErrorCode    bool
		expectErrorMessage bool
	}{
		{
			name:               "Provider disabled error with code 3370",
			err:                protocolerrors.DisabledRelayReceiverError,
			expectedGUID:       "test-guid-1",
			expectErrorCode:    true,
			expectErrorMessage: true,
		},
		{
			name:               "Consistency error with code 3368",
			err:                protocolerrors.ConsistencyError,
			expectedGUID:       "test-guid-2",
			expectErrorCode:    true,
			expectErrorMessage: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the error response
			errorResponse := plog.GetUniqueGuidResponseForError(tc.err, tc.expectedGUID)

			// Parse the response
			type ErrorData struct {
				Error_GUID    string `json:"Error_GUID"`
				Error         string `json:"Error,omitempty"`
				Error_Code    string `json:"Error_Code,omitempty"`
				Error_Message string `json:"Error_Message,omitempty"`
			}

			var errData ErrorData
			err := json.Unmarshal([]byte(errorResponse), &errData)
			require.NoError(t, err, "Failed to unmarshal error response")

			// Verify GUID
			assert.Equal(t, tc.expectedGUID, errData.Error_GUID, "GUID should match")

			// Verify raw error is present
			assert.NotEmpty(t, errData.Error, "Raw error should be present")

			// Verify structured fields based on expectations
			if tc.expectErrorCode {
				assert.NotEmpty(t, errData.Error_Code, "Error code should be extracted")
			}

			if tc.expectErrorMessage {
				assert.NotEmpty(t, errData.Error_Message, "Clean error message should be extracted")
				// Clean message should not contain gRPC artifacts
				assert.NotContains(t, errData.Error_Message, "rpc error:", "Clean message should not have gRPC prefix")
				assert.NotContains(t, errData.Error_Message, "code = Code(", "Clean message should not have code formatting")
			}
		})
	}
}

// TestStructuredErrorMasked verifies that when ReturnMaskedErrors is "true",
// no error details are returned (for security)
func TestStructuredErrorMasked(t *testing.T) {
	// Set ReturnMaskedErrors to "true" to mask errors
	originalValue := ReturnMaskedErrors
	ReturnMaskedErrors = "true"
	defer func() { ReturnMaskedErrors = originalValue }()

	plog, err := NewRPCConsumerLogs(nil, nil, nil, nil)
	require.NoError(t, err)

	testErr := protocolerrors.DisabledRelayReceiverError
	errorResponse := plog.GetUniqueGuidResponseForError(testErr, "test-guid")

	// Parse the response
	type ErrorData struct {
		Error_GUID    string `json:"Error_GUID"`
		Error         string `json:"Error,omitempty"`
		Error_Code    string `json:"Error_Code,omitempty"`
		Error_Message string `json:"Error_Message,omitempty"`
	}

	var errData ErrorData
	unmarshalErr := json.Unmarshal([]byte(errorResponse), &errData)
	require.NoError(t, unmarshalErr, "Failed to unmarshal error response")

	// Verify only GUID is present, no error details
	assert.Equal(t, "test-guid", errData.Error_GUID)
	assert.Empty(t, errData.Error, "Error should be masked")
	assert.Empty(t, errData.Error_Code, "Error code should be masked")
	assert.Empty(t, errData.Error_Message, "Error message should be masked")
}
