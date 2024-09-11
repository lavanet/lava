package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/lavanet/lava/v3/testutil/keeper"
	"github.com/lavanet/lava/v3/x/pairing/keeper"
	"github.com/lavanet/lava/v3/x/pairing/types"
)

// TODO: use or delete this function
func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.PairingKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
