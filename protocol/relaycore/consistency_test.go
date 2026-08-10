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
// admits 4*(5min/400ms)+1000 = 4000 blocks per update, far beyond anything the chain
// can actually produce in an entry's lifetime.
const solanaBlockTime = 400 * time.Millisecond

// TestSeenBlockImplausibleAdvanceRejected covers the consumer-side defense against a
// provider reporting a latest block from outside the chain's numeric domain.
//
// The seen block is taken from a provider's reported latest block and then sent to
// every other provider, which bail when they cannot reach it. Without this guard a
// single provider — buggy, adversarial, or reporting a different unit — pins the seen
// block out of reach and makes every honest provider look hopelessly behind for the
// entry's lifetime. The Solana slot-vs-block-height gap (~22M) is the naturally
// occurring instance.
func TestSeenBlockImplausibleAdvanceRejected(t *testing.T) {
	const (
		blockHeightDomain = int64(414654108) // what a pre-fix provider reports
		slotDomain        = int64(436597938) // what a post-fix provider reports
	)
	userData := common.UserData{DappId: "dapp", ConsumerIp: "1.1.1.1:443"}

	t.Run("cross-domain jump is discarded", func(t *testing.T) {
		consistency := NewConsistency("SOLANA", solanaBlockTime)
		consistency.SetSeenBlock(blockHeightDomain, userData)
		time.Sleep(4 * time.Millisecond)

		consistency.SetSeenBlock(slotDomain, userData)
		time.Sleep(4 * time.Millisecond)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, blockHeightDomain, block,
			"a ~22M jump cannot come from the chain advancing and must not pin the seen block")
	})

	t.Run("ordinary advance is still accepted", func(t *testing.T) {
		consistency := NewConsistency("SOLANA", solanaBlockTime)
		consistency.SetSeenBlock(slotDomain, userData)
		time.Sleep(4 * time.Millisecond)

		// A generous but realistic advance: more than the chain produces in an entry
		// lifetime at 400ms, still nowhere near the domain gap.
		advanced := slotDomain + 900
		consistency.SetSeenBlock(advanced, userData)
		time.Sleep(4 * time.Millisecond)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, advanced, block, "normal chain progress must not be gated")
	})

	t.Run("unknown block time fails open", func(t *testing.T) {
		consistency := NewConsistency("test", 0)
		consistency.SetSeenBlock(blockHeightDomain, userData)
		time.Sleep(4 * time.Millisecond)

		consistency.SetSeenBlock(slotDomain, userData)
		time.Sleep(4 * time.Millisecond)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, slotDomain, block,
			"with no average block time the guard cannot justify a bound and must not gate")
	})

	t.Run("first value has no baseline to check against", func(t *testing.T) {
		// Documents a known limit: the guard bounds advances, so a cold entry adopts
		// whatever it is first told. Closing this needs a cross-provider signal.
		consistency := NewConsistency("SOLANA", solanaBlockTime)
		consistency.SetSeenBlock(slotDomain, userData)
		time.Sleep(4 * time.Millisecond)

		block, found := consistency.GetSeenBlock(userData)
		require.True(t, found)
		require.Equal(t, slotDomain, block)
	})
}

func TestMaxPlausibleSeenBlockAdvance(t *testing.T) {
	// 5min/400ms = 750 blocks in an entry lifetime, *4 safety, +1000 floor.
	solana, ok := NewConsistency("SOLANA", solanaBlockTime).(*ConsistencyImpl)
	require.True(t, ok)
	require.Equal(t, int64(4000), solana.maxPlausibleSeenBlockAdvance())

	// A chain slower than the TTL per block rounds to zero and relies on the floor.
	slow, ok := NewConsistency("SLOW", time.Hour).(*ConsistencyImpl)
	require.True(t, ok)
	require.Equal(t, int64(seenBlockAdvanceFloor), slow.maxPlausibleSeenBlockAdvance())

	unknown, ok := NewConsistency("UNKNOWN", 0).(*ConsistencyImpl)
	require.True(t, ok)
	require.Equal(t, int64(math.MaxInt64), unknown.maxPlausibleSeenBlockAdvance())
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
