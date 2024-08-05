package rpcInterfaceMessages

import (
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
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
		testCase := testCase

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
