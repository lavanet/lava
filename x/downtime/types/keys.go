package types

import sdk "github.com/cosmos/cosmos-sdk/types"

const (
	// ModuleName defines the module name
	ModuleName = "downtime"
	// StoreKey is the default store key for the module.
	StoreKey = ModuleName
)

var (
	LastBlockTimeKey = []byte{0x01}
	DowntimeHeight   = []byte{0x02}
)

// GetDowntimeKey returns the downtime storage key given the height.
func GetDowntimeKey(height uint64) []byte {
	return append(DowntimeHeight, sdk.Uint64ToBigEndian(height)...)
}
