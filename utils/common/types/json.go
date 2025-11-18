package types

import (
	"encoding/hex"

	jsoniter "github.com/json-iterator/go"
)

// createCanonicalJSON creates a canonical form of JSON for comparison
func CreateCanonicalJSON(data []byte) (string, error) {
	var parsed interface{}
	json := jsoniter.ConfigCompatibleWithStandardLibrary

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

// CreateCanonicalForm creates a canonical form for comparison of both JSON and binary data
// For JSON data, it creates a normalized JSON string with sorted keys
// For binary data (like protobuf), it creates a hex-encoded string
func CreateCanonicalForm(data []byte) (string, error) {
	// First, try to parse as JSON
	canonicalJSON, err := CreateCanonicalJSON(data)
	if err == nil {
		return canonicalJSON, nil
	}

	// If JSON parsing fails, treat as binary data and use hex encoding for canonical comparison
	// This ensures binary data (like protobuf) can still be compared consistently
	return hex.EncodeToString(data), nil
}
