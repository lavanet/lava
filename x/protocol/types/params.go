package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyVersion     = []byte("Version")
	// TODO: Determine the default value
	DefaultVersion = Version{
		ProviderTarget: "",
		ProviderMin:    "",
		ConsumerTarget: "",
		ConsumerMin:    "",
	}
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	version Version,
) Params {
	return Params{
		Version: version,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultVersion,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyVersion, &p.Version, validateVersion),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateVersion(p.Version); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateVersion validates the Version param
func validateVersion(v interface{}) error {
	version, ok := v.(Version)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = providerVersion

	return nil
}
