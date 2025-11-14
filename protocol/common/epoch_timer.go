package common

import (
	"context"
	"sync"
	"time"

	"github.com/lavanet/lava/v5/utils"
)

// EpochTimer manages epochs based on absolute wall clock time
// All processes using the same epochDuration will calculate the same epoch number
// and update at the same absolute time boundaries
type EpochTimer struct {
	lock          sync.RWMutex
	epochDuration time.Duration
	epochZeroTime time.Time // Reference time for epoch 0 (fixed date in the past)
	currentTimer  *time.Timer
	callbacks     []func(uint64)
	stopChan      chan struct{}
}

// NewEpochTimer creates a new time-based epoch timer
// Epoch 0 starts at a fixed point in the past (January 1, 2024 00:00:00 UTC)
// This ensures epochs remain consistent even after long downtimes
func NewEpochTimer(epochDuration time.Duration) *EpochTimer {
	// Set epoch 0 to a fixed date far in the past
	// This ensures epoch numbers remain consistent even after weeks/months of downtime
	// Using January 1, 2024 00:00:00 UTC as the reference point
	epochZeroTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	return &EpochTimer{
		epochDuration: epochDuration,
		epochZeroTime: epochZeroTime,
		callbacks:     []func(uint64){},
		stopChan:      make(chan struct{}),
	}
}

// CalculateCurrentEpoch calculates epoch number from absolute time
// Formula: epoch = floor((currentTime - epochZeroTime) / epochDuration)
func (et *EpochTimer) CalculateCurrentEpoch() uint64 {
	et.lock.RLock()
	defer et.lock.RUnlock()

	// Time elapsed since epoch 0 (Jan 1, 2024)
	elapsed := time.Since(et.epochZeroTime)

	// Calculate epoch number (integer division)
	epochNumber := uint64(elapsed / et.epochDuration)

	return epochNumber
}

// GetCurrentEpoch returns current epoch (alias for clarity)
func (et *EpochTimer) GetCurrentEpoch() uint64 {
	return et.CalculateCurrentEpoch()
}

// GetTimeUntilNextEpoch calculates time until next epoch boundary
func (et *EpochTimer) GetTimeUntilNextEpoch() time.Duration {
	et.lock.RLock()
	defer et.lock.RUnlock()

	currentEpoch := et.CalculateCurrentEpoch()

	// Next epoch starts at: epochZeroTime + (currentEpoch + 1) * epochDuration
	nextEpochStart := et.epochZeroTime.Add(time.Duration(currentEpoch+1) * et.epochDuration)

	// Time until next epoch
	untilNext := time.Until(nextEpochStart)

	return untilNext
}

// GetEpochBoundaryTime returns the absolute time when an epoch starts
func (et *EpochTimer) GetEpochBoundaryTime(epoch uint64) time.Time {
	et.lock.RLock()
	defer et.lock.RUnlock()

	return et.epochZeroTime.Add(time.Duration(epoch) * et.epochDuration)
}

// GetEpochDuration returns the configured epoch duration
func (et *EpochTimer) GetEpochDuration() time.Duration {
	return et.epochDuration
}

// RegisterCallback registers a callback to be called on epoch updates
func (et *EpochTimer) RegisterCallback(callback func(uint64)) {
	et.lock.Lock()
	defer et.lock.Unlock()
	et.callbacks = append(et.callbacks, callback)
}

// Start begins the epoch timer with synchronized triggers at absolute boundaries
func (et *EpochTimer) Start(ctx context.Context) {
	currentEpoch := et.CalculateCurrentEpoch()
	timeUntilNext := et.GetTimeUntilNextEpoch()
	nextEpochTime := time.Now().Add(timeUntilNext)

	utils.LavaFormatInfo("EpochTimer starting",
		utils.LogAttr("epochDuration", et.epochDuration),
		utils.LogAttr("epochZeroTime", et.epochZeroTime.Format("2006-01-02 15:04:05 MST")),
		utils.LogAttr("currentTime", time.Now().Format("15:04:05 MST")),
		utils.LogAttr("currentEpoch", currentEpoch),
		utils.LogAttr("nextEpochTime", nextEpochTime.Format("15:04:05 MST")),
		utils.LogAttr("timeUntilNext", timeUntilNext),
	)

	// Immediately notify callbacks with current epoch
	et.notifyCallbacks(currentEpoch)

	// Schedule next update at absolute epoch boundary
	et.scheduleNextUpdate(ctx)
}

// scheduleNextUpdate schedules the next epoch update at the absolute epoch boundary
func (et *EpochTimer) scheduleNextUpdate(ctx context.Context) {
	timeUntilNext := et.GetTimeUntilNextEpoch()

	// Add small buffer (1 second) to ensure we cross the boundary
	timeUntilNext += time.Second

	currentEpoch := et.CalculateCurrentEpoch()
	nextEpochTime := et.GetEpochBoundaryTime(currentEpoch + 1)

	utils.LavaFormatDebug("EpochTimer: Scheduling next update",
		utils.LogAttr("currentEpoch", currentEpoch),
		utils.LogAttr("timeUntilNext", timeUntilNext),
		utils.LogAttr("nextEpochTime", nextEpochTime.Format("15:04:05 MST")),
		utils.LogAttr("nextEpoch", currentEpoch+1),
	)

	et.lock.Lock()
	et.currentTimer = time.NewTimer(timeUntilNext)
	timer := et.currentTimer
	et.lock.Unlock()

	go func() {
		select {
		case <-timer.C:
			et.onEpochBoundary(ctx)
		case <-et.stopChan:
			timer.Stop()
			return
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}()
}

// onEpochBoundary is called when we cross an epoch boundary
func (et *EpochTimer) onEpochBoundary(ctx context.Context) {
	newEpoch := et.CalculateCurrentEpoch()
	boundaryTime := et.GetEpochBoundaryTime(newEpoch)
	actualTime := time.Now()
	drift := actualTime.Sub(boundaryTime)

	utils.LavaFormatInfo("EpochTimer: Epoch boundary crossed",
		utils.LogAttr("newEpoch", newEpoch),
		utils.LogAttr("expectedTime", boundaryTime.Format("15:04:05.000 MST")),
		utils.LogAttr("actualTime", actualTime.Format("15:04:05.000 MST")),
		utils.LogAttr("drift", drift),
	)

	// Notify all callbacks
	et.notifyCallbacks(newEpoch)

	// Schedule next update
	et.scheduleNextUpdate(ctx)
}

// notifyCallbacks calls all registered callbacks
func (et *EpochTimer) notifyCallbacks(epoch uint64) {
	et.lock.RLock()
	callbacks := append([]func(uint64){}, et.callbacks...)
	et.lock.RUnlock()

	utils.LavaFormatDebug("EpochTimer: Notifying callbacks",
		utils.LogAttr("epoch", epoch),
		utils.LogAttr("numCallbacks", len(callbacks)),
	)

	for i, callback := range callbacks {
		// Call in goroutine to avoid blocking
		go func(cb func(uint64), index int) {
			defer func() {
				if r := recover(); r != nil {
					utils.LavaFormatError("Panic in epoch callback", nil,
						utils.LogAttr("panic", r),
						utils.LogAttr("epoch", epoch),
						utils.LogAttr("callbackIndex", index),
					)
				}
			}()
			cb(epoch)
		}(callback, i)
	}
}

// Stop stops the epoch timer
func (et *EpochTimer) Stop() {
	et.lock.Lock()
	if et.currentTimer != nil {
		et.currentTimer.Stop()
	}
	et.lock.Unlock()

	close(et.stopChan)

	utils.LavaFormatInfo("EpochTimer stopped")
}
