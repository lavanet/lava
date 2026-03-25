package metrics

import (
	"time"
)

type RelaySource int

const (
	GatewaySource RelaySource = iota + 1
)

type RelayMetrics struct {
	ProjectHash         string
	Timestamp           time.Time
	ChainID             string
	APIType             string
	Latency             int64
	Success             bool
	ComputeUnits        uint64
	Source              RelaySource
	Origin              string
	ApiMethod           string
	ProcessingTimestamp time.Time
	// Request classification fields — populated at relay call sites
	ProviderAddress string
	CacheHit        bool // true only when the response was served from the cache
	IsWrite         bool // stateful != 0; false means read
	IsArchive       bool
	IsDebugTrace    bool
	IsBatch         bool
	// Incident tracking fields — set during relay processing
	HedgeCount uint64 // number of hedge relays sent by the batch ticker for this request
}

type RelayAnalyticsDTO struct {
	ProjectHash  string
	Timestamp    string
	ChainID      string
	APIType      string
	Latency      uint64
	SuccessCount int64
	RelayCounts  int64
	TotalCu      uint64
	Source       RelaySource
	Origin       string
}

func NewRelayAnalytics(projectHash, chainId, apiType string) *RelayMetrics {
	return &RelayMetrics{
		Timestamp:   time.Now(),
		ProjectHash: projectHash,
		ChainID:     chainId,
		APIType:     apiType,
		Source:      GatewaySource,
	}
}

func (rm *RelayMetrics) SetProcessingTimestampBeforeRelay(timestamp time.Time) {
	if rm == nil {
		return
	}
	rm.ProcessingTimestamp = timestamp
}

func (rm *RelayMetrics) SetProcessingTimestampAfterRelay(timestamp time.Time) {
	if rm == nil {
		return
	}
	rm.ProcessingTimestamp = timestamp
}
