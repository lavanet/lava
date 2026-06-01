package metrics

import (
	"context"
	"math"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
	"github.com/lavanet/lava/v5/utils/rand"
	"golang.org/x/exp/maps"
)

// sanitizeFloat returns 0 if the value is NaN or Inf, otherwise returns the value
func sanitizeFloat(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

// ConsumerOptimizerQoSClient is a pure pass-through sampler — no internal
// aggregation, no rate calculations, no per-provider counters. On each
// sampling tick it asks every registered optimizer for its current QoS
// scores and emits one OTel log record per (chain, provider) through the
// usage sink. Downstream queries derive rates / averages by GROUP BY at
// query time.
//
// The only state retained is the optimizer registry, the per-chain
// provider address set (so the sampler knows which providers exist) and
// the current epoch. Everything else (selection counts, error rates,
// stake, RNG values) is removed — that data either lives inside the
// optimizer already or can be re-derived downstream from the per-relay
// usage events emitted by the chainlib transports.
type ConsumerOptimizerQoSClient struct {
	consumerHostname string
	consumerAddress  string
	geoLocation      uint64
	usageSink        UsageEventSink

	optimizers             map[string]OptimizerInf     // keys are chain ids
	chainIdToProviderStake map[string]map[string]int64 // current pairing-list stake per (chain, provider); replaced wholesale on each rotation
	currentEpoch           atomic.Uint64
	lock                   sync.RWMutex

	// latest is the most recent sample's records. Replaced wholesale on
	// each tick; not aggregated. Surfaces current scores via the optional
	// Prometheus-style scrape endpoint enabled by --optimizer-qos-listen.
	latest []OptimizerQoSReportToSend
}

// OptimizerQoSReport is what the optimizer hands back via
// CalculateQoSScoresForMetrics; the QoS client wraps these values into
// OptimizerQoSReportToSend records together with envelope metadata.
type OptimizerQoSReport struct {
	ProviderAddress string
	// Legacy fields - Raw EWMA values from score stores (NOT normalized for WRS)
	SyncScore         float64 // Raw sync lag in seconds from EWMA (lower is better)
	AvailabilityScore float64 // Raw availability from EWMA (0-1, higher is better)
	LatencyScore      float64 // Raw latency in seconds from EWMA (lower is better)
	GenericScore      float64 // Old composite score (deprecated, use SelectionComposite)
	EntryIndex        int     // Index in provider list
	// WRS normalized scores - Used in weighted random selection algorithm (0-1, higher is better)
	SelectionAvailability float64 // Normalized availability after Phase 1 rescaling
	SelectionLatency      float64 // Normalized latency after Phase 2 P10-P90
	SelectionSync         float64 // Normalized sync after Phase 2 P10-P90
	SelectionStake        float64 // Normalized stake after square root scaling
	SelectionComposite    float64 // Final composite score used for selection
	// Weighted contributions - How much each parameter contributes to SelectionComposite
	AvailabilityContribution float64 // SelectionAvailability × availability_weight
	LatencyContribution      float64 // SelectionLatency × latency_weight
	SyncContribution         float64 // SelectionSync × sync_weight
	StakeContribution        float64 // SelectionStake × stake_weight
}

// OptimizerQoSReportToSend is the wire-format envelope around an
// OptimizerQoSReport. Mirrors the relay-usage event shape: snake_case
// JSON tags, no derived/aggregated fields, downstream computes rates.
//
// The legacy raw EWMA fields (sync_score, availability_score,
// latency_score, generic_score) are intentionally omitted — they are
// the optimizer's internal state and feed the WRS normalized scores
// below, which are the values that actually drive selection. The
// downstream signal you almost always want is the normalized one.
type OptimizerQoSReportToSend struct {
	Timestamp        time.Time `json:"timestamp"`
	ProviderAddress  string    `json:"provider"`
	ConsumerHostname string    `json:"consumer_hostname"`
	ConsumerAddress  string    `json:"consumer_pub_address"`
	ChainId          string    `json:"chain_id"`
	GeoLocation      uint64    `json:"geo_location"`
	Epoch            uint64    `json:"epoch"`
	EntryIndex       int       `json:"entry_index"`
	ProviderStake    int64     `json:"provider_stake"` // current pairing-list stake (ulava); 0 if not yet known

	// WRS normalized scores (0-1, higher is better)
	SelectionAvailability float64 `json:"selection_availability"`
	SelectionLatency      float64 `json:"selection_latency"`
	SelectionSync         float64 `json:"selection_sync"`
	SelectionStake        float64 `json:"selection_stake"`
	SelectionComposite    float64 `json:"selection_composite"`

	// Weighted contributions to the composite
	AvailabilityContribution float64 `json:"availability_contribution"`
	LatencyContribution      float64 `json:"latency_contribution"`
	SyncContribution         float64 `json:"sync_contribution"`
	StakeContribution        float64 `json:"stake_contribution"`
}

type OptimizerInf interface {
	CalculateQoSScoresForMetrics(allAddresses []string, ignoredProviders map[string]struct{}) []*OptimizerQoSReport
}

// NewConsumerOptimizerQoSClient constructs a sampler that emits optimizer
// QoS snapshots through the given usage sink. Pass NoopUsageSink{} (or
// nil) to disable emission — the sampler still ticks but every emit is
// dropped.
func NewConsumerOptimizerQoSClient(consumerAddress string, sink UsageEventSink, geoLocation uint64) *ConsumerOptimizerQoSClient {
	hostname, err := os.Hostname()
	if err != nil {
		utils.LavaFormatWarning("Error while getting hostname for ConsumerOptimizerQoSClient", err)
		hostname = "unknown" + strconv.FormatUint(rand.Uint64(), 10)
	}
	return &ConsumerOptimizerQoSClient{
		consumerHostname:       hostname,
		consumerAddress:        consumerAddress,
		geoLocation:            geoLocation,
		usageSink:              sink,
		optimizers:             map[string]OptimizerInf{},
		chainIdToProviderStake: map[string]map[string]int64{},
	}
}

func (coqc *ConsumerOptimizerQoSClient) StartOptimizersQoSReportsCollecting(ctx context.Context, samplingInterval time.Duration) {
	if coqc == nil {
		return
	}

	utils.LavaFormatTrace("Starting ConsumerOptimizerQoSClient reports collecting")
	go func() {
		ticker := time.NewTicker(samplingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				utils.LavaFormatTrace("ConsumerOptimizerQoSClient context done")
				return
			case <-ticker.C:
				coqc.sampleAndEmit()
			}
		}
	}()
}

