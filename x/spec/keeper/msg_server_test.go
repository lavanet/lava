package keeper_test

import (
	"context"
	"testing"

	specutils "github.com/lavanet/lava/v4/utils/keeper"
	"github.com/lavanet/lava/v4/x/spec/keeper"
	"github.com/lavanet/lava/v4/x/spec/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := specutils.SpecKeeper(t)
	return keeper.NewMsgServerImpl(*k), ctx
}
