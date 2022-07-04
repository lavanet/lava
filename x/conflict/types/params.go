package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMajorityPercent = []byte("MajorityPercent")
	// TODO: Determine the default value
	DefaultMajorityPercent string = "majority_percent"
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	majorityPercent string,
) Params {
	return Params{
		MajorityPercent: majorityPercent,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMajorityPercent,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMajorityPercent, &p.MajorityPercent, validateMajorityPercent),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMajorityPercent(p.MajorityPercent); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateMajorityPercent validates the MajorityPercent param
func validateMajorityPercent(v interface{}) error {
	majorityPercent, ok := v.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = majorityPercent

	return nil
}
