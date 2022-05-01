package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMinStakeProvider            = []byte("MinStakeProvider")
	DefaultMinStakeProvider uint64 = 1000
)

var (
	KeyMinStakeClient            = []byte("MinStakeClient")
	DefaultMinStakeClient uint64 = 100
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
	KeyStakeRangeToCUList = []byte("StakeRangeToCUList")
	// TODO: Determine the default value
	DefaultStakeRangeToCUList StakeRangeToCUList = StakeRangeToCUList{List: []*StakeRangeToCU{{0, 500, 5000},
		{501, 2000, 15000},
		{2001, 5000, 50000},
		{5001, 10000, 250000},
		{10001, 99999999, 500000},
	}}
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	minStakeProvider uint64,
	minStakeClient uint64,
	mintCoinsPerCU sdk.Dec,
	burnCoinsPerCU sdk.Dec,
	fraudStakeSlashingFactor sdk.Dec,
	fraudSlashingAmount uint64,
	servicersToPairCount uint64,
	epochBlocksOverlap uint64,
	stakeRangeToCUList StakeRangeToCUList,
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
		StakeRangeToCUList:       stakeRangeToCUList,
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
		DefaultStakeRangeToCUList,
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
		paramtypes.NewParamSetPair(KeyStakeRangeToCUList, &p.StakeRangeToCUList, validateStakeRangeToCUList),
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

	if err := validateStakeRangeToCUList(p.StakeRangeToCUList); err != nil {
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
	minStakeProvider, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = minStakeProvider

	return nil
}

// validateMinStakeClient validates the MinStakeClient param
func validateMinStakeClient(v interface{}) error {
	minStakeClient, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = minStakeClient

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

	// TODO implement validation
	_ = fraudStakeSlashingFactor

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

// validateStakeRangeToCUList validates the StakeRangeToCUList param
func validateStakeRangeToCUList(v interface{}) error {
	epochBlocksOverlap, ok := v.(StakeRangeToCUList)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = epochBlocksOverlap

	return nil
}
