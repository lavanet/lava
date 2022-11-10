package v4_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite-hq/cli/ignite/pkg/cosmoscmd"
	"github.com/lavanet/lava/app"
	"github.com/lavanet/lava/relayer/sigs"
	keepertest "github.com/lavanet/lava/testutil/keeper"
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
	_, userAddress := sigs.GenerateFloatingKey()
	_, providerAddress := sigs.GenerateFloatingKey()
	allUniquePayments := keepers.Pairing.GetAllUniquePaymentStorageClientProvider(sdkctx)
	suite.Require().Empty(allUniquePayments)
	keepers.Pairing.AddUniquePaymentStorageClientProvider(sdkctx, "ETH1", 0, userAddress, providerAddress, "test", 50)
	allUniquePayments = keepers.Pairing.GetAllUniquePaymentStorageClientProvider(sdkctx)
	suite.Require().NotEmpty(allUniquePayments)
	// v4.DeleteStoreEntries(sdkctx, "pairing", v4.UniquePaymentStorageClientProviderKeyPrefix)
	// allUniquePayments = keepers.Pairing.GetAllUniquePaymentStorageClientProvider(sdkctx)
	// suite.Require().Empty(allUniquePayments)
}
