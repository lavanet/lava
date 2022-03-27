package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// SessionPaymentsKeyPrefix is the prefix to retrieve all SessionPayments
	SessionPaymentsKeyPrefix = "SessionPayments/value/"
)

// SessionPaymentsKey returns the store key to retrieve a SessionPayments from the index fields
func SessionPaymentsKey(
	index string,
) []byte {
	var key []byte

	indexBytes := []byte(index)
	key = append(key, indexBytes...)
	key = append(key, []byte("/")...)

	return key
}
