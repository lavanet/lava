package parser

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

type RPCInputTest struct {
	Params         interface{}
	Result         json.RawMessage
	Headers        []pairingtypes.Metadata
	ParseBlockFunc func(block string) (int64, error)
	GetHeadersFunc func() []pairingtypes.Metadata
}

func (rpcInputTest *RPCInputTest) GetMethod() string {
	return ""
}

func (rpcInputTest *RPCInputTest) GetParams() interface{} {
	return rpcInputTest.Params
}

func (rpcInputTest *RPCInputTest) GetResult() json.RawMessage {
	return rpcInputTest.Result
}

func (rpcInputTest *RPCInputTest) GetID() json.RawMessage {
	return nil
}

func (rpcInputTest *RPCInputTest) GetError() *rpcclient.JsonError {
	return nil
}

func (rpcInputTest *RPCInputTest) ParseBlock(block string) (int64, error) {
	if rpcInputTest.ParseBlockFunc == nil {
		return ParseDefaultBlockParameter(block)
	}
	return rpcInputTest.ParseBlockFunc(block)
}

func (rpcInputTest *RPCInputTest) GetHeaders() []pairingtypes.Metadata {
	return rpcInputTest.Headers
}

// TestAppendInterfaceToInterfaceArray tests append interface function
func TestAppendInterfaceToInterfaceArray(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []interface{}
	}{
		{
			name:     "Test with int value",
			input:    1,
			expected: []interface{}{1},
		},
		{
			name:     "Test with string value",
			input:    "hello",
			expected: []interface{}{"hello"},
		},
		{
			name:     "Test with struct value",
			input:    struct{ name string }{name: "John Doe"},
			expected: []interface{}{struct{ name string }{name: "John Doe"}},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := appendInterfaceToInterfaceArray(test.input)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Expected %v but got %v", test.expected, result)
			}
		})
	}
}

// TestParseArrayOfInterfaces tests parsing array of interfaces
func TestParseArrayOfInterfaces(t *testing.T) {
	tests := []struct {
		name     string
		data     []interface{}
		propName string
		sep      string
		expected []interface{}
	}{
		{
			name:     "Test with matching prop name",
			data:     []interface{}{"name:John Doe", "age:30", "gender:male"},
			propName: "name",
			sep:      ":",
			expected: []interface{}{"John Doe"},
		},
		{
			name:     "Test with non-matching prop name",
			data:     []interface{}{"name:John Doe", "age:30", "gender:male"},
			propName: "address",
			sep:      ":",
			expected: nil,
		},
		{
			name:     "Test with empty data array",
			data:     []interface{}{},
			propName: "name",
			sep:      ":",
			expected: nil,
		},
		{
			name:     "Test with non-string value in data array",
			data:     []interface{}{"name:John Doe", 30, "gender:male"},
			propName: "name",
			sep:      ":",
			expected: []interface{}{"John Doe"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := parseArrayOfInterfaces(test.data, test.propName, test.sep)
			if !reflect.DeepEqual(result, test.expected) {
				t.Errorf("Expected %v but got %v", test.expected, result)
			}
		})
	}
}

func TestParseResponseByEncoding(t *testing.T) {
	type data struct {
		bytes    []byte
		encoding string
	}

	testInputs := func(testData []data) {
		result0, err := parseResponseByEncoding(testData[0].bytes, testData[0].encoding)
		require.NoError(t, err)
		result1, err := parseResponseByEncoding(testData[1].bytes, testData[1].encoding)
		require.NoError(t, err)
		require.Equal(t, result0, result1)
	}
	// returned from lava blockchain rest vs tendermintrpc
	testData := []data{{bytes: []byte("9291EDC036AE254F9A6E0237F0EF13C452E7F08722E8DBD68B2F34CC8132C91D"), encoding: spectypes.EncodingHex}, {bytes: []byte("kpHtwDauJU+abgI38O8TxFLn8Ici6NvWiy80zIEyyR0="), encoding: spectypes.EncodingBase64}}
	testInputs(testData)
	// returned form evmos evm-jsonrpc vs rest
	testData = []data{{bytes: []byte("0x968ec00fd34eedc03b0577ee8116f74c75127b7d775e51c7a72519f760b821a8"), encoding: spectypes.EncodingHex}, {bytes: []byte("lo7AD9NO7cA7BXfugRb3THUSe313XlHHpyUZ92C4Iag="), encoding: spectypes.EncodingBase64}}
	testInputs(testData)
}

