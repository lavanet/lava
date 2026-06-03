package relaycore

import (
	"strconv"
	"testing"
	"time"

	"github.com/lavanet/lava/v5/protocol/chainstate"
	common "github.com/lavanet/lava/v5/protocol/common"
	"github.com/stretchr/testify/require"
)

func setupConsistency() Consistency {
	return NewConsistency("test", nil)
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

// --- Outlier guard tests ---

func TestSetSeenBlock_OutlierRejected(t *testing.T) {
	cs := &chainstate.ChainState{}
	cs.SetMajorityBaseline(1000, 200) // baseline=1000, threshold=200 → outlier if > 1200
	consistency := NewConsistency("test", cs)

	userData := common.UserData{DappId: "dapp1", ConsumerIp: "1.1.1.1"}

	// Normal block — should be accepted
	consistency.SetSeenBlock(1100, userData)
	time.Sleep(4 * time.Millisecond)
	block, found := consistency.GetSeenBlock(userData)
	require.True(t, found)
	require.Equal(t, int64(1100), block)

	// Outlier block — should be rejected
	consistency.SetSeenBlock(1201, userData)
	time.Sleep(4 * time.Millisecond)
	block, found = consistency.GetSeenBlock(userData)
	require.True(t, found)
	require.Equal(t, int64(1100), block) // unchanged

	// Extreme outlier (Feb 17 scenario) — should be rejected
	consistency.SetSeenBlock(20000, userData)
	time.Sleep(4 * time.Millisecond)
	block, found = consistency.GetSeenBlock(userData)
	require.True(t, found)
	require.Equal(t, int64(1100), block) // still unchanged
}

func TestSetSeenBlockFromKey_OutlierRejected(t *testing.T) {
	cs := &chainstate.ChainState{}
	cs.SetMajorityBaseline(1000, 200)
	consistency := NewConsistency("test", cs)

	key := "dapp1__1.1.1.1"

	// Normal — accepted
	consistency.SetSeenBlockFromKey(1100, key)
	time.Sleep(4 * time.Millisecond)
	block, found := consistency.(*ConsistencyImpl).GetLatestBlock(key)
	require.True(t, found)
	require.Equal(t, int64(1100), block)

	// Outlier — rejected
	consistency.SetSeenBlockFromKey(1201, key)
	time.Sleep(4 * time.Millisecond)
	block, found = consistency.(*ConsistencyImpl).GetLatestBlock(key)
	require.True(t, found)
	require.Equal(t, int64(1100), block) // unchanged
}

func TestSetSeenBlock_ColdStartAllowsAll(t *testing.T) {
	cs := &chainstate.ChainState{} // majorityBaseline=0 (cold start)
	consistency := NewConsistency("test", cs)

	userData := common.UserData{DappId: "dapp1", ConsumerIp: "1.1.1.1"}

	// Even extreme values should be allowed when majorityBaseline=0
	consistency.SetSeenBlock(999999, userData)
	time.Sleep(4 * time.Millisecond)
	block, found := consistency.GetSeenBlock(userData)
	require.True(t, found)
	require.Equal(t, int64(999999), block)
}

func TestSetSeenBlock_NilChainStateAllowsAll(t *testing.T) {
	consistency := NewConsistency("test", nil) // no chainState

	userData := common.UserData{DappId: "dapp1", ConsumerIp: "1.1.1.1"}

	// Any value should be allowed when chainState is nil
	consistency.SetSeenBlock(999999, userData)
	time.Sleep(4 * time.Millisecond)
	block, found := consistency.GetSeenBlock(userData)
	require.True(t, found)
	require.Equal(t, int64(999999), block)
}
