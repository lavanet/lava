package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMoveStake_SameTransaction_AllowsMultipleMoves(t *testing.T) {
	ts := newTester(t)
	// 1 provider, 2 specs, no clients needed
	SetupForSingleProviderTests(ts, 1, 2, 0)
	providerAcc, _ := ts.GetAccount("provider_", 0)
	provider := providerAcc.Addr
	src := SpecName(0)
	dst := SpecName(1)

	// Perform multiple moves within the same block/transaction context
	_, err := ts.TxPairingMoveStake(provider.String(), src, dst, testStake/10)
	require.NoError(t, err)

	// Same tx (no block time advance) should still be allowed
	_, err = ts.TxPairingMoveStake(provider.String(), dst, src, testStake/20)
	require.NoError(t, err)
}

func TestMoveStake_Cooldown_BlocksSecondMoveWithin24h(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 1, 2, 0)
	providerAcc, _ := ts.GetAccount("provider_", 0)
	provider := providerAcc.Addr
	src := SpecName(0)
	dst := SpecName(1)

	// First move succeeds
	_, err := ts.TxPairingMoveStake(provider.String(), src, dst, testStake/10)
	require.NoError(t, err)

	// Advance less than the cooldown window (24h)
	ts.AdvanceTimeHours(23 * time.Hour)

	// Second move should be rate-limited
	_, err = ts.TxPairingMoveStake(provider.String(), dst, src, testStake/20)
	require.Error(t, err)
}

func TestMoveStake_Cooldown_ExpiresAfter24h(t *testing.T) {
	ts := newTester(t)
	SetupForSingleProviderTests(ts, 1, 2, 0)
	providerAcc, _ := ts.GetAccount("provider_", 0)
	provider := providerAcc.Addr
	src := SpecName(0)
	dst := SpecName(1)

	// First move succeeds
	_, err := ts.TxPairingMoveStake(provider.String(), src, dst, testStake/10)
	require.NoError(t, err)

	// Advance full cooldown window (>= 24h)
	ts.AdvanceTimeHours(24 * time.Hour)

	// Second move should succeed after cooldown
	_, err = ts.TxPairingMoveStake(provider.String(), dst, src, testStake/20)
	require.NoError(t, err)
}
