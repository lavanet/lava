package score

import (
	"fmt"
	"math"
	"time"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/utils"
)

const (
	DecPrecision         int64 = 8
	InitialDataStaleness       = 24
)

// ScoreStore holds a score's numerator and denominator, last update timestamp, and a
// configuration object.
// When a ScoreStore updates it uses a decay exponent to lower the weight of old
// average samples and a weight parameter to determine the influence of the new sample.
// Keeping the score as a fracture helps calculating and updating weighted average calculations
// on the go.
// Currently, ScoreStore is used for QoS excellence metrics (see below).
type ScoreStore struct {
	Name   string
	Num    float64 // using float64 and not math/big for performance
	Denom  float64
	Time   time.Time
	Config Config
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
	cfg := Config{
		Weight:   DefaultWeight,
		HalfLife: DefaultHalfLifeTime,
	}
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
		latencyScoreStore, err := NewCustomScoreStore(scoreType, DefaultLatencyNum, 1, time.Now().Add(-InitialDataStaleness*time.Hour))
		if err != nil {
			utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", err)
		}
		return latencyScoreStore

	case SyncScoreType:
		// default sync: 100ms
		syncScoreStore, err := NewCustomScoreStore(scoreType, DefaultSyncNum, 1, time.Now().Add(-InitialDataStaleness*time.Hour))
		if err != nil {
			utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", err)
		}
		return syncScoreStore

	case AvailabilityScoreType:
		// default availability: 1
		availabilityScoreStore, err := NewCustomScoreStore(scoreType, DefaultAvailabilityNum, 1, time.Now().Add(-InitialDataStaleness*time.Hour))
		if err != nil {

		}
		return availabilityScoreStore
	default:
		utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", fmt.Errorf("unknown score type: %s", scoreType))
		return nil // not reached
	}
}

// String prints a ScoreStore's fields
func (ss *ScoreStore) String() string {
	return fmt.Sprintf("num: %f, denom: %f, last_update_time: %s, config: %s",
		ss.Num, ss.Denom, ss.Time.String(), ss.Config.String())
}

// Validate validates the ScoreStore's fields hold valid values
func (ss *ScoreStore) Validate() error {
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
	if err := ss.Validate(); err != nil {
		return 0, errors.Wrap(err, "cannot calculate "+ss.Name+" ScoreStore's score")
	}
	return ss.Num / ss.Denom, nil
}

// UpdateConfig updates the configuration of a ScoreStore
func (ss *ScoreStore) UpdateConfig(opts ...Option) error {
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

// update updates the ScoreStore's numerator and denominator with a new sample.
// The ScoreStore's score is calculated as a weighted average with a decay factor.
// The new sample is added by the following formula:
//
//	num = num * decay_factor + sample * weight
//	denom = denom * decay_factor + weight
//	decay_factor = exp(-time_since_last_update / half_life_time)
func (ss *ScoreStore) update(sample float64, sampleTime time.Time) error {
	if ss == nil {
		return fmt.Errorf("cannot update ScoreStore, ScoreStore is nil")
	}

	if sample < 0 {
		return fmt.Errorf("cannot update %s ScoreStore, sample is negative: %f", ss.Name, sample)
	}

	if ss.Time.After(sampleTime) {
		return fmt.Errorf("invalid %s ScoreStore: last update time in the future, last_update_time: %s, sample_time: %s", ss.Name, ss.Time.String(), sampleTime.String())
	}

	timeDiff := sampleTime.Sub(ss.Time).Seconds()
	if timeDiff < 0 {
		return fmt.Errorf("invalid time difference: %f seconds", timeDiff)
	}

	exponent := -(math.Ln2 * timeDiff) / ss.Config.HalfLife.Seconds()
	decayFactor := math.Exp(exponent)

	newNum := ss.Num*decayFactor + sample*ss.Config.Weight
	newDenom := ss.Denom*decayFactor + ss.Config.Weight

	if newDenom <= 0 {
		return fmt.Errorf("cannot update %s ScoreStore, invalid new denominator: %f", ss.Name, newDenom)
	}

	ss.Num = newNum
	ss.Denom = newDenom
	ss.Time = sampleTime

	if err := ss.Validate(); err != nil {
		return errors.Wrap(err, "cannot update "+ss.Name+" ScoreStore's num and denom")
	}

	return nil
}

func ConvertToDec(val float64) sdk.Dec {
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

const (
	DefaultLatencyNum      float64 = 0.01
	DefaultSyncNum         float64 = 0.1
	DefaultAvailabilityNum float64 = 1

	LatencyScoreType      = "latency"
	SyncScoreType         = "sync"
	AvailabilityScoreType = "availability"

	// Worst score results for each QoS excellence metric for truncation
	WorstLatencyScore float64 = 30      // seconds
	WorstSyncScore    float64 = 20 * 60 // seconds
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
	return ls.update(sample, sampleTime)
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
	return ss.update(sample, sampleTime)
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
	return as.update(sample, sampleTime)
}