func TestParseBlockHappyFlow(t *testing.T) {
	testCases := []struct {
		input          string
		expectedOutput int64
		expectError    bool
	}{
		{
			input:          "latest",
			expectedOutput: spectypes.LATEST_BLOCK,
		},
		{
			input:          "earliest",
			expectedOutput: spectypes.EARLIEST_BLOCK,
		},
		{
			input:          "pending",
			expectedOutput: spectypes.PENDING_BLOCK,
		},
		{
			input:          "safe",
			expectedOutput: spectypes.SAFE_BLOCK,
		},
		{
			input:          "finalized",
			expectedOutput: spectypes.FINALIZED_BLOCK,
		},
		{
			input:          "12",
			expectedOutput: 12,
		},
		{
			input:       "-6",
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		out, err := ParseDefaultBlockParameter(testCase.input)
		if !testCase.expectError {
			require.NoError(t, err)
			require.Equal(t, testCase.expectedOutput, out)
		} else {
			require.Error(t, err)
		}
	}
}

func TestParseBlockFromParamsHappyFlow(t *testing.T) {
	testCases := []struct {
		name          string
		message       RPCInputTest
		blockParser   spectypes.BlockParser
		expectedBlock int64
	}{
		{
			name:    "NilValue",
			message: RPCInputTest{},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{},
				ParserFunc: spectypes.PARSER_FUNC_EMPTY,
			},
			expectedBlock: spectypes.NOT_APPLICABLE,
		},
		{
			name:    "DefaultParsing",
			message: RPCInputTest{},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"latest"},
				ParserFunc: spectypes.PARSER_FUNC_DEFAULT,
			},
			expectedBlock: spectypes.LATEST_BLOCK,
		},
		{
			name: "ParseByArg",
			message: RPCInputTest{
				Params: []interface{}{"1"},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"0"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_BY_ARG,
			},
			expectedBlock: 1,
		},
		{
			name: "ParseCanonical__[]interface{}__Case",
			message: RPCInputTest{
				Params: []interface{}{
					map[string]interface{}{"block": int64(6)},
					map[string]interface{}{"block": int64(25)},
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"1", "block"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_CANONICAL,
			},
			expectedBlock: 25,
		},
		{
			name: "ParseCanonical__map[string]interface{}__Case",
			message: RPCInputTest{
				Params: map[string]interface{}{
					"data": map[string]interface{}{"block": int64(1234234)},
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"0", "data", "block"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_CANONICAL,
			},
			expectedBlock: 1234234,
		},
		{
			name: "ParseDictionary__[]interface{}__Case",
			message: RPCInputTest{
				Params: []interface{}{
					"block=10000",
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"block", "="},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_DICTIONARY,
			},
			expectedBlock: 10000,
		},
		{
			name: "ParseDictionary__map[string]interface{}__Case",
			message: RPCInputTest{
				Params: map[string]interface{}{
					"block": "6",
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"block", "unnecessary"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_DICTIONARY,
			},
			expectedBlock: 6,
		},
		{
			name: "ParseDictionaryOrOrdered__[]interface{}__PropName__Case",
			message: RPCInputTest{
				Params: []interface{}{
					"block=99",
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"block", "=", "0"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED,
			},
			expectedBlock: 99,
		},
		{
			name: "ParseDictionaryOrOrdered__[]interface{}__PropIndex__Case",
			message: RPCInputTest{
				Params: []interface{}{
					"765",
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"unused", "unused", "0"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED,
			},
			expectedBlock: 765,
		},
		{
			name: "ParseDictionaryOrOrdered__map[string]interface{}__PropName__Case",
			message: RPCInputTest{
				Params: map[string]interface{}{
					"block": "101",
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"block", "unused", "0"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED,
			},
			expectedBlock: 101,
		},
		{
			name: "ParseDictionaryOrOrdered__map[string]interface{}__KeyIndex__Case",
			message: RPCInputTest{
				Params: map[string]interface{}{
					"0": "103",
				},
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"unused", "unused", "0"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED,
			},
			expectedBlock: 103,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsedInput := ParseBlockFromParams(&testCase.message, testCase.blockParser, nil)
			require.Equal(t, testCase.expectedBlock, parsedInput.GetBlock())
		})
	}
}

func TestParseBlockFromReplyHappyFlow(t *testing.T) {
	testCases := []struct {
		name          string
		message       RPCInputTest
		blockParser   spectypes.BlockParser
		expectedBlock int64
	}{
		// { // TODO: Always fails, but here for future reference
		// 	name:    "EmptyParser",
		// 	message: RPCInputTest{},
		// 	blockParser: spectypes.BlockParser{
		// 		ParserArg:  []string{},
		// 		ParserFunc: spectypes.PARSER_FUNC_EMPTY,
		// 	},
		// 	expectedBlock: spectypes.NOT_APPLICABLE,
		// },
		{
			name:    "DefaultParsing",
			message: RPCInputTest{},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"latest"},
				ParserFunc: spectypes.PARSER_FUNC_DEFAULT,
			},
			expectedBlock: spectypes.LATEST_BLOCK,
		},
		{
			name: "ParseByArg",
			message: RPCInputTest{
				Result: []byte("1"),
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"0"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_BY_ARG,
			},
			expectedBlock: 1,
		},
		{
			name: "ParseCanonical",
			message: RPCInputTest{
				Result: []byte(
					"{\"block\" : 25}",
				),
			},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"0", "block"},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_CANONICAL,
			},
			expectedBlock: 25,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			parsedInput := ParseBlockFromReply(&testCase.message, testCase.blockParser, nil)
			require.Equal(t, testCase.expectedBlock, parsedInput.GetBlock())
		})
	}
}

