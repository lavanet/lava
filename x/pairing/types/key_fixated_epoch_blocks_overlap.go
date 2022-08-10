package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// FixatedEpochBlocksOverlapKeyPrefix is the prefix to retrieve all FixatedEpochBlocksOverlap
	FixatedEpochBlocksOverlapKeyPrefix = "FixatedEpochBlocksOverlap/value/"
)

// FixatedEpochBlocksOverlapKey returns the store key to retrieve a FixatedEpochBlocksOverlap from the index fields
func FixatedEpochBlocksOverlapKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
