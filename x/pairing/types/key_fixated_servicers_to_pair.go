package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// FixatedServicersToPairKeyPrefix is the prefix to retrieve all FixatedServicersToPair
	FixatedServicersToPairKeyPrefix = "FixatedServicersToPair/value/"
)

// FixatedServicersToPairKey returns the store key to retrieve a FixatedServicersToPair from the index fields
func FixatedServicersToPairKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
