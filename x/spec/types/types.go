package types

import (
	"bytes"
	"encoding/json"
)

const (
	GET_BLOCKNUM                = "getBlockNumber"
	GET_BLOCK_BY_NUM            = "getBlockByNumber"
	DEFAULT_PARSED_RESULT_INDEX = 0
)

var SupportedTags = [...]string{GET_BLOCKNUM, GET_BLOCK_BY_NUM}

//allows unmarshaling parser func
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
