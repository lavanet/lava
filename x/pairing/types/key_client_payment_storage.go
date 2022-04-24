package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ClientPaymentStorageKeyPrefix is the prefix to retrieve all ClientPaymentStorage
	ClientPaymentStorageKeyPrefix = "ClientPaymentStorage/value/"
)

// ClientPaymentStorageKey returns the store key to retrieve a ClientPaymentStorage from the index fields
func ClientPaymentStorageKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
