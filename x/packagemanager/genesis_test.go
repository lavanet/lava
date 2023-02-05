package packagemanager_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/packagemanager"
	"github.com/lavanet/lava/x/packagemanager/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		PackageVersionsStorageList: []types.PackageVersionsStorage{
			{
				PackageIndex: "0",
			},
			{
				PackageIndex: "1",
			},
		},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.PackagemanagerKeeper(t)
	packagemanager.InitGenesis(ctx, *k, genesisState)
	got := packagemanager.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.PackageVersionsStorageList, got.PackageVersionsStorageList)
	// this line is used by starport scaffolding # genesis/test/assert
}
