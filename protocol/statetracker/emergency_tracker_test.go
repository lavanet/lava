package statetracker

import (
	"testing"
	"time"

	downtimev1 "github.com/lavanet/lava/v2/x/downtime/v1"
	"github.com/stretchr/testify/require"
)

const (
	downtimeDuration  = 10 * time.Millisecond
	epochDuration     = 2 * downtimeDuration
	firstEpoch        = 10
	secondEpoch       = 20
	thirdEpoch        = 30
	zeroVirtualEpoch  = uint64(0)
	firstVirtualEpoch = uint64(1)
)

func TestEmergencyTrackerVirtualEpoch(t *testing.T) {
	emergencyTracker, emergencyCallback := NewEmergencyTracker(nil)
	emergencyTracker.SetDowntimeParams(downtimev1.Params{
		DowntimeDuration: downtimeDuration,
		EpochDuration:    epochDuration,
	})

	emergencyTracker.UpdateEpoch(firstEpoch)

	require.Equal(t, zeroVirtualEpoch, emergencyTracker.GetVirtualEpoch(firstEpoch))
	require.Equal(t, zeroVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())

	time.Sleep(epochDuration)
	emergencyTracker.UpdateEpoch(secondEpoch)
	latestBlockTime := time.Now()

	require.Equal(t, zeroVirtualEpoch, emergencyTracker.GetVirtualEpoch(firstEpoch))
	require.Equal(t, zeroVirtualEpoch, emergencyTracker.GetVirtualEpoch(secondEpoch))

	// wait until downtime starts, virtual_epoch = 0
	time.Sleep(downtimeDuration)
	emergencyCallback(latestBlockTime)
	require.Equal(t, zeroVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())

	// wait until the end of the normal epoch, virtual_epoch = 1
	time.Sleep(downtimeDuration)
	emergencyCallback(latestBlockTime)
	require.Equal(t, firstVirtualEpoch, emergencyTracker.GetVirtualEpoch(secondEpoch))
	require.Equal(t, firstVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())

	emergencyTracker.UpdateEpoch(thirdEpoch)
	require.Equal(t, zeroVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())
	require.Equal(t, firstVirtualEpoch, emergencyTracker.GetVirtualEpoch(secondEpoch))

	// test clear old virtual epoch
	emergencyTracker.UpdateEpoch(thirdEpoch + firstEpoch)
	require.Equal(t, firstVirtualEpoch, emergencyTracker.GetVirtualEpoch(secondEpoch))
	emergencyTracker.UpdateEpoch(thirdEpoch + secondEpoch)
	require.Equal(t, zeroVirtualEpoch, emergencyTracker.GetVirtualEpoch(secondEpoch))
}
