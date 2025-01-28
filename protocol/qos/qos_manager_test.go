package qos

import (
	"math"
	"sync"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	pairingtypes "github.com/lavanet/lava/v4/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateQoS(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)
	providerAddr := "provider1"

	// Test successful relay
	qosManager.CalculateQoS(
		epoch,
		sessionID,
		providerAddr,
		100*time.Millisecond,
		200*time.Millisecond,
		1,
		3,
		2,
	)

	report := qosManager.GetLastQoSReport(epoch, sessionID)
	require.NotNil(t, report)

	totalRelays := qosManager.GetTotalRelays(epoch, sessionID)
	require.Equal(t, uint64(1), totalRelays)

	answeredRelays := qosManager.GetAnsweredRelays(epoch, sessionID)
	require.Equal(t, uint64(1), answeredRelays)
}

func TestAddFailedRelay(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)

	qosManager.AddFailedRelay(epoch, sessionID)
	totalRelays := qosManager.GetTotalRelays(epoch, sessionID)
	require.Equal(t, uint64(1), totalRelays)

	answeredRelays := qosManager.GetAnsweredRelays(epoch, sessionID)
	require.Equal(t, uint64(0), answeredRelays)
}

func TestSetLastReputationQoSReport(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)

	testReport := &pairingtypes.QualityOfServiceReport{
		Latency:      sdk.NewDec(95),
		Availability: sdk.NewDec(100),
	}

	qosManager.SetLastReputationQoSReport(epoch, sessionID, testReport)
	report := qosManager.GetLastReputationQoSReport(epoch, sessionID)
	require.NotNil(t, report)
	require.Equal(t, testReport.Latency, report.Latency)
	require.Equal(t, testReport.Availability, report.Availability)
}

func TestMultipleEpochsAndSessions(t *testing.T) {
	qosManager := NewQoSManager()

	// Test multiple epochs and sessions simultaneously
	for epoch := uint64(1); epoch <= 3; epoch++ {
		for sessionID := int64(1); sessionID <= 3; sessionID++ {
			qosManager.CalculateQoS(
				epoch,
				sessionID,
				"provider1",
				100*time.Millisecond,
				200*time.Millisecond,
				1,
				3,
				2,
			)
		}
	}

	// Verify each epoch/session combination
	for epoch := uint64(1); epoch <= 3; epoch++ {
		for sessionID := int64(1); sessionID <= 3; sessionID++ {
			require.Equal(t, uint64(1), qosManager.GetTotalRelays(epoch, sessionID))
			require.NotNil(t, qosManager.GetLastQoSReport(epoch, sessionID))
		}
	}
}

func TestEdgeCaseLatencies(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)

	testCases := []struct {
		name            string
		latency         time.Duration
		expectedLatency time.Duration
	}{
		{"Zero Latency", 0, 100 * time.Millisecond},
		{"Extremely High Latency", 24 * time.Hour, 100 * time.Millisecond},
		{"Negative Expected Latency", 100 * time.Millisecond, -100 * time.Millisecond},
		{"Equal Latencies", 100 * time.Millisecond, 100 * time.Millisecond},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			qosManager.CalculateQoS(
				epoch,
				sessionID,
				"provider1",
				tc.latency,
				tc.expectedLatency,
				1,
				3,
				2,
			)
			require.NotNil(t, qosManager.GetLastQoSReport(epoch, sessionID))
		})
	}
}

func TestNilReportHandling(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)
	// Test setting nil report
	qosManager.SetLastReputationQoSReport(epoch, sessionID, nil)
	// Verify nil handling
	report := qosManager.GetLastReputationQoSReport(epoch, sessionID)
	require.Nil(t, report)

	// Test non-existent epoch/session
	require.Nil(t, qosManager.GetLastQoSReport(999, 999))
	require.Equal(t, uint64(0), qosManager.GetTotalRelays(999, 999))
	require.Equal(t, uint64(0), qosManager.GetAnsweredRelays(999, 999))
}

