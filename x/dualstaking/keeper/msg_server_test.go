package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
	"github.com/lavanet/lava/v4/x/dualstaking/keeper"
	"github.com/lavanet/lava/v4/x/dualstaking/types"
	"github.com/stretchr/testify/require"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.DualstakingKeeper(t)
	return keeper.NewMsgServerImpl(*k), ctx
}

func TestMsgServer(t *testing.T) {
	ms, ctx := setupMsgServer(t)
	require.NotNil(t, ms)
	require.NotNil(t, ctx)
}
