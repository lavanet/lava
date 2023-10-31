package rpcconsumer

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupConsumerConsistency() *ConsumerConsistency {
	return NewConsumerConsistency("test")
}

func TestSetGet(t *testing.T) {
	consumerConsistency := setupConsumerConsistency()
	const BLOCKVALUE = int64(5)
	for i := 0; i < 100; i++ {
		consumerConsistency.setLatestBlock(strconv.Itoa(i), BLOCKVALUE)
	}
	time.Sleep(4 * time.Millisecond)
	for i := 0; i < 100; i++ {
		block, found := consumerConsistency.getLatestBlock(strconv.Itoa(i))
		require.Equal(t, BLOCKVALUE, block)
		require.True(t, found)
	}
}

func TestBasic(t *testing.T) {
	consumerConsistency := setupConsumerConsistency()

	dappid := "/1245/"
	ip := "1.1.1.1:443"

	dappid_other := "/77777/"
	ip_other := "2.1.1.1:443"

	for i := 1; i < 100; i++ {
		consumerConsistency.SetSeenBlock(int64(i), dappid, ip)
		time.Sleep(4 * time.Millisecond) // need to let each set finish
	}
	consumerConsistency.SetSeenBlock(5, dappid_other, ip_other)
	time.Sleep(4 * time.Millisecond)
	// try to set older values and discard them
	consumerConsistency.SetSeenBlock(3, dappid_other, ip_other)
	time.Sleep(4 * time.Millisecond)
	consumerConsistency.SetSeenBlock(3, dappid, ip)
	time.Sleep(4 * time.Millisecond)
	block, found := consumerConsistency.GetSeenBlock(dappid, ip)
	require.True(t, found)
	require.Equal(t, int64(99), block)
	block, found = consumerConsistency.GetSeenBlock(dappid_other, ip_other)
	require.True(t, found)
	require.Equal(t, int64(5), block)
}
