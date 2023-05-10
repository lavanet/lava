package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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

// check badge's basic attributes compared to the same traits from the relay request
// TODO: check cu allocation
func (badge Badge) IsBadgeValid(clientAddr string, lavaChainID string, epoch uint64) bool {
	if badge.Address != clientAddr || badge.LavaChainId != lavaChainID ||
		badge.Epoch != epoch {
		return false
	}
	return true
}
