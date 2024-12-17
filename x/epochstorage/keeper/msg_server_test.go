package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
	"github.com/lavanet/lava/v4/x/epochstorage/keeper"
	"github.com/lavanet/lava/v4/x/epochstorage/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.EpochstorageKeeper(t)
	return keeper.NewMsgServerImpl(*k), ctx
}
