package lavasession

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReportedProvider(t *testing.T) {
	reportedProviders := NewReportedProviders()
	providers := []string{"p1", "p2", "p3"}
	reportedProviders.ReportProvider(providers[0], 0, 0, nil)
	require.True(t, reportedProviders.IsReported(providers[0]))
	require.False(t, reportedProviders.IsReported(providers[1]))
	reportedProviders.ReportProvider(providers[2], 0, 0, nil)
	require.True(t, reportedProviders.IsReported(providers[0]))
	require.True(t, reportedProviders.IsReported(providers[2]))
	require.False(t, reportedProviders.IsReported(providers[1]))
	found := false
	for _, reported := range reportedProviders.GetReportedProviders() {
		if reported.Address == providers[0] {
			found = true
			require.Equal(t, reported.Errors, uint64(0))
			require.Equal(t, reported.Disconnections, uint64(0))
		}
		require.False(t, reported.Address == providers[1])
	}
	require.True(t, found)
}

func TestReportedErrors(t *testing.T) {
	reportedProviders := NewReportedProviders()
	providers := []string{"p1", "p2", "p3"}
	reportedProviders.ReportProvider(providers[0], 5, 0, nil)
	require.True(t, reportedProviders.IsReported(providers[0]))
	require.False(t, reportedProviders.IsReported(providers[1]))
	reportedProviders.ReportProvider(providers[2], 5, 0, nil)
	reportedProviders.ReportProvider(providers[0], 5, 0, nil)
	require.True(t, reportedProviders.IsReported(providers[0]))
	require.True(t, reportedProviders.IsReported(providers[2]))
	require.False(t, reportedProviders.IsReported(providers[1]))
	found := false
	for _, reported := range reportedProviders.GetReportedProviders() {
		if reported.Address == providers[0] {
			found = true
			require.Equal(t, reported.Errors, uint64(10))
			require.Equal(t, reported.Disconnections, uint64(0))
		}
		require.False(t, reported.Address == providers[1])
	}
	require.True(t, found)
}

func TestReportedReconnect(t *testing.T) {
	reconnectAttempt := 0
	reconnected := func() error {
		reconnectAttempt++
		return nil
	}
	reconnectFail := func() error {
		reconnectAttempt++
		return fmt.Errorf("nope")
	}
	reportedProviders := NewReportedProviders()
	providers := []string{"p1", "p2", "p3", "p4"}
	reportedProviders.ReportProvider(providers[0], 0, 5, reconnected)
	require.True(t, reportedProviders.IsReported(providers[0]))
	require.False(t, reportedProviders.IsReported(providers[1]))
	reportedProviders.ReportProvider(providers[0], 1, 0, nil)
	reportedProviders.ReportProvider(providers[1], 0, 5, reconnected)
	reportedProviders.ReportProvider(providers[2], 5, 0, reconnected)
	reportedProviders.ReportProvider(providers[3], 0, 5, reconnectFail)
	require.True(t, reportedProviders.IsReported(providers[0]))
	require.True(t, reportedProviders.IsReported(providers[1]))
	require.True(t, reportedProviders.IsReported(providers[2]))
	require.True(t, reportedProviders.IsReported(providers[3]))
	require.Empty(t, reportedProviders.ReconnectCandidates())
	// set all entries in the past now
	for _, entry := range reportedProviders.addedToPurgeAndReport {
		timeAgo := time.Now().Add(-2 * ReconnectCandidateTime)
		entry.addedTime = timeAgo
	}
	candidates := reportedProviders.ReconnectCandidates()
	require.NotEmpty(t, candidates)
	require.Zero(t, reconnectAttempt)
	reportedProviders.ReconnectProviders()
	// expect only 1 and 3 to reconnect
	require.Equal(t, reconnectAttempt, 2)
	require.True(t, reportedProviders.IsReported(providers[3]))
	require.False(t, reportedProviders.IsReported(providers[1]))
}
