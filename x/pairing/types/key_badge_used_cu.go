package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// BadgeUsedCuKeyPrefix is the prefix to retrieve all BadgeUsedCu
	BadgeUsedCuKeyPrefix  = "BadgeUsedCu/value/"
	BadgeTimerStorePrefix = "BadgeTimerStore/"
)

// BadgeUsedCuKey returns the store key to retrieve a BadgeUsedCu from the index fields
func BadgeUsedCuKey(badgeSig []byte, providerAddress string) []byte {
	badgeSig = append(badgeSig, []byte(providerAddress)...)
	return append(badgeSig, byte('/'))
}
