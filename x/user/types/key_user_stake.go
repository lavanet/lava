package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// UserStakeKeyPrefix is the prefix to retrieve all UserStake
	UserStakeKeyPrefix = "UserStake/value/"
)

// UserStakeKey returns the store key to retrieve a UserStake from the index fields
func UserStakeKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
