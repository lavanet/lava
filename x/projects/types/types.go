package types

import (
	"bytes"
	"encoding/json"
)

const (
	MAX_PROJECT_NAME_LEN = 50
)

// set policy enum
type SetPolicyEnum int

const (
	SET_ADMIN_POLICY        SetPolicyEnum = 1
	SET_SUBSCRIPTION_POLICY SetPolicyEnum = 2
)

const (
	AddProjectKeyEventName = "add_key_to_project_event"
	DelProjectKeyEventName = "del_key_from_project_event"
)

// allows unmarshaling parser func
func (s SELECTED_PROVIDERS_MODE) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(SELECTED_PROVIDERS_MODE_name[int32(s)])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmashals a quoted json string to the enum value
func (s *SELECTED_PROVIDERS_MODE) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	// Note that if the string cannot be found then it will be set to the zero value, 'Created' in this case.
	*s = SELECTED_PROVIDERS_MODE(SELECTED_PROVIDERS_MODE_value[j])
	return nil
}