package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// UniquePaymentStorageClientProviderKeyPrefix is the prefix to retrieve all UniquePaymentStorageClientProvider
	UniquePaymentStorageClientProviderKeyPrefix = "UniquePaymentStorageClientProvider/value/"
	AdrLengthUser                               = 45
	AdrLengthProvider                           = 45
)

// UniquePaymentStorageClientProviderKey returns the store key to retrieve a UniquePaymentStorageClientProvider from the index fields
func UniquePaymentStorageClientProviderKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