func (coqc *ConsumerOptimizerQoSClient) RegisterOptimizer(optimizer OptimizerInf, chainId string) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	if _, found := coqc.optimizers[chainId]; found {
		utils.LavaFormatWarning("Optimizer already registered for chain", nil, utils.LogAttr("chainId", chainId))
		return
	}

	coqc.optimizers[chainId] = optimizer
}

// UpdatePairingListStake is called by the optimizer when the on-chain
// pairing list rotates. We snapshot the per-(chain, provider) stake
// map and the current epoch — both used only to envelope the next
// optimizer-qos sample. Wholesale replacement, no per-epoch history.
func (coqc *ConsumerOptimizerQoSClient) UpdatePairingListStake(stakeMap map[string]int64, chainId string, epoch uint64) {
	if coqc == nil {
		return
	}

	coqc.lock.Lock()
	defer coqc.lock.Unlock()

	coqc.currentEpoch.Store(epoch)

	stakes := make(map[string]int64, len(stakeMap))
	for addr, stake := range stakeMap {
		stakes[addr] = stake
	}
	coqc.chainIdToProviderStake[chainId] = stakes
}

func (coqc *ConsumerOptimizerQoSClient) sampleAndEmit() {
	if coqc == nil {
		return
	}

	coqc.lock.RLock()
	optimizers := make(map[string]OptimizerInf, len(coqc.optimizers))
	for chainId, opt := range coqc.optimizers {
		optimizers[chainId] = opt
	}
	stakesByChain := make(map[string]map[string]int64, len(coqc.chainIdToProviderStake))
	for chainId, stakes := range coqc.chainIdToProviderStake {
		copyStakes := make(map[string]int64, len(stakes))
		for addr, stake := range stakes {
			copyStakes[addr] = stake
		}
		stakesByChain[chainId] = copyStakes
	}
	coqc.lock.RUnlock()

	ignoredProviders := map[string]struct{}{}
	currentEpoch := coqc.currentEpoch.Load()
	now := time.Now()

	var batch []OptimizerQoSReportToSend
	for chainId, optimizer := range optimizers {
		stakes := stakesByChain[chainId]
		if len(stakes) == 0 {
			continue
		}
		addresses := maps.Keys(stakes)
		reports := optimizer.CalculateQoSScoresForMetrics(addresses, ignoredProviders)
		for _, report := range reports {
			if report == nil {
				continue
			}
			stake := stakes[report.ProviderAddress]
			batch = append(batch, coqc.buildAndEmit(now, chainId, currentEpoch, stake, report))
		}
	}

	coqc.lock.Lock()
	coqc.latest = batch
	coqc.lock.Unlock()
}

