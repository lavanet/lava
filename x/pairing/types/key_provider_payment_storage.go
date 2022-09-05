package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// ProviderPaymentStorageKeyPrefix is the prefix to retrieve all ProviderPaymentStorage
	ProviderPaymentStorageKeyPrefix = "ProviderPaymentStorage/value/"
)

// ProviderPaymentStorageKey returns the store key to retrieve a ProviderPaymentStorage from the index fields
func ProviderPaymentStorageKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
