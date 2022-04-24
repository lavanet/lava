package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// StakeStorageKeyPrefix is the prefix to retrieve all StakeStorage
	StakeStorageKeyPrefix = "StakeStorage/value/"
)

// StakeStorageKey returns the store key to retrieve a StakeStorage from the index fields
func StakeStorageKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
