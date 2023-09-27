package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// DelegatorRewardKeyPrefix is the prefix to retrieve all DelegatorReward
	DelegatorRewardKeyPrefix = "DelegatorReward/value/"
)

// DelegatorRewardKey returns the store key to retrieve a DelegatorReward from the index fields
func DelegatorRewardKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
