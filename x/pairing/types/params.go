package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyEpochBlocksOverlap = []byte("EpochBlocksOverlap")
	// TODO: Determine the default value
	DefaultEpochBlocksOverlap uint64 = 4
)

var (
	KeyQoSWeight             = []byte("QoSWeight")
	DefaultQoSWeight sdk.Dec = sdk.NewDecWithPrec(5, 1) // 0.5
)

var (
	KeyRecommendedEpochNumToCollectPayment            = []byte("RecommendedEpochNumToCollectPayment") // the recommended amount of max epochs that a provider should wait before collecting its payment (if he'll collect later, there's a higher chance to get punished)
	DefaultRecommendedEpochNumToCollectPayment uint64 = 3
)

var (
	KeyReputationVarianceStabilizationPeriod            = []byte("ReputationVarianceStabilizationPeriod")
	DefaultReputationVarianceStabilizationPeriod uint64 = 7 * 24 * 60 * 60 // week
)

var (
	KeyReputationLatencyOverSyncFactor                    = []byte("ReputationLatencyOverSyncFactor")
	DefaultReputationLatencyOverSyncFactor math.LegacyDec = sdk.NewDecWithPrec(1, 1) // 0.1
)

var (
	KeyReputationHalfLifeFactor            = []byte("ReputationHalfLifeFactor")
	DefaultReputationHalfLifeFactor uint64 = 12 // months
)

var (
	KeyReputationRelayFailureCost            = []byte("ReputationRelayFailureCost")
	DefaultReputationRelayFailureCost uint64 = 3 // seconds
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	epochBlocksOverlap uint64,
	qoSWeight sdk.Dec,
	recommendedEpochNumToCollectPayment uint64,
	reputationVarianceStabilizationPeriod uint64,
	reputationLatencyOverSyncFactor math.LegacyDec,
	reputationHalfLifeFactor uint64,
	reputationRelayFailureCost uint64,
) Params {
	return Params{
		EpochBlocksOverlap:                    epochBlocksOverlap,
		QoSWeight:                             qoSWeight,
		RecommendedEpochNumToCollectPayment:   recommendedEpochNumToCollectPayment,
		ReputationVarianceStabilizationPeriod: reputationVarianceStabilizationPeriod,
		ReputationLatencyOverSyncFactor:       reputationLatencyOverSyncFactor,
		ReputationHalfLifeFactor:              reputationHalfLifeFactor,
		ReputationRelayFailureCost:            reputationRelayFailureCost,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultEpochBlocksOverlap,
		DefaultQoSWeight,
		DefaultRecommendedEpochNumToCollectPayment,
		DefaultReputationVarianceStabilizationPeriod,
		DefaultReputationLatencyOverSyncFactor,
		DefaultReputationHalfLifeFactor,
		DefaultReputationRelayFailureCost,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyEpochBlocksOverlap, &p.EpochBlocksOverlap, validateEpochBlocksOverlap),
		paramtypes.NewParamSetPair(KeyQoSWeight, &p.QoSWeight, validateQoSWeight),
		paramtypes.NewParamSetPair(KeyRecommendedEpochNumToCollectPayment, &p.RecommendedEpochNumToCollectPayment, validateRecommendedEpochNumToCollectPayment),
		paramtypes.NewParamSetPair(KeyReputationVarianceStabilizationPeriod, &p.ReputationVarianceStabilizationPeriod, validateReputationVarianceStabilizationPeriod),
		paramtypes.NewParamSetPair(KeyReputationLatencyOverSyncFactor, &p.ReputationLatencyOverSyncFactor, validateReputationLatencyOverSyncFactor),
		paramtypes.NewParamSetPair(KeyReputationHalfLifeFactor, &p.ReputationHalfLifeFactor, validateReputationHalfLifeFactor),
		paramtypes.NewParamSetPair(KeyReputationRelayFailureCost, &p.ReputationRelayFailureCost, validateReputationRelayFailureCost),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateEpochBlocksOverlap(p.EpochBlocksOverlap); err != nil {
		return err
	}

	if err := validateQoSWeight(p.QoSWeight); err != nil {
		return err
	}

	if err := validateRecommendedEpochNumToCollectPayment(p.RecommendedEpochNumToCollectPayment); err != nil {
		return err
	}

	if err := validateReputationVarianceStabilizationPeriod(p.ReputationVarianceStabilizationPeriod); err != nil {
		return err
	}

	if err := validateReputationLatencyOverSyncFactor(p.ReputationLatencyOverSyncFactor); err != nil {
		return err
	}

	if err := validateReputationHalfLifeFactor(p.ReputationHalfLifeFactor); err != nil {
		return err
	}

	if err := validateReputationRelayFailureCost(p.ReputationRelayFailureCost); err != nil {
		return err
	}
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateEpochBlocksOverlap validates the EpochBlocksOverlap param
func validateEpochBlocksOverlap(v interface{}) error {
	epochBlocksOverlap, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = epochBlocksOverlap

	return nil
}

// validateDataReliabilityReward validates the param
func validateQoSWeight(v interface{}) error {
	QoSWeight, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if QoSWeight.GT(sdk.OneDec()) || QoSWeight.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter QoSWeight")
	}

	return nil
}

// validateRecommendedEpochNumToCollectPayment validates the RecommendedEpochNumToCollectPayment param
func validateRecommendedEpochNumToCollectPayment(v interface{}) error {
	recommendedEpochNumToCollectPayment, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = recommendedEpochNumToCollectPayment

	return nil
}

// validateReputationVarianceStabilizationPeriod validates the ReputationVarianceStabilizationPeriod param
func validateReputationVarianceStabilizationPeriod(v interface{}) error {
	reputationVarianceStabilizationPeriod, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = reputationVarianceStabilizationPeriod

	return nil
}

// validateReputationLatencyOverSyncFactor validates the ReputationLatencyOverSyncFactor param
func validateReputationLatencyOverSyncFactor(v interface{}) error {
	reputationLatencyOverSyncFactor, ok := v.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = reputationLatencyOverSyncFactor

	return nil
}

// validateReputationHalfLifeFactor validates the ReputationHalfLifeFactor param
func validateReputationHalfLifeFactor(v interface{}) error {
	reputationHalfLifeFactor, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = reputationHalfLifeFactor

	return nil
}

// validateReputationRelayFailureCost validates the ReputationRelayFailureCost param
func validateReputationRelayFailureCost(v interface{}) error {
	reputationRelayFailureCost, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = reputationRelayFailureCost

	return nil
}