func TestParseBlockFromParams(t *testing.T) {
	tests := []struct {
		name           string
		rpcInput       RPCInput
		blockParser    spectypes.BlockParser
		genericParsers []spectypes.GenericParser
		expected       int64
	}{
		{
			name: "generic_parser_happy_flow_default_value",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": []interface{}{
							map[string]interface{}{
								"baz": 123,
							},
						},
					},
				},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".params.foo.bar.[0].baz",
					Rule:      "=123",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.LATEST_BLOCK,
		},
		{
			name: "generic_parser_happy_flow_value",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{
					"foo": map[string]interface{}{
						"bar": []interface{}{
							map[string]interface{}{
								"baz": 123,
							},
						},
					},
				},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".params.foo.bar.[0].baz",
					ParseType: spectypes.PARSER_TYPE_BLOCK_LATEST,
				},
			},
			expected: 123,
		},
		{
			name:     "generic_parser_nil_params",
			rpcInput: &RPCInputTest{},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".params.foo",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.NOT_APPLICABLE,
		},
		{
			name: "generic_parser_fail_with_nil_var",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{
					"bar": 123,
				},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".params.foo",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.NOT_APPLICABLE,
		},
		{
			name: "generic_parser_fail_with_iter_error",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{
					"bar": 123,
				},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".params.bar.foo",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.NOT_APPLICABLE,
		},
		{
			name: "generic_parser_wrong_jq_path_no_default",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{
					"bar": 123,
				},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: "!@#$%^&*()",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.NOT_APPLICABLE,
		},
		{
			name: "generic_parser_wrong_jq_path_with_parser_func_default",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{
					"bar": 123,
				},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: "!@#$%^&*()",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			blockParser: spectypes.BlockParser{
				ParserFunc: spectypes.PARSER_FUNC_DEFAULT,
				ParserArg:  []string{"latest"},
			},
			expected: spectypes.LATEST_BLOCK,
		},
		{
			name: "generic_parser_and_block_parser_fail",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: "!@#$%^&*()",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			blockParser: spectypes.BlockParser{
				ParserFunc:   spectypes.PARSER_FUNC_PARSE_CANONICAL,
				ParserArg:    []string{"0", "block"},
				DefaultValue: "latest",
			},
			expected: spectypes.LATEST_BLOCK,
		},

		{
			name: "generic_parser_no_generic_parser",
			rpcInput: &RPCInputTest{
				Params: []interface{}{
					"block=10000",
				},
			},
			genericParsers: []spectypes.GenericParser{},
			blockParser: spectypes.BlockParser{
				ParserArg:  []string{"block", "="},
				ParserFunc: spectypes.PARSER_FUNC_PARSE_DICTIONARY,
			},
			expected: 10000,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := ParseBlockFromParams(test.rpcInput, test.blockParser, test.genericParsers)
			require.Equal(t, test.expected, result.GetBlock())
		})
	}
}

