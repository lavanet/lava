package types

import "encoding/json"

// jsonMarshal and jsonUnmarshal are thin wrappers around encoding/json used
// by types that need Marshal/Unmarshal methods compatible with
// protocopy.DeepCopyProtoObject.

func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
