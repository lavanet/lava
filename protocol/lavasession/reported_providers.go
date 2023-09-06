package lavasession

import (
	"sync"

	"github.com/lavanet/lava/utils"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

type ReportedProviders struct {
	addedToPurgeAndReport map[string]ReportedProviderEntry // list of purged providers to report for QoS unavailability. (easier to search maps.)
	lock                  sync.RWMutex
}

type ReportedProviderEntry struct {
	Disconnections  uint64
	Errors          uint64
	shouldReconnect bool
}

func (rp *ReportedProviders) Reset() {
	rp.addedToPurgeAndReport = make(map[string]ReportedProviderEntry, 0)
}

func NewReportedProviders() ReportedProviders {
	ret := ReportedProviders{}
	ret.Reset()
	return ret
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
		}
		reportedProviders = append(reportedProviders, &reportedProvider)
	}
	return reportedProviders
}

func (rp *ReportedProviders) ReportProvider(address string) {
	if _, ok := rp.addedToPurgeAndReport[address]; !ok { // verify it doesn't exist already
		utils.LavaFormatInfo("Reporting Provider for unresponsiveness", utils.Attribute{Key: "Provider address", Value: address})
		// TODO: add disconnections, errors, should reconnect
		rp.addedToPurgeAndReport[address] = ReportedProviderEntry{}
	}

}

func (rp *ReportedProviders) IsReported(address string) bool {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	_, ok := rp.addedToPurgeAndReport[address]
	return ok
}
