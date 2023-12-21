package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/lavanet/lava/testutil/common"
	planstypes "github.com/lavanet/lava/x/plans/types"
	rewardsTypes "github.com/lavanet/lava/x/rewards/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
	subscriptiontypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

const (
	testStake             int64 = 100000
	testBalance           int64 = 100000000
	allocationPoolBalance int64 = 30000000000000
	feeCollectorName            = authtypes.FeeCollectorName
)

type tester struct {
	common.Tester
	plan planstypes.Plan
	spec spectypes.Spec
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTesterRaw(t)}

	ts.addValidators(1)
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	ts.TxCreateValidator(val, math.NewIntFromUint64(uint64(testStake)))

	ts.plan = common.CreateMockPlan()
	monthlyProvidersPool := ts.Keepers.Rewards.TotalPoolTokens(ts.Ctx, rewardsTypes.ProviderRewardsDistributionPool)
	ts.plan.Price.Amount = monthlyProvidersPool.QuoRaw(5).AddRaw(5)
	ts.plan.PlanPolicy.EpochCuLimit = monthlyProvidersPool.Uint64() * 5
	ts.plan.PlanPolicy.TotalCuLimit = monthlyProvidersPool.Uint64() * 5
	ts.AddPlan(ts.plan.Index, ts.plan)
	ts.spec = ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")

	return ts
}

func (ts *tester) addValidators(count int) {
	start := len(ts.Accounts(common.VALIDATOR))
	for i := 0; i < count; i++ {
		_, _ = ts.AddAccount(common.VALIDATOR, start+i, testBalance)
	}
}

func (ts *tester) feeCollector() sdk.AccAddress {
	return sdk.AccAddress([]byte(feeCollectorName))
}

// deductParticipationFees calculates the validators and community participation
// fees and returns the providers reward after deducting them
func (ts *tester) DeductParticipationFees(reward math.Int) (updatedReward math.Int, valParticipation math.Int, communityParticipation math.Int) {
	valPerc, communityPerc, err := ts.Keepers.Rewards.CalculateContributionPercentages(ts.Ctx, reward, subscriptiontypes.ModuleName)
	require.Nil(ts.T, err)
	valParticipation = valPerc.MulInt(reward).TruncateInt()
	communityParticipation = communityPerc.MulInt(reward).TruncateInt()
	return reward.Sub(valParticipation).Sub(communityParticipation), valParticipation, communityParticipation
}
