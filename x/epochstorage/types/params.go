package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyUnstakeHoldBlocks            = []byte("UnstakeHoldBlocks")
	DefaultUnstakeHoldBlocks uint64 = 210
)

var (
	KeyEpochBlocks            = []byte("EpochBlocks")
	DefaultEpochBlocks uint64 = 20
)

var (
	KeyEpochsToSave            = []byte("EpochsToSave")
	DefaultEpochsToSave uint64 = 10
)

var (
	KeyLatestParamChange            = []byte("LatestParamChange")
	DefaultLatestParamChange uint64 = 0
)

var (
	KeyUnstakeHoldBlocksStatic            = []byte("UnstakeHoldBlocksStatic")
	DefaultUnstakeHoldBlocksStatic uint64 = 400
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(
	unstakeHoldBlocks uint64,
	epochBlocks uint64,
	epochsToSave uint64,
	latestParamChange uint64,
	unstakeHoldBlocksStatic uint64,
) Params {
	return Params{
		UnstakeHoldBlocks:       unstakeHoldBlocks,
		EpochBlocks:             epochBlocks,
		EpochsToSave:            epochsToSave,
		LatestParamChange:       latestParamChange,
		UnstakeHoldBlocksStatic: unstakeHoldBlocksStatic,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultUnstakeHoldBlocks,
		DefaultEpochBlocks,
		DefaultEpochsToSave,
		DefaultLatestParamChange,
		DefaultUnstakeHoldBlocksStatic,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyUnstakeHoldBlocks, &p.UnstakeHoldBlocks, validateUnstakeHoldBlocks),
		paramtypes.NewParamSetPair(KeyEpochBlocks, &p.EpochBlocks, validateEpochBlocks),
		paramtypes.NewParamSetPair(KeyEpochsToSave, &p.EpochsToSave, validateEpochsToSave),
		paramtypes.NewParamSetPair(KeyLatestParamChange, &p.LatestParamChange, validateLatestParamChange),
		paramtypes.NewParamSetPair(KeyUnstakeHoldBlocksStatic, &p.UnstakeHoldBlocksStatic, validateUnstakeHoldBlocksStatic),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateUnstakeHoldBlocks(p.UnstakeHoldBlocks); err != nil {
		return err
	}

	if err := validateEpochBlocks(p.EpochBlocks); err != nil {
		return err
	}

	if err := validateEpochsToSave(p.EpochsToSave); err != nil {
		return err
	}

	if err := validateUnstakeHoldBlocksStatic(p.UnstakeHoldBlocksStatic); err != nil {
		return err
	}

	if err := validateBlocksParams(p.UnstakeHoldBlocks, p.UnstakeHoldBlocksStatic, p.EpochBlocks*p.EpochsToSave); err != nil {
		return err
	}

	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// validateUnstakeHoldBlocks validates the UnstakeHoldBlocks param
func validateUnstakeHoldBlocks(v interface{}) error {
	unstakeHoldBlocks, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = unstakeHoldBlocks

	return nil
}

// validateEpochBlocks validates the EpochBlocks param
func validateEpochBlocks(v interface{}) error {
	epochBlocks, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}
	if epochBlocks == 0 {
		return fmt.Errorf("invalid parameter epochBlocks- cant be 0")
	}
	// TODO implement validation
	_ = epochBlocks

	return nil
}

// validateEpochsToSave validates the EpochsToSave param
func validateEpochsToSave(v interface{}) error {
	epochsToSave, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = epochsToSave

	return nil
}

// validateLatestParamChange validates the LatestParamChange param
func validateLatestParamChange(v interface{}) error {
	_, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	return nil
}

// validateUnstakeHoldBlocks validates the UnstakeHoldBlocks param
func validateUnstakeHoldBlocksStatic(v interface{}) error {
	unstakeHoldBlocks, ok := v.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", v)
	}

	// TODO implement validation
	_ = unstakeHoldBlocks

	return nil
}

// validateUnstakeHoldBlocks validates the UnstakeHoldBlocks param
func validateBlocksParams(unstakeHoldBlocks uint64, unstakeHoldBlocksStatic uint64, blocksToSave uint64) error {
	if !(unstakeHoldBlocksStatic > unstakeHoldBlocks && unstakeHoldBlocks > blocksToSave) {
		return fmt.Errorf("parameters do not follow the rule of: unstakeHoldBlocksStatic >  unstakeHoldBlocks > blocksToSave")
	}

	return nil
}
