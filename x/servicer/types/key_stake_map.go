package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// StakeMapKeyPrefix is the prefix to retrieve all StakeMap
	StakeMapKeyPrefix = "StakeMap/value/"
)

// StakeMapKey returns the store key to retrieve a StakeMap from the index fields
func StakeMapKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
