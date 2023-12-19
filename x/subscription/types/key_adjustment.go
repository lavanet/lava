package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// AdjustmentKeyPrefix is the prefix to retrieve all Adjustment
	AdjustmentKeyPrefix = "Adjustment/value/"
)

// AdjustmentKey returns the store key to retrieve an Adjustment from the index fields
func AdjustmentKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
