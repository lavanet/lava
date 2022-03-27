package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// UserPaymentStorageKeyPrefix is the prefix to retrieve all UserPaymentStorage
	UserPaymentStorageKeyPrefix = "UserPaymentStorage/value/"
)

// UserPaymentStorageKey returns the store key to retrieve a UserPaymentStorage from the index fields
func UserPaymentStorageKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
