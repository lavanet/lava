package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SpecKeyPrefix is the prefix to retrieve all Spec
	SpecKeyPrefix = "Spec/value/"
)

// SpecKey returns the store key to retrieve a Spec from the index fields
func SpecKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
