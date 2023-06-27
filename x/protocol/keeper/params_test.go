package keeper_test

import (
	"testing"

	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/protocol/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.ProtocolKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
	require.EqualValues(t, params.ProviderVersion, k.ProviderVersion(ctx))
	require.EqualValues(t, params.ProviderMinVersion, k.ProviderMinVersion(ctx))
	require.EqualValues(t, params.ConsumerVersion, k.ConsumerVersion(ctx))
	require.EqualValues(t, params.ConsumerMinVersion, k.ConsumerMinVersion(ctx))
}
