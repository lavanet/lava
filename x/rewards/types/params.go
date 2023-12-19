package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMinBondedTarget             = []byte("MinBondedTarget")
	DefaultMinBondedTarget sdk.Dec = sdk.NewDecWithPrec(6, 1) // 0.6
)

var (
	KeyMaxBondedTarget             = []byte("MaxBondedTarget")
	DefaultMaxBondedTarget sdk.Dec = sdk.NewDecWithPrec(8, 1) // 0.8
)

var (
	KeyLowFactor             = []byte("LowFactor")
	DefaultLowFactor sdk.Dec = sdk.NewDecWithPrec(5, 1) // 0.5
)

var (
	KeyLeftoverBurnRate             = []byte("LeftoverBurnRate")
	DefaultLeftOverBurnRate sdk.Dec = sdk.OneDec()
)

var (
	KeyMaxRewardBoost     = []byte("MaxRewardBoost")
	DefaultMaxRewardBoost = uint64(5)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	minBondedTarget sdk.Dec,
	maxBondedTarget sdk.Dec,
	lowFactor sdk.Dec,
	leftoverBurnRate sdk.Dec,
	maxRewardBoost uint64,
) Params {
	return Params{
		MinBondedTarget:  minBondedTarget,
		MaxBondedTarget:  maxBondedTarget,
		LowFactor:        lowFactor,
		LeftoverBurnRate: leftoverBurnRate,
		MaxRewardBoost:   maxRewardBoost,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMinBondedTarget,
		DefaultMaxBondedTarget,
		DefaultLowFactor,
		DefaultLeftOverBurnRate,
		DefaultMaxRewardBoost,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinBondedTarget, &p.MinBondedTarget, validateDec),
		paramtypes.NewParamSetPair(KeyMaxBondedTarget, &p.MaxBondedTarget, validateDec),
		paramtypes.NewParamSetPair(KeyLowFactor, &p.LowFactor, validateDec),
		paramtypes.NewParamSetPair(KeyLeftoverBurnRate, &p.LeftoverBurnRate, validateDec),
		paramtypes.NewParamSetPair(KeyMaxRewardBoost, &p.MaxRewardBoost, validateuint64),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateDec(p.MinBondedTarget); err != nil {
		return err
	}

	if err := validateDec(p.MaxBondedTarget); err != nil {
		return err
	}

	if p.MinBondedTarget.GT(p.MaxBondedTarget) {
		return fmt.Errorf("min_bonded_target cannot be greater than max_bonded_target")
	}

	if err := validateDec(p.LowFactor); err != nil {
		return err
	}

	if err := validateDec(p.LeftoverBurnRate); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateDec validates the Dec param is between 0 and 1
func validateDec(v interface{}) error {
	param, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if param.GT(sdk.OneDec()) || param.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter minBondedTarget")
	}

	return nil
}

func validateuint64(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}
