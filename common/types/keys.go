package types

import "encoding/binary"

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// FixationStore

const (
	FixationVersionKey = "Entry_Version"
	EntryIndexPrefix   = "Entry_Index_"
	EntryPrefix        = "Entry_Value_"
)

var (
	// use single byte to save storage.
	// (value cannot be part of a valid index)
	EntryIndexLive = []byte{1}
	EntryIndexDead = []byte{2}
)

// TimerStore

type TimerType int

const (
	BlockHeight TimerType = iota
	BlockTime
)

const (
	TimerVersionKey = "Entry_Version"
	NextTimerPrefix = "Timer_Next_"
	TimerPrefix     = "Timer_Value_"
)

var NextTimerKey = []string{
	"NextBlock", // for BlockHeight
	"NextDate",  // for BlockTime
}

var TimerTypePrefix = []string{
	"Block_", // for BlockHeight
	"Time_",  // for BlockTime
}

// Common encoder/decode

func EncodeBlockAndKey(block uint64, key []byte) []byte {
	encodedKey := make([]byte, 8+len(key))
	binary.BigEndian.PutUint64(encodedKey[0:8], block)
	copy(encodedKey[8:], key)
	return encodedKey
}

func DecodeBlockAndKey(encodedKey []byte) (uint64, []byte) {
	block := binary.BigEndian.Uint64(encodedKey[0:8])
	return block, encodedKey[8:]
}

func EncodeKey(key uint64) []byte {
	return EncodeBlockAndKey(key, []byte{})
}

func DecodeKey(encodedKey []byte) uint64 {
	block, _ := DecodeBlockAndKey(encodedKey)
	return block
}
