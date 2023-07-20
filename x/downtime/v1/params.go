package v1

import (
	"fmt"
	"time"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

var (
	ParamKeyDowntimeDuration                 = []byte("DowntimeDuration")
	DefaultParamKeyDowntimeDuration          = 30 * time.Minute
	ParamKeyGarbageCollectionDuration        = []byte("GarbageCollection")
	DefaultParamKeyGarbageCollectionDuration = uint64(10 * 20) // epochs to save * epoch blocks
)

func DefaultParams() Params {
	return Params{
		DowntimeDuration:        DefaultParamKeyDowntimeDuration,
		GarbageCollectionBlocks: DefaultParamKeyGarbageCollectionDuration,
	}
}

var _ paramtypes.ParamSet = (*Params)(nil)

func (m *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamKeyDowntimeDuration, &m.DowntimeDuration, validateDowntimeDuration),
		paramtypes.NewParamSetPair(ParamKeyGarbageCollectionDuration, &m.GarbageCollectionBlocks, validateGarbageCollectionBlocks),
	}
}

func (m *Params) Validate() error {
	if err := validateDowntimeDuration(m.DowntimeDuration); err != nil {
		return err
	}
	if err := validateDowntimeDuration(m.GarbageCollectionBlocks); err != nil {
		return err
	}
	return nil
}

func validateDowntimeDuration(value interface{}) error {
	concrete, ok := value.(time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T, wanted: %T", value, time.Duration(0))
	}

	if concrete <= 0 {
		return fmt.Errorf("invalid downtime duration: %s", concrete)
	}
	return nil
}

func validateGarbageCollectionBlocks(value interface{}) error {
	_, ok := value.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T, wanted: %T", value, uint64(0))
	}
	return nil
}

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}
