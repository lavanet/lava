package types

import (
	fmt "fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var (
	KeyMaxCU                         = []byte("MaxCU")
	KeyWhiteListExpeditedMsgs        = []byte("WhitelistedExpeditedMsgs")
	DefaultMaxCU              uint64 = 10000
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(maxCU uint64, whitelistExpeditedMsgs []string) Params {
	return Params{MaxCU: maxCU, WhitelistedExpeditedMsgs: whitelistExpeditedMsgs}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMaxCU, nil)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxCU, &p.MaxCU, validateMaxCU),
		paramtypes.NewParamSetPair(KeyWhiteListExpeditedMsgs, &p.WhitelistedExpeditedMsgs, validateWhitelistedExpeditedMsgs),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMaxCU(p.MaxCU); err != nil {
		return err
	}

	if err := validateWhitelistedExpeditedMsgs(p.WhitelistedExpeditedMsgs); err != nil {
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

	// TODO implement validation
	_ = maxCU

	return nil
}

func validateWhitelistedExpeditedMsgs(v interface{}) error {
	whitelistedExpeditedMsgs, ok := v.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// check for duplicates
	whitelistedExpeditedMsgsMap := make(map[string]struct{})
	for _, msg := range whitelistedExpeditedMsgs {
		if _, ok := whitelistedExpeditedMsgsMap[msg]; ok {
			return fmt.Errorf("duplicate message in WhitelistedExpeditedMessages: %s", msg)
		}
		whitelistedExpeditedMsgsMap[msg] = struct{}{}
	}

	return nil
}
