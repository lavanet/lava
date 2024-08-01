package parser

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/goccy/go-json"
	"github.com/itchyny/gojq"

	sdkerrors "cosmossdk.io/errors"
	"github.com/lavanet/lava/v2/utils"
	pairingtypes "github.com/lavanet/lava/v2/x/pairing/types"
	spectypes "github.com/lavanet/lava/v2/x/spec/types"
)

const (
	PARSE_PARAMS = 0
	PARSE_RESULT = 1
	debug        = false
)

var ValueNotSetError = sdkerrors.New("Value Not Set ", 6662, "when trying to parse, the value that we attempted to parse did not exist")

type RPCInput interface {
	GetParams() interface{}
	GetResult() json.RawMessage
	ParseBlock(block string) (int64, error)
	GetHeaders() []pairingtypes.Metadata
	GetMethod() string
}

func ParseDefaultBlockParameter(block string) (int64, error) {
	switch block {
	case "latest":
		return spectypes.LATEST_BLOCK, nil
	case "earliest":
		return spectypes.EARLIEST_BLOCK, nil
	case "pending":
		return spectypes.PENDING_BLOCK, nil
	case "safe":
		return spectypes.SAFE_BLOCK, nil
	case "finalized":
		return spectypes.FINALIZED_BLOCK, nil
	default:
		// try to parse a number
	}

	hashNoPrefix, found := strings.CutPrefix(block, "0x")
	if len(block) >= 64 && found {
		if len(hashNoPrefix)%64 == 0 {
			// this is a hash, and we can parse it
			return spectypes.LATEST_BLOCK, nil
		}
	}

	blockNum, err := strconv.ParseInt(block, 0, 64)
	if err != nil {
		return spectypes.NOT_APPLICABLE, fmt.Errorf("invalid block value, could not parse block %s, error: %s", block, err)
	}
	if blockNum < 0 {
		return spectypes.NOT_APPLICABLE, fmt.Errorf("invalid block value, block value was negative %d", blockNum)
	}
	return blockNum, nil
}

func parseInput(rpcInput RPCInput, blockParser spectypes.BlockParser, genericParsers []spectypes.GenericParser, dataSource int) ([]interface{}, error) {
	// first we try to parse the value with generic parsers, if they exist
	if len(genericParsers) > 0 {
		genericParserResult, genericParserErr := parseWithGenericParsers(rpcInput, genericParsers)
		if genericParserErr == nil {
			return genericParserResult, nil
		}
	}

	// first we try to parse the value with the block parser
	result, err := parse(rpcInput, blockParser, dataSource)
	if err == nil && result != nil {
		return result, nil
	}

	if ValueNotSetError.Is(err) && blockParser.DefaultValue != "" {
		return appendInterfaceToInterfaceArray(blockParser.DefaultValue), nil
	}

	return nil, err
}

// this function returns the block that was requested,
func ParseBlockFromParams(rpcInput RPCInput, blockParser spectypes.BlockParser, genericParsers []spectypes.GenericParser) (int64, error) {
	result, err := parseInput(rpcInput, blockParser, genericParsers, PARSE_PARAMS)
	if err != nil || result == nil {
		return spectypes.NOT_APPLICABLE, err
	}

	resString, ok := result[0].(string)
	if !ok {
		return spectypes.NOT_APPLICABLE, fmt.Errorf("ParseBlockFromParams - result[0].(string) - type assertion failed, type:" + fmt.Sprintf("%s", result[0]))
	}
	parsedBlock, err := rpcInput.ParseBlock(resString)
	if err != nil {
		if blockParser.DefaultValue != "" {
			utils.LavaFormatDebug("Failed parsing block from string, assuming default value",
				utils.LogAttr("params", rpcInput.GetParams()),
				utils.LogAttr("failed_parsed_value", resString),
				utils.LogAttr("default_value", blockParser.DefaultValue))
			return rpcInput.ParseBlock(blockParser.DefaultValue)
		}
	}
	return parsedBlock, err
}

