package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
	"github.com/lavanet/lava/v4/x/rewards/keeper"
	"github.com/lavanet/lava/v4/x/rewards/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.RewardsKeeper(t)
	return keeper.NewMsgServerImpl(*k), ctx
}
