package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/lavanet/lava/v2/utils/lavaslices"
	projectTypes "github.com/lavanet/lava/v2/x/projects/types"
	"github.com/stretchr/testify/require"
)

func TestStakeClientPairingimmediately(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(2, 1, 0) // 2 provider, 1 client, default providers-to-pair

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)

	epoch := ts.EpochStart()

	// check pairing in the same epoch
	_, _, err := ts.Keepers.Pairing.VerifyPairingData(ts.Ctx, ts.spec.Index, epoch)
	require.NoError(t, err)

	pairing, err := ts.QueryPairingGetPairing(ts.spec.Index, client1Addr)
	require.NoError(t, err)

	_, err = ts.QueryPairingVerifyPairing(ts.spec.Index, client1Addr, pairing.Providers[0].Address, epoch)
	require.NoError(t, err)
}

func TestCreateProjectAddKey(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 1)    // 0 sub, 0 adm, 1 dev
	ts.setupForPayments(2, 1, 0) // 2 provider, 1 client, default providers-to-pair

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, dev1Addr := ts.Account("dev1")

	// takes effect retroactively in the current epoch

	res1, err := ts.QuerySubscriptionListProjects(client1Addr)
	require.NoError(t, err)
	projects := res1.Projects

	err = ts.TxProjectAddKeys(projects[0], client1Addr, projectTypes.ProjectDeveloperKey(dev1Addr))
	require.NoError(t, err)

	ts.AdvanceBlock()

	projectData := projectTypes.ProjectData{
		Name:        "test",
		Enabled:     true,
		ProjectKeys: lavaslices.Slice(projectTypes.ProjectDeveloperKey(dev1Addr)),
		Policy:      nil,
	}

	// should fail, the key is in use
	err = ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.Error(t, err)

	epoch := ts.EpochStart()

	// check pairing in the same epoch (key added retroactively to this epoch)
	_, _, err = ts.Keepers.Pairing.VerifyPairingData(ts.Ctx, ts.spec.Index, epoch)
	require.NoError(t, err)

	res2, err := ts.QueryPairingGetPairing(ts.spec.Index, dev1Addr)
	require.NoError(t, err)
	pairing := res2.Providers

	_, err = ts.QueryPairingVerifyPairing(ts.spec.Index, dev1Addr, pairing[0].Address, ts.EpochStart())
	require.NoError(t, err)
}

func TestAddKeyCreateProject(t *testing.T) {
	ts := newTester(t)
	ts.SetupAccounts(0, 0, 1)    // 0 sub, 0 adm, 1 dev
	ts.setupForPayments(2, 1, 0) // 2 provider, 1 client, default providers-to-pair

	_, client1Addr := ts.GetAccount(common.CONSUMER, 0)
	_, dev1Addr := ts.Account("dev1")

	devkey := projectTypes.ProjectDeveloperKey(dev1Addr)

	res1, err := ts.QuerySubscriptionListProjects(client1Addr)
	require.NoError(t, err)
	projects := res1.Projects

	projectData := projectTypes.ProjectData{
		Name:        "test",
		Enabled:     true,
		ProjectKeys: lavaslices.Slice(devkey),
		Policy:      nil,
	}

	// should work, the key should take effect now
	err = ts.TxSubscriptionAddProject(client1Addr, projectData)
	require.NoError(t, err)

	// should fail, takes effect in the next epoch
	err = ts.TxProjectAddKeys(projects[0], client1Addr, devkey)
	require.Error(t, err)

	epoch := ts.EpochStart()

	ts.AdvanceEpoch()

	_, _, err = ts.Keepers.Pairing.VerifyPairingData(ts.Ctx, ts.spec.Index, epoch)
	require.NoError(t, err)

	res2, err := ts.QueryPairingGetPairing(ts.spec.Index, dev1Addr)
	require.NoError(t, err)
	pairing := res2.Providers

	_, err = ts.QueryPairingVerifyPairing(ts.spec.Index, dev1Addr, pairing[0].Address, ts.BlockHeight())
	require.NoError(t, err)
}
