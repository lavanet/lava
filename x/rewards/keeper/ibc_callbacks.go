package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/lavanet/lava/utils"
)

func (k Keeper) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) exported.Acknowledgement {
	utils.LogLavaEvent(ctx, k.Logger(ctx), "oren", nil, "you did it!!!")
	return nil
}
