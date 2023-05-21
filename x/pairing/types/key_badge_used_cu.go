package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// BadgeUsedCuKeyPrefix is the prefix to retrieve all BadgeUsedCu
	BadgeUsedCuKeyPrefix  = "BadgeUsedCu/value/"
	BadgeTimerStorePrefix = "BadgeTimerStore/"
)

// BadgeUsedCuKey returns the store key to retrieve a BadgeUsedCu from the index fields
func BadgeUsedCuKey(
	badgeUsedCuMapKey []byte,
) []byte {
	var key []byte

	key = append(key, badgeUsedCuMapKey...)
	key = append(key, []byte("/")...)

	return key
}
