package chainlib

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lavanet/lava/protocol/parser"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"
)

type RPCInputTest struct {
	Params         interface{}
	Result         json.RawMessage
	Headers        []pairingtypes.Metadata
	ParseBlockFunc func(block string) (int64, error)
	GetHeadersFunc func() []pairingtypes.Metadata
}

func (rpcInputTest *RPCInputTest) GetParams() interface{} {
	return rpcInputTest.Params
}
func (rpcInputTest *RPCInputTest) GetResult() json.RawMessage {
	return rpcInputTest.Result
}
func (rpcInputTest *RPCInputTest) ParseBlock(block string) (int64, error) {
	if rpcInputTest.ParseBlockFunc == nil {
		return parser.ParseDefaultBlockParameter(block)
	}
	return rpcInputTest.ParseBlockFunc(block)
}
func (rpcInputTest *RPCInputTest) GetHeaders() []pairingtypes.Metadata {
	return rpcInputTest.Headers
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
		out, err := parser.ParseDefaultBlockParameter(testCase.input)
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
			// TODO: DOES'NT WORK!!
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
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			block, err := parser.ParseBlockFromParams(&testCase.message, testCase.blockParser)
			require.NoError(t, err, fmt.Sprintf("Test case name: %s", testCase.name))
			require.Equal(t, testCase.expectedBlock, block)
		})
	}
}