// This returns the parsed response without decoding
func ParseFromReply(rpcInput RPCInput, blockParser spectypes.BlockParser) (string, error) {
	result, err := parse(rpcInput, blockParser, PARSE_RESULT)
	if err != nil || result == nil {
		return "", err
	}

	response, ok := result[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)
	if !ok {
		return "", utils.LavaFormatError("Failed to Convert blockData[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)", nil, utils.Attribute{Key: "blockData", Value: response[spectypes.DEFAULT_PARSED_RESULT_INDEX]})
	}

	if strings.Contains(response, "\"") {
		responseUnquoted, err := strconv.Unquote(response)
		if err != nil {
			return response, nil
		}
		return responseUnquoted, nil
	}

	return response, nil
}

func ParseBlockFromReply(rpcInput RPCInput, blockParser spectypes.BlockParser) (int64, error) {
	result, err := ParseFromReply(rpcInput, blockParser)
	if err != nil {
		return spectypes.NOT_APPLICABLE, err
	}
	return rpcInput.ParseBlock(result)
}

// This returns the parsed response after decoding
func ParseFromReplyAndDecode(rpcInput RPCInput, resultParser spectypes.BlockParser) (string, error) {
	response, err := ParseFromReply(rpcInput, resultParser)
	if err != nil {
		return "", err
	}
	return parseResponseByEncoding([]byte(response), resultParser.Encoding)
}