func TestParseBlockFromReply(t *testing.T) {
	tests := []struct {
		name           string
		rpcInput       RPCInput
		blockParser    spectypes.BlockParser
		genericParsers []spectypes.GenericParser
		expected       int64
		expectedError  string
	}{
		{
			name: "generic_parser_happy_flow_default_value",
			rpcInput: &RPCInputTest{
				Result: []byte(`
					{
						"foo": {
							"bar": [
								{
									"baz": 123
								}
							]
						}
					}
				`),
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".result.foo.bar.[0].baz",
					Rule:      "=123",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.LATEST_BLOCK,
		},
		{
			name: "generic_parser_happy_flow_value",
			rpcInput: &RPCInputTest{
				Result: []byte(`
					{
						"foo": {
							"bar": [
								{
									"baz": 123
								}
							]
						}
					}
				`),
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".result.foo.bar.[0].baz",
					ParseType: spectypes.PARSER_TYPE_BLOCK_LATEST,
				},
			},
			expected: 123,
		},
		{
			name:     "generic_parser_nil_params",
			rpcInput: &RPCInputTest{},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".result.foo",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.NOT_APPLICABLE,
		},
		{
			name: "generic_parser_fail_with_nil_var",
			rpcInput: &RPCInputTest{
				Result: []byte(`
					{
						"bar": 123
					}
				`),
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".result.foo",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.NOT_APPLICABLE,
		},
		{
			name: "generic_parser_fail_with_iter_error",
			rpcInput: &RPCInputTest{
				Result: []byte(`
					{
						"bar": 123
					}
				`),
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".result.bar.foo",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			expected: spectypes.NOT_APPLICABLE,
		},
		{
			name: "generic_parser_wrong_jq_path_with_parser_func_default",
			rpcInput: &RPCInputTest{
				Params: map[string]interface{}{
					"bar": 123,
				},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: "!@#$%^&*()",
					Value:     "latest",
					ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
				},
			},
			blockParser: spectypes.BlockParser{
				ParserFunc: spectypes.PARSER_FUNC_DEFAULT,
				ParserArg:  []string{"latest"},
			},
			expected: spectypes.LATEST_BLOCK,
		},
		{
			name: "generic_parser_parse_from_result_happy_flow",
			rpcInput: &RPCInputTest{
				Result: []byte(`
					{
						"foo": {
							"bar": 123
						}
					}
				`),
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".result.foo.bar",
					Value:     "123",
					ParseType: spectypes.PARSER_TYPE_RESULT,
				},
			},
			expected: 123,
		},
		{
			name: "generic_parser_parse_from_result_error",
			rpcInput: &RPCInputTest{
				Result: []byte(`
					{
						"foo": {
							"bar": 123
						}
					}
				`),
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".result.foo.bar",
					Value:     "321",
					ParseType: spectypes.PARSER_TYPE_RESULT,
				},
			},
			expected:      123,
			expectedError: "expected 321, received 123",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			parsedInput := ParseBlockFromReply(test.rpcInput, test.blockParser, test.genericParsers)
			if test.expectedError != "" {
				require.Equal(t, test.expectedError, parsedInput.GetParserError())
			} else {
				require.Empty(t, parsedInput.GetParserError())
			}
			require.Equal(t, test.expected, parsedInput.GetBlock())
		})
	}
}

func TestParseBlockFromParamsHash(t *testing.T) {
	tests := []struct {
		name           string
		rpcInput       RPCInput
		blockParser    spectypes.BlockParser
		genericParsers []spectypes.GenericParser
		expected       int64
		expectedHash   string
	}{
		{
			name: "generic_parser_hash",
			rpcInput: &RPCInputTest{
				Params: []interface{}{"a3f1c5e6b8d3e7f9c9d3b7c5e2f9d6e7f9c9a3b7c8d3e7f9c9d3b7c5e2f9d6e7"},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".params.[0]",
					ParseType: spectypes.PARSER_TYPE_BLOCK_HASH,
				},
			},
			blockParser: spectypes.BlockParser{
				ParserFunc:   spectypes.PARSER_FUNC_PARSE_CANONICAL,
				ParserArg:    []string{"0", "block"},
				DefaultValue: "latest",
			},
			expected:     spectypes.LATEST_BLOCK,
			expectedHash: "a3f1c5e6b8d3e7f9c9d3b7c5e2f9d6e7f9c9a3b7c8d3e7f9c9d3b7c5e2f9d6e7",
		},
		{
			name: "generic_parser_hash_bad_hash",
			rpcInput: &RPCInputTest{
				Params: []interface{}{"a3f1c5e6b8"},
			},
			genericParsers: []spectypes.GenericParser{
				{
					ParsePath: ".params.[0]",
					ParseType: spectypes.PARSER_TYPE_BLOCK_HASH,
				},
			},
			blockParser: spectypes.BlockParser{
				ParserFunc:   spectypes.PARSER_FUNC_PARSE_CANONICAL,
				ParserArg:    []string{"0", "block"},
				DefaultValue: "latest",
			},
			expected:     spectypes.LATEST_BLOCK,
			expectedHash: "",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			// t.Parallel()
			parsedInput := ParseBlockFromParams(test.rpcInput, test.blockParser, test.genericParsers)
			parsedHashes, err := parsedInput.GetBlockHashes()
			if test.expectedHash == "" {
				require.Error(t, err)
				require.Len(t, parsedHashes, 0)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedHash, parsedHashes[0])
			}
			require.Equal(t, test.expected, parsedInput.GetBlock())
		})
	}
}

