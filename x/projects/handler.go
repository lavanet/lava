package projects

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	legacyerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/lavanet/lava/v2/x/projects/keeper"
	"github.com/lavanet/lava/v2/x/projects/types"
)

// NewHandler ...
func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		_ = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *types.MsgAddKeys:
			res, err := msgServer.AddKeys(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgDelKeys:
			res, err := msgServer.DelKeys(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgSetPolicy:
			res, err := msgServer.SetPolicy(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgSetSubscriptionPolicy:
			res, err := msgServer.SetSubscriptionPolicy(sdk.WrapSDKContext(ctx), msg)
			return sdk.WrapServiceResult(ctx, res, err)
			// this line is used by starport scaffolding # 1
		default:
			errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, sdkerrors.Wrap(legacyerrors.ErrUnknownRequest, errMsg)
		}
	}
}
