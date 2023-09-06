package lavasession

import (
	"encoding/json"
	"sync"

	"github.com/lavanet/lava/utils"
)

type ReportedProviders struct {
	addedToPurgeAndReport map[string]struct{} // list of purged providers to report for QoS unavailability. (easier to search maps.)
	lock                  sync.RWMutex
}

func (rp *ReportedProviders) Reset() {
	rp.addedToPurgeAndReport = make(map[string]struct{}, 0)
}

func NewReportedProviders() ReportedProviders {
	ret := ReportedProviders{}
	ret.Reset()
	return ret
}

func (rp *ReportedProviders) GetReportedProviders() ([]byte, error) {
	rp.lock.RLock()
	defer rp.lock.RUnlock()
	keys := make([]string, 0, len(rp.addedToPurgeAndReport))
	for k := range rp.addedToPurgeAndReport {
		keys = append(keys, k)
	}
	bytes, err := json.Marshal(keys)
	return bytes, err
}

func (rp *ReportedProviders) ReportProvider(address string) {
	if _, ok := rp.addedToPurgeAndReport[address]; !ok { // verify it doesn't exist already
		utils.LavaFormatInfo("Reporting Provider for unresponsiveness", utils.Attribute{Key: "Provider address", Value: address})
		rp.addedToPurgeAndReport[address] = struct{}{}
	}

}
