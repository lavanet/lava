package lavasession

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test the basic functionality of the consumerSessionManager
func TestHappyFlowPSM(t *testing.T) {
	var a uint64 = 5
	res_a := atomic.CompareAndSwapUint64(&a, 5, 7)
	require.True(t, res_a)
	res_b := atomic.CompareAndSwapUint64(&a, 5, 7)
	require.False(t, res_b)
}
