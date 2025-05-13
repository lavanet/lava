package types

import (
	fmt "fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	common "github.com/lavanet/lava/v5/utils/common/types"
	"gopkg.in/yaml.v2"
)

var (
	KeyMaxCU                           = []byte("MaxCU")
	KeyallowlistExpeditedMsgs          = []byte("AllowlistedExpeditedMsgs")
	KeyProviderMinStake                = []byte("ProviderMinStake")
	DefaultMaxCU              uint64   = 10000
	DefaultProviderMinStake   sdk.Coin = sdk.NewCoin(common.TokenDenom, sdk.NewInt(5000000000)) // 5K LAVA
)

var _ paramtypes.ParamSet = (*Params)(nil)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(maxCU uint64, allowlistedExpeditedMsgs []string, providerMinStake sdk.Coin) Params {
	return Params{
		MaxCU:                    maxCU,
		AllowlistedExpeditedMsgs: allowlistedExpeditedMsgs,
		ProviderMinStake:         providerMinStake,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMaxCU, nil, DefaultProviderMinStake)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMaxCU, &p.MaxCU, validateMaxCU),
		paramtypes.NewParamSetPair(KeyallowlistExpeditedMsgs, &p.AllowlistedExpeditedMsgs, validateallowlistedExpeditedMsgs),
		paramtypes.NewParamSetPair(KeyProviderMinStake, &p.ProviderMinStake, validateProviderMinStake),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMaxCU(p.MaxCU); err != nil {
		return err
	}

	if err := validateallowlistedExpeditedMsgs(p.AllowlistedExpeditedMsgs); err != nil {
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

func validateallowlistedExpeditedMsgs(v interface{}) error {
	allowlistedExpeditedMsgs, ok := v.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// check for duplicates
	allowlistedExpeditedMsgsMap := make(map[string]struct{})
	for _, msg := range allowlistedExpeditedMsgs {
		if _, ok := allowlistedExpeditedMsgsMap[msg]; ok {
			return fmt.Errorf("duplicate message in allowlistedExpeditedMessages: %s", msg)
		}
		allowlistedExpeditedMsgsMap[msg] = struct{}{}
	}

	return nil
}

func validateProviderMinStake(v interface{}) error {
	providerMinStake, ok := v.(sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid provider min stake type %T", v)
	}

	if providerMinStake.Denom != common.TokenDenom {
		return fmt.Errorf("invalid provider min stake, invalid denom %s", providerMinStake.Denom)
	}

	if providerMinStake.Amount.IsNegative() {
		return fmt.Errorf("invalid provider min stake, negative amount %s", providerMinStake.Amount)
	}

	if providerMinStake.Amount.IsZero() {
		return fmt.Errorf("invalid provider min stake, zero amount %s", providerMinStake.Amount)
	}

	return nil
}
