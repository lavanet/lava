package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// FixatedStakeToMaxCuKeyPrefix is the prefix to retrieve all FixatedStakeToMaxCu
	FixatedStakeToMaxCuKeyPrefix = "FixatedStakeToMaxCu/value/"
)

// FixatedStakeToMaxCuKey returns the store key to retrieve a FixatedStakeToMaxCu from the index fields
func FixatedStakeToMaxCuKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
