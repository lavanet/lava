package metrics

import (
	"time"
)

type RelayMetrics struct {
	ProjectHash  string
	Timestamp    time.Time
	ChainID      string
	APIType      string
	Latency      int64
	Success      bool
	ComputeUnits uint64
	Source       string
}

type RelayAnalyticsDTO struct {
	ProjectHash  string
	Timestamp    time.Time
	ChainID      string
	APIType      string
	Latency      uint64
	SuccessCount int64
	RelayCounts  int64
	Source       string
}

func NewRelayAnalytics(projectHash string, chainId string, apiType string) *RelayMetrics {
	return &RelayMetrics{
		Timestamp:   time.Now(),
		ProjectHash: projectHash,
		ChainID:     chainId,
		APIType:     apiType,
	}
}