// GetReportsToSend returns the most recent sample's records. Used by
// the metrics manager's Prometheus-style scrape endpoint when
// --optimizer-qos-listen is on; not part of the OTel emission path.
func (coqc *ConsumerOptimizerQoSClient) GetReportsToSend() []OptimizerQoSReportToSend {
	if coqc == nil {
		return nil
	}
	coqc.lock.RLock()
	defer coqc.lock.RUnlock()
	return coqc.latest
}

// SetReportsToSend overwrites the latest-reports cache. Test-only helper
// — production fills the cache through sampleAndEmit on each tick.
func (coqc *ConsumerOptimizerQoSClient) SetReportsToSend(reports []OptimizerQoSReportToSend) {
	if coqc == nil {
		return
	}
	coqc.lock.Lock()
	defer coqc.lock.Unlock()
	coqc.latest = reports
}

// buildAndEmit constructs the wire-format record, hands it to the OTel
// sink, and returns it for the in-memory latest-batch cache.
func (coqc *ConsumerOptimizerQoSClient) buildAndEmit(ts time.Time, chainId string, epoch uint64, stake int64, r *OptimizerQoSReport) OptimizerQoSReportToSend {
	if r == nil {
		return OptimizerQoSReportToSend{}
	}
	out := OptimizerQoSReportToSend{
		Timestamp:        ts,
		ProviderAddress:  r.ProviderAddress,
		ConsumerHostname: coqc.consumerHostname,
		ConsumerAddress:  coqc.consumerAddress,
		ChainId:          chainId,
		GeoLocation:      coqc.geoLocation,
		Epoch:            epoch,
		EntryIndex:       r.EntryIndex,
		ProviderStake:    stake,
		// WRS normalized
		SelectionAvailability: sanitizeFloat(r.SelectionAvailability),
		SelectionLatency:      sanitizeFloat(r.SelectionLatency),
		SelectionSync:         sanitizeFloat(r.SelectionSync),
		SelectionStake:        sanitizeFloat(r.SelectionStake),
		SelectionComposite:    sanitizeFloat(r.SelectionComposite),
		// Weighted contributions
		AvailabilityContribution: sanitizeFloat(r.AvailabilityContribution),
		LatencyContribution:      sanitizeFloat(r.LatencyContribution),
		SyncContribution:         sanitizeFloat(r.SyncContribution),
		StakeContribution:        sanitizeFloat(r.StakeContribution),
	}
	if coqc.usageSink != nil {
		coqc.usageSink.EmitOptimizerQoS(out)
	}
	return out
}
