package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	commontypes "github.com/lavanet/lava/v2/utils/common/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMinSelfDelegation              = []byte("MinSelfDelegation")
	DefaultMinSelfDelegation sdk.Coin = sdk.NewCoin(commontypes.TokenDenom, sdk.NewInt(100000000)) // 100 lava = 100,000,000 ulava
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	minSelfDelegation sdk.Coin,
) Params {
	return Params{
		MinSelfDelegation: minSelfDelegation,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMinSelfDelegation)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMinSelfDelegation, &p.MinSelfDelegation, validateMinSelfDelegation),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func validateMinSelfDelegation(v interface{}) error {
	selfDelegation, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid min self delegation type %T", v)
	}

	if selfDelegation.Denom != commontypes.TokenDenom {
		return fmt.Errorf("invalid min self delegation denom %s", selfDelegation.Denom)
	}

	return nil
}
