package types

import "cosmossdk.io/math"

type SpecEmissionPart struct {
	ChainID  string
	Emission math.LegacyDec
}
