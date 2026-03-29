package types

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// ---------------------------------------------------------------------------
// EXTENSION
// ---------------------------------------------------------------------------

type EXTENSION int32

const (
	EXTENSION_NONE    EXTENSION = 0
	EXTENSION_ARCHIVE EXTENSION = 1
)

var EXTENSION_name = map[int32]string{
	0: "NONE",
	1: "ARCHIVE",
}

var EXTENSION_value = map[string]int32{
	"NONE":    0,
	"ARCHIVE": 1,
}

func (x EXTENSION) String() string {
	if name, ok := EXTENSION_name[int32(x)]; ok {
		return name
	}
	return fmt.Sprintf("EXTENSION(%d)", int32(x))
}

func (s EXTENSION) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s.String())
	buf.WriteString(`"`)
	return buf.Bytes(), nil
}

func (s *EXTENSION) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = EXTENSION(EXTENSION_value[j])
	return nil
}

// ---------------------------------------------------------------------------
// FUNCTION_TAG
// ---------------------------------------------------------------------------

type FUNCTION_TAG int32

const (
	FUNCTION_TAG_DISABLED               FUNCTION_TAG = 0
	FUNCTION_TAG_GET_BLOCKNUM           FUNCTION_TAG = 1
	FUNCTION_TAG_GET_BLOCK_BY_NUM       FUNCTION_TAG = 2
	FUNCTION_TAG_SET_LATEST_IN_METADATA FUNCTION_TAG = 3
	FUNCTION_TAG_SET_LATEST_IN_BODY     FUNCTION_TAG = 4
	FUNCTION_TAG_VERIFICATION           FUNCTION_TAG = 5
	FUNCTION_TAG_GET_EARLIEST_BLOCK     FUNCTION_TAG = 6
	FUNCTION_TAG_SUBSCRIBE              FUNCTION_TAG = 7
	FUNCTION_TAG_UNSUBSCRIBE            FUNCTION_TAG = 8
	FUNCTION_TAG_UNSUBSCRIBE_ALL        FUNCTION_TAG = 9
)

var FUNCTION_TAG_name = map[int32]string{
	0: "DISABLED",
	1: "GET_BLOCKNUM",
	2: "GET_BLOCK_BY_NUM",
	3: "SET_LATEST_IN_METADATA",
	4: "SET_LATEST_IN_BODY",
	5: "VERIFICATION",
	6: "GET_EARLIEST_BLOCK",
	7: "SUBSCRIBE",
	8: "UNSUBSCRIBE",
	9: "UNSUBSCRIBE_ALL",
}

var FUNCTION_TAG_value = map[string]int32{
	"DISABLED":               0,
	"GET_BLOCKNUM":           1,
	"GET_BLOCK_BY_NUM":       2,
	"SET_LATEST_IN_METADATA": 3,
	"SET_LATEST_IN_BODY":     4,
	"VERIFICATION":           5,
	"GET_EARLIEST_BLOCK":     6,
	"SUBSCRIBE":              7,
	"UNSUBSCRIBE":            8,
	"UNSUBSCRIBE_ALL":        9,
}

func (x FUNCTION_TAG) String() string {
	if name, ok := FUNCTION_TAG_name[int32(x)]; ok {
		return name
	}
	return fmt.Sprintf("FUNCTION_TAG(%d)", int32(x))
}

func (s FUNCTION_TAG) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s.String())
	buf.WriteString(`"`)
	return buf.Bytes(), nil
}

func (s *FUNCTION_TAG) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = FUNCTION_TAG(FUNCTION_TAG_value[j])
	return nil
}

// ---------------------------------------------------------------------------
// PARSER_TYPE
// ---------------------------------------------------------------------------

type PARSER_TYPE int32

const (
	PARSER_TYPE_NO_PARSER      PARSER_TYPE = 0
	PARSER_TYPE_BLOCK_LATEST   PARSER_TYPE = 1
	PARSER_TYPE_BLOCK_EARLIEST PARSER_TYPE = 2
	PARSER_TYPE_RESULT         PARSER_TYPE = 3
	PARSER_TYPE_EXTENSION_ARG  PARSER_TYPE = 4
	PARSER_TYPE_IDENTIFIER     PARSER_TYPE = 5
	PARSER_TYPE_DEFAULT_VALUE  PARSER_TYPE = 6
	PARSER_TYPE_BLOCK_HASH     PARSER_TYPE = 7
)

var PARSER_TYPE_name = map[int32]string{
	0: "NO_PARSER",
	1: "BLOCK_LATEST",
	2: "BLOCK_EARLIEST",
	3: "RESULT",
	4: "EXTENSION_ARG",
	5: "IDENTIFIER",
	6: "DEFAULT_VALUE",
	7: "BLOCK_HASH",
}

var PARSER_TYPE_value = map[string]int32{
	"NO_PARSER":      0,
	"BLOCK_LATEST":   1,
	"BLOCK_EARLIEST": 2,
	"RESULT":         3,
	"EXTENSION_ARG":  4,
	"IDENTIFIER":     5,
	"DEFAULT_VALUE":  6,
	"BLOCK_HASH":     7,
}

func (x PARSER_TYPE) String() string {
	if name, ok := PARSER_TYPE_name[int32(x)]; ok {
		return name
	}
	return fmt.Sprintf("PARSER_TYPE(%d)", int32(x))
}

func (s PARSER_TYPE) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s.String())
	buf.WriteString(`"`)
	return buf.Bytes(), nil
}