func parse(rpcInput RPCInput, blockParser spectypes.BlockParser, dataSource int) ([]interface{}, error) {
	var retval []interface{}
	var err error

	switch blockParser.ParserFunc {
	case spectypes.PARSER_FUNC_EMPTY:
		return nil, nil
	case spectypes.PARSER_FUNC_PARSE_BY_ARG:
		retval, err = parseByArg(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_CANONICAL:
		retval, err = parseCanonical(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_DICTIONARY:
		retval, err = parseDictionary(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED:
		retval, err = parseDictionaryOrOrdered(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_DEFAULT:
		retval = parseDefault(blockParser.ParserArg)
	default:
		return nil, fmt.Errorf("unsupported block parser parserFunc")
	}

	if err != nil {
		return nil, err
	}

	if debug {
		utils.LavaFormatDebug("parsed block:", utils.LogAttr("retval", retval))
	}

	return retval, nil
}

func parseWithGenericParsers(rpcInput RPCInput, genericParsers []spectypes.GenericParser) ([]interface{}, error) {
	if len(genericParsers) == 0 {
		return nil, fmt.Errorf("no generic parsers to use")
	}

	params := rpcInput.GetParams()
	if params == nil {
		return nil, utils.LavaFormatTrace("rpcInput params are nil", utils.LogAttr("rpcInput", rpcInput))
	}

	// We try to parse the params with all the generic parsers, the first one that succeeds, its value is returned
	for _, genericParser := range genericParsers {
		retval, err := parseGeneric(params, genericParser)
		if err == nil {
			return retval, nil
		}
	}

	// we didn't find a generic parser that worked
	return nil, utils.LavaFormatTrace("failed to parse with generic parsers",
		utils.LogAttr("params", rpcInput.GetParams()),
		utils.LogAttr("jsonParams", params),
		utils.LogAttr("genericParsers", genericParsers),
	)
}

func parseGeneric(input interface{}, genericParser spectypes.GenericParser) ([]interface{}, error) {
	value, err := findGenericParserValue(input, genericParser)
	if err != nil {
		return nil, err
	}

	utils.LavaFormatTrace("parsed generic value",
		utils.LogAttr("input", input),
		utils.LogAttr("genericParser", genericParser),
		utils.LogAttr("value", value),
	)

	switch genericParser.ParseType {
	case spectypes.PARSER_TYPE_BLOCK_LATEST:
		return appendInterfaceToInterfaceArray("latest"), nil
	default:
		return nil, fmt.Errorf("unsupported generic parser type")
	}
}

func findGenericParserValue(input interface{}, genericParser spectypes.GenericParser) (interface{}, error) {
	jqParser, err := gojq.Parse(genericParser.GetParsePath())
	if err != nil {
		return "", utils.LavaFormatError("failed to parse generic parser path", err, utils.LogAttr("path", genericParser.GetParsePath()))
	}

	iter := jqParser.Run(input)
	for {
		val, ok := iter.Next()
		if !ok {
			break
		}

		if val == nil {
			continue
		}

		if err, ok := val.(error); ok {
			utils.LavaFormatTrace("generic parser path returned an error",
				utils.LogAttr("error", err),
				utils.LogAttr("input", input),
				utils.LogAttr("path", genericParser.GetParsePath()),
			)

			continue
		}

		return val, nil
	}

	return "", utils.LavaFormatTrace("generic parser path did not return the expected value",
		utils.LogAttr("input", input),
		utils.LogAttr("path", genericParser.GetParsePath()),
	)
}

func parseDefault(input []string) []interface{} {
	retArr := make([]interface{}, 0)
	retArr = append(retArr, input[0])
	return retArr
}

// align hash encoding to base64 string, to save up on space and allow comparisons
func parseResponseByEncoding(rawResult []byte, encoding string) (string, error) {
	switch encoding {
	case spectypes.EncodingBase64:
		return string(rawResult), nil
	case spectypes.EncodingHex:
		hexString := strings.TrimPrefix(string(rawResult), "0x") // some protocols return 0x in their hex responses
		if len(hexString)%2 != 0 {
			// some hashes are hex but can't be encoded as base 64 without passing
			hexString = "0" + hexString
		}
		hexBytes, err := hex.DecodeString(hexString)
		if err != nil {
			return "", utils.LavaFormatError("tried decoding a hex response in parseResponseByEncoding but failed", err, utils.Attribute{Key: "data", Value: hexString})
		}
		return base64.StdEncoding.EncodeToString(hexBytes), nil
	default:
		return string(rawResult), nil
	}
}

// Move to RPCInput
func getDataToParse(rpcInput RPCInput, dataSource int) (interface{}, error) {
	switch dataSource {
	case PARSE_PARAMS:
		return rpcInput.GetParams(), nil
	case PARSE_RESULT:
		interfaceArr := []interface{}{}
		var data map[string]interface{}
		unmarshalled := rpcInput.GetResult()
		if len(unmarshalled) == 0 {
			return nil, fmt.Errorf("GetDataToParse failure Get.Result is empty")
		}
		// Try to unmarshal and if the data is unmarshalable then return the data itself
		err := json.Unmarshal(unmarshalled, &data)
		if err != nil {
			interfaceArr = append(interfaceArr, unmarshalled)
		} else {
			interfaceArr = append(interfaceArr, data)
		}

		return interfaceArr, nil
	default:
		return nil, fmt.Errorf("unsupported block parser parserFunc")
	}
}

func blockInterfaceToString(block interface{}) string {
	switch castedBlock := block.(type) {
	case string:
		return castedBlock
	case float64:
		return strconv.FormatFloat(castedBlock, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(castedBlock, 10)
	case uint64:
		return strconv.FormatUint(castedBlock, 10)
	default:
		return fmt.Sprintf("%s", block)
	}
}

func parseByArg(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	// specified block is one of the direct parameters, input should be one string defining the location of the block
	if len(input) != 1 {
		return nil, utils.LavaFormatProduction("invalid input format, input length", nil, utils.LogAttr("input_len", strconv.Itoa(len(input))))
	}
	inp := input[0]
	param_index, err := strconv.ParseUint(inp, 10, 32)
	if err != nil {
		return nil, utils.LavaFormatProduction("invalid input format, input isn't an unsigned index", err, utils.LogAttr("input", inp))
	}

	unmarshalledData, err := getDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, utils.LavaFormatProduction("invalid input format, data is not json", err, utils.LogAttr("data", unmarshalledData))
	}
	switch unmarshaledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		if uint64(len(unmarshaledDataTyped)) <= param_index {
			return nil, ValueNotSetError
		}
		block := unmarshaledDataTyped[param_index]
		// TODO: turn this into type assertion instead

		retArr := make([]interface{}, 0)
		retArr = append(retArr, blockInterfaceToString(block))
		return retArr, nil
	default:
		// Parse by arg can be only list as we dont have the name of the height property.
		return nil, utils.LavaFormatProduction("Parse type unsupported in parse by arg, only list parameters are currently supported", nil,
			utils.LogAttr("params", rpcInput.GetParams()),
			utils.LogAttr("request", unmarshaledDataTyped),
		)
	}
}

// expect input to be keys[a,b,c] and a canonical object such as
//
//	{
//	  "a": {
//	      "b": {
//	         "c": "wanted result"
//	       }
//	   }
//	}
//
// should output an interface array with "wanted result" in first index 0
func parseCanonical(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	unmarshalledData, err := getDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, data is not json: %s, error: %s", unmarshalledData, err)
	}

	switch unmarshalledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		inp := input[0]
		param_index, err := strconv.ParseUint(inp, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid input format, input isn't an unsigned index: %s, error: %s", inp, err)
		}
		if uint64(len(unmarshalledDataTyped)) <= param_index {
			return nil, ValueNotSetError
		}
		blockContainer := unmarshalledDataTyped[param_index]
		for _, key := range input[1:] {
			// type assertion for blockcontainer
			if blockContainer, ok := blockContainer.(map[string]interface{}); !ok {
				return nil, utils.LavaFormatWarning("invalid parser input format, blockContainer is not map[string]interface{}", ValueNotSetError,
					utils.LogAttr("params", rpcInput.GetParams()),
					utils.LogAttr("method", rpcInput.GetMethod()),
					utils.LogAttr("blockContainer", fmt.Sprintf("%v", blockContainer)),
					utils.LogAttr("key", key),
					utils.LogAttr("unmarshaledDataTyped", unmarshalledDataTyped),
				)
			}

			// assertion for key
			if container, ok := blockContainer.(map[string]interface{})[key]; ok {
				blockContainer = container
			} else {
				return nil, utils.LavaFormatWarning("invalid parser input format, blockContainer does not have the field searched inside", ValueNotSetError,
					utils.LogAttr("params", rpcInput.GetParams()),
					utils.LogAttr("method", rpcInput.GetMethod()),
					utils.LogAttr("blockContainer", fmt.Sprintf("%v", blockContainer)),
					utils.LogAttr("key", key),
					utils.LogAttr("unmarshaledDataTyped", unmarshalledDataTyped),
				)
			}
		}
		retArr := make([]interface{}, 0)
		retArr = append(retArr, blockInterfaceToString(blockContainer))
		return retArr, nil
	case map[string]interface{}:
		inp := input[0]
		_, err := strconv.ParseUint(inp, 10, 32)
		var relevantInput []string
		if err == nil {
			relevantInput = input[1:]
		} else {
			relevantInput = input
		}
		for idx, key := range relevantInput {
			if val, ok := unmarshalledDataTyped[key]; ok {
				if idx == (len(relevantInput) - 1) {
					retArr := make([]interface{}, 0)
					retArr = append(retArr, blockInterfaceToString(val))
					return retArr, nil
				}
				// if we didn't get to the last element continue deeper by changing unmarshaledDataTyped
				switch v := val.(type) {
				case map[string]interface{}:
					unmarshalledDataTyped = v
				default:
					return nil, fmt.Errorf("failed to parse, %s is not of type map[string]interface{} \nmore information: %s", v, unmarshalledData)
				}
			} else {
				return nil, ValueNotSetError
			}
		}
	case string:
		return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(unmarshalledDataTyped))
	default:
		// Parse by arg can be only list as we dont have the name of the height property.
		return nil, fmt.Errorf("not Supported ParseCanonical with other types %s", unmarshalledDataTyped)
	}
	return nil, fmt.Errorf("should not get here, parsing failed %s", unmarshalledData)
}

