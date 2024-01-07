package types

import (
	fmt "fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var (
	KeyMaxCU            = []byte("MaxCU")
	DefaultMaxCU uint64 = 10000
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(maxCU uint64) Params {
	return Params{MaxCU: maxCU}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMaxCU)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxCU, &p.MaxCU, validateMaxCU),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMaxCU(p.MaxCU); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func validateMaxCU(v interface{}) error {
	maxCU, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}
	if maxCU == 0 {
		return fmt.Errorf("invalid parameter maxCU - can't be 0")
	}
	// TODO implement validation
	_ = maxCU

	return nil
}
