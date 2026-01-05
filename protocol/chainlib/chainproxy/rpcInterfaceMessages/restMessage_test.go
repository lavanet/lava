package rpcInterfaceMessages

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestMessage(t *testing.T) {
	// Test GetParams method
	restMessage := RestMessage{
		Path:     "blocks/latest",
		SpecPath: "blocks/latest",
	}

	// Test GetParams method
	params := restMessage.GetParams()
	require.Nil(t, params)

	// Test GetResult method
	result := restMessage.GetResult()
	if result != nil {
		t.Errorf("Expected nil, but got %v", result)
	}
}

func TestRestParseBlock(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "Default block param",
			input:    "latest",
			expected: -2,
		},
		{
			name:     "String representation of int64",
			input:    "80",
			expected: 80,
		},
		{
			name:     "Hex representation of int64",
			input:    "0x26D",
			expected: 621,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			restMessage := RestMessage{}

			block, err := restMessage.ParseBlock(testCase.input)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if block != testCase.expected {
				t.Errorf("Expected %v, but got %v", testCase.expected, block)
			}
		})
	}
}

// TestCheckResponseError_CosmosTransaction_Success tests successful Cosmos transaction (code=0)
func TestCheckResponseError_CosmosTransaction_Success(t *testing.T) {
	restMsg := RestMessage{}

	// Cosmos SDK transaction success response (HTTP 200 + code=0)
	successResponse := `{
		"tx_response": {
			"height": "12345",
			"txhash": "ABC123DEF456789",
			"codespace": "",
			"code": 0,
			"data": "0A1E0A1C2F636F736D6F732E62616E6B2E763162657461312E4D736753656E64",
			"raw_log": "[]",
			"logs": [{"msg_index": 0, "log": "", "events": []}],
			"info": "",
			"gas_wanted": "200000",
			"gas_used": "85123",
			"tx": {},
			"timestamp": "2025-01-15T10:30:00Z",
			"events": []
		}
	}`

	hasError, errorMessage := restMsg.CheckResponseError([]byte(successResponse), 200)

	require.False(t, hasError, "Should not detect error for code=0")
	require.Empty(t, errorMessage, "Error message should be empty for success")
}

// TestCheckResponseError_CosmosTransaction_SequenceMismatch tests sequence mismatch error (code=32)
func TestCheckResponseError_CosmosTransaction_SequenceMismatch(t *testing.T) {
	restMsg := RestMessage{}

	// Cosmos SDK transaction error (HTTP 200 + code=32)
	errorResponse := `{
		"tx_response": {
			"height": "0",
			"txhash": "CB584A80FE7EA54E4CCEFCA6BE89B4FABCF4B0087766733FB8E1741ACB61EFFE",
			"codespace": "sdk",
			"code": 32,
			"data": "",
			"raw_log": "account sequence mismatch, expected 379, got 392: incorrect account sequence",
			"logs": [],
			"info": "",
			"gas_wanted": "0",
			"gas_used": "0",
			"tx": null,
			"timestamp": "",
			"events": []
		}
	}`

	hasError, errorMessage := restMsg.CheckResponseError([]byte(errorResponse), 200)

	require.True(t, hasError, "Should detect error for code=32")
	require.Contains(t, errorMessage, "account sequence mismatch", "Should return raw_log as error message")
}

