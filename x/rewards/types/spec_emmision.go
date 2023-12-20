package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type SpecEmmisionPart struct {
	ChainID  string
	Emission sdk.Dec
}
