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
	projectstypes "github.com/lavanet/lava/x/projects/types"
	"github.com/stretchr/testify/require"
)

type testStruct struct {
	ctx     context.Context
	keepers *testkeeper.Keepers
}

func (ts *testStruct) advanceEpochUntilStale() {
	block := sdk.UnwrapSDKContext(ts.ctx).BlockHeight() + commontypes.STALE_ENTRY_TIME
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

// createTestPlans returns a slice of plans for testing
func createTestPlans(planAmount int, withSameIndex bool, startIndex int) []types.Plan {
	testPlans := []types.Plan{}
	policy := projectstypes.Policy{
		TotalCuLimit:       1000,
		EpochCuLimit:       100,
		MaxProvidersToPair: 3,
	}

	for i := startIndex; i < startIndex+planAmount; i++ {
		planIndex := "mockplan" + strconv.Itoa(i)
		overuseRate := uint64(10)

		dummyPlan := types.Plan{
			Index:                    planIndex,
			Description:              "plan to test",
			Type:                     "rpc",
			Block:                    100,
			Price:                    sdk.NewCoin("ulava", sdk.OneInt()),
			PlanPolicy:               policy,
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

	// advance an epoch and create a plan
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	testPlans := createTestPlans(1, true, 0)

	// simulate a plan proposal of the first plans
	err := testkeeper.SimulatePlansAddProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, []types.Plan{testPlans[0]})
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// simulate a plan proposal of the second plans
	err = testkeeper.SimulatePlansAddProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, []types.Plan{testPlans[1]})
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

	// advance an epoch and create a plan
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	testPlans := createTestPlans(1, true, 0)

	// simulate a proposal of the plans
	err := testkeeper.SimulatePlansAddProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlans)
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
	PRICE_FIELD = iota + 1
	OVERUSE_FIELDS
	CU_FIELD_TOTAL
	CU_FIELD_EPOCH
	SERVICERS_FIELD
	DESCRIPTION_FIELD
	TYPE_FIELD
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
		{"InvalidPriceTest", 1},
		{"InvalidOveruseTest", 2},
		{"InvalidTotalCuTest", 3},
		{"InvalidEpochCuTest", 4},
		{"InvalidServicersToPairTest", 5},
		{"InvalidDescriptionTest", 6},
		{"InvalidTypeTest", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			planToTest := createTestPlans(1, false, 0)

			// each test, change one field to an invalid value
			switch tt.fieldIndex {
			case PRICE_FIELD:
				planToTest[0].Price = sdk.NewCoin(epochstoragetypes.TokenDenom, sdk.ZeroInt())
			case OVERUSE_FIELDS:
				planToTest[0].AllowOveruse = true
				planToTest[0].OveruseRate = 0
			case CU_FIELD_TOTAL:
				planToTest[0].PlanPolicy.TotalCuLimit = 0
			case CU_FIELD_EPOCH:
				planToTest[0].PlanPolicy.EpochCuLimit = 0
			case SERVICERS_FIELD:
				planToTest[0].PlanPolicy.MaxProvidersToPair = 1
			case DESCRIPTION_FIELD:
				planToTest[0].Description = strings.Repeat("a", types.MAX_LEN_PACKAGE_DESCRIPTION+1)
			case TYPE_FIELD:
				planToTest[0].Type = strings.Repeat("a", types.MAX_LEN_PACKAGE_TYPE+1)
			}

			// simulate a plan proposal - should fail
			err := testkeeper.SimulatePlansAddProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, planToTest)
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
	testPlansWithDifferentIDs := createTestPlans(TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT, false, 0)
	testPlansWithSameIDs := createTestPlans(TEST_PACKAGES_WITH_SAME_ID_AMOUNT, true, TEST_PACKAGES_WITH_DIFFERENT_ID_AMOUNT)

	// simulate a plan proposal of testPlansWithDifferentIDs
	err := testkeeper.SimulatePlansAddProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlansWithDifferentIDs)
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)

	// simulate a plan proposal of testPlansWithSameIDs
	err = testkeeper.SimulatePlansAddProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlansWithSameIDs)
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

	// advance an epoch and create a plan
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	testPlans := createTestPlans(3, false, 0)

	// make one of the plans invalid
	testPlans[2].PlanPolicy.TotalCuLimit = 0

	// simulate a plan proposal of testPlans (note, inside SimulatePlansAddProposal
	// it fails the test when a plan is invalid. So we avoid checking the error to
	// make sure later there are no plans in the storage)
	_ = testkeeper.SimulatePlansAddProposal(sdk.UnwrapSDKContext(ts.ctx), ts.keepers.Plans, testPlans)

	// check there are no plans in the storage
	plansIndices := ts.keepers.Plans.GetAllPlanIndices(sdk.UnwrapSDKContext(ts.ctx))
	require.Equal(t, 0, len(plansIndices))
}

