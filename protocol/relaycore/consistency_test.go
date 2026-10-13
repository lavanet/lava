package relaycore

import (
	"math"
	"strconv"
	"testing"
	"time"

	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

func setupConsistency() Consistency {
	return NewConsistency("test", 0)
}

// solanaBlockTime is SOLANA's average_block_time from the spec. At 400ms the guard
// admits 4*(5min/400ms) = 3000 blocks per update, far beyond anything the chain can
// actually produce in an entry's lifetime.
const solanaBlockTime = 400 * time.Millisecond

// setSeenBlockSync records a seen block and flushes the cache so the write is readable
// before the test continues. Sleeping instead would make these tests depend on runner
// scheduling, which is a source of intermittent CI failures rather than a guarantee.
func setSeenBlockSync(t *testing.T, consistency Consistency, blockSeen int64, userData common.UserData) {
	t.Helper()
	consistency.SetSeenBlock(blockSeen, userData)
	impl, ok := consistency.(*ConsistencyImpl)
	require.True(t, ok, "expected *ConsistencyImpl")
	impl.cache.Wait()
}

// TestConsistency_ImplausibleSeenBlockAdvanceRejected covers the consumer-side defense
// against a provider reporting a latest block from outside the chain's numeric domain.
//
// The seen block is taken from a provider's reported latest block and then sent to
// every other provider, which bail when they cannot reach it. Without this guard a
// single provider — buggy, adversarial, or reporting a different unit — pins the seen
// block out of reach and makes every honest provider look hopelessly behind for the
// entry's lifetime. The Solana slot-vs-block-height gap (~22M) is the naturally
// occurring instance.
func TestConsistency_ImplausibleSeenBlockAdvanceRejected(t *testing.T) {
	const (
		blockHeightDomain = int64(414654108) // what a pre-fix provider reports
		slotDomain        = int64(436597938) // what a post-fix provider reports
	)
	userData := common.UserData{DappId: "dapp", ConsumerIp: "1.1.1.1:443"}

	t.Run("cross-domain jump is discarded", func(t *testing.T) {
		consistency := NewConsistency("SOLANA", solanaBlockTime)
		setSeenBlockSync(t, consistency, blockHeightDomain, userData)
		setSeenBlockSync(t, consistency, slotDomain, userData)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, blockHeightDomain, block,
			"a ~22M jump cannot come from the chain advancing and must not pin the seen block")
	})

	t.Run("ordinary advance is still accepted", func(t *testing.T) {
		consistency := NewConsistency("SOLANA", solanaBlockTime)
		setSeenBlockSync(t, consistency, slotDomain, userData)

		// A generous but realistic advance: more than the chain produces in an entry
		// lifetime at 400ms, still nowhere near the domain gap.
		advanced := slotDomain + 900
		setSeenBlockSync(t, consistency, advanced, userData)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, advanced, block, "normal chain progress must not be gated")
	})

	t.Run("unknown block time fails open", func(t *testing.T) {
		consistency := NewConsistency("test", 0)
		setSeenBlockSync(t, consistency, blockHeightDomain, userData)
		setSeenBlockSync(t, consistency, slotDomain, userData)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, slotDomain, block,
			"with no average block time the guard cannot justify a bound and must not gate")
	})

	t.Run("first value has no baseline to check against", func(t *testing.T) {
		// Documents a known limit: the guard bounds advances, so a cold entry adopts
		// whatever it is first told. Closing this needs a cross-provider signal.
		consistency := NewConsistency("SOLANA", solanaBlockTime)
		setSeenBlockSync(t, consistency, slotDomain, userData)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, slotDomain, block)
	})
}

