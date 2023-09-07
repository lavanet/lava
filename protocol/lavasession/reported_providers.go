package lavasession

import (
	"sync"
	"time"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type ReportedProviders struct {
	addedToPurgeAndReport map[string]*ReportedProviderEntry // list of purged providers to report for QoS unavailability. (easier to search maps.)
	lock                  sync.RWMutex
}

type ReportedProviderEntry struct {
	Disconnections uint64
	Errors         uint64
	addedTime      time.Time
}

func (rp *ReportedProviders) Reset() {
	rp.addedToPurgeAndReport = make(map[string]*ReportedProviderEntry, 0)
}

func NewReportedProviders() ReportedProviders {
	return ReportedProviders{addedToPurgeAndReport: map[string]*ReportedProviderEntry{}}
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

func (rp *ReportedProviders) ReportProvider(address string, errors uint64, disconnections uint64) {
	rp.lock.Lock()
	defer rp.lock.Unlock()
	if _, ok := rp.addedToPurgeAndReport[address]; !ok { // verify it doesn't exist already
		utils.LavaFormatInfo("Reporting Provider for unresponsiveness", utils.Attribute{Key: "Provider address", Value: address})
		rp.addedToPurgeAndReport[address] = &ReportedProviderEntry{}
	}
	rp.addedToPurgeAndReport[address].Disconnections += disconnections
	rp.addedToPurgeAndReport[address].Errors += errors
	rp.addedToPurgeAndReport[address].addedTime = time.Now()
}

func (rp *ReportedProviders) IsReported(address string) bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	_, ok := rp.addedToPurgeAndReport[address]
	return ok
}
