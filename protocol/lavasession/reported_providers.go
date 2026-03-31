package lavasession

import (
	"sync"
	"time"

	metrics "github.com/lavanet/lava/v5/protocol/metrics"
	"github.com/lavanet/lava/v5/utils"
	pairingtypes "github.com/lavanet/lava/v5/types/relay"
)

const (
	ReconnectCandidateTime = 30 * time.Second
	debugReportedProviders = false
)

type ReportedProviders struct {
	addedToPurgeAndReport map[string]*ReportedProviderEntry // list of purged providers to report for QoS unavailability. (easier to search maps.)
	lock                  sync.RWMutex
	generation            uint64 // incremented on Reset(); used to discard stale reconnect candidates
	reporter              metrics.Reporter
	chainId               string
}

type ReportedProviderEntry struct {
	Disconnections uint64
	Errors         uint64
	addedTime      time.Time
	reconnectCB    func() error
}

func (rp *ReportedProviders) Reset() {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	if debugReportedProviders {
		utils.LavaFormatDebug("[debugReportedProviders] Reset called")
	}
	rp.addedToPurgeAndReport = make(map[string]*ReportedProviderEntry, 0)
	rp.generation++
}

func (rp *ReportedProviders) GetReportedProviders() []*pairingtypes.ReportedProvider {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	reportedProviders := make([]*pairingtypes.ReportedProvider, 0, len(rp.addedToPurgeAndReport))
	for provider, reportedProviderEntry := range rp.addedToPurgeAndReport {
		reportedProvider := pairingtypes.ReportedProvider{
			Address:        provider,
			Disconnections: reportedProviderEntry.Disconnections,
			Errors:         reportedProviderEntry.Errors,
			TimestampS:     reportedProviderEntry.addedTime.Unix(),
		}
		reportedProviders = append(reportedProviders, &reportedProvider)
	}
	return reportedProviders
}

func (rp *ReportedProviders) ReportProvider(providerAddr string, errors uint64, disconnections uint64, reconnectCB func() error, errorsForReport []error) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	rp.reportProviderLocked(providerAddr, errors, disconnections, reconnectCB, errorsForReport)
}

// reportProviderLocked performs the actual report. Caller must hold rp.lock.Lock().
func (rp *ReportedProviders) reportProviderLocked(providerAddr string, errors uint64, disconnections uint64, reconnectCB func() error, errorsForReport []error) {
	if _, ok := rp.addedToPurgeAndReport[providerAddr]; !ok { // add if it doesn't exist already
		utils.LavaFormatInfo("Reporting Provider for unresponsiveness", utils.Attribute{Key: "Provider address", Value: providerAddr})
		rp.addedToPurgeAndReport[providerAddr] = &ReportedProviderEntry{}
		rp.addedToPurgeAndReport[providerAddr].addedTime = time.Now()
	}
	rp.addedToPurgeAndReport[providerAddr].Disconnections += disconnections
	rp.addedToPurgeAndReport[providerAddr].Errors += errors
	if reconnectCB != nil {
		rp.addedToPurgeAndReport[providerAddr].reconnectCB = reconnectCB
	}
	if debugReportedProviders {
		utils.LavaFormatDebug("[debugReportedProviders] adding provider to reported providers", utils.LogAttr("rp.addedToPurgeAndReport", rp.addedToPurgeAndReport))
	}

	// update metrics on the report.
	go rp.AppendReport(metrics.NewReportsRequest(providerAddr, errorsForReport, rp.chainId))
}

// will be called after a disconnected provider got a valid connection
func (rp *ReportedProviders) RemoveReport(address string) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	if debugReportedProviders {
		utils.LavaFormatDebug("[debugReportedProviders] Removing Report", utils.LogAttr("address", address))
	}
	delete(rp.addedToPurgeAndReport, address)
}

func (rp *ReportedProviders) IsReported(address string) bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	_, ok := rp.addedToPurgeAndReport[address]
	return ok
}

type reconnectCandidate struct {
	address     string
	reconnectCB func() error
}

type reconnectCandidatesResult struct {
	candidates []reconnectCandidate
	generation uint64
}

func (rp *ReportedProviders) ReconnectCandidates() reconnectCandidatesResult {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	candidates := []reconnectCandidate{}
	if debugReportedProviders {
		utils.LavaFormatDebug("[debugReportedProviders] Reconnect candidates", utils.LogAttr("candidate list", rp.addedToPurgeAndReport))
	}
	for address, entry := range rp.addedToPurgeAndReport {
		// only reconnect providers that didn't have consecutive errors
		if entry.Errors == 0 && time.Since(entry.addedTime) > ReconnectCandidateTime {
			candidate := reconnectCandidate{
				address:     address,
				reconnectCB: entry.reconnectCB,
			}
			candidates = append(candidates, candidate)
		}
	}
	return reconnectCandidatesResult{candidates: candidates, generation: rp.generation}
}

func (rp *ReportedProviders) ReconnectProviders() {
	result := rp.ReconnectCandidates()
	for _, candidate := range result.candidates {
		if candidate.reconnectCB != nil {
			if debugReportedProviders {
				utils.LavaFormatDebug("[debugReportedProviders] Trying to reconnect candidate", utils.LogAttr("candidate", candidate.address))
			}
			err := candidate.reconnectCB()
			// Generation check and mutation must happen under the same lock
			// to prevent Reset() from interleaving between check and mutation.
			stale := func() bool {
				rp.lock.Lock()
				defer rp.lock.Unlock()
				if rp.generation != result.generation {
					return true
				}
				if err == nil {
					if debugReportedProviders {
						utils.LavaFormatDebug("[debugReportedProviders] Removing Report", utils.LogAttr("address", candidate.address))
					}
					delete(rp.addedToPurgeAndReport, candidate.address)
				} else {
					rp.reportProviderLocked(candidate.address, 0, 1, nil, []error{err})
				}
				return false
			}()
			if stale {
				utils.LavaFormatDebug("skipping stale reconnect candidate after epoch transition",
					utils.Attribute{Key: "provider", Value: candidate.address})
				continue
			}
			utils.LavaFormatDebug("reconnect attempt", utils.Attribute{Key: "provider", Value: candidate.address}, utils.Attribute{Key: "success", Value: err == nil})
		}
	}
}

func (rp *ReportedProviders) AppendReport(report metrics.ReportsRequest) {
	if rp == nil || rp.reporter == nil {
		return
	}
	if debugReportedProviders {
		utils.LavaFormatDebug("[debugReportedProviders] Sending report on provider", utils.LogAttr("provider", report.Provider))
	}
	rp.reporter.AppendReport(report)
}

func NewReportedProviders(reporter metrics.Reporter, chainId string) *ReportedProviders {
	rp := &ReportedProviders{addedToPurgeAndReport: map[string]*ReportedProviderEntry{}, reporter: reporter, chainId: chainId}
	go func() {
		ticker := time.NewTicker(ReconnectCandidateTime)
		defer ticker.Stop()
		for range ticker.C {
			rp.ReconnectProviders()
		}
	}()
	return rp
}
