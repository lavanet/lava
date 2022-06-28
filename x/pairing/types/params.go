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
	KeyMinStakeProvider              = []byte("MinStakeProvider")
	DefaultMinStakeProvider sdk.Coin = sdk.NewCoin("ulava", sdk.NewInt(1000))
)

var (
	KeyMinStakeClient              = []byte("MinStakeClient")
	DefaultMinStakeClient sdk.Coin = sdk.NewCoin("ulava", sdk.NewInt(100))
)

var (
	KeyMintCoinsPerCU             = []byte("MintCoinsPerCU")
	DefaultMintCoinsPerCU sdk.Dec = sdk.NewDecWithPrec(1, 1) //0.1
)

var (
	KeyBurnCoinsPerCU             = []byte("BurnCoinsPerCU")
	DefaultBurnCoinsPerCU sdk.Dec = sdk.NewDecWithPrec(5, 2) //0.05
)

var (
	KeyFraudStakeSlashingFactor = []byte("FraudStakeSlashingFactor")
	// TODO: Determine the default value
	DefaultFraudStakeSlashingFactor sdk.Dec = sdk.NewDecWithPrec(0, 0) //0
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

		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(0)}, 5000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(500)}, 15000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(2000)}, 50000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(5000)}, 250000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(100000)}, 500000},
		{sdk.Coin{Denom: epochstoragetypes.TokenDenom, Amount: sdk.NewIntFromUint64(9999900000)}, 9999999999},
	}}
)

var (
	KeyUnpayLimit             = []byte("UnpayLimit")
	DefaultUnpayLimit sdk.Dec = sdk.NewDecWithPrec(1, 1) //0.1 = 10%
)
var (
	KeySlashLimit             = []byte("SlashLimit")
	DefaultSlashLimit sdk.Dec = sdk.NewDecWithPrec(2, 1) //0.2 = 20%
)

var (
	KeyDataReliabilityReward             = []byte("DataReliabilityReward")
	DefaultDataReliabilityReward sdk.Dec = sdk.NewDecWithPrec(5, 2) //0.05
)

var (
	KeyQoSWeight             = []byte("QoSWeight")
	DefaultQoSWeight sdk.Dec = sdk.NewDecWithPrec(5, 1) //0.5
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	minStakeProvider sdk.Coin,
	minStakeClient sdk.Coin,
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
) Params {
	return Params{
		MinStakeProvider:         minStakeProvider,
		MinStakeClient:           minStakeClient,
		MintCoinsPerCU:           mintCoinsPerCU,
		BurnCoinsPerCU:           burnCoinsPerCU,
		FraudStakeSlashingFactor: fraudStakeSlashingFactor,
		FraudSlashingAmount:      fraudSlashingAmount,
		ServicersToPairCount:     servicersToPairCount,
		EpochBlocksOverlap:       epochBlocksOverlap,
		StakeToMaxCUList:         stakeToMaxCUList,
		UnpayLimit:               unpayLimit,
		SlashLimit:               slashLimit,
		DataReliabilityReward:    dataReliabilityReward,
		QoSWeight:                qoSWeight,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMinStakeProvider,
		DefaultMinStakeClient,
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
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinStakeProvider, &p.MinStakeProvider, validateMinStakeProvider),
		paramtypes.NewParamSetPair(KeyMinStakeClient, &p.MinStakeClient, validateMinStakeClient),
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
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMinStakeProvider(p.MinStakeProvider); err != nil {
		return err
	}

	if err := validateMinStakeClient(p.MinStakeClient); err != nil {
		return err
	}

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
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateMinStakeProvider validates the MinStakeProvider param
func validateMinStakeProvider(v interface{}) error {
	minStakeProvider, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if minStakeProvider.Denom != epochstoragetypes.TokenDenom {
		return fmt.Errorf("invalid denom %s on provider minstake param, should be %s", minStakeProvider.Denom, epochstoragetypes.TokenDenom)
	}

	return nil
}

// validateMinStakeClient validates the MinStakeClient param
func validateMinStakeClient(v interface{}) error {
	minStakeClient, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if minStakeClient.Denom != epochstoragetypes.TokenDenom {
		return fmt.Errorf("invalid denom %s on consumer minstake param,, should be %s", minStakeClient.Denom, epochstoragetypes.TokenDenom)
	}

	return nil
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
		if i > 0 {
			if stakeToMaxCU.StakeThreshold.IsLT(stakeToMaxCUList.List[i-1].StakeThreshold) ||
				stakeToMaxCU.MaxComputeUnits <= stakeToMaxCUList.List[i-1].MaxComputeUnits {
				return fmt.Errorf("invalid parameter order: %T", v)
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
