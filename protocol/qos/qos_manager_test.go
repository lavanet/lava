package qos

import (
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
	doneChan := qosManager.CalculateQoS(
		epoch,
		sessionID,
		providerAddr,
		100*time.Millisecond,
		200*time.Millisecond,
		1,
		3,
		2,
	)

	<-doneChan // Wait for processing

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

	doneChan := qosManager.AddFailedRelay(epoch, sessionID)
	<-doneChan // Wait for processing

	totalRelays := qosManager.GetTotalRelays(epoch, sessionID)
	require.Equal(t, uint64(1), totalRelays)

	answeredRelays := qosManager.GetAnsweredRelays(epoch, sessionID)
	require.Equal(t, uint64(0), answeredRelays)
}

func TestSetLastReputationQoSReportRaw(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)

	testReport := &pairingtypes.QualityOfServiceReport{
		Latency:      sdk.NewDec(95),
		Availability: sdk.NewDec(100),
	}

	doneChan := qosManager.SetLastReputationQoSReportRaw(epoch, sessionID, testReport)
	<-doneChan // Wait for processing

	report := qosManager.GetLastReputationQoSReportRaw(epoch, sessionID)
	require.NotNil(t, report)
	require.Equal(t, testReport.Latency, report.Latency)
	require.Equal(t, testReport.Availability, report.Availability)
}

func TestConcurrentAccess(t *testing.T) {
	qosManager := NewQoSManager()
	epoch := uint64(1)
	sessionID := int64(1)
	providerAddr := "provider1"

	// Run multiple operations concurrently
	go func() {
		for i := 0; i < 100; i++ {
			doneChan := qosManager.CalculateQoS(
				epoch,
				sessionID,
				providerAddr,
				100*time.Millisecond,
				200*time.Millisecond,
				1,
				3,
				2,
			)
			<-doneChan
		}
	}()

	go func() {
		for i := 0; i < 50; i++ {
			doneChan := qosManager.AddFailedRelay(epoch, sessionID)
			<-doneChan
		}
	}()

	// Give time for goroutines to complete
	time.Sleep(100 * time.Millisecond)

	// Verify the total number of relays
	totalRelays := qosManager.GetTotalRelays(epoch, sessionID)
	require.Equal(t, uint64(150), totalRelays) // 100 successful + 50 failed
}
