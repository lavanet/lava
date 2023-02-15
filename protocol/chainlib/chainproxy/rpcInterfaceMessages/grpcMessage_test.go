package rpcInterfaceMessages

import (
	"fmt"
	"testing"

	"github.com/lavanet/lava/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCMessage(t *testing.T) {
	// Test GetParams method
	restMessage := GrpcMessage{
		Path: "eth_getTransactionByHash",
		Msg:  []byte{1, 2, 3, 4, 5},
	}

	// Test GetParams method
	params := restMessage.GetParams()
	if params != nil {
		t.Errorf("Expected nil, but got %v", params)
	}

	// Test GetResult method
	result := restMessage.GetResult()
	if result != nil {
		t.Errorf("Expected nil, but got %v", result)
	}
}

func TestGRPCParseBlock(t *testing.T) {
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

			restMessage := GrpcMessage{}

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

func TestReflectionSupport(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name   string
		err    error
		result error
	}{
		{
			name:   "nil error",
			err:    nil,
			result: nil,
		},
		{
			name:   "non-unimplemented error",
			err:    fmt.Errorf("error"),
			result: fmt.Errorf("error"),
		},
		{
			name: "unimplemented error",
			err:  status.Error(codes.Unimplemented, "unimplemented"),
			result: utils.LavaFormatError("server does not support the reflection API",
				status.Error(codes.Unimplemented, "unimplemented"), nil),
		},
	}

	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			result := ReflectionSupport(testCase.err)

			if testCase.err == nil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, testCase.result.Error(), result.Error())
			}
		})
	}
}

func TestParseSymbol(t *testing.T) {
	t.Parallel()

	testTable := []struct {
		name      string
		input     string
		expectedS string
		expectedM string
	}{
		{
			name:      "Parse ServiceName/MethodName",
			input:     "ServiceName/MethodName",
			expectedS: "ServiceName",
			expectedM: "MethodName",
		},
		{
			name:      "Parse ServiceName.MethodName",
			input:     "ServiceName.MethodName",
			expectedS: "ServiceName",
			expectedM: "MethodName",
		},
		{
			name:      "Parse ServiceName",
			input:     "ServiceName",
			expectedS: "",
			expectedM: "",
		},
		{
			name:      "Parse empty string",
			input:     "",
			expectedS: "",
			expectedM: "",
		},
	}

	for _, testCase := range testTable {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			s, m := ParseSymbol(testCase.input)
			if s != testCase.expectedS {
				t.Errorf("expected %q, but got %q", testCase.expectedS, s)
			}
			if m != testCase.expectedM {
				t.Errorf("expected %q, but got %q", testCase.expectedM, m)
			}
		})
	}
}
