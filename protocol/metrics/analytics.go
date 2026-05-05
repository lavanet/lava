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

// RelayUsageEvent is the wire format for raw per-relay usage events emitted as
// OTel logs (body="relay_usage") to the host-local collector. One event per
// relay, no consumer-side aggregation — the destination rolls up by
// (timestamp, project, chain, ...) at query time. JSON tags are kept short to
// minimize payload size at scale.
type RelayUsageEvent struct {
	TimestampNs     int64  `json:"ts"`
	ProjectHash     string `json:"project"`
	ChainID         string `json:"chain"`
	APIInterface    string `json:"api_interface"`
	APIMethod       string `json:"api_method,omitempty"`
	ComputeUnits    uint64 `json:"cu"`
	LatencyMs       int64  `json:"latency_ms"`
	Success         bool   `json:"success"`
	CacheHit        bool   `json:"cache_hit,omitempty"`
	IsWrite         bool   `json:"is_write,omitempty"`
	IsArchive       bool   `json:"is_archive,omitempty"`
	IsBatch         bool   `json:"is_batch,omitempty"`
	IsDebugTrace    bool   `json:"is_debug_trace,omitempty"`
	HedgeCount      uint64 `json:"hedge_count,omitempty"`
	ProviderAddress string `json:"provider,omitempty"`
	Origin          string `json:"origin,omitempty"`
}

func NewRelayUsageEvent(rm *RelayMetrics) RelayUsageEvent {
	return RelayUsageEvent{
		TimestampNs:     rm.Timestamp.UnixNano(),
		ProjectHash:     rm.ProjectHash,
		ChainID:         rm.ChainID,
		APIInterface:    rm.APIType,
		APIMethod:       rm.ApiMethod,
		ComputeUnits:    rm.ComputeUnits,
		LatencyMs:       rm.Latency,
		Success:         rm.Success,
		CacheHit:        rm.CacheHit,
		IsWrite:         rm.IsWrite,
		IsArchive:       rm.IsArchive,
		IsBatch:         rm.IsBatch,
		IsDebugTrace:    rm.IsDebugTrace,
		HedgeCount:      rm.HedgeCount,
		ProviderAddress: rm.ProviderAddress,
		Origin:          rm.Origin,
	}
}
