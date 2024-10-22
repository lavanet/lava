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
	DecPrecision_Refactor         int64 = 8
	InitialDataStaleness_Refactor       = 24
)

// ScoreStore is a decaying weighted average object that is used to collect
// providers performace metrics samples (see QoS excellence comment below).
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
type ScoreStore_Refactor struct {
	Name   string
	Num    float64 // using float64 and not math/big for performance
	Denom  float64
	Time   time.Time
	Config Config_Refactor
}

// ScoreStorer defines the interface for all score stores
type ScoreStorer_Refactor interface {
	Update(sample float64, sampleTime time.Time) error
	Resolve() (float64, error)
	Validate() error
	String() string
	UpdateConfig(opts ...Option_Refactor) error

	GetName() string
	GetNum() float64
	GetDenom() float64
	GetLastUpdateTime() time.Time
	GetConfig() Config_Refactor
}

// NewCustomScoreStore creates a new custom ScoreStorer based on the score type
func NewCustomScoreStore_Refactor(scoreType string, num, denom float64, t time.Time, opts ...Option_Refactor) (ScoreStorer_Refactor, error) {
	cfg := defaultConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("cannot create %s ScoreStore, invalid configuration: %w", scoreType, err)
	}

	base := &ScoreStore_Refactor{
		Num:    num,
		Denom:  denom,
		Time:   t,
		Config: cfg,
	}

	if err := base.Validate(); err != nil {
		return nil, fmt.Errorf("cannot create %s ScoreStore, invalid parameters: %w", scoreType, err)
	}

	switch scoreType {
	case LatencyScoreType_Refactor:
		base.Name = LatencyScoreType_Refactor
		return &LatencyScoreStore_Refactor{ScoreStore_Refactor: base}, nil
	case SyncScoreType_Refactor:
		base.Name = SyncScoreType_Refactor
		return &SyncScoreStore_Refactor{ScoreStore_Refactor: base}, nil
	case AvailabilityScoreType_Refactor:
		base.Name = AvailabilityScoreType_Refactor
		return &AvailabilityScoreStore_Refactor{ScoreStore_Refactor: base}, nil
	default:
		return nil, fmt.Errorf("unknown score type: %s", scoreType)
	}
}

// NewScoreStore creates a new default ScoreStorer based on the score type
func NewScoreStore_Refactor(scoreType string) ScoreStorer_Refactor {
	switch scoreType {
	case LatencyScoreType_Refactor:
		// default latency: 10ms
		latencyScoreStore, err := NewCustomScoreStore_Refactor(scoreType, DefaultLatencyNum_Refactor, 1, time.Now().Add(-InitialDataStaleness_Refactor*time.Hour))
		if err != nil {
			utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", err)
		}
		return latencyScoreStore

	case SyncScoreType_Refactor:
		// default sync: 100ms
		syncScoreStore, err := NewCustomScoreStore_Refactor(scoreType, DefaultSyncNum_Refactor, 1, time.Now().Add(-InitialDataStaleness_Refactor*time.Hour))
		if err != nil {
			utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", err)
		}
		return syncScoreStore

	case AvailabilityScoreType_Refactor:
		// default availability: 1
		availabilityScoreStore, err := NewCustomScoreStore_Refactor(scoreType, DefaultAvailabilityNum_Refactor, 1, time.Now().Add(-InitialDataStaleness_Refactor*time.Hour))
		if err != nil {

		}
		return availabilityScoreStore
	default:
		utils.LavaFormatFatal("cannot create default "+scoreType+" ScoreStore", fmt.Errorf("unknown score type: %s", scoreType))
		return nil // not reached
	}
}

// String prints a ScoreStore's fields
func (ss *ScoreStore_Refactor) String() string {
	return fmt.Sprintf("num: %f, denom: %f, last_update_time: %s, config: %s",
		ss.Num, ss.Denom, ss.Time.String(), ss.Config.String())
}

