package types

import sdk "github.com/cosmos/cosmos-sdk/types"

type SpecEmissionPart struct {
	ChainID  string
	Emission sdk.Dec
}
