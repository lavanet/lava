package types

import "encoding/binary"

var _ binary.ByteOrder

const (
	// PackageEntryKeyPrefix is the prefix to retrieve all PackageEntry
	PackageEntryKeyPrefix = "PackageEntry/value/"
)

// PackageEntryKey returns the store key to retrieve a PackageEntry from the index fields
func PackageEntryKey(
	packageIndex string,
) []byte {
	var key []byte

	packageIndexBytes := []byte(packageIndex)
	key = append(key, packageIndexBytes...)
	key = append(key, []byte("/")...)

	return key
}
