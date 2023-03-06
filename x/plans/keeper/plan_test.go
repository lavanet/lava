package keeper_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	commontypes "github.com/lavanet/lava/common/types"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/testutil/nullify"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"github.com/lavanet/lava/x/plans/keeper"
	"github.com/lavanet/lava/x/plans/types"
	"github.com/stretchr/testify/require"
)

// Prevent strconv unused error
var _ = strconv.IntSize

type testStruct struct {
	ctx     context.Context
	keepers *testkeeper.Keepers
}
func (ts *testStruct) advanceEpochUntilStale() {
	block := sdk.UnwrapSDKContext(ts.ctx).BlockHeight() + int64(commontypes.STALE_ENTRY_TIME+1)
	for block > sdk.UnwrapSDKContext(ts.ctx).BlockHeight() {
		ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	}
}

func createNPlanEntry(keeper *keeper.Keeper, ctx sdk.Context, n int) []types.Plan {
	items := make([]types.Plan, n)
	for i := range items {
		items[i].Index = strconv.Itoa(i)

		keeper.AddPlan(ctx, items[i])
	}
	return items
}

func TestPlanEntryGet(t *testing.T) {
	keeper, ctx := testkeeper.PlanKeeper(t)
	items := createNPlanEntry(keeper, ctx, 10)
	for _, item := range items {
		tempPlan, found := keeper.FindPlan(ctx, item.GetIndex(), uint64(ctx.BlockHeight()))
		require.True(t, found)
		require.Equal(t,
			nullify.Fill(&item),
			nullify.Fill(&tempPlan),
		)
	}
}

// Function to create an array of test plans. Can output an array with plans with the same ID
func CreateTestPlans(planAmount uint64, withSameIndex bool, startIndex uint64) []types.Plan {
	testPlans := []types.Plan{}

	// create dummy plans in a loop according to planAmount
	for i := startIndex; i < startIndex+planAmount; i++ {
		// create distinct plan index by the loop counter
		planIndex := "mockPlan" + strconv.FormatUint(i, 10)
		overuseRate := uint64(10)

		// create dummy plan and append to the testPlans array
		dummyPlan := types.Plan{
			Index:                    planIndex,
			Name:                     "test plan",
			Description:              "plan to test",
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
		testPlans = append(testPlans, dummyPlan)

		// if we need to create a plan with the same index, create an additional one
		// with a different overuseRate and append it to testPlans (we increase the
		// counter so we won't get more plans than planAmount)
		if withSameIndex {
			overuseRate2 := uint64(15)
			dummyPlan2 := dummyPlan
			dummyPlan2.OveruseRate = overuseRate2
			testPlans = append(testPlans, dummyPlan2)
		}
	}

	return testPlans
}

// Test that the process of: plan is added, an update is added, stale version
// is removed works correctly. Make sure that a stale plan with subs is not removed
func TestPlanAdditionDifferentEpoch(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create plans (both plans have the same ID. They only differ in the overuseRate field)
	testPlans := CreateTestPlans(1, true, uint64(0))

	// simulate a plan proposal of the first plans
	err := testkeeper.SimulatePlansProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, []types.Plan{testPlans[0]})
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// simulate a plan proposal of the second plans
	err = testkeeper.SimulatePlansProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, []types.Plan{testPlans[1]})
	require.Nil(t, err)

	// get the plan storage and verify that there are two plans in the plan storage
	plansIndices := ts.keepers.Plans.GetAllPlanIndices(sdk.UnwrapSDKContext(ts.ctx))
	require.Equal(t, 1, len(plansIndices))

	// verify that testPlans[1] is the latest plan version (its index should be first in storageIndexList)
	planLatestVersion, found := ts.keepers.Plans.FindPlan(
		sdk.UnwrapSDKContext(ts.ctx),
		plansIndices[0],
		uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()),
	)
	require.True(t, found)
	require.Equal(t, testPlans[1].OveruseRate, planLatestVersion.GetOveruseRate())
}

