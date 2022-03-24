package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SessionStorageForSpecKeyPrefix is the prefix to retrieve all SessionStorageForSpec
	SessionStorageForSpecKeyPrefix = "SessionStorageForSpec/value/"
)

// SessionStorageForSpecKey returns the store key to retrieve a SessionStorageForSpec from the index fields
func SessionStorageForSpecKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
