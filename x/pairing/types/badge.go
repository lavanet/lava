package types

import (
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BadgeData struct {
	Badge       Badge
	BadgeSigner sdk.AccAddress
}

func CreateBadge(cuAllocation uint64, epoch uint64, address sdk.AccAddress, lavaChainID string, sig []byte) *Badge {
	badge := Badge{
		CuAllocation: cuAllocation,
		Epoch:        epoch,
		Address:      address.String(),
		LavaChainId:  lavaChainID,
		ProjectSig:   sig,
	}

	return &badge
}

func CreateAddressEpochBadgeMapKey(address string, epoch uint64) string {
	return address + "_" + strconv.FormatUint(epoch, 10)
}

// check badge's basic attributes compared to the same traits from the relay request
// TODO: check cu allocation
func (badge Badge) IsBadgeValid(clientAddr string, lavaChainID string, epoch uint64) bool {
	if badge.Address != clientAddr || badge.LavaChainId != lavaChainID ||
		badge.Epoch != epoch {
		return false
	}
	return true
}