// TestCheckResponseError_CosmosTransaction_VariousErrorCodes tests various Cosmos transaction error codes
func TestCheckResponseError_CosmosTransaction_VariousErrorCodes(t *testing.T) {
	testCases := []struct {
		name        string
		code        int
		rawLog      string
		expectError bool
	}{
		{
			name:        "Success (code=0)",
			code:        0,
			rawLog:      "[]",
			expectError: false,
		},
		{
			name:        "Invalid Signature (code=4)",
			code:        4,
			rawLog:      "signature verification failed",
			expectError: true,
		},
		{
			name:        "Out of Gas (code=11)",
			code:        11,
			rawLog:      "out of gas",
			expectError: true,
		},
		{
			name:        "Insufficient Fee (code=13)",
			code:        13,
			rawLog:      "insufficient fees",
			expectError: true,
		},
		{
			name:        "Invalid Transaction (code=18)",
			code:        18,
			rawLog:      "invalid transaction: must contain at least one message",
			expectError: true,
		},
		{
			name:        "TX Already in Cache (code=19)",
			code:        19,
			rawLog:      "tx already exists in cache",
			expectError: true,
		},
		{
			name:        "Sequence Mismatch (code=32)",
			code:        32,
			rawLog:      "account sequence mismatch",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restMsg := RestMessage{}

			response := fmt.Sprintf(`{
				"tx_response": {
					"height": "12345",
					"txhash": "ABC123DEF456",
					"code": %d,
					"raw_log": "%s",
					"logs": [],
					"gas_wanted": "200000",
					"gas_used": "150000"
				}
			}`, tc.code, tc.rawLog)

			hasError, errorMessage := restMsg.CheckResponseError([]byte(response), 200)

			require.Equal(t, tc.expectError, hasError, "Error detection mismatch for %s", tc.name)
			if tc.expectError {
				require.Equal(t, tc.rawLog, errorMessage, "Error message should match raw_log for %s", tc.name)
			}
		})
	}
}

// TestCheckResponseError_HTTPStatusCodeVariations tests different HTTP status codes
func TestCheckResponseError_HTTPStatusCodeVariations(t *testing.T) {
	testCases := []struct {
		name          string
		httpStatus    int
		response      string
		expectedError bool
		errorCheck    func(t *testing.T, hasError bool, errorMessage string)
	}{
		{
			name:       "HTTP 200 + code=0 (Success)",
			httpStatus: 200,
			response: `{
				"tx_response": {
					"code": 0,
					"raw_log": "[]"
				}
			}`,
			expectedError: false,
		},
		{
			name:       "HTTP 200 + code=32 (Error)",
			httpStatus: 200,
			response: `{
				"tx_response": {
					"code": 32,
					"raw_log": "sequence mismatch"
				}
			}`,
			expectedError: true,
			errorCheck: func(t *testing.T, hasError bool, errorMessage string) {
				require.Contains(t, errorMessage, "sequence mismatch")
			},
		},
		{
			name:       "HTTP 400 (Bad Request)",
			httpStatus: 400,
			response: `{
				"message": "bad request",
				"code": 400
			}`,
			expectedError: true,
			errorCheck: func(t *testing.T, hasError bool, errorMessage string) {
				require.Contains(t, errorMessage, "bad request")
			},
		},
		{
			name:       "HTTP 500 (Server Error)",
			httpStatus: 500,
			response: `{
				"message": "internal server error",
				"code": 500
			}`,
			expectedError: true,
			errorCheck: func(t *testing.T, hasError bool, errorMessage string) {
				require.Contains(t, errorMessage, "internal server error")
			},
		},
		{
			name:       "HTTP 404 (Not Found)",
			httpStatus: 404,
			response: `{
				"message": "not found",
				"code": 404
			}`,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restMsg := RestMessage{}

			hasError, errorMessage := restMsg.CheckResponseError([]byte(tc.response), tc.httpStatus)

			require.Equal(t, tc.expectedError, hasError, "Error detection mismatch for %s", tc.name)
			if tc.errorCheck != nil {
				tc.errorCheck(t, hasError, errorMessage)
			}
		})
	}
}

// TestCheckResponseError_NonTransactionCalls tests non-transaction REST calls (regression)
func TestCheckResponseError_NonTransactionCalls(t *testing.T) {
	testCases := []struct {
		name          string
		response      string
		httpStatus    int
		expectedError bool
	}{
		{
			name: "Query Balance - Success",
			response: `{
				"balances": [
					{"denom": "ulava", "amount": "1000000"}
				],
				"pagination": {}
			}`,
			httpStatus:    200,
			expectedError: false,
		},
		{
			name: "Query TX by Hash - Success",
			response: `{
				"tx": {},
				"tx_response": {
					"code": 0,
					"txhash": "ABC123"
				}
			}`,
			httpStatus:    200,
			expectedError: false,
		},
		{
			name: "Query TX by Hash - Not Found",
			response: `{
				"message": "tx not found",
				"code": 404
			}`,
			httpStatus:    404,
			expectedError: true,
		},
		{
			name: "Query Block - Success",
			response: `{
				"block": {
					"header": {
						"height": "12345"
					}
				}
			}`,
			httpStatus:    200,
			expectedError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restMsg := RestMessage{}

			hasError, _ := restMsg.CheckResponseError([]byte(tc.response), tc.httpStatus)

			require.Equal(t, tc.expectedError, hasError, "Error detection mismatch for %s", tc.name)
		})
	}
}

