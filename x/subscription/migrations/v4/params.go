package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*ParamsV4)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&ParamsV4{})
}

// NewParams creates a new Params instance
func NewParams() ParamsV4 {
	return ParamsV4{}
}

// DefaultParams returns a default set of parameters
func DefaultParams() ParamsV4 {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (p *ParamsV4) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

// Validate validates the set of params
func (p ParamsV4) Validate() error {
	return nil
}

// String implements the Stringer interface.
func (p ParamsV4) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}
