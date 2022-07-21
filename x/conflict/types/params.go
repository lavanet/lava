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
	KeyWinnerRewardPercent             = []byte("WinnerRewardPercent")
	DefaultWinnerRewardPercent sdk.Dec = sdk.NewDecWithPrec(15, 2)
)

var (
	KeyClientRewardPercent             = []byte("ClientRewardPercent")
	DefaultClientRewardPercent sdk.Dec = sdk.NewDecWithPrec(10, 2)
)

var (
	KeyVotersRewardPercent             = []byte("VotersRewardPercent")
	DefaultVotersRewardPercent sdk.Dec = sdk.NewDecWithPrec(15, 2)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	majorityPercent sdk.Dec, voteStartSpan uint64, votePeriod uint64, winnerRewardPercent sdk.Dec, clientRewardPercent sdk.Dec, votersRewardPercent sdk.Dec) Params {
	return Params{
		MajorityPercent:     majorityPercent,
		VoteStartSpan:       voteStartSpan,
		VotePeriod:          votePeriod,
		WinnerRewardPercent: winnerRewardPercent,
		ClientRewardPercent: clientRewardPercent,
		VotersRewardPercent: votersRewardPercent,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMajorityPercent,
		DefaultVoteStartSpan,
		DefaultVotePeriod,
		DefaultWinnerRewardPercent,
		DefaultClientRewardPercent,
		DefaultVotersRewardPercent,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMajorityPercent, &p.MajorityPercent, validateMajorityPercent),
		paramtypes.NewParamSetPair(KeyVoteStartSpan, &p.VoteStartSpan, validateVoteStartSpan),
		paramtypes.NewParamSetPair(KeyVotePeriod, &p.VotePeriod, validateVotePeriod),
		paramtypes.NewParamSetPair(KeyWinnerRewardPercent, &p.WinnerRewardPercent, validateWinnerRewardPercent),
		paramtypes.NewParamSetPair(KeyClientRewardPercent, &p.ClientRewardPercent, validateClientRewardPercent),
		paramtypes.NewParamSetPair(KeyVotersRewardPercent, &p.VotersRewardPercent, validateVotersRewardPercent),
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

	if err := validateWinnerRewardPercent(p.WinnerRewardPercent); err != nil {
		return err
	}

	if err := validateClientRewardPercent(p.ClientRewardPercent); err != nil {
		return err
	}

	if err := validateVotersRewardPercent(p.VotersRewardPercent); err != nil {
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

func validateWinnerRewardPercent(v interface{}) error {
	winnerRewardPercent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if winnerRewardPercent.GT(sdk.OneDec()) || winnerRewardPercent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter winnerRewardPercent")
	}

	return nil
}

func validateClientRewardPercent(v interface{}) error {
	clientRewardPercent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if clientRewardPercent.GT(sdk.OneDec()) || clientRewardPercent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter clientRewardPercent")
	}

	return nil
}

func validateVotersRewardPercent(v interface{}) error {
	votersRewardPercent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if votersRewardPercent.GT(sdk.OneDec()) || votersRewardPercent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter votersRewardPercent")
	}

	return nil
}
