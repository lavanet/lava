package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/lavanet/lava/v4/testutil/keeper"
	"github.com/lavanet/lava/v4/x/pairing/keeper"
	"github.com/lavanet/lava/v4/x/pairing/types"
)

// TODO: use or delete this function
func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.PairingKeeper(t)
	return keeper.NewMsgServerImpl(*k), ctx
}
