package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SpecStakeStorageKeyPrefix is the prefix to retrieve all SpecStakeStorage
	SpecStakeStorageKeyPrefix = "SpecStakeStorage/value/"
)

// SpecStakeStorageKey returns the store key to retrieve a SpecStakeStorage from the index fields
func SpecStakeStorageKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
