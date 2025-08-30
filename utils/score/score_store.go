package score

import (
	"fmt"
	"math"
	"sync"
	"time"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v5/utils"
)

const (
	DecPrecision         int64 = 8
	InitialDataStaleness       = 24 * time.Hour
)

// ScoreStore is a decaying weighted average object that is used to collect
// providers performance metrics samples (see QoS excellence comment below).
// These are used to calculate the providers QoS excellence score, used
// by the provider optimizer when choosing providers to be paired with a consumer.
//
// ScoreStore holds a score's numerator and denominator, last update timestamp, and a
// configuration object. When a ScoreStore updates it uses a decay exponent to lower
// the weight of old average samples and a weight parameter to determine the influence
// of the new sample.
//
// Resolving the ScoreStore's num and denom means to divide the num by the denom to get
// the score. Keeping the score as a fracture helps calculating and updating weighted
// average calculations on the go.
type ScoreStore struct {
	Name   string
	Num    float64 // using float64 and not math/big for performance
	Denom  float64
	Time   time.Time
	Config Config
	lock   sync.RWMutex
}

// ScoreStorer defines the interface for all score stores
type ScoreStorer interface {
	Update(sample float64, sampleTime time.Time) error
	Resolve() (float64, error)
	Validate() error
	String() string
	UpdateConfig(opts ...Option) error

	GetName() string
	GetNum() float64
	GetDenom() float64
	GetLastUpdateTime() time.Time
	GetConfig() Config
}

// NewCustomScoreStore creates a new custom ScoreStorer based on the score type
func NewCustomScoreStore(scoreType string, num, denom float64, t time.Time, opts ...Option) (ScoreStorer, error) {
	cfg := defaultConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("cannot create %s ScoreStore, invalid configuration: %w", scoreType, err)
	}

	base := &ScoreStore{
		Num:    num,
		Denom:  denom,
		Time:   t,
		Config: cfg,
	}

	if err := base.Validate(); err != nil {
		return nil, fmt.Errorf("cannot create %s ScoreStore, invalid parameters: %w", scoreType, err)
	}

	switch scoreType {
	case LatencyScoreType:
		base.Name = LatencyScoreType
		return &LatencyScoreStore{ScoreStore: base}, nil
	case SyncScoreType:
		base.Name = SyncScoreType
		return &SyncScoreStore{ScoreStore: base}, nil
	case AvailabilityScoreType:
		base.Name = AvailabilityScoreType
		return &AvailabilityScoreStore{ScoreStore: base}, nil
	default:
		return nil, fmt.Errorf("unknown score type: %s", scoreType)
	}
}

// NewScoreStore creates a new default ScoreStorer based on the score type
func NewScoreStore(scoreType string) ScoreStorer {
	switch scoreType {
	case LatencyScoreType:
		// default latency: 10ms
		latencyScoreStore, err := NewCustomScoreStore(scoreType, DefaultLatencyNum, 1, time.Now().Add(-InitialDataStaleness))
		if err != nil {
			utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", err)
		}
		return latencyScoreStore

	case SyncScoreType:
		// default sync: 100ms
		syncScoreStore, err := NewCustomScoreStore(scoreType, DefaultSyncNum, 1, time.Now().Add(-InitialDataStaleness))
		if err != nil {
			utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", err)
		}
		return syncScoreStore

	case AvailabilityScoreType:
		// default availability: 1
		availabilityScoreStore, err := NewCustomScoreStore(scoreType, DefaultAvailabilityNum, 1, time.Now().Add(-InitialDataStaleness))
		if err != nil {
			utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", err)
		}
		return availabilityScoreStore
	default:
		utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", fmt.Errorf("unknown score type: %s", scoreType))
		return nil // not reached
	}
}

// String prints a ScoreStore's fields
func (ss *ScoreStore) String() string {
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	return fmt.Sprintf("num: %f, denom: %f, last_update_time: %s, config: %s",
		ss.Num, ss.Denom, ss.Time.String(), ss.Config.String())
}

// Validate validates the ScoreStore's fields hold valid values
func (ss *ScoreStore) Validate() error {
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	return ss.validateInner()
}

func (ss *ScoreStore) validateInner() error {
	if ss.Num < 0 || ss.Denom <= 0 {
		return fmt.Errorf("invalid %s ScoreStore: num or denom are non-positives, num: %f, denom: %f", ss.Name, ss.Num, ss.Denom)
	}
	if err := ss.Config.Validate(); err != nil {
		return errors.Wrap(err, "invalid "+ss.Name+" ScoreStore")
	}
	return nil
}

