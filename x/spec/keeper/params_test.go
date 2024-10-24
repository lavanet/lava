package keeper_test

import (
	"testing"

	specutils "github.com/lavanet/lava/v4/utils/keeper"
	"github.com/lavanet/lava/v4/x/spec/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := specutils.SpecKeeper(t)
	params := types.DefaultParams()

	k.SetParams(ctx, params)

	require.EqualValues(t, params, k.GetParams(ctx))
}
