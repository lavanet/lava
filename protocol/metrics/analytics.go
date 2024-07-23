package metrics

import (
	"time"
)

type RelaySource int

const (
	SdkSource RelaySource = iota + 1
	GatewaySource
	BadgesSource
)

type RelayMetrics struct {
	ProjectHash                        string
	Timestamp                          time.Time
	ChainID                            string
	APIType                            string
	Latency                            int64
	Success                            bool
	ComputeUnits                       uint64
	Source                             RelaySource
	Origin                             string
	ApiMethod                          string
	ProcessingTimestamp                time.Time
	MeasureAfterProviderProcessingTime bool // we measure processing time only on first relay success so we use this to indicate that the after provider measurement should occur (not true for all code flows)
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
	// we use this flag to make sure the relay passed successfully. (only the first relay has the RelayMetrics)
	rm.MeasureAfterProviderProcessingTime = true
	rm.ProcessingTimestamp = timestamp
}
