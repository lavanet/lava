package parser

import (
	"fmt"
	"strconv"

	spectypes "github.com/lavanet/lava/x/spec/types"
)

const NOT_APPLICABLE int64 = -1
const LATEST_BLOCK int64 = -2
const EARLIEST_BLOCK int64 = -3
const PENDING_BLOCK int64 = -4

type RPCInput interface {
	GetParams() []interface{}
	ParseBlock(block string) (int64, error)
}

func ParseDefaultBlockParameter(block string) (int64, error) {
	switch block {
	case "latest":
		return LATEST_BLOCK, nil
	case "earliest":
		return EARLIEST_BLOCK, nil
	case "pending":
		return PENDING_BLOCK, nil
	default:
		//try to parse a number
	}
	blockNum, err := strconv.ParseInt(block, 0, 64)
	if err != nil {
		return NOT_APPLICABLE, fmt.Errorf("invalid block value, could not parse block %s, error: %s", block, err)
	}
	if blockNum < 0 {
		return NOT_APPLICABLE, fmt.Errorf("invalid block value, block value was negative %d", blockNum)
	}
	return blockNum, nil
}

//this function returns the block that was requested,
func Parse(rpcInput RPCInput, blockParser spectypes.BlockParser) (int64, error) {
	switch blockParser.ParserFunc {
	case spectypes.PARSER_FUNC_EMPTY:
		return NOT_APPLICABLE, nil
	case spectypes.PARSER_FUNC_PARSE_PARAM_BY_ARG:
		return ParseParamByArg(rpcInput, blockParser.ParserArg)
	case spectypes.PARSER_FUNC_PARSE_PARAM_CANONICAL:
		return ParseParamCanonical(rpcInput, blockParser.ParserArg)
	default:
		return NOT_APPLICABLE, fmt.Errorf("unsupported block parser parserFunc")
	}
}

func ParseParamByArg(rpcInput RPCInput, input []string) (int64, error) {
	//specified block is one of the direct parameters, input should be one string defining the location of the block
	if len(input) != 1 {
		return NOT_APPLICABLE, fmt.Errorf("invalid input format, input length: %d", len(input))
	}
	inp := input[0]
	param_index, err := strconv.ParseUint(inp, 10, 32)
	if err != nil {
		return NOT_APPLICABLE, fmt.Errorf("invalid input format, input isn't an unsigned index: %s, error: %s", inp, err)
	}
	params := rpcInput.GetParams()
	if uint64(len(params)) < param_index {
		return NOT_APPLICABLE, fmt.Errorf("invalid rpc input and input index: wanted param: %d params: %s", param_index, params)
	}
	block := params[param_index]

	return rpcInput.ParseBlock(fmt.Sprintf("%s", block))
}

func ParseParamCanonical(rpcInput RPCInput, input []string) (int64, error) {
	if len(input) != 2 {
		return NOT_APPLICABLE, fmt.Errorf("invalid input format, input length: %d and needs to be 2", len(input))
	}
	inp := input[0]
	param_index, err := strconv.ParseUint(inp, 10, 32)
	if err != nil {
		return NOT_APPLICABLE, fmt.Errorf("invalid input format, input isn't an unsigned index: %s, error: %s", inp, err)
	}
	params := rpcInput.GetParams()
	if uint64(len(params)) < param_index {
		return NOT_APPLICABLE, fmt.Errorf("invalid rpc input and input index: wanted param: %d params: %s", param_index, params)
	}
	blockContainer := params[param_index]
	if container, ok := blockContainer.(map[string]interface{}); ok {
		//TODO: add default
		return rpcInput.ParseBlock(fmt.Sprintf("%s", container[input[1]]))
	}
	return NOT_APPLICABLE, fmt.Errorf("invalid input format, blockContainer is %s and tried to get a field inside: %s", blockContainer, input)
}
