package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMintCoinsPerCU             = []byte("MintCoinsPerCU")
	DefaultMintCoinsPerCU sdk.Dec = sdk.NewDecWithPrec(1, 1) // 0.1
)

var (
	KeyFraudStakeSlashingFactor = []byte("FraudStakeSlashingFactor")
	// TODO: Determine the default value
	DefaultFraudStakeSlashingFactor sdk.Dec = sdk.NewDecWithPrec(0, 0) // 0
)

var (
	KeyFraudSlashingAmount = []byte("FraudSlashingAmount")
	// TODO: Determine the default value
	DefaultFraudSlashingAmount uint64 = 0
)

var (
	KeyEpochBlocksOverlap = []byte("EpochBlocksOverlap")
	// TODO: Determine the default value
	DefaultEpochBlocksOverlap uint64 = 5
)

var (
	KeyUnpayLimit             = []byte("UnpayLimit")
	DefaultUnpayLimit sdk.Dec = sdk.NewDecWithPrec(1, 1) // 0.1 = 10%
)

var (
	KeySlashLimit             = []byte("SlashLimit")
	DefaultSlashLimit sdk.Dec = sdk.NewDecWithPrec(2, 1) // 0.2 = 20%
)

var (
	KeyDataReliabilityReward             = []byte("DataReliabilityReward")
	DefaultDataReliabilityReward sdk.Dec = sdk.NewDecWithPrec(5, 2) // 0.05
)

var (
	KeyQoSWeight             = []byte("QoSWeight")
	DefaultQoSWeight sdk.Dec = sdk.NewDecWithPrec(5, 1) // 0.5
)

var (
	KeyRecommendedEpochNumToCollectPayment            = []byte("RecommendedEpochNumToCollectPayment") // the recommended amount of max epochs that a provider should wait before collecting its payment (if he'll collect later, there's a higher chance to get punished)
	DefaultRecommendedEpochNumToCollectPayment uint64 = 3
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	mintCoinsPerCU sdk.Dec,
	fraudStakeSlashingFactor sdk.Dec,
	fraudSlashingAmount uint64,
	epochBlocksOverlap uint64,
	unpayLimit sdk.Dec,
	slashLimit sdk.Dec,
	dataReliabilityReward sdk.Dec,
	qoSWeight sdk.Dec,
	recommendedEpochNumToCollectPayment uint64,
) Params {
	return Params{
		MintCoinsPerCU:                      mintCoinsPerCU,
		FraudStakeSlashingFactor:            fraudStakeSlashingFactor,
		FraudSlashingAmount:                 fraudSlashingAmount,
		EpochBlocksOverlap:                  epochBlocksOverlap,
		UnpayLimit:                          unpayLimit,
		SlashLimit:                          slashLimit,
		DataReliabilityReward:               dataReliabilityReward,
		QoSWeight:                           qoSWeight,
		RecommendedEpochNumToCollectPayment: recommendedEpochNumToCollectPayment,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMintCoinsPerCU,
		DefaultFraudStakeSlashingFactor,
		DefaultFraudSlashingAmount,
		DefaultEpochBlocksOverlap,
		DefaultUnpayLimit,
		DefaultSlashLimit,
		DefaultDataReliabilityReward,
		DefaultQoSWeight,
		DefaultRecommendedEpochNumToCollectPayment,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMintCoinsPerCU, &p.MintCoinsPerCU, validateMintCoinsPerCU),
		paramtypes.NewParamSetPair(KeyFraudStakeSlashingFactor, &p.FraudStakeSlashingFactor, validateFraudStakeSlashingFactor),
		paramtypes.NewParamSetPair(KeyFraudSlashingAmount, &p.FraudSlashingAmount, validateFraudSlashingAmount),
		paramtypes.NewParamSetPair(KeyEpochBlocksOverlap, &p.EpochBlocksOverlap, validateEpochBlocksOverlap),
		paramtypes.NewParamSetPair(KeyUnpayLimit, &p.UnpayLimit, validateUnpayLimit),
		paramtypes.NewParamSetPair(KeySlashLimit, &p.SlashLimit, validateSlashLimit),
		paramtypes.NewParamSetPair(KeyDataReliabilityReward, &p.DataReliabilityReward, validateDataReliabilityReward),
		paramtypes.NewParamSetPair(KeyQoSWeight, &p.QoSWeight, validateQoSWeight),
		paramtypes.NewParamSetPair(KeyRecommendedEpochNumToCollectPayment, &p.RecommendedEpochNumToCollectPayment, validateRecommendedEpochNumToCollectPayment),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMintCoinsPerCU(p.MintCoinsPerCU); err != nil {
		return err
	}

	if err := validateFraudStakeSlashingFactor(p.FraudStakeSlashingFactor); err != nil {
		return err
	}

	if err := validateFraudSlashingAmount(p.FraudSlashingAmount); err != nil {
		return err
	}

	if err := validateEpochBlocksOverlap(p.EpochBlocksOverlap); err != nil {
		return err
	}

	if err := validateUnpayLimit(p.UnpayLimit); err != nil {
		return err
	}

	if err := validateSlashLimit(p.SlashLimit); err != nil {
		return err
	}
	if err := validateDataReliabilityReward(p.DataReliabilityReward); err != nil {
		return err
	}
	if err := validateRecommendedEpochNumToCollectPayment(p.RecommendedEpochNumToCollectPayment); err != nil {
		return err
	}
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateMintCoinsPerCU validates the MintCoinsPerCU param
func validateMintCoinsPerCU(v interface{}) error {
	mintCoinsPerCU, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = mintCoinsPerCU

	return nil
}

// validateBurnCoinsPerCU validates the BurnCoinsPerCU param
func validateBurnCoinsPerCU(v interface{}) error {
	burnCoinsPerCU, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = burnCoinsPerCU

	return nil
}

// validateFraudStakeSlashingFactor validates the FraudStakeSlashingFactor param
func validateFraudStakeSlashingFactor(v interface{}) error {
	fraudStakeSlashingFactor, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if fraudStakeSlashingFactor.GT(sdk.OneDec()) || fraudStakeSlashingFactor.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter fraudStakeSlashingFactor")
	}

	return nil
}

// validateFraudSlashingAmount validates the FraudSlashingAmount param
func validateFraudSlashingAmount(v interface{}) error {
	fraudSlashingAmount, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = fraudSlashingAmount

	return nil
}

// validateServicersToPairCount validates the ServicersToPairCount param
func validateServicersToPairCount(v interface{}) error {
	servicersToPairCount, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if servicersToPairCount <= 0 {
		return fmt.Errorf("invalid parameter, servicersToPairCount can't be zero")
	}
	// TODO implement validation
	_ = servicersToPairCount

	return nil
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

// validateUnpayLimit validates the UnpayLimit param
func validateUnpayLimit(v interface{}) error {
	unpayLimit, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if unpayLimit.GT(sdk.OneDec()) || unpayLimit.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter unpayLimit")
	}

	return nil
}

// validateSlashLimit validates the SlashLimit param
func validateSlashLimit(v interface{}) error {
	slashLimit, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if slashLimit.GT(sdk.OneDec()) || slashLimit.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter unpayLimit")
	}

	return nil
}

// validateDataReliabilityReward validates the param
func validateDataReliabilityReward(v interface{}) error {
	dataReliabilityReward, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if dataReliabilityReward.GT(sdk.OneDec()) || dataReliabilityReward.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter DataReliabilityReward")
	}

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
