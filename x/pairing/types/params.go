package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	epochstoragetypes "github.com/lavanet/lava/x/epochstorage/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMintCoinsPerCU             = []byte("MintCoinsPerCU")
	DefaultMintCoinsPerCU sdk.Dec = sdk.NewDecWithPrec(1, 1) // 0.1
)

var (
	KeyBurnCoinsPerCU             = []byte("BurnCoinsPerCU")
	DefaultBurnCoinsPerCU sdk.Dec = sdk.NewDecWithPrec(5, 2) // 0.05
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
	KeyServicersToPairCount = []byte("ServicersToPairCount")
	// TODO: Determine the default value
	DefaultServicersToPairCount uint64 = 2
)

var (
	KeyEpochBlocksOverlap = []byte("EpochBlocksOverlap")
	// TODO: Determine the default value
	DefaultEpochBlocksOverlap uint64 = 5
)

var (
	KeyStakeToMaxCUList = []byte("StakeToMaxCUList")
	// TODO: Determine the default value
	DefaultStakeToMaxCUList StakeToMaxCUList = StakeToMaxCUList{List: []StakeToMaxCU{
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(1)}, 5000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(500)}, 15000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(2000)}, 50000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(5000)}, 250000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(100000)}, 500000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(9999900000)}, 9999999999},
	}}
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
	burnCoinsPerCU sdk.Dec,
	fraudStakeSlashingFactor sdk.Dec,
	fraudSlashingAmount uint64,
	servicersToPairCount uint64,
	epochBlocksOverlap uint64,
	stakeToMaxCUList StakeToMaxCUList,
	unpayLimit sdk.Dec,
	slashLimit sdk.Dec,
	dataReliabilityReward sdk.Dec,
	qoSWeight sdk.Dec,
	recommendedEpochNumToCollectPayment uint64,
) Params {
	return Params{
		MintCoinsPerCU:                      mintCoinsPerCU,
		BurnCoinsPerCU:                      burnCoinsPerCU,
		FraudStakeSlashingFactor:            fraudStakeSlashingFactor,
		FraudSlashingAmount:                 fraudSlashingAmount,
		ServicersToPairCount:                servicersToPairCount,
		EpochBlocksOverlap:                  epochBlocksOverlap,
		StakeToMaxCUList:                    stakeToMaxCUList,
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
		DefaultBurnCoinsPerCU,
		DefaultFraudStakeSlashingFactor,
		DefaultFraudSlashingAmount,
		DefaultServicersToPairCount,
		DefaultEpochBlocksOverlap,
		DefaultStakeToMaxCUList,
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
		paramtypes.NewParamSetPair(KeyBurnCoinsPerCU, &p.BurnCoinsPerCU, validateBurnCoinsPerCU),
		paramtypes.NewParamSetPair(KeyFraudStakeSlashingFactor, &p.FraudStakeSlashingFactor, validateFraudStakeSlashingFactor),
		paramtypes.NewParamSetPair(KeyFraudSlashingAmount, &p.FraudSlashingAmount, validateFraudSlashingAmount),
		paramtypes.NewParamSetPair(KeyServicersToPairCount, &p.ServicersToPairCount, validateServicersToPairCount),
		paramtypes.NewParamSetPair(KeyEpochBlocksOverlap, &p.EpochBlocksOverlap, validateEpochBlocksOverlap),
		paramtypes.NewParamSetPair(KeyStakeToMaxCUList, &p.StakeToMaxCUList, validateStakeToMaxCUList),
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

	if err := validateBurnCoinsPerCU(p.BurnCoinsPerCU); err != nil {
		return err
	}

	if err := validateFraudStakeSlashingFactor(p.FraudStakeSlashingFactor); err != nil {
		return err
	}

	if err := validateFraudSlashingAmount(p.FraudSlashingAmount); err != nil {
		return err
	}

	if err := validateServicersToPairCount(p.ServicersToPairCount); err != nil {
		return err
	}

	if err := validateEpochBlocksOverlap(p.EpochBlocksOverlap); err != nil {
		return err
	}

	if err := validateStakeToMaxCUList(p.StakeToMaxCUList); err != nil {
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

// validateStakeToMaxCUList validates the StakeToMaxCUList param
func validateStakeToMaxCUList(v interface{}) error {
	stakeToMaxCUList, ok := v.(StakeToMaxCUList)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	for i, stakeToMaxCU := range stakeToMaxCUList.List {
		if stakeToMaxCU.StakeThreshold.Amount.Sign() == -1 || stakeToMaxCU.StakeThreshold.Amount.Int64() == 0 {
			return fmt.Errorf("invalid stakeThreshold %v. Must be non-zero positive integer", stakeToMaxCU.StakeThreshold)
		}
		if i > 0 {
			if stakeToMaxCU.StakeThreshold.IsLT(stakeToMaxCUList.List[i-1].StakeThreshold) ||
				stakeToMaxCU.MaxComputeUnits <= stakeToMaxCUList.List[i-1].MaxComputeUnits {
				return fmt.Errorf("invalid parameter stakeToMaxCUList order, the order must be ascending: index %d value %+v and index %d value %+v", i, stakeToMaxCU.StakeThreshold, i-1, stakeToMaxCUList.List[i-1].StakeThreshold)
			}
		}
	}

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
