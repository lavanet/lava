package keeper_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/packages/keeper"
	"github.com/lavanet/lava/x/packages/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

type testStruct struct {
	ctx     context.Context
	keepers *testkeeper.Keepers
}

func createNPackageVersionsStorage(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.PackageVersionsStorage {
	items := make([]types.PackageVersionsStorage, n)
	for i := range items {
		items[i].PackageIndex = strconv.Itoa(i)

		keeper.SetPackageVersionsStorage(ctx, items[i])
	}
	return items
}

func TestPackageVersionsStorageGet(t *testing.T) {
	keeper, ctx := testkeeper.PackagesKeeper(t)
	items := createNPackageVersionsStorage(keeper, ctx, 10)
	for _, item := range items {
		rst, found := keeper.GetPackageVersionsStorage(ctx,
			item.PackageIndex,
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestPackageVersionsStorageRemove(t *testing.T) {
	keeper, ctx := testkeeper.PackagesKeeper(t)
	items := createNPackageVersionsStorage(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePackageVersionsStorage(ctx,
			item.PackageIndex,
		)
		_, found := keeper.GetPackageVersionsStorage(ctx,
			item.PackageIndex,
		)
		require.False(t, found)
	}
}

func TestPackageVersionsStorageGetAll(t *testing.T) {
	keeper, ctx := testkeeper.PackagesKeeper(t)
	items := createNPackageVersionsStorage(keeper, ctx, 10)
	require.ElementsMatch(t,
		nullify.Fill(items),
		nullify.Fill(keeper.GetAllPackageVersionsStorage(ctx)),
	)
}

// Function to create an array of test packages. Can output an array with packages with the same ID
func CreateTestPackages(packageAmount uint64, withSameIndex bool) []types.Package {
	testPackages := []types.Package{}

	// create dummy packages in a loop according to packageAmount
	for i := uint64(0); i < packageAmount; i++ {
		// create distinct package index by the loop counter
		packageIndex := "mockPackage" + strconv.FormatUint(i, 10)
		overuseRate := uint64(10)

		// create dummy package and append to the testPackages array
		dummyPackage := types.Package{
			Index:                packageIndex,
			Name:                 "test package",
			Description:          "package to test",
			Type:                 "rpc",
			Duration:             200,
			Epoch:                100,
			Price:                sdk.NewCoin("ulava", sdk.OneInt()),
			ComputeUnits:         1000,
			ComputeUnitsPerEpoch: 100,
			ServicersToPair:      3,
			AllowOveruse:         true,
			OveruseRate:          overuseRate,
		}
		testPackages = append(testPackages, dummyPackage)

		// if we need to create a package with the same index, create an additional one with a different overuseRate and append it to testPackages (we increase the counter so we won't get more packages than packageAmount)
		if withSameIndex {
			overuseRate2 := uint64(15)
			dummyPackage2 := dummyPackage
			dummyPackage2.OveruseRate = overuseRate2
			testPackages = append(testPackages, dummyPackage2)
			i++
		}
	}

	return testPackages
}

// Test that the process of: package is added, an update is added, stale version is removed works correctly. Make sure that a stale package with subs is not removed
func TestPackageAdditionAndRemoval(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create packages (both packages have the same ID. They only differ in the overuseRate field)
	testPackages := CreateTestPackages(2, true)

	// simulate a package proposal of the first package
	err := testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, []types.Package{testPackages[0]})
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// simulate a package proposal of the second package
	err = testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, []types.Package{testPackages[1]})
	require.Nil(t, err)

	// get the package storage and verify that there are two packages in the package storage
	packageStorage, found := ts.keepers.Packages.GetPackageVersionsStorage(sdk.UnwrapSDKContext(ts.ctx), testPackages[0].GetIndex())
	require.True(t, found)
	require.Equal(t, 2, len(packageStorage.GetPackageStorage().GetEntryList()))

	// verify that testPackages[1] is the latest package version (should be first in the entryList)
	packageLatestVersion, err := ts.keepers.Packages.GetPackageLatestVersion(sdk.UnwrapSDKContext(ts.ctx), packageStorage.GetPackageIndex())
	require.Equal(t, testPackages[1].OveruseRate, packageLatestVersion.GetOveruseRate())

	// advance duration[epochs]+1 epochs and test that there are still two packages (packages are deleted when currentEpoch > updatedPackageEpoch + oldPackageDuration. The epoch field of packages is always the next epoch from their proposal epoch)
	packageDurationInEpochs := testPackages[0].GetDuration() / ts.keepers.Epochstorage.EpochBlocksRaw(sdk.UnwrapSDKContext(ts.ctx))
	for i := 0; i < int(packageDurationInEpochs)+1; i++ {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}
	require.Equal(t, 2, len(packageStorage.GetPackageStorage().GetEntryList()))

	// advance one more epoch to remove the stale package (the deletePackage function is invoked in the start of every epoch)
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// test that there is only one package in the storage and it's the newer one
	packageStorage, found = ts.keepers.Packages.GetPackageVersionsStorage(sdk.UnwrapSDKContext(ts.ctx), testPackages[0].GetIndex())
	require.True(t, found)
	packageLeft, err := ts.keepers.Packages.GetPackageLatestVersion(sdk.UnwrapSDKContext(ts.ctx), packageStorage.GetPackageIndex())
	require.Nil(t, err)
	require.Equal(t, 1, len(packageStorage.GetPackageStorage().GetEntryList()))
	require.Equal(t, testPackages[1].GetOveruseRate(), packageLeft.GetOveruseRate())
}

// Test that if two packages with the same index are added, then we keep only the latest one
func TestUpdatePackageInSameEpoch(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create packages (both packages have the same ID. They only differ in the overuseRate field)
	testPackages := CreateTestPackages(2, true)

	// simulate a proposal of the packages
	err := testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, testPackages)
	require.Nil(t, err)

	// test that there is only one package in the storage and it's the newer one
	packageStorage, found := ts.keepers.Packages.GetPackageVersionsStorage(sdk.UnwrapSDKContext(ts.ctx), testPackages[0].GetIndex())
	require.True(t, found)
	packageLeft, err := ts.keepers.Packages.GetPackageLatestVersion(sdk.UnwrapSDKContext(ts.ctx), packageStorage.GetPackageIndex())
	require.Nil(t, err)
	require.Equal(t, 1, len(packageStorage.GetPackageStorage().GetEntryList()))
	require.Equal(t, testPackages[1].GetOveruseRate(), packageLeft.GetOveruseRate())
}

const (
	DURATION_FIELD    = 1
	PRICE_FIELD       = 2
	OVERUSE_FIELDS    = 3
	CU_FIELD          = 4
	SERVICERS_FIELD   = 5
	NAME_FIELD        = 6
	DESCRIPTION_FIELD = 7
	TYPE_FIELD        = 8
)

func TestInvalidPackageAddition(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// test invalid package addition
	tests := []struct {
		name       string
		fieldIndex int
	}{
		{"InvalidDurationTest", 1},
		{"InvalidPriceTest", 2},
		{"InvalidOveruseTest", 3},
		{"InvalidCuTest", 4},
		{"InvalidServicersToPairTest", 5},
		{"InvalidNameTest", 6},
		{"InvalidDescriptionTest", 7},
		{"InvalidTypeTest", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create a test package
			packageToTest := CreateTestPackages(1, false)

			// each test, change one field to an invalid value
			switch tt.fieldIndex {
			case DURATION_FIELD:
				packageToTest[0].Duration = 0
			case PRICE_FIELD:
				packageToTest[0].Price = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())
			case OVERUSE_FIELDS:
				packageToTest[0].AllowOveruse = true
				packageToTest[0].OveruseRate = 0
			case CU_FIELD:
				packageToTest[0].ComputeUnits = 0
			case SERVICERS_FIELD:
				packageToTest[0].ServicersToPair = 1
			case NAME_FIELD:
				packageToTest[0].Name = strings.Repeat("a", types.MAX_LEN_PACKAGE_NAME+1)
			case DESCRIPTION_FIELD:
				packageToTest[0].Description = strings.Repeat("a", types.MAX_LEN_PACKAGE_DESCRIPTION+1)
			case TYPE_FIELD:
				packageToTest[0].Type = strings.Repeat("a", types.MAX_LEN_PACKAGE_TYPE+1)
			}

			// simulate a package proposal - should fail
			err := testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, packageToTest)
			require.NotNil(t, err)
		})
	}
}