// Resolve resolves the ScoreStore's frac by dividing the numerator by the denominator
func (ss *ScoreStore) Resolve() (float64, error) {
	ss.lock.RLock()
	defer ss.lock.RUnlock()
	if err := ss.validateInner(); err != nil {
		return 0, errors.Wrap(err, "cannot calculate "+ss.Name+" ScoreStore's score")
	}
	return ss.Num / ss.Denom, nil
}

// UpdateConfig updates the configuration of a ScoreStore
func (ss *ScoreStore) UpdateConfig(opts ...Option) error {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	cfg := ss.Config
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	ss.Config = cfg

	return nil
}

// update updates the ScoreStore's numerator and denominator with a new sample.
// The ScoreStore's score is calculated as a weighted average with a decay factor.
// The new sample is added by the following formula:
//
//	num = num * decay_factor + sample * weight
//	denom = denom * decay_factor + weight
//	decay_factor = exp(-time_since_last_update / half_life_time)
func (ss *ScoreStore) Update(sample float64, sampleTime time.Time) error {
	if ss == nil {
		return fmt.Errorf("cannot update ScoreStore, ScoreStore is nil")
	}
	ss.lock.Lock()
	defer ss.lock.Unlock()

	if sample < 0 {
		return fmt.Errorf("cannot update %s ScoreStore, sample is negative: %f", ss.Name, sample)
	}

	if ss.Time.After(sampleTime) {
		utils.LavaFormatTrace("TimeConflictingScoresError", utils.LogAttr("ss.Time", ss.Time), utils.LogAttr("sampleTime", sampleTime))
		return TimeConflictingScoresError
	}

	timeDiff := sampleTime.Sub(ss.Time).Seconds()
	if timeDiff < 0 {
		return fmt.Errorf("invalid time difference: %f seconds", timeDiff)
	}

	exponent := -(math.Ln2 * timeDiff) / ss.Config.HalfLife.Seconds()
	decayFactor := math.Exp(exponent)
	if decayFactor > 1 {
		return fmt.Errorf("invalid larger than 1 decay factor, factor: %f", decayFactor)
	}

	newNum, err := ss.calcNewNum(sample, decayFactor)
	if err != nil {
		return err
	}
	newDenom, err := ss.calcNewDenom(decayFactor)
	if err != nil {
		return err
	}

	ss.Num = newNum
	ss.Denom = newDenom
	ss.Time = sampleTime

	if err := ss.validateInner(); err != nil {
		return errors.Wrap(err, "cannot update "+ss.Name+" ScoreStore's num and denom")
	}

	return nil
}

// calcNewNum calculates the new numerator update and verifies it's not negative or overflowing
func (ss *ScoreStore) calcNewNum(sample float64, decayFactor float64) (float64, error) {
	if math.IsInf(ss.Num*decayFactor, 0) || math.IsInf(sample*ss.Config.Weight, 0) {
		return 0, utils.LavaFormatError("cannot ScoreStore update numerator", fmt.Errorf("potential overflow"),
			utils.LogAttr("score_store_name", ss.Name),
			utils.LogAttr("current_num", ss.Num),
			utils.LogAttr("decay_factor", decayFactor),
			utils.LogAttr("sample", sample),
			utils.LogAttr("weight", ss.Config.Weight),
		)
	}

	newNum := ss.Num*decayFactor + sample*ss.Config.Weight
	if newNum < 0 {
		return 0, fmt.Errorf("cannot update %s ScoreStore, invalid negative numerator: %f", ss.Name, newNum)
	}
	return newNum, nil
}

// calcNewDenom calculates the new denominator update and verifies it's strictly positive or not overflowing
func (ss *ScoreStore) calcNewDenom(decayFactor float64) (float64, error) {
	if math.IsInf(ss.Denom*decayFactor, 0) || math.IsInf(ss.Config.Weight, 0) {
		return 0, utils.LavaFormatError("cannot ScoreStore update denominator", fmt.Errorf("potential overflow"),
			utils.LogAttr("score_store_name", ss.Name),
			utils.LogAttr("current_denom", ss.Denom),
			utils.LogAttr("decay_factor", decayFactor),
			utils.LogAttr("weight", ss.Config.Weight),
		)
	}

	newDenom := ss.Denom*decayFactor + ss.Config.Weight
	if newDenom <= 0 {
		return 0, fmt.Errorf("cannot update %s ScoreStore, invalid non-positive denominator: %f", ss.Name, newDenom)
	}
	return newDenom, nil
}

