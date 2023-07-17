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
	DefaultParamKeyGarbageCollectionDuration = 24 * time.Hour * 30 // 30 days
)

func DefaultParams() Params {
	return Params{
		DowntimeDuration:          DefaultParamKeyDowntimeDuration,
		GarbageCollectionDuration: DefaultParamKeyGarbageCollectionDuration,
	}
}

var _ paramtypes.ParamSet = (*Params)(nil)

func (m *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamKeyDowntimeDuration, &m.DowntimeDuration, validateDowntimeDuration),
		paramtypes.NewParamSetPair(ParamKeyGarbageCollectionDuration, &m.GarbageCollectionDuration, validateDowntimeDuration),
	}
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

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}
