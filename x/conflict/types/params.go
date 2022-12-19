package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMajorityPercent             = []byte("MajorityPercent")
	DefaultMajorityPercent sdk.Dec = sdk.NewDecWithPrec(95, 2)
)

var (
	KeyVoteStartSpan            = []byte("VoteStartSpan")
	DefaultVoteStartSpan uint64 = 3
)

var (
	KeyVotePeriod            = []byte("VotePeriod")
	DefaultVotePeriod uint64 = 2
)

var (
	KeyRewards             = []byte("WinnerRewardPercent")
	DefaultRewards Rewards = Rewards{WinnerRewardPercent: sdk.NewDecWithPrec(15, 2), ClientRewardPercent: sdk.NewDecWithPrec(10, 2), VotersRewardPercent: sdk.NewDecWithPrec(15, 2)}
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	majorityPercent sdk.Dec, voteStartSpan uint64, votePeriod uint64, rewards Rewards,
) Params {
	return Params{
		MajorityPercent: majorityPercent,
		VoteStartSpan:   voteStartSpan,
		VotePeriod:      votePeriod,
		Rewards:         rewards,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMajorityPercent,
		DefaultVoteStartSpan,
		DefaultVotePeriod,
		DefaultRewards,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMajorityPercent, &p.MajorityPercent, validateMajorityPercent),
		paramtypes.NewParamSetPair(KeyVoteStartSpan, &p.VoteStartSpan, validateVoteStartSpan),
		paramtypes.NewParamSetPair(KeyVotePeriod, &p.VotePeriod, validateVotePeriod),
		paramtypes.NewParamSetPair(KeyRewards, &p.Rewards, validateRewards),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMajorityPercent(p.MajorityPercent); err != nil {
		return err
	}

	if err := validateVoteStartSpan(p.VoteStartSpan); err != nil {
		return err
	}

	if err := validateVotePeriod(p.VotePeriod); err != nil {
		return err
	}

	if err := validateRewards(p.Rewards); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateMajorityPercent validates the majorityPercent param
func validateMajorityPercent(v interface{}) error {
	majorityPercent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if majorityPercent.GT(sdk.OneDec()) || majorityPercent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter majorityPercent")
	}

	return nil
}

func validateVoteStartSpan(v interface{}) error {
	voteStartSpan, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = voteStartSpan

	return nil
}

func validateVotePeriod(v interface{}) error {
	votePeriod, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = votePeriod

	return nil
}

func validateRewards(v interface{}) error {
	rewards, ok := v.(Rewards)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if rewards.ClientRewardPercent.GT(sdk.OneDec()) || rewards.ClientRewardPercent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter ClientRewardPercent")
	}

	if rewards.VotersRewardPercent.GT(sdk.OneDec()) || rewards.VotersRewardPercent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter VotersRewardPercent")
	}

	if rewards.WinnerRewardPercent.GT(sdk.OneDec()) || rewards.WinnerRewardPercent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter WinnerRewardPercent")
	}

	if rewards.ClientRewardPercent.Add(rewards.VotersRewardPercent).Add(rewards.WinnerRewardPercent).GT(sdk.OneDec()) {
		return fmt.Errorf("sum of all rewards is bigger than 100 percent")
	}

	return nil
}
