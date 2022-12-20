package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewMinter returns a new Minter object with the given inflation and annual
// provisions values.
func NewMinter(inflation, annualProvisions sdk.Dec, phase, startPhaseBlock uint64, targetSupply sdk.Int) Minter {
	return Minter{
		Inflation:        inflation,
		AnnualProvisions: annualProvisions,
		Phase:            phase,
		StartPhaseBlock:  startPhaseBlock,
		TargetSupply:     targetSupply,
	}
}

// InitialMinter returns an initial Minter object with a given inflation value.
func InitialMinter(inflation sdk.Dec) Minter {
	return NewMinter(
		inflation,
		sdk.NewDec(0),
		0,
		0,
		sdk.NewInt(0),
	)
}

// DefaultInitialMinter returns a default initial Minter object for a new chain
// which uses an inflation rate of 13%.
func DefaultInitialMinter() Minter {
	return InitialMinter(
		sdk.NewDecWithPrec(13, 2),
	)
}

// validate minter
func ValidateMinter(minter Minter) error {
	if minter.Inflation.IsNegative() {
		return fmt.Errorf("mint parameter Inflation should be positive, is %s",
			minter.Inflation.String())
	}
	return nil
}

// PhaseInflationRate returns the inflation rate by phase.
func (m Minter) PhaseInflationRate(phase uint64) sdk.Dec {
	switch {
	case phase > 12:
		return sdk.ZeroDec()

	case phase == 1:
		return sdk.NewDecWithPrec(40, 2)

	case phase == 2:
		return sdk.NewDecWithPrec(20, 2)

	case phase == 3:
		return sdk.NewDecWithPrec(10, 2)

	default:
		// Phase4:  9%
		// Phase5:  8%
		// Phase6:  7%
		// ...
		// Phase12: 1%
		return sdk.NewDecWithPrec(13-int64(phase), 2)
	}
}

// NextPhase returns the new phase.
func (m Minter) NextPhase(params Params, currentSupply sdk.Int) uint64 {
	nonePhase := m.Phase == 0
	if nonePhase {
		return 1
	}

	if currentSupply.LT(m.TargetSupply) {
		return m.Phase
	}

	return m.Phase + 1
}

// NextAnnualProvisions returns the annual provisions based on current total
// supply and inflation rate.
func (m Minter) NextAnnualProvisions(_ Params, totalSupply sdk.Int) sdk.Dec {
	return m.Inflation.MulInt(totalSupply)
}

// BlockProvision returns the provisions for a block based on the annual
// provisions rate.
func (m Minter) BlockProvision(params Params, totalSupply sdk.Int) sdk.Coin {
	provisionAmt := m.AnnualProvisions.QuoInt(sdk.NewInt(int64(params.BlocksPerYear)))

	// Because of rounding, we might mint too many tokens in this phase, let's limit it
	futureSupply := totalSupply.Add(provisionAmt.TruncateInt())
	if futureSupply.GT(m.TargetSupply) {
		return sdk.NewCoin(params.MintDenom, m.TargetSupply.Sub(totalSupply))
	}

	return sdk.NewCoin(params.MintDenom, provisionAmt.TruncateInt())
}