// TestCheckResponseError_EdgeCases tests edge cases and malformed data
func TestCheckResponseError_EdgeCases(t *testing.T) {
	testCases := []struct {
		name          string
		response      string
		httpStatus    int
		expectedError bool
	}{
		{
			name:          "Empty Response",
			response:      "",
			httpStatus:    200,
			expectedError: false,
		},
		{
			name:          "Invalid JSON",
			response:      "{invalid json",
			httpStatus:    200,
			expectedError: false,
		},
		{
			name: "Missing tx_response field",
			response: `{
				"height": "12345",
				"txhash": "ABC123"
			}`,
			httpStatus:    200,
			expectedError: false,
		},
		{
			name: "tx_response with missing code field",
			response: `{
				"tx_response": {
					"txhash": "ABC123",
					"raw_log": "some log"
				}
			}`,
			httpStatus:    200,
			expectedError: false, // code defaults to 0 (success)
		},
		{
			name: "Empty tx_response",
			response: `{
				"tx_response": {}
			}`,
			httpStatus:    200,
			expectedError: false, // code defaults to 0
		},
		{
			name: "tx_response with only code field",
			response: `{
				"tx_response": {
					"code": 19
				}
			}`,
			httpStatus:    200,
			expectedError: true, // code=19 is an error, even without raw_log
		},
		{
			name: "Nested JSON with tx_response",
			response: `{
				"data": {
					"tx_response": {
						"code": 0
					}
				}
			}`,
			httpStatus:    200,
			expectedError: false, // Nested structure won't be parsed as tx_response
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restMsg := RestMessage{}

			hasError, _ := restMsg.CheckResponseError([]byte(tc.response), tc.httpStatus)

			require.Equal(t, tc.expectedError, hasError, "Error detection mismatch for %s", tc.name)
		})
	}
}

// TestCheckResponseError_RealWorldScenarios tests real-world response examples
func TestCheckResponseError_RealWorldScenarios(t *testing.T) {
	testCases := []struct {
		name          string
		response      string
		httpStatus    int
		expectedError bool
		errorContains string
	}{
		{
			name: "Real Success Response",
			response: `{
				"tx_response": {
					"height": "8123456",
					"txhash": "5F4A8B9C7D6E1F2A3B4C5D6E7F8A9B0C1D2E3F4A5B6C7D8E9F0A1B2C3D4E5F6A",
					"codespace": "",
					"code": 0,
					"data": "0A1E0A1C2F636F736D6F732E62616E6B2E763162657461312E4D736753656E64",
					"raw_log": "[{\"events\":[{\"type\":\"coin_spent\",\"attributes\":[{\"key\":\"spender\",\"value\":\"lava1...\"},{\"key\":\"amount\",\"value\":\"1000ulava\"}]}]}]",
					"logs": [{"msg_index": 0, "log": "", "events": [{"type": "coin_spent"}]}],
					"info": "",
					"gas_wanted": "200000",
					"gas_used": "85123",
					"timestamp": "2025-10-16T14:30:00Z"
				}
			}`,
			httpStatus:    200,
			expectedError: false,
		},
		{
			name: "Real Sequence Error",
			response: `{
				"tx_response": {
					"height": "0",
					"txhash": "CB584A80FE7EA54E4CCEFCA6BE89B4FABCF4B0087766733FB8E1741ACB61EFFE",
					"codespace": "sdk",
					"code": 32,
					"data": "",
					"raw_log": "account sequence mismatch, expected 379, got 392: incorrect account sequence",
					"logs": [],
					"info": "",
					"gas_wanted": "0",
					"gas_used": "0",
					"timestamp": ""
				}
			}`,
			httpStatus:    200,
			expectedError: true,
			errorContains: "account sequence mismatch",
		},
		{
			name: "Real TX in Mempool Cache Error",
			response: `{
				"tx_response": {
					"height": "0",
					"txhash": "A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6",
					"codespace": "sdk",
					"code": 19,
					"data": "",
					"raw_log": "tx already exists in cache",
					"logs": [],
					"info": "",
					"gas_wanted": "0",
					"gas_used": "0",
					"timestamp": ""
				}
			}`,
			httpStatus:    200,
			expectedError: true,
			errorContains: "tx already exists in cache",
		},
		{
			name: "Real Invalid Transaction Error",
			response: `{
				"tx_response": {
					"height": "0",
					"txhash": "",
					"codespace": "sdk",
					"code": 18,
					"data": "",
					"raw_log": "invalid transaction: must contain at least one message",
					"logs": [],
					"info": "",
					"gas_wanted": "0",
					"gas_used": "0",
					"timestamp": ""
				}
			}`,
			httpStatus:    200,
			expectedError: true,
			errorContains: "invalid transaction",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restMsg := RestMessage{}

			hasError, errorMessage := restMsg.CheckResponseError([]byte(tc.response), tc.httpStatus)

			require.Equal(t, tc.expectedError, hasError, "Error detection mismatch for %s", tc.name)
			if tc.errorContains != "" {
				require.Contains(t, errorMessage, tc.errorContains, "Error message should contain expected text")
			}
		})
	}
}

