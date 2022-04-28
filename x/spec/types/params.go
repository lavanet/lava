package types

import (
	fmt "fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var (
	KeyGeolocationCount            = []byte("GeolocationCount")
	DefaultGeolocationCount uint64 = 8
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
func NewParams(geolocationCount uint64, maxCU uint64) Params {
	return Params{GeolocationCount: geolocationCount, MaxCU: maxCU}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultGeolocationCount, DefaultMaxCU)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyGeolocationCount, &p.GeolocationCount, validateGeolocationCount),
		paramtypes.NewParamSetPair(KeyMaxCU, &p.MaxCU, validateMaxCU),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateGeolocationCount(p.GeolocationCount); err != nil {
		return err
	}

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

func validateGeolocationCount(v interface{}) error {
	geolocationCount, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = geolocationCount

	return nil
}

func validateMaxCU(v interface{}) error {
	maxCU, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = maxCU

	return nil
}
