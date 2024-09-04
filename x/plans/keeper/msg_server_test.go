package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/v3/testutil/keeper"
	"github.com/lavanet/lava/v3/x/plans/keeper"
	"github.com/lavanet/lava/v3/x/plans/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.PlanKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
