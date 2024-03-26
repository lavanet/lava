package v2

import "encoding/binary"

var _ binary.ByteOrder

const (
	// EpochPaymentsKeyPrefix is the prefix to retrieve all EpochPayments
	EpochPaymentsKeyPrefix                      = "EpochPayments/value/"
	ProviderPaymentStorageKeyPrefix             = "ProviderPaymentStorage/value/"
	UniquePaymentStorageClientProviderKeyPrefix = "UniquePaymentStorageClientProvider/value/"
)

// EpochPaymentsKey returns the store key to retrieve a EpochPayments from the index fields
func EpochPaymentsKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}

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

func UniquePaymentStorageClientProviderKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