// Test that creates 3 versions of a plan and checks the removal of the first two
func TestPlansStaleRemoval(t *testing.T) {
	// setup the testStruct
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch and create a plan
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	testPlans := createTestPlans(1, true, 0)

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
	ts.keepers.Plans.PutPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[0].GetIndex(), firstPlanBlockHeight)
	ts.keepers.Plans.PutPlan(sdk.UnwrapSDKContext(ts.ctx), testPlans[1].GetIndex(), secondPlanBlockHeight)

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

// Test that creates a plan and then deletes it
func TestPlansDelete(t *testing.T) {
	ts := &testStruct{}
	_, ts.keepers, ts.ctx = testkeeper.InitAllKeepers(t)

	// advance an epoch and create a plan
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ctx := sdk.UnwrapSDKContext(ts.ctx)

	testPlans := createTestPlans(1, true, 0)
	index := testPlans[0].Index

	err := testkeeper.SimulatePlansAddProposal(ctx, ts.keepers.Plans, []types.Plan{testPlans[0]})
	require.Nil(t, err)

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ctx = sdk.UnwrapSDKContext(ts.ctx)
	block1 := uint64(ctx.BlockHeight())

	// verify there is one plan visible
	plansIndices := ts.keepers.Plans.GetAllPlanIndices(ctx)
	require.Equal(t, 1, len(plansIndices))

	// advance an epoch
	ts.ctx = testkeeper.AdvanceEpoch(ts.ctx, ts.keepers)
	ctx = sdk.UnwrapSDKContext(ts.ctx)

	// now delete the plan
	err = testkeeper.SimulatePlansDelProposal(ctx, ts.keepers.Plans, []string{index})
	require.Nil(t, err)

	block2 := uint64(ctx.BlockHeight())

	// verify there are zero plans visible
	plansIndices = ts.keepers.Plans.GetAllPlanIndices(ctx)
	require.Equal(t, 0, len(plansIndices))
	_, found := ts.keepers.Plans.GetPlan(ctx, index)
	require.False(t, found)

	// but the plan is not stale yet, so can be found (until block2)
	_, found = ts.keepers.Plans.FindPlan(ctx, index, block1)
	require.True(t, found)
	_, found = ts.keepers.Plans.FindPlan(ctx, index, block2)
	require.False(t, found)

	// advance epoch until the plan becomes stale
	ts.advanceEpochUntilStale()
	ctx = sdk.UnwrapSDKContext(ts.ctx)

	// because testutils.AdvanceBlock() and testutils.AdvanceEpoch() are sloppy
	ts.keepers.Plans.BeginBlock(ctx)

	_, found = ts.keepers.Plans.FindPlan(ctx, index, block1)
	require.False(t, found)
	_, found = ts.keepers.Plans.FindPlan(ctx, index, block2)
	require.False(t, found)

	// fail attempt to delete the plan again
	err = testkeeper.SimulatePlansDelProposal(ctx, ts.keepers.Plans, []string{index})
	require.NotNil(t, err)
}
