package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyMajorityPrecent             = []byte("MajorityPrecent")
	DefaultMajorityPrecent sdk.Dec = sdk.NewDecWithPrec(95, 2)
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
	KeyWinnerRewardPrecent             = []byte("WinnerRewardPrecent")
	DefaultWinnerRewardPrecent sdk.Dec = sdk.NewDecWithPrec(15, 2)
)

var (
	KeyClientRewardPrecent             = []byte("ClientRewardPrecent")
	DefaultClientRewardPrecent sdk.Dec = sdk.NewDecWithPrec(10, 2)
)

var (
	KeyVotersRewardPrecent             = []byte("VotersRewardPrecent")
	DefaultVotersRewardPrecent sdk.Dec = sdk.NewDecWithPrec(15, 2)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	majorityPrecent sdk.Dec, voteStartSpan uint64, votePeriod uint64, winnerRewardPrecent sdk.Dec, clientRewardPrecent sdk.Dec, votersRewardPrecent sdk.Dec) Params {
	return Params{
		MajorityPercent:     majorityPrecent,
		VoteStartSpan:       voteStartSpan,
		VotePeriod:          votePeriod,
		WinnerRewardPercent: winnerRewardPrecent,
		ClientRewardPercent: clientRewardPrecent,
		VotersRewardPercent: votersRewardPrecent,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultMajorityPrecent,
		DefaultVoteStartSpan,
		DefaultVotePeriod,
		DefaultWinnerRewardPrecent,
		DefaultClientRewardPrecent,
		DefaultVotersRewardPrecent,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMajorityPrecent, &p.MajorityPercent, validateMajorityPrecent),
		paramtypes.NewParamSetPair(KeyVoteStartSpan, &p.VoteStartSpan, validateVoteStartSpan),
		paramtypes.NewParamSetPair(KeyVotePeriod, &p.VotePeriod, validateVotePeriod),
		paramtypes.NewParamSetPair(KeyWinnerRewardPrecent, &p.WinnerRewardPercent, validateWinnerRewardPrecent),
		paramtypes.NewParamSetPair(KeyClientRewardPrecent, &p.ClientRewardPercent, validateClientRewardPrecent),
		paramtypes.NewParamSetPair(KeyVotersRewardPrecent, &p.VotersRewardPercent, validateVotersRewardPrecent),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateMajorityPrecent(p.MajorityPercent); err != nil {
		return err
	}

	if err := validateVoteStartSpan(p.VoteStartSpan); err != nil {
		return err
	}

	if err := validateVotePeriod(p.VotePeriod); err != nil {
		return err
	}

	if err := validateWinnerRewardPrecent(p.WinnerRewardPercent); err != nil {
		return err
	}

	if err := validateClientRewardPrecent(p.ClientRewardPercent); err != nil {
		return err
	}

	if err := validateVotersRewardPrecent(p.VotersRewardPercent); err != nil {
		return err
	}
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateMajorityPrecent validates the majorityPrecent param
func validateMajorityPrecent(v interface{}) error {
	majorityPrecent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if majorityPrecent.GT(sdk.OneDec()) || majorityPrecent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter majorityPrecent")
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

func validateWinnerRewardPrecent(v interface{}) error {
	winnerRewardPrecent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if winnerRewardPrecent.GT(sdk.OneDec()) || winnerRewardPrecent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter winnerRewardPrecent")
	}

	return nil
}

func validateClientRewardPrecent(v interface{}) error {
	clientRewardPrecent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if clientRewardPrecent.GT(sdk.OneDec()) || clientRewardPrecent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter clientRewardPrecent")
	}

	return nil
}

func validateVotersRewardPrecent(v interface{}) error {
	votersRewardPrecent, ok := v.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	if votersRewardPrecent.GT(sdk.OneDec()) || votersRewardPrecent.LT(sdk.ZeroDec()) {
		return fmt.Errorf("invalid parameter votersRewardPrecent")
	}

	return nil
}
