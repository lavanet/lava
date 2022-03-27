package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// UniquePaymentStorageUserServicerKeyPrefix is the prefix to retrieve all UniquePaymentStorageUserServicer
	UniquePaymentStorageUserServicerKeyPrefix = "UniquePaymentStorageUserServicer/value/"
)

// UniquePaymentStorageUserServicerKey returns the store key to retrieve a UniquePaymentStorageUserServicer from the index fields
func UniquePaymentStorageUserServicerKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
