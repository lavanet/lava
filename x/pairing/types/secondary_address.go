package types

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func NewProviderSecondaryAddressAllowance(sdkMessages []sdk.Msg) (*feegrant.AllowedMsgAllowance, error) {
	allowedSecondaryAddressMessages := []string{}
	for _, sdkMsg := range sdkMessages {
		allowedSecondaryAddressMessages = append(allowedSecondaryAddressMessages, sdk.MsgTypeURL(sdkMsg))
	}
	return feegrant.NewAllowedMsgAllowance(&feegrant.BasicAllowance{}, allowedSecondaryAddressMessages)
}

func EmptyProviderSecondaryAddressAllowance(ctx sdk.Context, revocationTime time.Duration) *feegrant.BasicAllowance {
	blockTime := ctx.BlockTime().Add(revocationTime) // after the revocation period for stake entries
	return &feegrant.BasicAllowance{
		Expiration: &blockTime,
	}
}
