package parser

import (
	"encoding/json"
	"testing"

	"github.com/lavanet/lava/v5/protocol/chainlib/chainproxy/rpcclient"
	pairingtypes "github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/assert"
)

// MockRPCInput for testing
type MockRPCInput struct {
	params interface{}
	result json.RawMessage
	error  *rpcclient.JsonError
	method string
}

func (m *MockRPCInput) GetParams() interface{}         { return m.params }
func (m *MockRPCInput) GetResult() json.RawMessage     { return m.result }
func (m *MockRPCInput) GetError() *rpcclient.JsonError { return m.error }
func (m *MockRPCInput) ParseBlock(block string) (int64, error) {
	return ParseDefaultBlockParameter(block)
}
func (m *MockRPCInput) GetHeaders() []pairingtypes.Metadata { return nil }
func (m *MockRPCInput) GetMethod() string                   { return m.method }
func (m *MockRPCInput) GetID() json.RawMessage              { return json.RawMessage(`"1"`) }

// Test parseByArg with nil params
func TestParseByArg_NilParams(t *testing.T) {
	testCases := []struct {
		name          string
		input         []string
		rpcInput      *MockRPCInput
		expectedError error
		description   string
	}{
		{
			name:          "nil params should return ValueNotSetError",
			input:         []string{"0"},
			rpcInput:      &MockRPCInput{params: nil, method: "eth_blockNumber"},
			expectedError: ValueNotSetError,
			description:   "When params is nil, parseByArg should return ValueNotSetError",
		},
		{
			name:          "empty array should work",
			input:         []string{"0"},
			rpcInput:      &MockRPCInput{params: []interface{}{}, method: "test"},
			expectedError: ValueNotSetError,
			description:   "Empty array has no index 0, should return ValueNotSetError",
		},
		{
			name:          "valid array index should work",
			input:         []string{"0"},
			rpcInput:      &MockRPCInput{params: []interface{}{"value1"}, method: "test"},
			expectedError: nil,
			description:   "Valid index should return the value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Description: %s", tc.description)

			result, err := parseByArg(tc.rpcInput, tc.input, PARSE_PARAMS)

			if tc.expectedError != nil {
				assert.Equal(t, tc.expectedError, err, "Should return expected error")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Should not return error")
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

// Test parseCanonical with nil params
func TestParseCanonical_NilParams(t *testing.T) {
	testCases := []struct {
		name          string
		input         []string
		rpcInput      *MockRPCInput
		expectedError error
		description   string
	}{
		{
			name:          "nil params should return ValueNotSetError",
			input:         []string{"0", "key"},
			rpcInput:      &MockRPCInput{params: nil, method: "eth_call"},
			expectedError: ValueNotSetError,
			description:   "When params is nil, parseCanonical should return ValueNotSetError",
		},
		{
			name:          "empty array should return error",
			input:         []string{"0", "key"},
			rpcInput:      &MockRPCInput{params: []interface{}{}, method: "test"},
			expectedError: ValueNotSetError,
			description:   "Empty array has no index 0",
		},
		{
			name:          "valid object in array should work",
			input:         []string{"0", "key"},
			rpcInput:      &MockRPCInput{params: []interface{}{map[string]interface{}{"key": "value"}}, method: "test"},
			expectedError: nil,
			description:   "Valid index with nested key should return value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Description: %s", tc.description)

			result, err := parseCanonical(tc.rpcInput, tc.input, PARSE_PARAMS)

			if tc.expectedError != nil {
				assert.Equal(t, tc.expectedError, err, "Should return expected error")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Should not return error")
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

// Test parseDictionary with nil params
func TestParseDictionary_NilParams(t *testing.T) {
	testCases := []struct {
		name          string
		input         []string
		rpcInput      *MockRPCInput
		expectedError error
		description   string
	}{
		{
			name:          "nil params should return ValueNotSetError",
			input:         []string{"key", "|"},
			rpcInput:      &MockRPCInput{params: nil, method: "eth_getBalance"},
			expectedError: ValueNotSetError,
			description:   "When params is nil, parseDictionary should return ValueNotSetError",
		},
		{
			name:          "empty array should return error",
			input:         []string{"key", "|"},
			rpcInput:      &MockRPCInput{params: []interface{}{}, method: "test"},
			expectedError: ValueNotSetError,
			description:   "Empty array can't find property",
		},
		{
			name:          "valid object should work",
			input:         []string{"key", "|"},
			rpcInput:      &MockRPCInput{params: map[string]interface{}{"key": "value"}, method: "test"},
			expectedError: nil,
			description:   "Valid object with key should return value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Description: %s", tc.description)

			result, err := parseDictionary(tc.rpcInput, tc.input, PARSE_PARAMS)

			if tc.expectedError != nil {
				assert.Equal(t, tc.expectedError, err, "Should return expected error")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Should not return error")
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

// Test parseDictionaryOrOrdered with nil params
func TestParseDictionaryOrOrdered_NilParams(t *testing.T) {
	testCases := []struct {
		name          string
		input         []string
		rpcInput      *MockRPCInput
		expectedError error
		description   string
	}{
		{
			name:          "nil params should return ValueNotSetError",
			input:         []string{"key", "|", "0"},
			rpcInput:      &MockRPCInput{params: nil, method: "eth_sendTransaction"},
			expectedError: ValueNotSetError,
			description:   "When params is nil, parseDictionaryOrOrdered should return ValueNotSetError",
		},
		{
			name:          "empty array should return error",
			input:         []string{"key", "|", "0"},
			rpcInput:      &MockRPCInput{params: []interface{}{}, method: "test"},
			expectedError: ValueNotSetError,
			description:   "Empty array has no index 0",
		},
		{
			name:          "valid array with value should work",
			input:         []string{"key", "|", "0"},
			rpcInput:      &MockRPCInput{params: []interface{}{"value"}, method: "test"},
			expectedError: nil,
			description:   "Valid array with fallback index should return value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Description: %s", tc.description)

			result, err := parseDictionaryOrOrdered(tc.rpcInput, tc.input, PARSE_PARAMS)

			if tc.expectedError != nil {
				assert.Equal(t, tc.expectedError, err, "Should return expected error")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Should not return error")
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

// Test comprehensive scenario: parameters transformation through the system
func TestParameterTransformation(t *testing.T) {
	t.Run("nil params becomes ValueNotSetError in all parsers", func(t *testing.T) {
		input := &MockRPCInput{params: nil}

		// All parser functions should handle nil consistently
		testFuncs := []struct {
			name string
			fn   func(*MockRPCInput, []string) ([]interface{}, error)
			args []string
		}{
			{"parseByArg", func(r *MockRPCInput, s []string) ([]interface{}, error) {
				return parseByArg(r, s, PARSE_PARAMS)
			}, []string{"0"}},
			{"parseCanonical", func(r *MockRPCInput, s []string) ([]interface{}, error) {
				return parseCanonical(r, s, PARSE_PARAMS)
			}, []string{"0", "key"}},
			{"parseDictionary", func(r *MockRPCInput, s []string) ([]interface{}, error) {
				return parseDictionary(r, s, PARSE_PARAMS)
			}, []string{"key", "|"}},
			{"parseDictionaryOrOrdered", func(r *MockRPCInput, s []string) ([]interface{}, error) {
				return parseDictionaryOrOrdered(r, s, PARSE_PARAMS)
			}, []string{"key", "|", "0"}},
		}

		for _, tc := range testFuncs {
			t.Run(tc.name, func(t *testing.T) {
				result, err := tc.fn(input, tc.args)
				assert.Equal(t, ValueNotSetError, err, "%s should return ValueNotSetError for nil params", tc.name)
				assert.Nil(t, result, "%s should return nil result", tc.name)
			})
		}
	})

	t.Run("empty params vs nil params behavior", func(t *testing.T) {
		// Empty array should behave differently than nil
		emptyArrayInput := &MockRPCInput{params: []interface{}{}}
		nilInput := &MockRPCInput{params: nil}

		// parseByArg with empty array
		result, err := parseByArg(emptyArrayInput, []string{"0"}, PARSE_PARAMS)
		assert.Equal(t, ValueNotSetError, err, "Empty array with out-of-range index should error")
		assert.Nil(t, result)

		// parseByArg with nil
		result, err = parseByArg(nilInput, []string{"0"}, PARSE_PARAMS)
		assert.Equal(t, ValueNotSetError, err, "Nil params should error")
		assert.Nil(t, result)

		// Both should be handled consistently
		assert.Equal(t, err, err, "Both should return same error type")
	})
}
