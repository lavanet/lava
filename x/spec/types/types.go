package types

import (
	"bytes"
	"encoding/json"
)

const (
	APIInterfaceJsonRPC       = "jsonrpc"
	APIInterfaceTendermintRPC = "tendermintrpc"
	APIInterfaceRest          = "rest"
	APIInterfaceGrpc          = "grpc"
)

const (
	ParamChangeEventName = "param_change"
	SpecAddEventName     = "spec_add"
	SpecModifyEventName  = "spec_modify"
)

const (
	EncodingBase64 = "base64"
	EncodingHex    = "hex"
)

const (
	NOT_APPLICABLE  int64 = -1
	LATEST_BLOCK    int64 = -2
	EARLIEST_BLOCK  int64 = -3
	PENDING_BLOCK   int64 = -4
	SAFE_BLOCK      int64 = -5
	FINALIZED_BLOCK int64 = -6
)

const (
	GET_BLOCKNUM                = "getBlockNumber"
	GET_BLOCK_BY_NUM            = "getBlockByNumber"
	GET_CHAIN_ID                = "getChainId"
	DEFAULT_PARSED_RESULT_INDEX = 0
)

var SupportedTags = [...]string{GET_BLOCKNUM, GET_BLOCK_BY_NUM, GET_CHAIN_ID}

// allows unmarshaling parser func
func (s PARSER_FUNC) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(PARSER_FUNC_name[int32(s)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *PARSER_FUNC) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*s = PARSER_FUNC(PARSER_FUNC_value[j])
	return nil
}

func IsFinalizedBlock(requestedBlock int64, latestBlock int64, finalizationCriteria uint32) bool {
	switch requestedBlock {
	case NOT_APPLICABLE:
		return false
		// TODO: handle safe & finalized key words, currently returns false
	default:
		if requestedBlock < 0 {
			return false
		}
		if requestedBlock <= latestBlock-int64(finalizationCriteria) {
			return true
		}
	}
	return false
}
