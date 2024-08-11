package keeper_test

import (
	"testing"

	"github.com/lavanet/lava/v2/testutil/common"
	"github.com/stretchr/testify/require"
)

// TestPairingQueryCache tests the following:
// 1. The pairing query cache is reset every epoch
// 2. Getting pairing with a query using an existent cache entry consumes fewer gas than without one
func TestPairingQueryCache(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	_, consumer := ts.GetAccount(common.CONSUMER, 0)

	getPairingGas := func(ts *tester) uint64 {
		gm := ts.Ctx.GasMeter()
		before := gm.GasConsumed()
		_, err := ts.QueryPairingGetPairing(ts.spec.Index, consumer)
		require.NoError(t, err)
		return gm.GasConsumed() - before
	}

	// query for pairing for the first time - empty cache
	emptyCacheGas := getPairingGas(ts)

	// query for pairing for the second time - non-empty cache
	filledCacheGas := getPairingGas(ts)

	// second time gas should be smaller than first time
	require.Less(t, filledCacheGas, emptyCacheGas)

	// advance block to test it stays the same (should still be less than empty cache gas)
	ts.AdvanceBlock()
	filledAfterBlockCacheGas := getPairingGas(ts)
	require.Less(t, filledAfterBlockCacheGas, emptyCacheGas)

	// advance epoch to reset the cache
	ts.AdvanceEpoch()
	emptyCacheAgainGas := getPairingGas(ts)
	require.Equal(t, emptyCacheGas, emptyCacheAgainGas)
}

// TestPairingQueryCache tests the following:
// 1. The pairing relay cache is reset every block
// 2. Getting pairing in relay payment using an existent cache entry consumes fewer gas than without one
func TestPairingRelayCache(t *testing.T) {
	ts := newTester(t)
	ts.setupForPayments(1, 1, 0) // 1 provider, 1 client, default providers-to-pair

	consumer, _ := ts.GetAccount(common.CONSUMER, 0)
	provider, _ := ts.GetAccount(common.PROVIDER, 0)

	getRelayPairingGas := func(ts *tester) uint64 {
		gm := ts.Ctx.GasMeter()
		before := gm.GasConsumed()
		relayPayment := sendRelay(ts, provider.Addr.String(), consumer, []string{ts.spec.Index})
		_, err := ts.TxPairingRelayPayment(relayPayment.Creator, relayPayment.Relays[0])
		require.NoError(t, err)
		return gm.GasConsumed() - before
	}

	// query for pairing for the first time - empty cache
	emptyCacheGas := getRelayPairingGas(ts)

	// query for pairing for the second time - non-empty cache
	filledCacheGas := getRelayPairingGas(ts)

	// second time gas should be smaller than first time
	require.Less(t, filledCacheGas, emptyCacheGas)

	// advance block to to reset the cache
	ts.AdvanceBlock()
	emptyCacheAgainGas := getRelayPairingGas(ts)
	require.InEpsilon(t, emptyCacheGas, emptyCacheAgainGas, 0.05)
}
