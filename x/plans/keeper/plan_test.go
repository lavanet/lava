package keeper_test

import (
	"strconv"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v2/testutil/common"
	testkeeper "github.com/lavanet/lava/v2/testutil/keeper"
	"github.com/lavanet/lava/v2/testutil/nullify"
	"github.com/lavanet/lava/v2/x/plans/types"
	"github.com/stretchr/testify/require"
)

type tester struct {
	common.Tester
}

func newTester(t *testing.T) *tester {
	return &tester{Tester: *common.NewTester(t)}
}

func TestPlanEntryGet(t *testing.T) {
	ts := newTester(t)

	plan := common.CreateMockPlan()
	plan.Block = ts.BlockHeight()

	err := ts.TxProposalAddPlans(plan)
	require.NoError(t, err)

	res, found := ts.FindPlan(plan.Index, ts.BlockHeight())
	require.True(t, found)
	require.Equal(t,
		nullify.Fill(&plan),
		nullify.Fill(&res),
	)
}

// createTestPlans returns a slice of plans for testing
func (ts *tester) createTestPlans(count int, withSameIndex bool, startIndex int) []types.Plan {
	var plans []types.Plan

	policy := types.Policy{
		TotalCuLimit:       1000,
		EpochCuLimit:       100,
		MaxProvidersToPair: 3,
	}

	for i := startIndex; i < startIndex+count; i++ {
		planIndex := "mockplan" + strconv.Itoa(i)
		overuseRate := uint64(10)

		plan := types.Plan{
			Index:                    planIndex,
			Description:              "plan to test",
			Type:                     "rpc",
			Block:                    100,
			Price:                    common.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), 1),
			PlanPolicy:               policy,
			AllowOveruse:             true,
			OveruseRate:              overuseRate,
			AnnualDiscountPercentage: 20,
			ProjectsLimit:            10,
		}

		plans = append(plans, plan)

		// if withSameIndex is true, create an additional plan with a
		// different overuseRate and append it to plans
		if withSameIndex {
			plan.OveruseRate = uint64(15)
			plans = append(plans, plan)
		}
	}

	return plans
}

// TestAddAndUpdateOtherEpoch tests the lifecycle of a plan: add, update,
// old version becomes stale and removed (unless it has subscriptions)
func TestAddAndUpdateOtherEpoch(t *testing.T) {
	ts := newTester(t)
	plans := ts.createTestPlans(1, true, 0)

	// proposal with a plan
	ts.AdvanceEpoch()
	err := ts.TxProposalAddPlans(plans[0])
	require.NoError(t, err)

	// proposal with plan update
	ts.AdvanceEpoch()
	err = ts.TxProposalAddPlans(plans[1])
	require.NoError(t, err)

	indices := ts.Keepers.Plans.GetAllPlanIndices(ts.Ctx)
	require.Equal(t, 1, len(indices))

	// updated plan should be the latest
	plan, found := ts.FindPlan(indices[0], ts.BlockHeight())
	require.True(t, found)
	require.Equal(t, plans[1].OveruseRate, plan.GetOveruseRate())
}

// TestAddAndUpdateSameBlock addding the same plan twice in the same block;
// Only the latter should prevail.
func TestUpdatePlanInSameEpoch(t *testing.T) {
	ts := newTester(t)
	plans := ts.createTestPlans(1, true, 0)

	// proposal with a plan
	ts.AdvanceEpoch()
	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans, false)
	require.NoError(t, err)

	plan, found := ts.FindPlan(plans[0].Index, ts.BlockHeight())
	require.True(t, found)
	require.Equal(t, plans[1].GetOveruseRate(), plan.GetOveruseRate())
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

