package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// PackageKeyPrefix is the prefix to retrieve all Package
	PackageKeyPrefix = "Package/value/"
)

// PackageKey returns the store key to retrieve a Package from the index fields
func PackageKey(
	packageIndex string,
) []byte {
	var key []byte

	packageIndexBytes := []byte(packageIndex)
	key = append(key, packageIndexBytes...)
	key = append(key, []byte("/")...)

	return key
}
