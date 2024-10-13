package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/lavanet/lava/v3/testutil/common"
	"github.com/lavanet/lava/v3/x/pairing/keeper"
	"github.com/stretchr/testify/require"
)

func TestMigration4To5(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 0, 3, 0)

	provider0, _ := ts.AddAccount(common.PROVIDER, 0, testBalance)
	provider1, _ := ts.AddAccount(common.PROVIDER, 1, testBalance)

	// stake provider 0
	d := common.MockDescription()
	err := ts.StakeProviderExtra(provider0.GetVaultAddr(), provider0.Addr.String(), ts.Spec(SpecName(0)), testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details)
	require.NoError(ts.T, err)

	// stake provider 1
	err = ts.StakeProviderExtra(provider1.Addr.String(), provider1.Addr.String(), ts.Spec(SpecName(1)), testStake, nil, 0, "mon2", d.Identity, d.Website, d.SecurityContact, d.Details)
	require.NoError(ts.T, err)

	// mimic the state where one provider has vault and another dont

	entry, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, SpecName(0), provider0.Addr.String())
	ts.Keepers.Epochstorage.RemoveStakeEntryCurrent(ts.Ctx, entry.Chain, entry.Address)
	require.True(t, found)

	// provider 0 address is now provider1 with vault of provider 0
	entry.Address = provider1.Addr.String()
	entry.DelegateCommission = 49
	entry.Description = d
	ts.Keepers.Epochstorage.SetStakeEntryCurrent(ts.Ctx, entry)

	keeper.NewMigrator(ts.Keepers.Pairing).MigrateVersion4To5(ts.Ctx)

	// unbond from validator with provider1
	val, _ := ts.GetAccount(common.VALIDATOR, 0)
	_, err = ts.TxUnbondValidator(provider1, val, math.NewInt(testStake))
	require.NoError(t, err)

	// check stake entries are correct
	res, err := ts.QueryPairingProvider(provider1.Addr.String(), "")
	require.NoError(t, err)
	require.Len(t, res.StakeEntries, 2)
	for _, entry := range res.StakeEntries {
		require.Equal(t, entry.Address, provider1.Addr.String())
		require.Equal(t, entry.Vault, provider0.GetVaultAddr())
	}

	mds, err := ts.Keepers.Epochstorage.GetAllMetadata(ts.Ctx)
	require.NoError(t, err)
	require.Len(t, mds, 1)
	require.Equal(t, provider1.Addr.String(), mds[0].Provider)
	require.Equal(t, provider0.GetVaultAddr(), mds[0].Vault)
	require.Equal(t, []string{SpecName(0), SpecName(1)}, mds[0].Chains)
	require.Equal(t, uint64(49), mds[0].DelegateCommission)
	require.Equal(t, d, mds[0].Description)
}