// parseDictionary return a value of prop specified in args if exists in dictionary
// if not return an error
func parseDictionary(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	// Validate number of arguments
	// The number of arguments should be 2
	// [prop_name,separator]
	if len(input) != 2 {
		return nil, fmt.Errorf("invalid input format, input length: %d and needs to be 2", len(input))
	}

	// Unmarshall data
	unmarshalledData, err := getDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, data is not json: %s, error: %s", unmarshalledData, err)
	}

	// Extract arguments
	propName := input[0]
	innerSeparator := input[1]

	switch unmarshalledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		// If value attribute with propName exists in array return it
		value := parseArrayOfInterfaces(unmarshalledDataTyped, propName, innerSeparator)
		if value != nil {
			return value, nil
		}

		// Else return an error
		return nil, ValueNotSetError
	case map[string]interface{}:
		// If attribute with key propName exists return value
		if val, ok := unmarshalledDataTyped[propName]; ok {
			return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(val))
		}

		// Else return an error
		return nil, ValueNotSetError
	case string:
		return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(unmarshalledDataTyped))
	default:
		return nil, fmt.Errorf("not Supported ParseDictionary with other types: %T", unmarshalledData)
	}
}

// parseDictionaryOrOrdered return a value of prop specified in args if exists in dictionary
// if not return an item from specified index
func parseDictionaryOrOrdered(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	// Validate number of arguments
	// The number of arguments should be 3
	// [prop_name,separator,parameter order if not found]
	if len(input) != 3 {
		return nil, fmt.Errorf("ParseDictionaryOrOrdered: invalid input format, input length: %d and needs to be 3: %s", len(input), strings.Join(input, ","))
	}

	// Unmarshall data
	unmarshalledData, err := getDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, data is not json: %s, error: %s", unmarshalledData, err)
	}

	// Extract arguments
	propName := input[0]
	innerSeparator := input[1]
	inp := input[2]

	// Convert prop index to the uint
	propIndex, err := strconv.ParseUint(inp, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, input isn't an unsigned index: %s, error: %s", inp, err)
	}

	switch unmarshalledDataTyped := unmarshalledData.(type) {
	case []interface{}:
		// If value attribute with propName exists in array return it
		value := parseArrayOfInterfaces(unmarshalledDataTyped, propName, innerSeparator)
		if value != nil {
			return value, nil
		}

		// If not make sure there are enough elements
		if uint64(len(unmarshalledDataTyped)) <= propIndex {
			return nil, ValueNotSetError
		}

		// Fetch value using prop index
		block := unmarshalledDataTyped[propIndex]
		return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(block))
	case map[string]interface{}:
		// If attribute with key propName exists return value
		if val, ok := unmarshalledDataTyped[propName]; ok {
			return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(val))
		}

		// If attribute with key index exists return value
		if val, ok := unmarshalledDataTyped[inp]; ok {
			return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(val))
		}

		// Else return not set error)
		utils.LavaFormatDebug("Failed parsing parseDictionaryOrOrdered",
			utils.LogAttr("params", rpcInput.GetParams()),
			utils.LogAttr("propName", propName),
			utils.LogAttr("inp", inp),
			utils.LogAttr("unmarshalledDataTyped", unmarshalledDataTyped),
			utils.LogAttr("method", rpcInput.GetMethod()),
			utils.LogAttr("err", ValueNotSetError),
		)

		return nil, ValueNotSetError
	case string:
		return appendInterfaceToInterfaceArrayWithError(blockInterfaceToString(unmarshalledDataTyped))
	default:
		return nil, fmt.Errorf("not Supported ParseDictionary with other types: %T", unmarshalledData)
	}
}

