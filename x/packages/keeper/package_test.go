package keeper_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/common"
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

func createNPackageEntry(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Package {
	items := make([]types.Package, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.AddPackage(ctx, items[i])
	}
	return items
}

func TestPackageEntryGet(t *testing.T) {
	keeper, ctx := testkeeper.PackagesKeeper(t)
	items := createNPackageEntry(keeper, ctx, 10)
	for _, item := range items {
		rst, found, _ := keeper.GetPackageForBlock(ctx,
			item.GetIndex(), uint64(ctx.BlockHeight()),
		)
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&rst),
		)
	}
}

func TestPackageEntryRemove(t *testing.T) {
	keeper, ctx := testkeeper.PackagesKeeper(t)
	items := createNPackageEntry(keeper, ctx, 10)
	for _, item := range items {
		keeper.RemovePackage(ctx,
			item.GetIndex(),
		)
		_, found, _ := keeper.GetPackageForBlock(ctx,
			item.GetIndex(), uint64(ctx.BlockHeight()),
		)
		require.False(t, found)
	}
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
			Index:                    packageIndex,
			Name:                     "test package",
			Description:              "package to test",
			Type:                     "rpc",
			Duration:                 200,
			Block:                    100,
			Price:                    sdk.NewCoin("ulava", sdk.OneInt()),
			ComputeUnits:             1000,
			ComputeUnitsPerEpoch:     100,
			ServicersToPair:          3,
			AllowOveruse:             true,
			OveruseRate:              overuseRate,
			AnnualDiscountPercentage: 20,
		}
		testPackages = append(testPackages, dummyPackage)

		// if we need to create a package with the same index, create an additional one with a different overuseRate and append it to testPackages (we increase the counter so we won't get more packages than packageAmount)
		if withSameIndex {
			overuseRate2 := uint64(15)
			dummyPackage2 := dummyPackage
			dummyPackage2.OveruseRate = overuseRate2
			testPackages = append(testPackages, dummyPackage2)
		}
	}

	return testPackages
}

// Test that the process of: package is added, an update is added, stale version is removed works correctly. Make sure that a stale package with subs is not removed
func TestPackageAdditionDifferentEpoch(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create packages (both packages have the same ID. They only differ in the overuseRate field)
	testPackages := CreateTestPackages(1, true)

	// simulate a package proposal of the first package
	err := testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, []types.Package{testPackages[0]})
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// simulate a package proposal of the second package
	err = testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, []types.Package{testPackages[1]})
	require.Nil(t, err)

	// get the package storage and verify that there are two packages in the package storage
	packages := getAllEntriesFromStorage(t, ts)
	require.Equal(t, 2, len(packages))

	// verify that testPackages[1] is the latest package version (its index should be first in storageIndexList)
	packageLatestVersion, err := ts.keepers.Packages.GetPackageLatestVersion(sdk.UnwrapSDKContext(ts.ctx), packages[0].GetIndex())
	require.Equal(t, testPackages[1].OveruseRate, packageLatestVersion.GetOveruseRate())
}

// Test that if two packages with the same index are added in the same epoch then we keep only the latest one
func TestUpdatePackageInSameEpoch(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create packages (both packages have the same ID. They only differ in the overuseRate field)
	testPackages := CreateTestPackages(1, true)

	// simulate a proposal of the packages
	err := testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, testPackages)
	require.Nil(t, err)

	// test that there's a single package in the storage
	packages := ts.keepers.Packages.GetAllPackageVersions(sdk.UnwrapSDKContext(ts.ctx), testPackages[0].GetIndex())
	require.Equal(t, 1, len(packages))

	// verify it's the latest one (testPackages[1] that is the last element in the testPackages array)
	packageLeft, err := ts.keepers.Packages.GetPackageLatestVersion(sdk.UnwrapSDKContext(ts.ctx), testPackages[0].GetIndex())
	require.Nil(t, err)
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

// Test that the package verification before adding it to the package storage is working correctly
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

const (
	TEST_PACKAGES_WITH_SAME_ID_AMOUNT      = 3
	TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT = 5
)

// Test multiple package addition and removals
func TestMultiplePackagesAdditions(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create packages (both packages which have the same ID and different ID)
	testPackagesWithDifferentIDs := CreateTestPackages(TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT, false)
	testPackagesWithSameIDs := CreateTestPackages(TEST_PACKAGES_WITH_SAME_ID_AMOUNT, true)

	// simulate a package proposal of testPackagesWithDifferentIDs
	err := testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, testPackagesWithDifferentIDs)
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// simulate a package proposal of testPackagesWithSameIDs
	err = testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, testPackagesWithSameIDs)
	require.Nil(t, err)

	// check there are enough packages in the storage (should not be TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT+2*(TEST_PACKAGES_WITH_SAME_ID_AMOUNT)) since we propose the duplicate packages in a single epoch so only the latest are kept
	packages := getAllEntriesFromStorage(t, ts)
	require.Equal(t, TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT+TEST_PACKAGES_WITH_SAME_ID_AMOUNT, len(packages))
}

// Test that proposes two valid packages and an invalid one and checks that none have passed
func TestProposeBadAndGoodPackages(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create packages
	testPackages := CreateTestPackages(3, false)

	// make one of the packages invalid
	testPackages[2].ComputeUnits = 0

	// simulate a package proposal of testPackages (note, inside SimulatePackageProposal it fails the test when a package is invalid. So we avoid checking the error to make sure later there are no packages in the storage)
	_ = testkeeper.SimulatePackageProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages, testPackages)

	// check there are enough packages in the storage (should not be TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT+2*(TEST_PACKAGES_WITH_SAME_ID_AMOUNT)) since we propose the duplicate packages in a single epoch so only the latest are kept
	packages := getAllEntriesFromStorage(t, ts)
	require.Equal(t, 0, len(packages))
}

// Helper function to get all the entries of all indices using the uniqueIndex list
func getAllEntriesFromStorage(t *testing.T, ts *testStruct) []types.Package {
	uniqueIndices := common.GetAllFixationEntryUniqueIndex(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Packages.GetStoreKey(), ts.keepers.Packages.GetCdc(), types.UniqueIndexKeyPrefix())
	packages := []types.Package{}
	for _, uniqueIndex := range uniqueIndices {
		uniqueIndexAllVersionsPackages := ts.keepers.Packages.GetAllPackageVersions(sdk.UnwrapSDKContext(ts.ctx), uniqueIndex.GetUniqueIndex())
		packages = append(packages, uniqueIndexAllVersionsPackages...)
	}

	return packages
}
