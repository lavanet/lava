package fixationstore_test

import (
	"testing"

	"github.com/lavanet/lava/testutil/common"
	planstypes "github.com/lavanet/lava/x/plans/types"
	spectypes "github.com/lavanet/lava/x/spec/types"
)

const (
	testStake   int64 = 100000
	testBalance int64 = 100000000
)

type tester struct {
	common.Tester
	plan planstypes.Plan
	spec spectypes.Spec
}

func newTester(t *testing.T) *tester {
	ts := &tester{Tester: *common.NewTester(t)}

	ts.plan = ts.AddPlan("mock", common.CreateMockPlan()).Plan("mock")
	ts.spec = ts.AddSpec("mock", common.CreateMockSpec()).Spec("mock")

	ts.AdvanceEpoch()

	return ts
}
