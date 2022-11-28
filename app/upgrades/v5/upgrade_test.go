//go:build ignore

package v4_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/conflict/keeper"
	"github.com/lavanet/lava/x/conflict/types"
	"github.com/stretchr/testify/suite"
)

type UpgradeTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app cosmoscmd.App
}

func (suite *UpgradeTestSuite) SetupTestApp() {
	suite.app, suite.ctx = app.TestSetup()
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(UpgradeTestSuite))
}

func (suite *UpgradeTestSuite) TestBody() {
	suite.SetupTestApp() // setup test app
	suite.T().Log("test")
}

func (suite *UpgradeTestSuite) TestDeletingOldPrefixDataFromStore() {
	_, keepers, ctx := keepertest.InitAllKeepers(suite.T())
	sdkctx := sdk.UnwrapSDKContext(ctx)
	suite.T().Log("TestDeletingOldPrefixDataFromStore")

	vote := types.ConflictVote{}
	keepers.Conflict.SetConflictVote(sdkctx, vote)

	allvotes := keepers.Conflict.GetAllConflictVote(sdkctx)
	suite.Require().NotEqual(0, len(allvotes))

	migrator := keeper.NewMigrator(keepers.Conflict)
	migrator.MigrateToV5(sdkctx)

	allvotes = keepers.Conflict.GetAllConflictVote(sdkctx)
	suite.Require().Equal(0, len(allvotes))
}
