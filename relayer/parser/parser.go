package parser

import (
	spectypes "github.com/lavanet/lava/x/spec/types"
)

func Parse(blockParser spectypes.BlockParser) int64 {
	switch blockParser.ParserFunc {
	case spectypes.PARSER_FUNC_EMPTY:
		return -1
	case spectypes.PARSER_FUNC_PARSE_PARAM_BY_ARG:
		return ParseParamByArg(blockParser.ParserArg)
	default:
		return -1
	}
}

func ParseParamByArg(input []string) int64 {
	return -1
}