// TestAddInvalid Plan tests plan verification before addition
func TestAddInvalidPlan(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

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
			plans := ts.createTestPlans(1, false, 0)

			switch tt.fieldIndex {
			case PRICE_FIELD:
				plans[0].Price = common.NewCoin(ts.Keepers.StakingKeeper.BondDenom(ts.Ctx), 0)
			case OVERUSE_FIELDS:
				plans[0].AllowOveruse = true
				plans[0].OveruseRate = 0
			case CU_FIELD_TOTAL:
				plans[0].PlanPolicy.TotalCuLimit = 0
			case CU_FIELD_EPOCH:
				plans[0].PlanPolicy.EpochCuLimit = 0
			case SERVICERS_FIELD:
				plans[0].PlanPolicy.MaxProvidersToPair = 1
			case DESCRIPTION_FIELD:
				plans[0].Description = strings.Repeat("a", types.MAX_LEN_PLAN_DESCRIPTION+1)
			case TYPE_FIELD:
				plans[0].Type = strings.Repeat("a", types.MAX_LEN_PLAN_TYPE+1)
			}

			err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans, false)
			require.Error(t, err)
		})
	}
}

const (
	TEST_PLANS_DIFF_ID = 3
	TEST_PLANS_SAME_ID = 5
)

// TestAddRemove tests multiple plan addition (some in same block)
func TestAddRemove(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

	plans1 := ts.createTestPlans(TEST_PLANS_DIFF_ID, false, 0)
	plans2 := ts.createTestPlans(TEST_PLANS_SAME_ID, true, len(plans1))

	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans1, false)
	require.NoError(t, err)

	ts.AdvanceEpoch()

	err = testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans2, false)
	require.NoError(t, err)

	// plans with same ID added in same block: only 1 (of each pair) prevailed
	indices := ts.Keepers.Plans.GetAllPlanIndices(ts.Ctx)
	require.Equal(t, TEST_PLANS_SAME_ID+TEST_PLANS_DIFF_ID, len(indices))
}

// TestAddBadAndGoodPlans tests a mix of good and bad plans - none passes.
func TestAddBadAndGoodPlans(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

	// create plans and spoil one of them
	plans := ts.createTestPlans(3, false, 0)
	// make one of the plans invalid
	plans[2].PlanPolicy.TotalCuLimit = 0

	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans, false)
	require.Error(t, err)

	indices := ts.Keepers.Plans.GetAllPlanIndices(ts.Ctx)
	require.Equal(t, 0, len(indices))
}

// TestPlansStaleRemoval tests that removal of stale versions
func TestPlansStaleRemoval(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

	plans := ts.createTestPlans(1, true, 0)

	// add 1st plan and keep a reference
	err := ts.TxProposalAddPlans(plans[0])
	require.NoError(t, err)
	plans[0].Block = ts.BlockHeight()
	res, found := ts.Keepers.Plans.GetPlan(ts.Ctx, plans[0].Index)
	require.True(t, found)
	require.Equal(t, plans[0], res)

	ts.AdvanceEpoch()

	// add 2nd plan and keep a reference
	err = ts.TxProposalAddPlans(plans[1])
	require.NoError(t, err)
	plans[1].Block = ts.BlockHeight()
	res, found = ts.Keepers.Plans.GetPlan(ts.Ctx, plans[1].Index)
	require.True(t, found)
	require.Equal(t, plans[1], res)

	ts.AdvanceEpoch()

	// add 3rd plan (so the prevous two would become stale)
	plan := plans[1]
	plan.OveruseRate += 20
	err = ts.TxProposalAddPlans(plan)
	require.NoError(t, err)
	plan.Block = ts.BlockHeight()

	// advance enough epochs to make removed entries become stale
	ts.AdvanceEpochUntilStale()

	// the first two plans should not be deleted (refcount > 0)
	res, found = ts.FindPlan(plans[0].Index, plans[0].Block)
	require.True(t, found)
	require.Equal(t, plans[0], res)
	res, found = ts.FindPlan(plans[1].Index, plans[1].Block)
	require.True(t, found)
	require.Equal(t, plans[1], res)

	// decremenet the old plans' refCount
	ts.Keepers.Plans.PutPlan(ts.Ctx, plans[0].Index, plans[0].Block)
	ts.Keepers.Plans.PutPlan(ts.Ctx, plans[1].Index, plans[1].Block)

	ts.AdvanceEpochUntilStale()
	ts.AdvanceBlock()

	// check that the old plans were removed
	_, found = ts.FindPlan(plans[0].Index, plans[0].Block)
	require.False(t, found)
	_, found = ts.FindPlan(plans[1].Index, plans[1].Block)
	require.False(t, found)

	// verify that the newest is still there
	res, found = ts.FindPlan(plan.Index, plan.Block)
	require.True(t, found)
	require.Equal(t, plan, res)
}

