package keeper_test

import (
	"math"
	"testing"

	"github.com/lavanet/lava/v5/testutil/common"
	testutils "github.com/lavanet/lava/v5/testutil/keeper"
	"github.com/lavanet/lava/v5/x/pairing/types"
	"github.com/stretchr/testify/require"
)

// TestJailProposalSingleChain verifies that a JailProposal freezes a provider and sets JailEndTime.
func TestJailProposalSingleChain(t *testing.T) {
	ts := newTester(t)
	require.NoError(t, ts.addProvider(1))
	ts.AdvanceEpoch()

	_, provider := ts.GetAccount(common.PROVIDER, 0)
	chainID := ts.spec.Index

	jailEndTime := ts.Ctx.BlockTime().Unix() + 3600 // 1 hour from now

	err := testutils.SimulateJailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: provider, ChainId: chainID, Reason: "test jail", JailEndTime: jailEndTime},
	})
	require.NoError(t, err)

	se, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, chainID, provider)
	require.True(t, found)
	require.True(t, se.IsFrozen(), "provider should be frozen after jail proposal")
	require.Equal(t, jailEndTime, se.JailEndTime, "JailEndTime should match proposal value")
	require.EqualValues(t, 1, se.Jails, "Jails counter should be incremented")
}

// TestJailProposalPermanent verifies that jail_end_time = 0 results in a permanent jail (math.MaxInt64).
func TestJailProposalPermanent(t *testing.T) {
	ts := newTester(t)
	require.NoError(t, ts.addProvider(1))
	ts.AdvanceEpoch()

	_, provider := ts.GetAccount(common.PROVIDER, 0)
	chainID := ts.spec.Index

	err := testutils.SimulateJailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: provider, ChainId: chainID, Reason: "permanent test", JailEndTime: 0},
	})
	require.NoError(t, err)

	se, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, chainID, provider)
	require.True(t, found)
	require.True(t, se.IsFrozen(), "provider should be frozen")
	require.Equal(t, int64(math.MaxInt64), se.JailEndTime, "JailEndTime should be MaxInt64 for permanent jail")
}

// TestJailProposalWildcard verifies that chain_id = "*" jails a provider across all chains it is staked on.
func TestJailProposalWildcard(t *testing.T) {
	ts := newTester(t)

	// Add a second spec and stake the same provider on both specs.
	spec2 := ts.AddSpec("mock2", common.CreateMockSpec()).Spec("mock2")

	require.NoError(t, ts.addProvider(1))
	ts.AdvanceEpoch()

	acc, provider := ts.GetAccount(common.PROVIDER, 0)
	d := common.MockDescription()
	require.NoError(t, ts.StakeProviderExtra(acc.GetVaultAddr(), provider, spec2, testStake, nil, 0, d.Moniker, d.Identity, d.Website, d.SecurityContact, d.Details))
	ts.AdvanceEpoch()

	// Jail across all chains via wildcard.
	err := testutils.SimulateJailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: provider, ChainId: "*", Reason: "wildcard test", JailEndTime: 0},
	})
	require.NoError(t, err)

	for _, chainID := range []string{ts.spec.Index, spec2.Index} {
		se, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, chainID, provider)
		require.True(t, found, "stake entry missing for chain %s", chainID)
		require.True(t, se.IsFrozen(), "provider should be frozen on chain %s", chainID)
	}
}

// TestJailProposalProviderNotFound verifies that jailing a non-staked provider does not return an error
// (the handler logs it as not-found and continues).
func TestJailProposalProviderNotFound(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

	nonExistentProvider := "lava@1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq5pv00v"

	err := testutils.SimulateJailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: nonExistentProvider, ChainId: ts.spec.Index, Reason: "not found test", JailEndTime: 0},
	})
	// The handler should not return an error — it logs not-found entries and continues.
	require.NoError(t, err)
}

// TestUnjailProposal verifies that an UnjailProposal resets JailEndTime, Jails, and unfreezes the provider.
func TestUnjailProposal(t *testing.T) {
	ts := newTester(t)
	require.NoError(t, ts.addProvider(1))
	ts.AdvanceEpoch()

	_, provider := ts.GetAccount(common.PROVIDER, 0)
	chainID := ts.spec.Index

	// First jail the provider permanently.
	require.NoError(t, testutils.SimulateJailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: provider, ChainId: chainID, Reason: "setup jail", JailEndTime: 0},
	}))

	se, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, chainID, provider)
	require.True(t, found)
	require.True(t, se.IsFrozen(), "should be frozen before unjail")

	// Now unjail.
	err := testutils.SimulateUnjailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: provider, ChainId: chainID},
	})
	require.NoError(t, err)

	se, found = ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, chainID, provider)
	require.True(t, found)
	require.False(t, se.IsFrozen(), "provider should be unfrozen after unjail")
	require.EqualValues(t, 0, se.JailEndTime, "JailEndTime should be reset to 0")
	require.EqualValues(t, 0, se.Jails, "Jails counter should be reset to 0")
}

// TestUnjailProposalProviderNotFound verifies that unjailing a non-staked provider does not return an error.
func TestUnjailProposalProviderNotFound(t *testing.T) {
	ts := newTester(t)
	ts.AdvanceEpoch()

	nonExistentProvider := "lava@1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq5pv00v"

	err := testutils.SimulateUnjailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: nonExistentProvider, ChainId: ts.spec.Index},
	})
	require.NoError(t, err)
}

// TestUnjailProposalEpochBoundary verifies that after an unjail the StakeAppliedBlock is set
// to GetCurrentNextEpoch + 1, so the provider re-enters pairing at the correct epoch boundary.
func TestUnjailProposalEpochBoundary(t *testing.T) {
	ts := newTester(t)
	require.NoError(t, ts.addProvider(1))
	ts.AdvanceEpoch()

	_, provider := ts.GetAccount(common.PROVIDER, 0)
	chainID := ts.spec.Index

	// Jail permanently.
	require.NoError(t, testutils.SimulateJailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: provider, ChainId: chainID, JailEndTime: 0},
	}))

	expectedUnfreezeBlock := ts.Keepers.Epochstorage.GetCurrentNextEpoch(ts.Ctx) + 1

	// Unjail.
	require.NoError(t, testutils.SimulateUnjailProposal(ts.Ctx, ts.Keepers.Pairing, []types.ProviderJailInfo{
		{Provider: provider, ChainId: chainID},
	}))

	se, found := ts.Keepers.Epochstorage.GetStakeEntryCurrent(ts.Ctx, chainID, provider)
	require.True(t, found)
	require.Equal(t, expectedUnfreezeBlock, se.StakeAppliedBlock,
		"StakeAppliedBlock should be set to currentNextEpoch+1 after unjail")
}
