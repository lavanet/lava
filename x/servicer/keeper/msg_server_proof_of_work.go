package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/x/servicer/types"
)

func (k msgServer) ProofOfWork(goCtx context.Context, msg *types.MsgProofOfWork) (*types.MsgProofOfWorkResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// TODO: Handling the message
	_ = ctx

	return &types.MsgProofOfWorkResponse{}, nil
}
