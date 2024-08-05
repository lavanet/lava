package rpcInterfaceMessages

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	tenderminttypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/lavanet/lava/v2/protocol/chainlib/chainproxy/rpcclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTendermintrpcMessage(t *testing.T) {
	tendermintMessage := TendermintrpcMessage{
		JsonrpcMessage: JsonrpcMessage{
			Params: []int{1, 2, 3},
			Result: json.RawMessage(`"test_result"`),
		},
	}

	// Test GetParams method
	params := tendermintMessage.GetParams()
	if params == nil {
		t.Errorf("Expected params to be []int{1, 2, 3}, got nil")
	}

	// Test the GetResult method
	result := tendermintMessage.GetResult()
	if result == nil {
		t.Errorf("Expected result to be test_result, got nil")
	}
}

func TestTendermintrpcMessage_ParseBlock(t *testing.T) {
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

			restMessage := TendermintrpcMessage{}

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

func TestGetTendermintRPCError(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name        string
		jsonError   *rpcclient.JsonError
		expectedErr error
	}{
		{
			name:        "nil json error",
			jsonError:   nil,
			expectedErr: nil,
		},
		{
			name: "conversion to string fails",
			jsonError: &rpcclient.JsonError{
				Code:    0,
				Message: "error message",
				Data:    []int{1, 2, 3},
			},
			expectedErr: fmt.Errorf("(rpcMsg.Error.Data).(string) conversion failed {data:[1 2 3]}"),
		},
		{
			name: "conversion succeeds",
			jsonError: &rpcclient.JsonError{
				Code:    0,
				Message: "error message",
				Data:    "error data",
			},
			expectedErr: nil,
		},
	}

	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rpcError, err := GetTendermintRPCError(testCase.jsonError)
			if testCase.expectedErr != nil {
				require.Error(t, err)
				require.Nil(t, rpcError)
				require.Equal(t, testCase.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				if rpcError != nil {
					require.Equal(t, testCase.jsonError.Code, rpcError.Code)
					require.Equal(t, testCase.jsonError.Message, rpcError.Message)
					require.Equal(t, testCase.jsonError.Data, rpcError.Data)
				}
			}
		})
	}
}

func TestConvertErrorToRPCError(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name        string
		err         error
		expectedRPC *tenderminttypes.RPCError
	}{
		{
			name: "valid error string",
			err:  fmt.Errorf("test error"),
			expectedRPC: &tenderminttypes.RPCError{
				Code:    -1,
				Message: "Rpc Error",
				Data:    "test error",
			},
		},
	}

	for _, testCase := range testTable {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			errMsg := ""

			if testCase.err != nil {
				errMsg = testCase.err.Error()
			}

			result := ConvertErrorToRPCError(errMsg, -1)
			if !reflect.DeepEqual(result, testCase.expectedRPC) {
				t.Errorf("ConvertErrorToRPCError(%v) = %v, expected %v", testCase.err, result, testCase.expectedRPC)
			}
		})
	}
}

func TestIdFromRawMessage(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name           string
		rawID          json.RawMessage
		expectedResult jsonrpcId
		expectedErr    bool
	}{
		{
			name:           "Unmarshal string ID",
			rawID:          []byte(`"string-id"`),
			expectedResult: JSONRPCStringID("string-id"),
			expectedErr:    false,
		},
		{
			name:           "Unmarshal integer ID",
			rawID:          []byte(`100`),
			expectedResult: JSONRPCIntID(100),
			expectedErr:    false,
		},
		{
			name:           "Unmarshal invalid ID",
			rawID:          []byte(`{"invalid": "id"}`),
			expectedResult: nil,
			expectedErr:    true,
		},
		{
			name:           "Unmarshal json",
			rawID:          []byte(`{invalid: "id"}`),
			expectedResult: nil,
			expectedErr:    true,
		},
	}

	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			result, err := IdFromRawMessage(testCase.rawID)
			if testCase.expectedErr == false {
				assert.Equal(t, testCase.expectedResult, result)
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestConvertTendermintMsg(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name        string
		rpcMsg      *rpcclient.JsonrpcMessage
		expected    *RPCResponse
		expectErr   bool
		expectedErr string
	}{
		{
			"nil input",
			nil,
			nil,
			true,
			ErrFailedToConvertMessage.Error(),
		},
		{
			"successful conversion",
			&rpcclient.JsonrpcMessage{
				Version: "2.0",
				ID:      json.RawMessage(`"abc"`),
				Result:  json.RawMessage(`{"key":"value"}`),
				Error:   nil,
			},
			&RPCResponse{
				JSONRPC: "2.0",
				ID:      JSONRPCStringID("abc"),
				Result:  json.RawMessage(`{"key":"value"}`),
				Error:   nil,
			},
			false,
			"",
		},
		{
			"successful conversion no data in jsonrpc",
			&rpcclient.JsonrpcMessage{
				Version: "2.0",
				ID:      json.RawMessage(`"abc"`),
				Result:  json.RawMessage(`{"key":"value"}`),
				Error: &rpcclient.JsonError{
					Code:    0,
					Message: "error message",
					Data:    nil,
				},
			},
			&RPCResponse{
				JSONRPC: "2.0",
				ID:      JSONRPCStringID("abc"),
				Result:  json.RawMessage(`{"key":"value"}`),
				Error: &tenderminttypes.RPCError{
					Code:    0,
					Message: "error message",
					Data:    "",
				},
			},
			false,
			"",
		},
		{
			"error in GetTendermintRPCError",
			&rpcclient.JsonrpcMessage{
				Version: "2.0",
				ID:      json.RawMessage(`[]`),
				Result:  json.RawMessage(`{"key":"value"}`),
				Error: &rpcclient.JsonError{
					Code:    0,
					Message: "error message",
					Data:    []int{1, 2, 3},
				},
			},
			nil,
			true,
			"(rpcMsg.Error.Data).(string) conversion failed {data:[1 2 3]}",
		},
		{
			"error in IdFromRawMessage",
			&rpcclient.JsonrpcMessage{
				Version: "2.0",
				ID:      json.RawMessage(`[]`),
				Result:  json.RawMessage(`{"key":"value"}`),
				Error:   nil,
			},
			nil,
			true,
			"failed to unmarshal id not a string or float",
		},
	}
	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			res, err := ConvertTendermintMsg(testCase.rpcMsg)
			if testCase.expectErr {
				require.Error(t, err)
				fmt.Println(err)
				fmt.Println(testCase.expectedErr)
				assert.Contains(t, err.Error(), testCase.expectedErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testCase.expected, res)
			}
		})
	}
}

func TestJSONRPCID(t *testing.T) {
	stringID := JSONRPCStringID("12345")
	expected := "12345"
	if stringID.String() != expected {
		t.Errorf("Expected %q, but got %q", expected, stringID.String())
	}

	intID := JSONRPCIntID(123)
	expected = "123"
	if intID.String() != expected {
		t.Errorf("Expected %q, but got %q", expected, intID.String())
	}
}
