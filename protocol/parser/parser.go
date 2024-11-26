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
	"github.com/lavanet/lava/v4/protocol/chainlib/chainproxy/rpcclient"
	"github.com/lavanet/lava/v4/utils"
	"github.com/lavanet/lava/v4/utils/lavaslices"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	spectypes "github.com/lavanet/lava/v4/x/spec/types"
)

const (
	PARSE_PARAMS = 0
	PARSE_RESULT = 1

	MinimumHashLength = 32 // minimum hash length is 32 bits, which is MD5. SHA-1=40, SHA256=64, SHA512=128.
)

var ValueNotSetError = sdkerrors.New("Value Not Set ", 6662, "when trying to parse, the value that we attempted to parse did not exist")

type RPCInput interface {
	GetParams() interface{}
	GetResult() json.RawMessage
	GetError() *rpcclient.JsonError
	ParseBlock(block string) (int64, error)
	GetHeaders() []pairingtypes.Metadata
	GetMethod() string
	GetID() json.RawMessage
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

	block = unquoteString(block)
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

func parseInputWithGenericParsers(rpcInput RPCInput, genericParsers []spectypes.GenericParser) (*ParsedInput, bool) {
	managedToParse := false
	if len(genericParsers) == 0 {
		return nil, managedToParse
	}

	genericParserResult, genericParserErr := ParseWithGenericParsers(rpcInput, genericParsers)
	if genericParserErr != nil {
		return nil, managedToParse
	}

	_, err := genericParserResult.GetBlockHashes()
	if genericParserResult.GetParserError() == "" && (err == nil || genericParserResult.GetRawParsedData() != "") {
		// if we got here, there is no parser error and we successfully parsed either the block hashes or the raw parsed data
		managedToParse = true
	}

	return genericParserResult, managedToParse
}

// ParseRawBlock attempts to parse a block from rpcInput and store it in parsedInput.
// If parsing fails or rawBlock is empty, it uses defaultValue if provided.
// If parsing the defaultValue also fails, it sets the block to NOT_APPLICABLE.
func ParseRawBlock(rpcInput RPCInput, parsedInput *ParsedInput, defaultValue string) {
	rawBlock := parsedInput.GetRawParsedData()
	var parsedBlock int64
	var err error
	if rawBlock != "" {
		parsedBlock, err = rpcInput.ParseBlock(rawBlock)
	}
	if rawBlock == "" || err != nil {
		if defaultValue != "" {
			parsedBlock, err = rpcInput.ParseBlock(defaultValue)
			if err != nil {
				utils.LavaFormatError("Failed parsing default value, setting to NOT_APPLICABLE", err,
					utils.LogAttr("default_value", defaultValue),
				)
				parsedBlock = spectypes.NOT_APPLICABLE
			} else {
				parsedInput.UsedDefaultValue = true
			}
			utils.LavaFormatDebug("Failed parsing block from string, assuming default value",
				utils.LogAttr("params", rpcInput.GetParams()),
				utils.LogAttr("failed_parsed_value", rawBlock),
				utils.LogAttr("default_value", defaultValue),
				utils.LogAttr("parsedBlock", parsedBlock),
			)
		} else {
			parsedBlock = spectypes.NOT_APPLICABLE
		}
	}
	parsedInput.SetBlock(parsedBlock)
}

func parseInputWithLegacyBlockParser(rpcInput RPCInput, blockParser spectypes.BlockParser, dataSource int) (string, bool, error) {
	if rpcInput.GetError() != nil {
		return "", false, utils.LavaFormatError("blockParsing - rpcInput is error", nil, utils.LogAttr("rpcInput.GetError()", rpcInput.GetError()))
	}

	result, usedDefaultValue, err := legacyParse(rpcInput, blockParser, dataSource)
	if err != nil || result == nil {
		return "", usedDefaultValue, fmt.Errorf("blockParsing - parse failed. result=%v error=%w", result, err)
	}

	resString, ok := result[spectypes.DEFAULT_PARSED_RESULT_INDEX].(string)
	if !ok {
		return "", usedDefaultValue, utils.LavaFormatDebug("blockParsing - result[0].(string) - type assertion failed", utils.LogAttr("result[0]", result[0]))
	}

	return resString, usedDefaultValue, nil
}

// parseBlock processes the given RPC input using either generic parsers or a legacy block parser.
// It first attempts to parse the input with the provided generic parsers. If successful, it returns
// the parsed information after unquoting the raw parsed data. If the generic parsing fails, it falls
// back to using a legacy block parser.
//
// Parameters:
// - rpcInput: The input data to be parsed.
// - blockParser: The legacy block parser to use if generic parsing fails.
// - genericParsers: A slice of generic parsers to attempt first.
// - dataSource: An integer representing the source of the input: either PARSE_PARAMS or PARSE_RESULT.
//
// Returns:
// - A pointer to a ParsedInput struct containing the parsed data.
func parseBlock(rpcInput RPCInput, blockParser spectypes.BlockParser, genericParsers []spectypes.GenericParser, dataSource int) *ParsedInput {
	parsedBlockInfo, _ := parseInputWithGenericParsers(rpcInput, genericParsers)
	if parsedBlockInfo == nil {
		parsedBlockInfo = NewParsedInput()
	} else {
		rawBlockFromGenericParser := parsedBlockInfo.parsedDataRaw
		if rawBlockFromGenericParser != "" {
			parsedBlockInfo.parsedDataRaw = unquoteString(rawBlockFromGenericParser)
			return parsedBlockInfo
		}
	}

	parsedRawBlock, usedDefaultValue, err := parseInputWithLegacyBlockParser(rpcInput, blockParser, dataSource)
	if err == nil {
		parsedBlockInfo.UsedDefaultValue = usedDefaultValue
	}
	parsedBlockInfo.parsedDataRaw = unquoteString(parsedRawBlock)
	return parsedBlockInfo
}

func ParseBlockFromParams(rpcInput RPCInput, blockParser spectypes.BlockParser, genericParsers []spectypes.GenericParser) *ParsedInput {
	parsedInput := parseBlock(rpcInput, blockParser, genericParsers, PARSE_PARAMS)
	ParseRawBlock(rpcInput, parsedInput, blockParser.DefaultValue)
	return parsedInput
}

func ParseBlockFromReply(rpcInput RPCInput, blockParser spectypes.BlockParser, genericParsers []spectypes.GenericParser) *ParsedInput {
	parsedInput := parseBlock(rpcInput, blockParser, genericParsers, PARSE_RESULT)
	ParseRawBlock(rpcInput, parsedInput, blockParser.DefaultValue)
	return parsedInput
}

func unquoteString(str string) string {
	if !strings.Contains(str, "\"") {
		return str
	}

	unquoted, err := strconv.Unquote(str)
	if err != nil {
		return str
	}
	return unquoted
}

// This returns the parsed response after decoding
func ParseBlockHashFromReplyAndDecode(rpcInput RPCInput, resultParser spectypes.BlockParser, genericParsers []spectypes.GenericParser) (string, error) {
	parsedInput, parsedSuccessfully := parseInputWithGenericParsers(rpcInput, genericParsers)
	if parsedInput == nil {
		parsedBlockHashFromBlockParser, _, err := parseInputWithLegacyBlockParser(rpcInput, resultParser, PARSE_RESULT)
		if err != nil {
			parseErrorLogLevel := utils.LAVA_LOG_WARN
			if parsedSuccessfully {
				// found a hash, no need to log warning later
				parseErrorLogLevel = utils.LAVA_LOG_DEBUG
			}

			return "", utils.LavaFormatLog("failed to parse with legacy block parser", err,
				lavaslices.Slice(
					utils.LogAttr("rpcInput", rpcInput),
					utils.LogAttr("resultParser", resultParser),
				),
				uint(parseErrorLogLevel),
			)
		}
		return parseResponseByEncoding([]byte(parsedBlockHashFromBlockParser), resultParser.Encoding)
	}

	parsedBlockHashes, err := parsedInput.GetBlockHashes()
	if err != nil {
		return "", err
	}

	numberOfParsedHashes := len(parsedBlockHashes)
	if numberOfParsedHashes != 1 {
		return "", utils.LavaFormatError("[ParseBlockHashFromReplyAndDecode] expected parsed hashes length 1", nil, utils.LogAttr("rpcInput.GetResult()", rpcInput.GetResult()), utils.LogAttr("hashes_length", numberOfParsedHashes))
	}
	return parseResponseByEncoding([]byte(parsedBlockHashes[0]), resultParser.Encoding)
}

func legacyParse(rpcInput RPCInput, blockParser spectypes.BlockParser, dataSource int) ([]interface{}, bool, error) {
	var retval []interface{}
	var err error
	var usedDefaultValue bool
	switch blockParser.ParserFunc {
	case spectypes.PARSER_FUNC_EMPTY:
		return nil, usedDefaultValue, nil
	case spectypes.PARSER_FUNC_PARSE_BY_ARG:
		retval, err = parseByArg(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_CANONICAL:
		retval, err = parseCanonical(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_DICTIONARY:
		// currently, parseDictionary does not log warnings. If it ever will, we need to pass parserOptions
		retval, err = parseDictionary(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED:
		// currently, parseDictionaryOrOrdered does not log warnings. If it ever will, we need to pass parserOptions
		retval, err = parseDictionaryOrOrdered(rpcInput, blockParser.ParserArg, dataSource)
	case spectypes.PARSER_FUNC_DEFAULT:
		retval = parseDefault(blockParser.ParserArg)
		usedDefaultValue = true
	default:
		return nil, usedDefaultValue, fmt.Errorf("unsupported block parser parserFunc")
	}

	if err != nil {
		if ValueNotSetError.Is(err) && blockParser.DefaultValue != "" {
			// means this parsing failed because the value did not exist on an optional param
			retval = appendInterfaceToInterfaceArray(blockParser.DefaultValue)
			usedDefaultValue = true
		} else {
			return nil, usedDefaultValue, err
		}
	}

	return retval, usedDefaultValue, nil
}

type ParsedInput struct {
	parsedDataRaw    string
	parsedBlock      int64
	parsedHashes     []string
	parserError      string
	UsedDefaultValue bool
}

const RAW_NOT_APPLICABLE = "-1"

func NewParsedInput() *ParsedInput {
	return &ParsedInput{
		parsedDataRaw: RAW_NOT_APPLICABLE,
		parsedBlock:   spectypes.NOT_APPLICABLE,
		parsedHashes:  make([]string, 0),
	}
}

func (p *ParsedInput) SetBlock(block int64) {
	p.parsedBlock = block
}

func (p *ParsedInput) GetRawParsedData() string {
	return p.parsedDataRaw
}

func (p *ParsedInput) GetBlock() int64 {
	return p.parsedBlock
}

func (p *ParsedInput) GetParserError() string {
	return p.parserError
}

func (p *ParsedInput) GetBlockHashes() ([]string, error) {
	if len(p.parsedHashes) == 0 {
		return nil, fmt.Errorf("no parsed hashes found")
	}
	return p.parsedHashes, nil
}

func getMapForParse(rpcInput RPCInput) map[string]interface{} {
	var result map[string]interface{}
	rpcInputResult := rpcInput.GetResult()
	if rpcInputResult != nil {
		json.Unmarshal(rpcInputResult, &result)
	}

	return map[string]interface{}{"params": rpcInput.GetParams(), "result": result, "error": rpcInput.GetError().ToMap()}
}

func ParseWithGenericParsers(rpcInput RPCInput, genericParsers []spectypes.GenericParser) (*ParsedInput, error) {
	if len(genericParsers) == 0 {
		return nil, fmt.Errorf("no generic parsers to use")
	}

	parsingMap := getMapForParse(rpcInput)

	// We try to parse the params with all the generic parsers, the first one that succeeds, its value is returned
	for _, genericParser := range genericParsers {
		retval, err := parseGeneric(parsingMap, genericParser)
		if err == nil {
			return retval, nil
		}
	}

	// we didn't find a generic parser that worked
	return nil, utils.LavaFormatTrace("failed to parse with generic parsers",
		utils.LogAttr("parsingMap", parsingMap),
		utils.LogAttr("genericParsers", genericParsers),
	)
}

func parseRule(rule string, valueInterface interface{}) bool {
	// Split the rule by "||" to handle OR conditions
	if rule == "" {
		return true
	}

	// convert to string
	value := blockInterfaceToString(valueInterface)
	conditions := strings.Split(rule, "||")

	for _, condition := range conditions {
		condition = strings.TrimSpace(condition)
		if len(condition) <= 1 {
			continue
		}

		operator := condition[:1]
		operand := strings.TrimSpace(condition[1:])

		switch operator {
		case "=":
			if value == operand {
				return true
			}
		// Implement other operators when needed
		// case "<":
		// 	// Handle "<" logic
		// case ">":
		// 	// Handle ">" logic
		default:
			utils.LavaFormatError("Unsupported operator", nil, utils.LogAttr("operator", operator), utils.LogAttr("rule", rule), utils.LogAttr("value", value))
			continue
		}
	}

	return false
}

func parseGeneric(input interface{}, genericParser spectypes.GenericParser) (*ParsedInput, error) {
	value, err := findGenericParserValue(input, genericParser)
	if err != nil {
		return nil, err
	}

	if !parseRule(genericParser.Rule, value) {
		return nil, utils.LavaFormatWarning("PARSER_TYPE_DEFAULT_VALUE Did not match any rule", nil, utils.LogAttr("value", value), utils.LogAttr("rules", genericParser.Rule))
	}

	utils.LavaFormatTrace("parsed generic value",
		utils.LogAttr("input", input),
		utils.LogAttr("genericParser", genericParser),
		utils.LogAttr("value", value),
	)

	switch genericParser.ParseType {
	// Case Default value setting parsed block as the value set on spec after being parsed by ParseDefaultBlockParameter
	// regardless of the value provided by the user. for example .finality: final
	case spectypes.PARSER_TYPE_DEFAULT_VALUE:
		parsed := NewParsedInput()
		parsed.parsedDataRaw = genericParser.Value

		return parsed, nil
	// Case Block Latest, setting the value set by the user given a json path hit.
	// Example: block_id: 100, will result in requested block 100.
	case spectypes.PARSER_TYPE_RESULT:
		parsed := NewParsedInput()
		strValue := blockInterfaceToString(value)
		parsed.parsedDataRaw = strValue
		if genericParser.Value != "*" && strValue != genericParser.Value {
			parsed.parserError = fmt.Sprintf("expected %s, received %s", genericParser.Value, strValue)
		}
		return parsed, nil
	case spectypes.PARSER_TYPE_BLOCK_LATEST:
		parsed := NewParsedInput()
		parsed.parsedDataRaw = blockInterfaceToString(value)
		return parsed, nil
	case spectypes.PARSER_TYPE_BLOCK_HASH:
		return parseGenericParserBlockHash(value)
	// TODO: Implement other cases for different parsers
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

func parseGenericParserBlockHash(value interface{}) (*ParsedInput, error) {
	parsed := NewParsedInput()

	strVal, ok := value.(string)
	if !ok {
		return parsed, utils.LavaFormatDebug("failed to cast generic parser value to string", utils.LogAttr("value", value))
	}
	if len(strVal) < MinimumHashLength {
		return parsed, utils.LavaFormatDebug("value length is below minimum hash length", utils.LogAttr("strVal", strVal), utils.LogAttr("len(strVal)", len(strVal)), utils.LogAttr("MinimumHashLength", MinimumHashLength))
	}

	parsed.parsedHashes = append(parsed.parsedHashes, strVal)
	return parsed, nil
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
	case int:
		return strconv.Itoa(castedBlock)
	default:
		return fmt.Sprintf("%s", block)
	}
}

func parseByArg(rpcInput RPCInput, input []string, dataSource int) ([]interface{}, error) {
	// specified block is one of the direct parameters, input should be one string defining the location of the block
	if len(input) != 1 {
		return nil, fmt.Errorf("invalid input format, input length: %d and needs to be 1", len(input))
	}

	inp := input[0]
	param_index, err := strconv.ParseUint(inp, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, input isn't an unsigned index. input=%s error=%w", inp, err)
	}

	unmarshalledData, err := getDataToParse(rpcInput, dataSource)
	if err != nil {
		return nil, fmt.Errorf("invalid input format, data is not json. data=%s err=%w", unmarshalledData, err)
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
		// Parse by arg can be only list as we don't have the name of the height property.
		return nil, fmt.Errorf("parse type unsupported in parse by arg, only list parameters are currently supported. param=%s request=%s", rpcInput.GetParams(), unmarshaledDataTyped)
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
			// type assertion for blockContainer
			if blockContainer, ok := blockContainer.(map[string]interface{}); !ok {
				return nil, ValueNotSetError.Wrapf("invalid parser input format, blockContainer is not map[string]interface{}. "+
					"params=%v method=%v blockContainer=%v key=%s unmarshaledDataTyped=%v", rpcInput.GetParams(), rpcInput.GetMethod(), blockContainer, key, unmarshalledDataTyped)
			}

			// assertion for key
			if container, ok := blockContainer.(map[string]interface{})[key]; ok {
				blockContainer = container
			} else {
				return nil, ValueNotSetError.Wrapf("invalid parser input format, blockContainer does not have the field searched inside."+
					"params=%v method=%v blockContainer=%v key=%s unmarshaledDataTyped=%v", rpcInput.GetParams(), rpcInput.GetMethod(), blockContainer, key, unmarshalledDataTyped)
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