// parseArrayOfInterfaces returns value of item with specified prop name
// If it doesn't exist return nil
func parseArrayOfInterfaces(data []interface{}, propName, innerSeparator string) []interface{} {
	// Iterate over unmarshalled data
	for _, val := range data {
		if prop, ok := val.(string); ok {
			// split the value by innerSeparator
			valueArr := strings.SplitN(prop, innerSeparator, 2)

			// Check if the name match prop name
			if valueArr[0] != propName || len(valueArr) < 2 {
				// if not continue
				continue
			} else {
				// if yes return the value
				return appendInterfaceToInterfaceArray(valueArr[1])
			}
		}
		if m, ok := val.(map[string]interface{}); ok {
			if propValue, foundProp := m[propName]; foundProp {
				return appendInterfaceToInterfaceArray(blockInterfaceToString(propValue))
			}
		}
	}

	return nil
}

// appendInterfaceToInterfaceArray appends interface to interface array
func appendInterfaceToInterfaceArray(value interface{}) []interface{} {
	retArr := make([]interface{}, 0)
	retArr = append(retArr, value)
	return retArr
}

// appendInterfaceToInterfaceArrayWithError appends interface to interface array
// returns a valueNotSetError if the value is an empty string
func appendInterfaceToInterfaceArrayWithError(value string) ([]interface{}, error) {
	if value == "" || value == "0" || value == "%!s(<nil>)" {
		return nil, ValueNotSetError
	}
	return appendInterfaceToInterfaceArray(value), nil
}
