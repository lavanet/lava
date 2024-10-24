package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	specutils "github.com/lavanet/lava/v4/utils/keeper"
	"github.com/lavanet/lava/v4/x/spec/keeper"
	"github.com/lavanet/lava/v4/x/spec/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := specutils.SpecKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