func TestParseRule(t *testing.T) {
	tests := []struct {
		rule     string
		value    string
		expected bool
	}{
		{"=final || =optimistic", "final", true},
		{"=final || =optimistic", "optimistic", true},
		{"=final || =optimistic", "pessimistic", false},
		{"=final", "final", true},
		{"=final", "notfinal", false},
		{"=final || =optimistic || =neutral", "neutral", true},
		{"=final || =optimistic", "finale", false},
	}

	for _, test := range tests {
		result := parseRule(test.rule, test.value)
		if result != test.expected {
			t.Errorf("parseRule(%q, %q) = %v; expected %v", test.rule, test.value, result, test.expected)
		}
	}
}

func TestParseGeneric(t *testing.T) {
	jsonMap := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "nil",
		"method":  "block",
		"params": map[string]interface{}{
			"finality": "final",
		},
	}
	parsedInput, err := parseGeneric(jsonMap, spectypes.GenericParser{
		ParsePath: ".params.finality",
		Value:     "latest",
		ParseType: spectypes.PARSER_TYPE_DEFAULT_VALUE,
		Rule:      "=final || =optimistic",
	})
	require.NoError(t, err)
	require.Equal(t, "latest", parsedInput.GetRawParsedData())
}

func TestHashLengthValidation(t *testing.T) {
	_, err := parseGenericParserBlockHash("123")
	require.Error(t, err)
	_, err = parseGenericParserBlockHash("123456789,123456789,123456789,12")
	require.NoError(t, err)
}

func TestParseRawBlock(t *testing.T) {
	defaultValue := "defaultValue"
	defaultBlock := int64(1)
	rawBlock := "123"
	expectedBlock := int64(123)

	t.Run("with raw block parsed", func(t *testing.T) {
		parsedInput := &ParsedInput{
			parsedDataRaw: rawBlock,
		}

		rpcInput := RPCInputTest{
			ParseBlockFunc: func(block string) (int64, error) {
				require.Equal(t, parsedInput.parsedDataRaw, block)
				return expectedBlock, nil
			},
		}

		ParseRawBlock(&rpcInput, parsedInput, defaultValue)
		require.Equal(t, expectedBlock, parsedInput.GetBlock())
	})

	t.Run("without raw block parsed, with default value", func(t *testing.T) {
		parsedInput := &ParsedInput{}

		rpcInput := RPCInputTest{
			ParseBlockFunc: func(block string) (int64, error) {
				require.Equal(t, defaultValue, block)
				return defaultBlock, nil
			},
		}

		ParseRawBlock(&rpcInput, parsedInput, defaultValue)
		require.Equal(t, defaultBlock, parsedInput.GetBlock())
	})

	t.Run("without raw block parsed, with default value parse error", func(t *testing.T) {
		parsedInput := &ParsedInput{}

		rpcInput := RPCInputTest{
			ParseBlockFunc: func(block string) (int64, error) {
				require.Equal(t, defaultValue, block)
				return 0, fmt.Errorf("parse error")
			},
		}

		ParseRawBlock(&rpcInput, parsedInput, defaultValue)
		require.Equal(t, spectypes.NOT_APPLICABLE, parsedInput.GetBlock())
	})

	t.Run("without raw block parsed, without default value", func(t *testing.T) {
		parsedInput := &ParsedInput{}

		rpcInput := RPCInputTest{
			ParseBlockFunc: func(block string) (int64, error) {
				require.Fail(t, "should not be called")
				return 0, nil
			},
		}

		ParseRawBlock(&rpcInput, parsedInput, "")
		require.Equal(t, spectypes.NOT_APPLICABLE, parsedInput.GetBlock())
	})
}