// Validate validates the ScoreStore's fields hold valid values
func (ss *ScoreStore_Refactor) Validate() error {
	if ss.Num < 0 || ss.Denom <= 0 {
		return fmt.Errorf("invalid %s ScoreStore: num or denom are non-positives, num: %f, denom: %f", ss.Name, ss.Num, ss.Denom)
	}

	if err := ss.Config.Validate(); err != nil {
		return errors.Wrap(err, "invalid "+ss.Name+" ScoreStore")
	}
	return nil
}

// Resolve resolves the ScoreStore's frac by dividing the numerator by the denominator
func (ss *ScoreStore_Refactor) Resolve() (float64, error) {
	if err := ss.Validate(); err != nil {
		return 0, errors.Wrap(err, "cannot calculate "+ss.Name+" ScoreStore's score")
	}
	return ss.Num / ss.Denom, nil
}

// UpdateConfig updates the configuration of a ScoreStore
func (ss *ScoreStore_Refactor) UpdateConfig(opts ...Option_Refactor) error {
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
func (ss *ScoreStore_Refactor) Update(sample float64, sampleTime time.Time) error {
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

func (ss *ScoreStore_Refactor) GetName() string {
	return ss.Name
}

func (ss *ScoreStore_Refactor) GetNum() float64 {
	return ss.Num
}

func (ss *ScoreStore_Refactor) GetDenom() float64 {
	return ss.Denom
}

func (ss *ScoreStore_Refactor) GetLastUpdateTime() time.Time {
	return ss.Time
}

func (ss *ScoreStore_Refactor) GetConfig() Config_Refactor {
	return ss.Config
}

func ConvertToDec(val float64) sdk.Dec {
	intScore := int64(math.Round(val * math.Pow(10, float64(DecPrecision_Refactor))))
	return sdk.NewDecWithPrec(intScore, DecPrecision_Refactor)
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
	DefaultLatencyNum_Refactor      float64 = 0.01
	DefaultSyncNum_Refactor         float64 = 0.1
	DefaultAvailabilityNum_Refactor float64 = 1

	LatencyScoreType_Refactor      = "latency"
	SyncScoreType_Refactor         = "sync"
	AvailabilityScoreType_Refactor = "availability"

	// Worst score results for each QoS excellence metric for truncation
	WorstLatencyScore_Refactor      float64 = 30      // seconds
	WorstSyncScore_Refactor         float64 = 20 * 60 // seconds
	WorstAvailabilityScore_Refactor float64 = 0
)

/* ########## Latency ScoreStore ############ */

type LatencyScoreStore_Refactor struct {
	*ScoreStore_Refactor
}

// Update updates the Latency ScoreStore's numerator and denominator with a new sample.
func (ls *LatencyScoreStore_Refactor) Update(sample float64, sampleTime time.Time) error {
	if ls == nil {
		return fmt.Errorf("LatencyScoreStore is nil")
	}

	// normalize the sample with the latency CU factor
	sample = sample * ls.ScoreStore_Refactor.Config.LatencyCuFactor

	return ls.ScoreStore_Refactor.Update(sample, sampleTime)
}

/* ########## Sync ScoreStore ############ */

type SyncScoreStore_Refactor struct {
	*ScoreStore_Refactor
}

// Update updates the Sync ScoreStore's numerator and denominator with a new sample.
func (ss *SyncScoreStore_Refactor) Update(sample float64, sampleTime time.Time) error {
	if ss == nil {
		return fmt.Errorf("SyncScoreStore is nil")
	}
	return ss.ScoreStore_Refactor.Update(sample, sampleTime)
}

/* ########## Availability ScoreStore ############ */

type AvailabilityScoreStore_Refactor struct {
	*ScoreStore_Refactor
}

// Update updates the availability ScoreStore's numerator and denominator with a new sample.
// The new sample must be 0 or 1.
func (as *AvailabilityScoreStore_Refactor) Update(sample float64, sampleTime time.Time) error {
	if as == nil {
		return fmt.Errorf("AvailabilityScoreStore is nil")
	}
	if sample != float64(0) && sample != float64(1) {
		return fmt.Errorf("availability must be 0 (false) or 1 (true), got %f", sample)
	}
	return as.ScoreStore_Refactor.Update(sample, sampleTime)
}
