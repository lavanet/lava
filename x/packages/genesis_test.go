package packages_test

import (
	"testing"

	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	"github.com/lavanet/lava/x/packages"
	"github.com/lavanet/lava/x/packages/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		PackageEntryList: []types.PackageEntry{
			{
				PackageIndex: "0",
			},
			{
				PackageIndex: "1",
			},
		},
		PackageUniqueIndexList: []types.PackageUniqueIndex{
			{
				Id: 0,
			},
			{
				Id: 1,
			},
		},
		PackageUniqueIndexCount: 2,
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.PackagesKeeper(t)
	packages.InitGenesis(ctx, *k, genesisState)
	got := packages.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.PackageEntryList, got.PackageEntryList)
	require.ElementsMatch(t, genesisState.PackageUniqueIndexList, got.PackageUniqueIndexList)
	require.Equal(t, genesisState.PackageUniqueIndexCount, got.PackageUniqueIndexCount)
	// this line is used by starport scaffolding # genesis/test/assert
}
