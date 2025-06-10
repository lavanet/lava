package types

import (
	"encoding/json"
)

// createCanonicalJSON creates a canonical form of JSON for comparison
func CreateCanonicalJSON(data []byte) (string, error) {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", err
	}

	// Marshal back to JSON with sorted keys
	// The json is sorted in the implementation of the json package (pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/encoding/json/encode.go - line 750)
	result, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
