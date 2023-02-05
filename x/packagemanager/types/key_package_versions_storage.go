package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// PackageVersionsStorageKeyPrefix is the prefix to retrieve all PackageVersionsStorage
	PackageVersionsStorageKeyPrefix = "PackageVersionsStorage/value/"
)

// PackageVersionsStorageKey returns the store key to retrieve a PackageVersionsStorage from the index fields
func PackageVersionsStorageKey(
	packageIndex string,
) []byte {
	var key []byte

	packageIndexBytes := []byte(packageIndex)
	key = append(key, packageIndexBytes...)
	key = append(key, []byte("/")...)

	return key
}
