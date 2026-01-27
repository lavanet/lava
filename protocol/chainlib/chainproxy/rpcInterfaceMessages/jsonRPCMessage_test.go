package rpcInterfaceMessages

import (
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertJsonRPCMsg_Success(t *testing.T) {
	rpcMsg := &rpcclient.JsonrpcMessage{
		Version: "2.0",
		ID:      json.RawMessage(`"1"`),
		Method:  "test",
		Params:  json.RawMessage(`"test_params"`),
		Error:   nil,
		Result:  json.RawMessage(`"test_result"`),
	}

	msg, err := ConvertJsonRPCMsg(rpcMsg)
	assert.NoError(t, err)
	assert.Equal(t, "2.0", msg.Version)
	assert.Equal(t, json.RawMessage(`"1"`), msg.ID)
	assert.Equal(t, "test", msg.Method)
	assert.Equal(t, json.RawMessage(`"test_params"`), msg.Params)
	assert.Nil(t, msg.Error)
	assert.Equal(t, json.RawMessage(`"test_result"`), msg.Result)
}

func TestConvertJsonRPCMsg_Nil(t *testing.T) {
	msg, err := ConvertJsonRPCMsg(nil)
	assert.EqualError(t, err, ErrFailedToConvertMessage.Error())
	assert.Nil(t, msg)
}

func TestJsonrpcMessage_GetParams(t *testing.T) {
	cp := JsonrpcMessage{
		Params: "test_params",
	}

	assert.Equal(t, "test_params", cp.GetParams())
}

func TestJsonrpcMessage_GetResult(t *testing.T) {
	cp := JsonrpcMessage{
		Result: json.RawMessage(`"test_result"`),
	}

	assert.Equal(t, json.RawMessage(`"test_result"`), cp.GetResult())
}

func TestJsonrpcMessage_ParseBlock(t *testing.T) {
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

			restMessage := JsonrpcMessage{}

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

func TestParseJsonRPCMsg(t *testing.T) {
	// Test Case 1: Valid JSON input
	data := []byte(`{"jsonrpc": "2.0", "id": 1, "method": "getblock", "params": [], "result": {"block": "block data"}}`)
	msgs, err := ParseJsonRPCMsg(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	require.Len(t, msgs, 1)
	msg := msgs[0]
	if msg.Version != "2.0" {
		t.Errorf("Expected msg.Version to be 2.0, but got %s", msg.Version)
	}
	if msg.Method != "getblock" {
		t.Errorf("Expected msg.Method to be getblock, but got %s", msg.Method)
	}

	// Test Case 2: Invalid JSON input
	data = []byte(`{"jsonrpc": "2.0", "id": 1, "method": "getblock", "params": []`)
	_, err = ParseJsonRPCMsg(data)
	if err == nil {
		t.Errorf("Expected error, but got nil")
	}
}

func TestParseJsonRPCMissingId(t *testing.T) {
	// Test Case 1: Valid JSON input
	data := []byte(`{"jsonrpc": "2.0", "id": nil, "method": "getblock", "params": []}`)
	_, err := ParseJsonRPCMsg(data)
	require.Error(t, err, err)

	data = []byte(`{"jsonrpc": "2.0", "method": "getblock", "params": []}`)
	msg, err := ParseJsonRPCMsg(data)
	require.NoError(t, err)
	require.Equal(t, json.RawMessage([]byte("null")), msg[0].ID)
}

func TestParseJsonRPCBatch(t *testing.T) {
	// Test Case 1: Valid JSON input
	data := []byte(`[{"method":"eth_chainId","params":[],"id":1,"jsonrpc":"2.0"},{"method":"eth_accounts","params":[],"id":2,"jsonrpc":"2.0"},{"method":"eth_blockNumber","params":[],"id":3,"jsonrpc":"2.0"}]`)
	msgs, err := ParseJsonRPCMsg(data)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	values := []string{"eth_chainId", "eth_accounts", "eth_blockNumber"}
	for idx, msg := range msgs {
		if msg.Version != "2.0" {
			t.Errorf("Expected msg.Version to be 2.0, but got %s", msg.Version)
		}
		require.Equal(t, values[idx], msg.Method, "Expected msg.Method to be %s, but got %s", values[idx], msg.Method)

		// Test Case 2: Invalid JSON input
		data = []byte(`{"jsonrpc": "2.0", "id": 1, "method": "getblock", "params": []`)
		_, err = ParseJsonRPCMsg(data)
		if err == nil {
			t.Errorf("Expected error, but got nil")
		}
	}
}

func TestCheckResponseErrorForJsonRpcBatch(t *testing.T) {
	t.Run("all_success_no_error", func(t *testing.T) {
		// All sub-requests succeeded
		data := []byte(`[
			{"jsonrpc":"2.0","id":1,"result":{"blockNumber":"0x123"}},
			{"jsonrpc":"2.0","id":2,"result":{"blockNumber":"0x124"}},
			{"jsonrpc":"2.0","id":3,"result":{"blockNumber":"0x125"}}
		]`)
		hasError, errorMsg := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.False(t, hasError, "All success should not have error")
		require.Empty(t, errorMsg)
	})

	t.Run("all_errors_should_return_error", func(t *testing.T) {
		// All sub-requests failed
		data := []byte(`[
			{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Slot was skipped"}},
			{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"Slot was skipped"}},
			{"jsonrpc":"2.0","id":3,"error":{"code":-32000,"message":"Slot was skipped"}}
		]`)
		hasError, errorMsg := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.True(t, hasError, "All errors should return error")
		require.Contains(t, errorMsg, "Slot was skipped")
	})

	t.Run("partial_success_should_not_error", func(t *testing.T) {
		// Some sub-requests succeeded, some failed - should NOT be considered an error
		data := []byte(`[
			{"jsonrpc":"2.0","id":1,"result":{"blockNumber":"0x123"}},
			{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"Slot was skipped"}},
			{"jsonrpc":"2.0","id":3,"result":{"blockNumber":"0x125"}}
		]`)
		hasError, errorMsg := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.False(t, hasError, "Partial success should not be considered an error")
		require.Empty(t, errorMsg)
	})

	t.Run("single_success_among_errors_should_not_error", func(t *testing.T) {
		// Only one success among multiple errors - should NOT be considered an error
		data := []byte(`[
			{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Slot was skipped"}},
			{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"Slot was skipped"}},
			{"jsonrpc":"2.0","id":3,"result":{"blockNumber":"0x125"}}
		]`)
		hasError, errorMsg := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.False(t, hasError, "Single success should prevent error classification")
		require.Empty(t, errorMsg)
	})

	t.Run("empty_batch_no_error", func(t *testing.T) {
		data := []byte(`[]`)
		hasError, errorMsg := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.False(t, hasError)
		require.Empty(t, errorMsg)
	})

	t.Run("invalid_json_no_error", func(t *testing.T) {
		data := []byte(`not valid json`)
		hasError, errorMsg := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.False(t, hasError, "Invalid JSON should not return error (fail-open)")
		require.Empty(t, errorMsg)
	})

	t.Run("null_result_with_no_error_is_success", func(t *testing.T) {
		// null result without error is still a valid response (API returned successfully)
		data := []byte(`[
			{"jsonrpc":"2.0","id":1,"result":null},
			{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"Some error"}}
		]`)
		// null result is a valid JSON-RPC response - it means the method executed successfully
		// and returned null. This counts as a success.
		hasError, _ := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.False(t, hasError, "null result is a valid success response")
	})

	t.Run("real_world_solana_batch_partial_error", func(t *testing.T) {
		// Real-world scenario: Solana batch request where some slots are skipped
		data := []byte(`[
			{"jsonrpc":"2.0","id":1,"result":{"slot":123,"blockhash":"abc"}},
			{"jsonrpc":"2.0","id":2,"error":{"code":-32007,"message":"Slot 124 was skipped, or missing in long-term storage"}},
			{"jsonrpc":"2.0","id":3,"result":{"slot":125,"blockhash":"def"}}
		]`)
		hasError, errorMsg := CheckResponseErrorForJsonRpcBatch(data, 200)
		require.False(t, hasError, "Solana batch with partial skipped slots should not trigger retry")
		require.Empty(t, errorMsg)
	})
}
