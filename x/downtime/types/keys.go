package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName defines the module name
	ModuleName = "downtime"
	// StoreKey is the default store key for the module.
	StoreKey = ModuleName
)

var (
	LastBlockTimeKey         = []byte{0x01}
	DowntimeHeightKey        = []byte{0x02}
	DowntimeHeightGarbageKey = []byte{0x03}
)

// GetDowntimeKey returns the downtime storage key given the height.
func GetDowntimeKey(height uint64) []byte {
	return append(DowntimeHeightKey, sdk.Uint64ToBigEndian(height)...)
}

// GetDowntimeGarbageKey returns the downtime garbage storage key given the height.
func GetDowntimeGarbageKey(time time.Time) []byte {
	return append(DowntimeHeightGarbageKey, sdk.FormatTimeBytes(time)...)
}