// Test that if two plans with the same index are added in the same epoch then we keep only the latest one
func TestUpdatePlanInSameEpoch(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create plans (both plans have the same ID. They only differ in the overuseRate field)
	testPlans := CreateTestPlans(1, true, uint64(0))

	// simulate a proposal of the plans
	err := testkeeper.SimulatePlansProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlans)
	require.Nil(t, err)

	// verify the latest one is kept (testPlans[1] that is the last element in the testPlans array)
	planLatestVersion, found := ts.keepers.Plans.FindPlan(
		sdk.UnwrapSDKContext(ts.ctx),
		testPlans[0].GetIndex(),
		uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight()),
	)
	require.True(t, found)
	require.Equal(t, testPlans[1].GetOveruseRate(), planLatestVersion.GetOveruseRate())
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

// Test that the plan verification before adding it to the plan storage is working correctly
func TestInvalidPlanAddition(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// test invalid plan addition
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
			// create a test plans
			planToTest := CreateTestPlans(1, false, uint64(0))

			// each test, change one field to an invalid value
			switch tt.fieldIndex {
			case DURATION_FIELD:
				planToTest[0].Duration = 0
			case PRICE_FIELD:
				planToTest[0].Price = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())
			case OVERUSE_FIELDS:
				planToTest[0].AllowOveruse = true
				planToTest[0].OveruseRate = 0
			case CU_FIELD:
				planToTest[0].ComputeUnits = 0
			case SERVICERS_FIELD:
				planToTest[0].ServicersToPair = 1
			case NAME_FIELD:
				planToTest[0].Name = strings.Repeat("a", types.MAX_LEN_PACKAGE_NAME+1)
			case DESCRIPTION_FIELD:
				planToTest[0].Description = strings.Repeat("a", types.MAX_LEN_PACKAGE_DESCRIPTION+1)
			case TYPE_FIELD:
				planToTest[0].Type = strings.Repeat("a", types.MAX_LEN_PACKAGE_TYPE+1)
			}

			// simulate a plan proposal - should fail
			err := testkeeper.SimulatePlansProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, planToTest)
			require.NotNil(t, err)
		})
	}
}

const (
	TEST_PACKAGES_WITH_SAME_ID_AMOUNT      = 3
	TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT = 5
)

// Test multiple plan addition and removals
func TestMultiplePlansAdditions(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create plans (both plans which have the same ID and different ID)
	testPlansWithDifferentIDs := CreateTestPlans(TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT, false, uint64(0))
	testPlansWithSameIDs := CreateTestPlans(TEST_PACKAGES_WITH_SAME_ID_AMOUNT, true, uint64(TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT+1))

	// simulate a plan proposal of testPlansWithDifferentIDs
	err := testkeeper.SimulatePlansProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlansWithDifferentIDs)
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// simulate a plan proposal of testPlansWithSameIDs
	err = testkeeper.SimulatePlansProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlansWithSameIDs)
	require.Nil(t, err)

	// check there are enough plans in the storage (should not be
	// TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT+2*TEST_PACKAGES_WITH_SAME_ID_AMOUNT) since we
	// propose the duplicate plans in a single block so only the latest are kept
	plansIndices := ts.keepers.Plans.GetAllPlanIndices(sdk.UnwrapSDKContext(ts.ctx))
	require.Equal(t, TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT+TEST_PACKAGES_WITH_SAME_ID_AMOUNT, len(plansIndices))
}

// Test that proposes two valid plans and an invalid one and checks that none have passed
func TestProposeBadAndGoodPlans(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create plans
	testPlans := CreateTestPlans(3, false, uint64(0))

	// make one of the plans invalid
	testPlans[2].ComputeUnits = 0

	// simulate a plan proposal of testPlans (note, inside SimulatePlansProposal
	// it fails the test when a plan is invalid. So we avoid checking the error to
	// make sure later there are no plans in the storage)
	_ = testkeeper.SimulatePlansProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlans)

	// check there are no plans in the storage
	plansIndices := ts.keepers.Plans.GetAllPlanIndices(sdk.UnwrapSDKContext(ts.ctx))
	require.Equal(t, 0, len(plansIndices))
}

