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
	SpecRefreshEventName = "spec_refresh"
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
	DEFAULT_PARSED_RESULT_INDEX = 0
)

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

// allows unmarshaling generic parser type
func (s PARSER_TYPE) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(PARSER_TYPE_name[int32(s)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *PARSER_TYPE) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'NO_PARSER' in this case.
	*s = PARSER_TYPE(PARSER_TYPE_value[j])
	return nil
}

// allows unmarshaling parser func
func (s FUNCTION_TAG) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(s.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *FUNCTION_TAG) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*s = FUNCTION_TAG(FUNCTION_TAG_value[j])
	return nil
}

// allows unmarshaling header type
func (s Header_HeaderType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(Header_HeaderType_name[int32(s)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshals a quoted json string to the enum value
func (s *Header_HeaderType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*s = Header_HeaderType(Header_HeaderType_value[j])
	return nil
}

// allows unmarshaling header Verification Severity
func (s ParseValue_VerificationSeverity) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(ParseValue_VerificationSeverity_name[int32(s)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshals a quoted json string to the enum value
func (s *ParseValue_VerificationSeverity) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*s = ParseValue_VerificationSeverity(ParseValue_VerificationSeverity_value[j])
	return nil
}

func IsFinalizedBlock(requestedBlock, latestBlock, finalizationCriteria int64) bool {
	switch requestedBlock {
	case NOT_APPLICABLE:
		return false
		// TODO: handle safe & finalized key words, currently returns false
	default:
		if requestedBlock < 0 {
			return false
		}
		if requestedBlock <= latestBlock-finalizationCriteria {
			return true
		}
	}
	return false
}
