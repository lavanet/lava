package statetracker

import (
	"testing"
	"time"

	downtimev1 "github.com/lavanet/lava/x/downtime/v1"
	"github.com/stretchr/testify/require"
)

const (
	DowntimeDuration  = 10 * time.Millisecond
	EpochDuration     = 2 * DowntimeDuration
	FirstEpoch        = 10
	SecondEpoch       = 20
	ThirdEpoch        = 30
	ZeroVirtualEpoch  = uint64(0)
	FirstVirtualEpoch = uint64(1)
)

func TestEmergencyTrackerVirtualEpoch(t *testing.T) {
	emergencyTracker, emergencyCallback := NewEmergencyTracker(nil)
	emergencyTracker.SetDowntimeParams(downtimev1.Params{
		DowntimeDuration: DowntimeDuration,
		EpochDuration:    EpochDuration,
	})

	emergencyTracker.UpdateEpoch(FirstEpoch)

	require.Equal(t, ZeroVirtualEpoch, emergencyTracker.GetVirtualEpoch(FirstEpoch))
	require.Equal(t, ZeroVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())

	time.Sleep(EpochDuration)
	emergencyTracker.UpdateEpoch(SecondEpoch)
	latestBlockTime := time.Now()

	require.Equal(t, ZeroVirtualEpoch, emergencyTracker.GetVirtualEpoch(FirstEpoch))
	require.Equal(t, ZeroVirtualEpoch, emergencyTracker.GetVirtualEpoch(SecondEpoch))

	// wait until downtime starts, virtual_epoch = 0
	time.Sleep(DowntimeDuration)
	emergencyCallback(latestBlockTime)
	require.Equal(t, ZeroVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())

	// wait until the end of the normal epoch, virtual_epoch = 1
	time.Sleep(DowntimeDuration)
	emergencyCallback(latestBlockTime)
	require.Equal(t, FirstVirtualEpoch, emergencyTracker.GetVirtualEpoch(SecondEpoch))
	require.Equal(t, FirstVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())

	emergencyTracker.UpdateEpoch(ThirdEpoch)
	require.Equal(t, ZeroVirtualEpoch, emergencyTracker.GetLatestVirtualEpoch())
	require.Equal(t, FirstVirtualEpoch, emergencyTracker.GetVirtualEpoch(SecondEpoch))
}
