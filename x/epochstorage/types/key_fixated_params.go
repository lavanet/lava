package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// FixatedParamsKeyPrefix is the prefix to retrieve all FixatedParams
	FixatedParamsKeyPrefix = "FixatedParams/value/"
)

// FixatedParamsKey returns the store key to retrieve a FixatedParams from the index fields
func FixatedParamsKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