func TestHighConcurrencyScenario(t *testing.T) {
	qosManager := NewQoSManager()
	numGoroutines := 10
	operationsPerGoroutine := 1000

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 different operation types

	// Launch multiple goroutines for CalculateQoS
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				qosManager.CalculateQoS(
					uint64(routineID),
					int64(j),
					"provider1",
					100*time.Millisecond,
					200*time.Millisecond,
					1,
					3,
					2,
				)
			}
		}(i)
	}

	// Launch multiple goroutines for AddFailedRelay
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				qosManager.AddFailedRelay(uint64(routineID), int64(j))
			}
		}(i)
	}

	// Launch multiple goroutines for SetLastReputationQoSReport
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				report := &pairingtypes.QualityOfServiceReport{
					Latency:      sdk.NewDec(95),
					Availability: sdk.NewDec(100),
				}
				qosManager.SetLastReputationQoSReport(uint64(routineID), int64(j), report)
			}
		}(i)
	}

	wg.Wait()

	// Verify some results
	for i := 0; i < numGoroutines; i++ {
		for j := 0; j < operationsPerGoroutine; j++ {
			totalRelays := qosManager.GetTotalRelays(uint64(i), int64(j))
			require.Equal(t, uint64(2), totalRelays) // 1 successful + 1 failed relay
			require.NotNil(t, qosManager.GetLastReputationQoSReport(uint64(i), int64(j)))
		}
	}
}

func TestQoSParameterBoundaries(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)

	testCases := []struct {
		name             string
		latency          time.Duration
		expectedLatency  time.Duration
		blockHeightDiff  int64
		numOfProviders   int
		servicersToCount int64
	}{
		{"Max Values", time.Duration(math.MaxInt64), time.Duration(math.MaxInt64), math.MaxInt, math.MaxInt, math.MaxInt},
		{"Min Values", 1, 1, 1, 1, 1},
		{"Zero Values", 0, 0, 0, 0, 0},
		{"Inverted Weights", 100 * time.Millisecond, 100 * time.Millisecond, 10, 5, 7},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			qosManager.CalculateQoS(
				epoch,
				sessionID,
				"provider1",
				tc.latency,
				tc.expectedLatency,
				tc.blockHeightDiff,
				tc.numOfProviders,
				tc.servicersToCount,
			)
			// Verify that the manager doesn't panic and returns a report
			report := qosManager.GetLastQoSReport(epoch, sessionID)
			require.NotNil(t, report)
		})
	}
}

func TestSequentialOperations(t *testing.T) {
	t.Run("Mixed Operations Sequence", func(t *testing.T) {
		qosManager := NewQoSManager()
		epoch := uint64(1)
		sessionID := int64(1)

		// Sequence: Calculate -> Fail -> Calculate
		qosManager.CalculateQoS(
			epoch,
			sessionID,
			"provider1",
			100*time.Millisecond,
			200*time.Millisecond,
			1, 3, 2,
		)
		qosManager.AddFailedRelay(epoch, sessionID)
		qosManager.CalculateQoS(
			epoch,
			sessionID,
			"provider1",
			100*time.Millisecond,
			200*time.Millisecond,
			1, 3, 2,
		)
		require.Equal(t, uint64(3), qosManager.GetTotalRelays(epoch, sessionID))
		require.Equal(t, uint64(2), qosManager.GetAnsweredRelays(epoch, sessionID))
	})
}

// TODO: Enable this test when we register the QoSManager to epoch updater
// func TestMemoryManagement(t *testing.T) {
// 	qosManager := NewQoSManager()

// 	// Create data for multiple epochs
// 	for epoch := uint64(1); epoch <= 100; epoch++ {
// 		doneChan := qosManager.CalculateQoS(
// 			epoch,
// 			1,
// 			"provider1",
// 			100*time.Millisecond,
// 			200*time.Millisecond,
// 			1, 3, 2,
// 		)
// 		<-doneChan
// 	}

// 	// Verify old data is not taking up memory (if cleanup is implemented)
// 	// Note: This test might need adjustment based on actual cleanup implementation
// 	t.Run("Memory Cleanup", func(t *testing.T) {
// 		// Add implementation-specific verification here
// 		// For example, verify that very old epochs are cleaned up
// 		veryOldEpoch := uint64(1)
// 		report := qosManager.GetLastQoSReport(veryOldEpoch, 1)
// 		require.Nil(t, report, "Old epoch data should be cleaned up")
// 		t.Log("Memory cleanup behavior should be verified based on implementation")
// 	})
// }
