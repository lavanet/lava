package types

import (
	"strconv"

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

func CreateAddressEpochBadgeMapKey(address string, epoch uint64) string {
	return address + "_" + strconv.FormatUint(epoch, 10)
}

// check badge's basic attributes compared to the same traits from the relay request
func (badge Badge) IsBadgeValid(clientAddr string, lavaChainID string, epoch uint64) bool {
	if badge.Address != clientAddr || badge.LavaChainId != lavaChainID ||
		badge.Epoch != epoch {
		return false
	}
	return true
}

func (b Badge) GetSignature() []byte {
	return b.ProjectSig
}

func (b Badge) DataToSign() []byte {
	b.ProjectSig = []byte{}
	return []byte(b.String())
}

func (b Badge) HashRounds() int {
	return 1
}
