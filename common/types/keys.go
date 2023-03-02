package types

import "encoding/binary"

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	EntryIndexKey string = "Entry_Index_"
	EntryKey      string = "Entry_Value_"
)

func EncodeKey(key uint64) []byte {
	encodedKey := make([]byte, 8)
	binary.BigEndian.PutUint64(encodedKey, key)
	return encodedKey
}