func (ss *ScoreStore) GetName() string {
	return ss.Name
}

func (ss *ScoreStore) GetNum() float64 {
	return ss.Num
}

func (ss *ScoreStore) GetDenom() float64 {
	return ss.Denom
}

func (ss *ScoreStore) GetLastUpdateTime() time.Time {
	return ss.Time
}

func (ss *ScoreStore) GetConfig() Config {
	return ss.Config
}

func ConvertToDec(val float64) sdk.Dec {
	if val > 0 && val < math.Pow(10, -float64(DecPrecision)) {
		// If value is positive but would round to zero, return smallest possible value
		return sdk.NewDecWithPrec(1, DecPrecision)
	}
	intScore := int64(math.Round(val * math.Pow(10, float64(DecPrecision))))
	return sdk.NewDecWithPrec(intScore, DecPrecision)
}

// QoS excellence is a collection of performance metrics that measure a provider's
// performance in terms of latency, sync, and availability.
// These are calculated when the consumer processes responses from the provider.
// The consumer measures the provider's response latency, its reported last seen block
// (to check for sync) and whether the provider is responsive in general (availability).
// All three metrics are saved using the ScoreStore objects that implement the ScoreStorer
// interface.
// The QoS excellence score influences a provider's chance to be selected in the consumer
// pairing process.
// The metrics are:
// 	1. Latency: the time it takes the provider to answer to consumer relays.
//
//  2. Sync: the difference between the latest block as the provider percieves it
//           compared to the actual last block of the chain it serves.
//
//  3. Availability: the provider's up time.

const (
	DefaultLatencyNum      float64 = 0.01
	DefaultSyncNum         float64 = 0.1
	DefaultAvailabilityNum float64 = 1

	LatencyScoreType      = "latency"
	SyncScoreType         = "sync"
	AvailabilityScoreType = "availability"
	TotalScoreType        = "total"

	// Worst score results for each QoS excellence metric for truncation
	WorstLatencyScore      float64 = 30      // seconds
	WorstSyncScore         float64 = 20 * 60 // seconds
	WorstAvailabilityScore float64 = 0.00001 // very small value to avoid score = 0
)

/* ########## Latency ScoreStore ############ */

type LatencyScoreStore struct {
	*ScoreStore
}

// Update updates the Latency ScoreStore's numerator and denominator with a new sample.
func (ls *LatencyScoreStore) Update(sample float64, sampleTime time.Time) error {
	if ls == nil {
		return fmt.Errorf("LatencyScoreStore is nil")
	}

	// normalize the sample with the latency CU factor
	sample *= ls.ScoreStore.Config.LatencyCuFactor

	return ls.ScoreStore.Update(sample, sampleTime)
}

/* ########## Sync ScoreStore ############ */

type SyncScoreStore struct {
	*ScoreStore
}

// Update updates the Sync ScoreStore's numerator and denominator with a new sample.
func (ss *SyncScoreStore) Update(sample float64, sampleTime time.Time) error {
	if ss == nil {
		return fmt.Errorf("SyncScoreStore is nil")
	}
	return ss.ScoreStore.Update(sample, sampleTime)
}

/* ########## Availability ScoreStore ############ */

type AvailabilityScoreStore struct {
	*ScoreStore
}

// Update updates the availability ScoreStore's numerator and denominator with a new sample.
// The new sample must be 0 or 1.
func (as *AvailabilityScoreStore) Update(sample float64, sampleTime time.Time) error {
	if as == nil {
		return fmt.Errorf("AvailabilityScoreStore is nil")
	}
	if sample != float64(0) && sample != float64(1) {
		return fmt.Errorf("availability must be 0 (false) or 1 (true), got %f", sample)
	}
	return as.ScoreStore.Update(sample, sampleTime)
}

func (as *AvailabilityScoreStore) Resolve() (float64, error) {
	if as == nil {
		return 0, fmt.Errorf("AvailabilityScoreStore is nil")
	}
	score, err := as.ScoreStore.Resolve()
	if err != nil {
		return 0, err
	}

	// if the resolved score is equal to zero, return a very small number
	// instead of zero since in the QoS Compute() method we divide by
	// the availability score
	if score <= WorstAvailabilityScore {
		score = WorstAvailabilityScore
	}
	return score, nil
}
