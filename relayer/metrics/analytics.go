package metrics

import (
	"time"
)

type RelayAnalytics struct {
	ProjectHash string // imported - REST
	Timestamp   time.Time
	ChainID     string /// DONE - SendRelay
	APIType     string /// DONE - SendRelay
	// latency in ms
	Latency      int64  // DONE - SendRelay
	Success      bool   // Done - SendRelay
	ComputeUnits uint64 // Done - SendRelay
}

func NewRelayAnalytics(projectHash string, chainId string, apiType string) *RelayAnalytics {
	return &RelayAnalytics{
		Timestamp:   time.Now(),
		ProjectHash: projectHash,
		ChainID:     chainId,
		APIType:     apiType,
	}
}
