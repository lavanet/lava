package v4_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	"github.com/stretchr/testify/suite"
)

type UpgradeTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *cosmoscmd.App
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