func (s *PARSER_TYPE) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = PARSER_TYPE(PARSER_TYPE_value[j])
	return nil
}

// ---------------------------------------------------------------------------
// PARSER_FUNC
// ---------------------------------------------------------------------------

type PARSER_FUNC int32

const (
	PARSER_FUNC_EMPTY                       PARSER_FUNC = 0
	PARSER_FUNC_PARSE_BY_ARG                PARSER_FUNC = 1
	PARSER_FUNC_PARSE_CANONICAL             PARSER_FUNC = 2
	PARSER_FUNC_PARSE_DICTIONARY            PARSER_FUNC = 3
	PARSER_FUNC_PARSE_DICTIONARY_OR_ORDERED PARSER_FUNC = 4
	PARSER_FUNC_DEFAULT                     PARSER_FUNC = 6
)

var PARSER_FUNC_name = map[int32]string{
	0: "EMPTY",
	1: "PARSE_BY_ARG",
	2: "PARSE_CANONICAL",
	3: "PARSE_DICTIONARY",
	4: "PARSE_DICTIONARY_OR_ORDERED",
	6: "DEFAULT",
}

var PARSER_FUNC_value = map[string]int32{
	"EMPTY":                       0,
	"PARSE_BY_ARG":                1,
	"PARSE_CANONICAL":             2,
	"PARSE_DICTIONARY":            3,
	"PARSE_DICTIONARY_OR_ORDERED": 4,
	"DEFAULT":                     6,
}

func (x PARSER_FUNC) String() string {
	if name, ok := PARSER_FUNC_name[int32(x)]; ok {
		return name
	}
	return fmt.Sprintf("PARSER_FUNC(%d)", int32(x))
}

func (s PARSER_FUNC) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s.String())
	buf.WriteString(`"`)
	return buf.Bytes(), nil
}

func (s *PARSER_FUNC) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = PARSER_FUNC(PARSER_FUNC_value[j])
	return nil
}

// ---------------------------------------------------------------------------
// Header_HeaderType
// ---------------------------------------------------------------------------

type Header_HeaderType int32

const (
	Header_pass_send     Header_HeaderType = 0
	Header_pass_reply    Header_HeaderType = 1
	Header_pass_both     Header_HeaderType = 2
	Header_pass_ignore   Header_HeaderType = 3
	Header_pass_nullify  Header_HeaderType = 4
	Header_pass_override Header_HeaderType = 5
)

var Header_HeaderType_name = map[int32]string{
	0: "pass_send",
	1: "pass_reply",
	2: "pass_both",
	3: "pass_ignore",
	4: "pass_nullify",
	5: "pass_override",
}

var Header_HeaderType_value = map[string]int32{
	"pass_send":     0,
	"pass_reply":    1,
	"pass_both":     2,
	"pass_ignore":   3,
	"pass_nullify":  4,
	"pass_override": 5,
}

func (x Header_HeaderType) String() string {
	if name, ok := Header_HeaderType_name[int32(x)]; ok {
		return name
	}
	return fmt.Sprintf("Header_HeaderType(%d)", int32(x))
}

func (s Header_HeaderType) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s.String())
	buf.WriteString(`"`)
	return buf.Bytes(), nil
}

func (s *Header_HeaderType) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = Header_HeaderType(Header_HeaderType_value[j])
	return nil
}

// ---------------------------------------------------------------------------
// ParseValue_VerificationSeverity
// ---------------------------------------------------------------------------

type ParseValue_VerificationSeverity int32

const (
	ParseValue_Fail    ParseValue_VerificationSeverity = 0
	ParseValue_Warning ParseValue_VerificationSeverity = 1
)

var ParseValue_VerificationSeverity_name = map[int32]string{
	0: "Fail",
	1: "Warning",
}

var ParseValue_VerificationSeverity_value = map[string]int32{
	"Fail":    0,
	"Warning": 1,
}

func (x ParseValue_VerificationSeverity) String() string {
	if name, ok := ParseValue_VerificationSeverity_name[int32(x)]; ok {
		return name
	}
	return fmt.Sprintf("ParseValue_VerificationSeverity(%d)", int32(x))
}

func (s ParseValue_VerificationSeverity) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s.String())
	buf.WriteString(`"`)
	return buf.Bytes(), nil
}

func (s *ParseValue_VerificationSeverity) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = ParseValue_VerificationSeverity(ParseValue_VerificationSeverity_value[j])
	return nil
}

// ---------------------------------------------------------------------------
// Spec_ProvidersTypes
// ---------------------------------------------------------------------------

type Spec_ProvidersTypes int32

const (
	Spec_dynamic Spec_ProvidersTypes = 0
	Spec_static  Spec_ProvidersTypes = 1
)

var Spec_ProvidersTypes_name = map[int32]string{
	0: "dynamic",
	1: "static",
}

var Spec_ProvidersTypes_value = map[string]int32{
	"dynamic": 0,
	"static":  1,
}

func (x Spec_ProvidersTypes) String() string {
	if name, ok := Spec_ProvidersTypes_name[int32(x)]; ok {
		return name
	}
	return fmt.Sprintf("Spec_ProvidersTypes(%d)", int32(x))
}

func (s Spec_ProvidersTypes) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString(`"`)
	buf.WriteString(s.String())
	buf.WriteString(`"`)
	return buf.Bytes(), nil
}

func (s *Spec_ProvidersTypes) UnmarshalJSON(b []byte) error {
	var j string
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	*s = Spec_ProvidersTypes(Spec_ProvidersTypes_value[j])
	return nil
}
