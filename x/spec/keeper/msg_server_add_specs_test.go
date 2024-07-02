package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/x/spec/keeper"
	"github.com/lavanet/lava/x/spec/types"
	"github.com/stretchr/testify/require"

	testcommon "github.com/lavanet/lava/testutil/common"
)

func TestSpecAddPermissions(t *testing.T) {
	ts := newTester(t)
	spec := testcommon.CreateMockSpec()
	spec.DataReliabilityEnabled = false
	contributorPercentage := math.LegacyMustNewDecFromStr("0.5")
	spec.ContributorPercentage = &contributorPercentage

	msg := types.MsgAddSpecs{}
	msg.Specs = append(msg.Specs, spec)

	// adding spec from user 1
	User1, _ := ts.AddAccount("user", 0, 10000)
	msg.Creator = User1.Addr.String()
	_, err := ts.Servers.SpecServer.AddSpecs(ts.Ctx, &msg)
	require.NoError(t, err)

	// trying to add same spec from user 2
	User2, _ := ts.AddAccount("user", 1, 10000)
	msg.Creator = User2.Addr.String()
	_, err = ts.Servers.SpecServer.AddSpecs(ts.Ctx, &msg)
	require.Error(t, err)

	// adding spec from user 1
	msg.Creator = User1.Addr.String()
	_, err = ts.Servers.SpecServer.AddSpecs(ts.Ctx, &msg)
	require.NoError(t, err)

	val, found := ts.Keepers.Spec.GetSpec(ts.Ctx, spec.Index)
	require.True(t, found)
	require.True(t, val.ContributorPercentage.Equal(math.LegacyMustNewDecFromStr(keeper.USER_SPEC_CONTRIBUTION)))

	// add the spec from gov
	msg.Creator = ts.Keepers.Spec.GetAuthority()
	_, err = ts.Servers.SpecServer.AddSpecs(ts.Ctx, &msg)
	require.NoError(t, err)

	// try to add again from user 1
	msg.Creator = User1.Addr.String()
	_, err = ts.Servers.SpecServer.AddSpecs(ts.Ctx, &msg)
	require.Error(t, err)

	val, found = ts.Keepers.Spec.GetSpec(ts.Ctx, spec.Index)
	require.True(t, found)
	require.False(t, val.ContributorPercentage.Equal(math.LegacyMustNewDecFromStr(keeper.USER_SPEC_CONTRIBUTION)))
}