// Test that creates 3 versions of a plan and checks the deletion of the first two
func TestPlansDeletion(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// create plans (creates two plans with the same index but different overuse rate)
	testPlans := CreateTestPlans(1, true, uint64(0))

	// save the first plan in the KVstore
	err := ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[0])
	firstPlanBlockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	testPlans[0].Block = firstPlanBlockHeight
	require.Nil(t, err)

	// increase the first plans' refCount
	firstPlanFromStore, found := ts.keepers.Plans.GetPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[0].GetIndex())
	require.True(t, found)
	require.Equal(t, testPlans[0], firstPlanFromStore)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// save the second plan in the KVstore
	err = ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[1])
	secondPlanBlockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	testPlans[1].Block = secondPlanBlockHeight
	require.Nil(t, err)

	// increase the second plans' refCount
	secondPlanFromStore, found := ts.keepers.Plans.GetPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[1].GetIndex())
	require.True(t, found)
	require.Equal(t, testPlans[1], secondPlanFromStore)

	// advance enough epochs so the first two packages will be stale
	ts.advanceEpochUntilStale()

	// create an additional plan and add it to the store to trigger plan deletion code
	newPlan := testPlans[1]
	newPlan.OveruseRate += 20
	newPlanBlockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	newPlan.Block = newPlanBlockHeight
	err = ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), newPlan)
	require.Nil(t, err)

	// check that the plans were not(!) deleted since their refCount is > 0
	firstPlanFromStore, found = ts.keepers.Plans.FindPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[0].GetIndex(), firstPlanBlockHeight)
	require.True(t, found)
	require.Equal(t, testPlans[0], firstPlanFromStore)
	secondPlanFromStore, found = ts.keepers.Plans.FindPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[1].GetIndex(), secondPlanBlockHeight)
	require.True(t, found)
	require.Equal(t, testPlans[1], secondPlanFromStore)

	// decrease the old plans' refCount
	found = ts.keepers.Plans.PutPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[0].GetIndex(), firstPlanBlockHeight)
	require.True(t, found)
	found = ts.keepers.Plans.PutPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[1].GetIndex(), secondPlanBlockHeight)
	require.True(t, found)

	// advance an epoch and create an newer plan to add (and trigger the plan deletion)
	ts.advanceEpochUntilStale()

	newerPlanBlockHeight := uint64(sdk.UnwrapSDKContext(ts.ctx).BlockHeight())
	newerPlan := newPlan
	newerPlan.OveruseRate += 20
	newerPlan.Block = newerPlanBlockHeight
	err = ts.keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ts.ctx), newerPlan)
	require.Nil(t, err)

	// check that the old plans were removed
	_, found = ts.keepers.Plans.FindPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[0].GetIndex(), firstPlanBlockHeight)
	require.False(t, found)
	_, found = ts.keepers.Plans.FindPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[1].GetIndex(), secondPlanBlockHeight)
	require.False(t, found)

	// get the new and newer plans from the store
	newPlanFromStore, found := ts.keepers.Plans.FindPlan(sdk.UnwrapSDKContext(ts.ctx), newPlan.GetIndex(), newPlanBlockHeight)
	require.True(t, found)
	require.Equal(t, newPlan, newPlanFromStore)
	newerPlanFromStore, found := ts.keepers.Plans.FindPlan(sdk.UnwrapSDKContext(ts.ctx), newerPlan.GetIndex(), newerPlanBlockHeight)
	require.True(t, found)
	require.Equal(t, newerPlan, newerPlanFromStore)
}
