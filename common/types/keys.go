package types

import "encoding/binary"

func KeyPrefix(p string) []byte {
	return []byte(p)
}

// FixationStore

const (
	FixationVersionKey string = "Entry_Version"
	EntryIndexPrefix   string = "Entry_Index_"
	EntryPrefix        string = "Entry_Value_"
)

// TimerStore

type TimerType int

const (
	BlockHeight TimerType = iota
	BlockTime
)

const (
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

func EncodeKey(key uint64) []byte {
	encodedKey := make([]byte, 8)
	binary.BigEndian.PutUint64(encodedKey, key)
	return encodedKey
}

func DecodeKey(encodedKey []byte) uint64 {
	return binary.BigEndian.Uint64(encodedKey)
}
