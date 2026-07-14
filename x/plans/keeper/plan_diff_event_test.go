package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/testutil/common"
	testkeeper "github.com/lavanet/lava/v5/testutil/keeper"
	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/x/plans/types"
	"github.com/stretchr/testify/require"
)

// getLastPlanAddEvent returns the attributes of the most recent plan-add event
func getLastPlanAddEvent(ctx sdk.Context) (attributes map[string]string, found bool) {
	eventName := utils.EventPrefix + types.PlanAddEventName
	for _, event := range ctx.EventManager().Events() {
		if event.Type != eventName {
			continue
		}
		found = true
		attributes = map[string]string{}
		for _, attr := range event.Attributes {
			attributes[attr.Key] = attr.Value
		}
	}
	return attributes, found
}

// TestPlanUpdateEventReflectsChanges verifies that when a plans-add proposal
// updates an existing plan, the emitted event reflects what changed
func TestPlanUpdateEventReflectsChanges(t *testing.T) {
	ts := newTester(t)

	plan := common.CreateMockPlan()
	err := testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []types.Plan{plan}, false)
	require.NoError(t, err)

	// a brand new plan: no "changes" attribute in the event
	attributes, found := getLastPlanAddEvent(ts.Ctx)
	require.True(t, found)
	_, hasChanges := attributes["changes"]
	require.False(t, hasChanges)

	ts.AdvanceEpoch()

	// propose an update of the plan (same index, new version)
	update := common.CreateMockPlan()
	update.Description = "an updated description"
	update.PlanPolicy.EpochCuLimit += 50
	err = testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []types.Plan{update}, false)
	require.NoError(t, err)

	attributes, found = getLastPlanAddEvent(ts.Ctx)
	require.True(t, found)
	require.Contains(t, attributes, "changes")
	require.Contains(t, attributes["changes"], `description: "plan for testing" -> "an updated description"`)
	require.Contains(t, attributes["changes"], "plan_policy.epoch_cu_limit: 10000 -> 10050")

	ts.AdvanceEpoch()

	// propose the same plan content again: the event must state nothing changed
	err = testkeeper.SimulatePlansAddProposal(ts.Ctx, ts.Keepers.Plans, []types.Plan{update}, false)
	require.NoError(t, err)

	attributes, found = getLastPlanAddEvent(ts.Ctx)
	require.True(t, found)
	require.Contains(t, attributes, "changes")
	require.Contains(t, attributes["changes"], "none")
}
