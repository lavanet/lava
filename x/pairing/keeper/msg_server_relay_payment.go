package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/pairing/types"
)

func (k msgServer) RelayPayment(goCtx context.Context, msg *types.MsgRelayPayment) (*types.MsgRelayPaymentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgRelayPaymentResponse{}, nil
}
