package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/testutil/common"
	testkeeper "github.com/lavanet/lava/testutil/keeper"
	"github.com/lavanet/lava/x/pairing/types"
	projectTypes "github.com/lavanet/lava/x/projects/types"
	subTypes "github.com/lavanet/lava/x/subscription/types"
	"github.com/stretchr/testify/require"
)

func TestStakeClientPairingimmediately(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	var balance int64 = 10000
	consumer := common.CreateNewAccount(ctx, *keepers, balance)
	provider1 := common.CreateNewAccount(ctx, *keepers, balance)
	provider2 := common.CreateNewAccount(ctx, *keepers, balance)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	plan := common.CreateMockPlan()
	keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ctx), plan)

	stake := balance / 10
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	common.StakeAccount(t, ctx, *keepers, *servers, provider1, spec, stake)
	common.StakeAccount(t, ctx, *keepers, *servers, provider2, spec, stake)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	common.BuySubscription(t, ctx, *keepers, *servers, consumer, plan.Index)

	epoch := keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ctx))

	// check pairing in the same epoch
	_, err := keepers.Pairing.VerifyPairingData(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr, epoch)
	require.Nil(t, err)

	pairing, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, consumer.Addr)
	require.Nil(t, err)

	_, err = keepers.Pairing.VerifyPairing(ctx, &types.QueryVerifyPairingRequest{ChainID: spec.Index, Client: consumer.Addr.String(), Provider: pairing[0].Address, Block: epoch})
	require.Nil(t, err)
}

func TestCreateProjectAddKey(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	var balance int64 = 10000
	subscriptionConsumer := common.CreateNewAccount(ctx, *keepers, balance)
	developer := common.CreateNewAccount(ctx, *keepers, balance)
	provider1 := common.CreateNewAccount(ctx, *keepers, balance)
	provider2 := common.CreateNewAccount(ctx, *keepers, balance)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	plan := common.CreateMockPlan()
	keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ctx), plan)

	stake := balance / 10
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	common.StakeAccount(t, ctx, *keepers, *servers, provider1, spec, stake)
	common.StakeAccount(t, ctx, *keepers, *servers, provider2, spec, stake)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	common.BuySubscription(t, ctx, *keepers, *servers, subscriptionConsumer, plan.Index)
	projects := keepers.Projects.GetAllProjectsForSubscription(sdk.UnwrapSDKContext(ctx), subscriptionConsumer.Addr.String())

	// takes effect retroactively in the current epoch
	_, err := servers.ProjectServer.AddKeys(ctx, &projectTypes.MsgAddKeys{
		Creator:     subscriptionConsumer.Addr.String(),
		Project:     projects[0],
		ProjectKeys: []projectTypes.ProjectKey{projectTypes.ProjectDeveloperKey(developer.Addr.String())},
	})
	require.Nil(t, err)

	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	// should fail, the key is in use
	_, err = servers.SubscriptionServer.AddProject(ctx,
		&subTypes.MsgAddProject{
			Creator: subscriptionConsumer.Addr.String(),
			ProjectData: projectTypes.ProjectData{
				Name:        "test",
				Enabled:     true,
				ProjectKeys: []projectTypes.ProjectKey{projectTypes.ProjectDeveloperKey(developer.Addr.String())},
				Policy:      nil,
			},
		})

	require.NotNil(t, err)

	epoch := keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ctx))

	// check pairing in the same epoch (key added retroactively to this epoch)
	_, err = keepers.Pairing.VerifyPairingData(sdk.UnwrapSDKContext(ctx), spec.Index, developer.Addr, epoch)
	require.Nil(t, err)

	pairing, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, developer.Addr)
	require.Nil(t, err)

	_, err = keepers.Pairing.VerifyPairing(ctx, &types.QueryVerifyPairingRequest{ChainID: spec.Index, Client: developer.Addr.String(), Provider: pairing[0].Address, Block: uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())})
	require.Nil(t, err)
}

func TestAddKeyCreateProject(t *testing.T) {
	servers, keepers, ctx := testkeeper.InitAllKeepers(t)

	// init keepers state
	var balance int64 = 10000
	subscriptionConsumer := common.CreateNewAccount(ctx, *keepers, balance)
	developer := common.CreateNewAccount(ctx, *keepers, balance)
	provider1 := common.CreateNewAccount(ctx, *keepers, balance)
	provider2 := common.CreateNewAccount(ctx, *keepers, balance)

	spec := common.CreateMockSpec()
	keepers.Spec.SetSpec(sdk.UnwrapSDKContext(ctx), spec)

	plan := common.CreateMockPlan()
	keepers.Plans.AddPlan(sdk.UnwrapSDKContext(ctx), plan)

	stake := balance / 10
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	common.StakeAccount(t, ctx, *keepers, *servers, provider1, spec, stake)
	common.StakeAccount(t, ctx, *keepers, *servers, provider2, spec, stake)

	ctx = testkeeper.AdvanceEpoch(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)
	ctx = testkeeper.AdvanceBlock(ctx, keepers)

	common.BuySubscription(t, ctx, *keepers, *servers, subscriptionConsumer, plan.Index)
	projects := keepers.Projects.GetAllProjectsForSubscription(sdk.UnwrapSDKContext(ctx), subscriptionConsumer.Addr.String())

	// should work, the key should take effect now
	_, err := servers.SubscriptionServer.AddProject(ctx,
		&subTypes.MsgAddProject{
			Creator: subscriptionConsumer.Addr.String(),
			ProjectData: projectTypes.ProjectData{
				Name:        "test",
				Enabled:     true,
				ProjectKeys: []projectTypes.ProjectKey{projectTypes.ProjectDeveloperKey(developer.Addr.String())},
				Policy:      nil,
			},
		})

	require.Nil(t, err)

	// should fail, takes effect in the next epoch
	_, err = servers.ProjectServer.AddKeys(ctx, &projectTypes.MsgAddKeys{
		Creator:     subscriptionConsumer.Addr.String(),
		Project:     projects[0],
		ProjectKeys: []projectTypes.ProjectKey{projectTypes.ProjectDeveloperKey(developer.Addr.String())},
	})
	require.NotNil(t, err)

	epoch := keepers.Epochstorage.GetEpochStart(sdk.UnwrapSDKContext(ctx))

	// check pairing in the same epoch
	ctx = testkeeper.AdvanceEpoch(ctx, keepers)

	// check pairing in the next epoch
	_, err = keepers.Pairing.VerifyPairingData(sdk.UnwrapSDKContext(ctx), spec.Index, developer.Addr, epoch)
	require.Nil(t, err)

	pairing, err := keepers.Pairing.GetPairingForClient(sdk.UnwrapSDKContext(ctx), spec.Index, developer.Addr)
	require.Nil(t, err)

	_, err = keepers.Pairing.VerifyPairing(ctx, &types.QueryVerifyPairingRequest{ChainID: spec.Index, Client: developer.Addr.String(), Provider: pairing[0].Address, Block: uint64(sdk.UnwrapSDKContext(ctx).BlockHeight())})
	require.Nil(t, err)
}