func TestConsistency_MaxPlausibleSeenBlockAdvance(t *testing.T) {
	tests := []struct {
		name             string
		averageBlockTime time.Duration
		expected         int64
	}{
		{
			// 5min/400ms = 750 blocks in an entry lifetime, times the safety factor.
			name:             "fast chain uses the computed bound",
			averageBlockTime: solanaBlockTime,
			expected:         3000,
		},
		{
			// 5min/12s = 25 blocks, still far above any real provider spread.
			name:             "medium chain uses the computed bound",
			averageBlockTime: 12 * time.Second,
			expected:         100,
		},
		{
			// BTC produces a block every 10 minutes, so fewer than one fits in the TTL
			// and the computed bound truncates to zero. The minimum keeps the guard
			// usable without handing slow chains a huge allowance — the bound stays far
			// below the thousands of blocks a flat floor would have permitted.
			name:             "chain slower than the TTL falls back to the minimum",
			averageBlockTime: 10 * time.Minute,
			expected:         minSeenBlockAdvance,
		},
		{
			name:             "unknown block time disables the guard",
			averageBlockTime: 0,
			expected:         math.MaxInt64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consistency, ok := NewConsistency("TEST", tt.averageBlockTime).(*ConsistencyImpl)
			require.True(t, ok)
			require.Equal(t, tt.expected, consistency.maxPlausibleSeenBlockAdvance())
		})
	}
}

// TestConsistency_GuardAppliesToBackToBackUpdates exercises the ordering the relay
// path actually produces: two updates for the same key with no flush between them,
// which is how consecutive provider responses arrive.
//
// The hazard it covers is that cache writes are applied asynchronously, so the
// baseline read at the start of the second update can miss a write that has not
// drained yet — and a missing baseline skips the guard entirely. Honest caveat: the
// window is narrow and the buffer drains quickly, so this test does not reliably
// reproduce it, and it passes against code without the flush-and-recheck. It is kept
// as an assertion of the ordering's outcome, not as a regression lock; the flush in
// SetSeenBlockFromKey is what closes the window regardless of timing.
func TestConsistency_GuardAppliesToBackToBackUpdates(t *testing.T) {
	const (
		baseline = int64(414654108)
		poisoned = int64(436597938)
	)
	userData := common.UserData{DappId: "dapp", ConsumerIp: "1.1.1.1:443"}

	consistency := NewConsistency("SOLANA", solanaBlockTime)
	// No flush between the two calls: this is the ordering the relay path produces.
	consistency.SetSeenBlock(baseline, userData)
	consistency.SetSeenBlock(poisoned, userData)

	impl, ok := consistency.(*ConsistencyImpl)
	require.True(t, ok)
	impl.cache.Wait()

	block, found := consistency.GetSeenBlock(userData)
	require.True(t, found)
	require.Equal(t, baseline, block,
		"the guard must apply even when the baseline write has not drained yet")
}

func TestSetGet(t *testing.T) {
	consistency, ok := setupConsistency().(*ConsistencyImpl)
	require.True(t, ok, "setupConsistency should return *ConsistencyImpl")
	const BLOCKVALUE = int64(5)
	for i := 0; i < 100; i++ {
		consistency.SetLatestBlock(strconv.Itoa(i), BLOCKVALUE)
	}
	time.Sleep(4 * time.Millisecond)
	for i := 0; i < 100; i++ {
		block, found := consistency.GetLatestBlock(strconv.Itoa(i))
		require.Equal(t, BLOCKVALUE, block)
		require.True(t, found)
	}
}

func TestBasic(t *testing.T) {
	consistency := setupConsistency()

	dappid := "/1245/"
	ip := "1.1.1.1:443"

	dappid_other := "/77777/"
	ip_other := "2.1.1.1:443"

	userDataOne := common.UserData{DappId: dappid, ConsumerIp: ip}
	userDataOther := common.UserData{DappId: dappid_other, ConsumerIp: ip_other}

	for i := 1; i < 100; i++ {
		consistency.SetSeenBlock(int64(i), userDataOne)
		time.Sleep(4 * time.Millisecond) // need to let each set finish
	}
	consistency.SetSeenBlock(5, userDataOther)
	time.Sleep(4 * time.Millisecond)
	// try to set older values and discard them
	consistency.SetSeenBlock(3, userDataOther)
	time.Sleep(4 * time.Millisecond)
	consistency.SetSeenBlock(3, userDataOne)
	time.Sleep(4 * time.Millisecond)
	block, found := consistency.GetSeenBlock(userDataOne)
	require.True(t, found)
	require.Equal(t, int64(99), block)
	block, found = consistency.GetSeenBlock(userDataOther)
	require.True(t, found)
	require.Equal(t, int64(5), block)
}
