//go:build ignore

package vx_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	keepertest "github.com/lavanet/lava/testutil/keeper"
	v020 "github.com/lavanet/lava/x/spec/migrations/v0.2.0"
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

func (suite *UpgradeTestSuite) TestAddingBlockLastUpdated() {
	_, keepers, ctx := keepertest.InitAllKeepers(suite.T())
	sdkctx := sdk.UnwrapSDKContext(ctx)
	suite.T().Log("TestAddingBlockLastUpdated")

	migrator := v020.NewMigrator(keepers.Spec)
	migrator.MigrateToV020(sdkctx)

	allSpec := keepers.Spec.GetAllSpec(sdk.UnwrapSDKContext(ctx))
	for _, spec := range allSpec {
		suite.Require().Equal(0, spec.BlockLastUpdated)
	}
}