// TestAddAndDelete tests creation and deleteion of plans
func TestAddAndDelete(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

	plans := ts.createTestPlans(1, true, 0)
	index := plans[0].Index

	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, plans[0:1], false)
	require.NoError(t, err)
	indices := ts.Keepers.Plans.GetAllPlanIndices(ts.Ctx)
	require.Equal(t, 1, len(indices))

	ts.AdvanceEpoch()
	block1 := ts.BlockHeight()

	err = testkeeper.SimulatePlansDelProposal(ts.Ctx, ts.Keepers.Plans, []string{index})
	require.NoError(t, err)

	// delete only takes effect next epoch, so for now still visible
	block2 := ts.BlockHeight()
	_, found := ts.FindPlan(index, block2)
	require.True(t, found)

	// advance an epoch (so the delete will take effect)
	ts.AdvanceEpoch()

	// verify there are zero plans visible
	indices = ts.Keepers.Plans.GetAllPlanIndices(ts.Ctx)
	require.Equal(t, 0, len(indices))
	_, found = ts.Keepers.Plans.GetPlan(ts.Ctx, index)
	require.False(t, found)

	// but the plan is not stale yet, so can be found (until block2 + ~epoch)
	_, found = ts.FindPlan(index, block1)
	require.True(t, found)
	_, found = ts.FindPlan(index, block2+30)
	require.False(t, found)

	// advance epoch until the plan becomes stale
	ts.AdvanceEpochUntilStale()
	ts.AdvanceBlock()

	_, found = ts.FindPlan(index, block1)
	require.False(t, found)
	_, found = ts.FindPlan(index, block2)
	require.False(t, found)

	// fail attempt to delete the plan again
	err = testkeeper.SimulatePlansDelProposal(ts.Ctx, ts.Keepers.Plans, []string{index})
	require.Error(t, err)
}

// TestModifyPlan checks that plan modification acts as expected:
// 1. there should be no new version in the planFS (latest plan should have the same block)
// 2. plan price cannot be changed
func TestModifyPlan(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

	plans := ts.createTestPlans(1, false, 0)

	// add a new plan
	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []types.Plan{plans[0]}, false)
	require.NoError(t, err)
	indices := ts.Keepers.Plans.GetAllPlanIndices(ts.Ctx)
	require.Equal(t, 1, len(indices))
	originalPlan, found := ts.Keepers.Plans.GetPlan(ts.Ctx, plans[0].Index)
	require.True(t, found)
	planBlock := originalPlan.Block

	ts.AdvanceEpoch()

	// modify the plan (block should stay the same, change should happen)
	originalPlan.AnnualDiscountPercentage = 42
	err = testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []types.Plan{originalPlan}, true)
	require.NoError(t, err)
	indices = ts.Keepers.Plans.GetAllPlanIndices(ts.Ctx)
	require.Equal(t, 1, len(indices))
	modifiedPlan, found := ts.Keepers.Plans.GetPlan(ts.Ctx, plans[0].Index)
	require.True(t, found)
	require.Equal(t, planBlock, modifiedPlan.Block)
	require.Equal(t, uint64(42), modifiedPlan.AnnualDiscountPercentage)

	// modify the plan by increasing its price. proposal should fail
	originalPlan.Price = originalPlan.Price.AddAmount(math.NewIntFromUint64(1))
	err = testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []types.Plan{originalPlan}, true)
	require.Error(t, err)

	// modify the plan by decreasing its price. proposal should fail
	originalPlan.Price = originalPlan.Price.SubAmount(math.NewIntFromUint64(2))
	err = testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []types.Plan{originalPlan}, true)
	require.Error(t, err)
}