// TestCheckResponseError_ServerErrors tests 5xx and 429 handling (Phase 4 corrections)
func TestCheckResponseError_ServerErrors(t *testing.T) {
	testCases := []struct {
		name          string
		httpStatus    int
		response      string
		expectedError bool  // Should be node error?
		errorCheck    func(t *testing.T, errorMessage string)
	}{
		{
			name:          "503 Service Unavailable with JSON error",
			httpStatus:    503,
			response:      `{"error":"service temporarily unavailable"}`,
			expectedError: true,  // Node error - triggers retry
			errorCheck: func(t *testing.T, errorMessage string) {
				require.Contains(t, errorMessage, "service temporarily unavailable")
			},
		},
		{
			name:          "503 with plain text body",
			httpStatus:    503,
			response:      `Service Unavailable`,
			expectedError: true,  // Node error even without JSON
			errorCheck: func(t *testing.T, errorMessage string) {
				require.Contains(t, errorMessage, "Service Unavailable")
			},
		},
		{
			name:          "500 Internal Server Error",
			httpStatus:    500,
			response:      `{"message":"internal server error"}`,
			expectedError: true,  // Node error
			errorCheck: func(t *testing.T, errorMessage string) {
				require.Contains(t, errorMessage, "internal server error")
			},
		},
		{
			name:          "502 Bad Gateway",
			httpStatus:    502,
			response:      `Bad Gateway`,
			expectedError: true,  // Node error
		},
		{
			name:          "429 Rate Limit Exceeded",
			httpStatus:    429,
			response:      `{"error":"rate limit exceeded"}`,
			expectedError: true,  // Node error (triggers backoff/retry)
			errorCheck: func(t *testing.T, errorMessage string) {
				require.Contains(t, errorMessage, "rate limit exceeded")
			},
		},
		{
			name:          "429 with empty body",
			httpStatus:    429,
			response:      ``,
			expectedError: true,  // Still node error
			errorCheck: func(t *testing.T, errorMessage string) {
				require.Equal(t, "HTTP 429", errorMessage)  // Fallback to status
			},
		},
		{
			name:          "404 Not Found (client error)",
			httpStatus:    404,
			response:      `{"code":5,"message":"block not found"}`,
			expectedError: false,  // NOT a node error - client error
		},
		{
			name:          "400 Bad Request (client error)",
			httpStatus:    400,
			response:      `{"error":"invalid request"}`,
			expectedError: false,  // NOT a node error
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			restMsg := RestMessage{}

			hasError, errorMessage := restMsg.CheckResponseError([]byte(tc.response), tc.httpStatus)

			require.Equal(t, tc.expectedError, hasError,
				"Expected node error=%v for HTTP %d", tc.expectedError, tc.httpStatus)
			
			if tc.errorCheck != nil {
				tc.errorCheck(t, errorMessage)
			}
		})
	}
}
