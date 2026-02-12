package lavasession

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCalcWeightsByStake_NonStaticProviders(t *testing.T) {
	// Non-static providers should be unaffected by the static-provider stake logic.
	p1 := NewConsumerSessionWithProvider("p1", nil, 1, 1, sdk.NewInt64Coin("ulava", 7))
	p2 := NewConsumerSessionWithProvider("p2", nil, 1, 1, sdk.NewInt64Coin("ulava", 11))

	providers := map[uint64]*ConsumerSessionsWithProvider{
		0: p1,
		1: p2,
	}

	weights := CalcWeightsByStake(providers)
	require.Equal(t, int64(7), weights["p1"])
	require.Equal(t, int64(11), weights["p2"])
}

func TestCalcWeightsByStake_StaticProviders_DefaultWhenStakeOmitted(t *testing.T) {
	// If stake is omitted for static providers, stakeSize is 0 and they get the legacy boost.
	// With ONLY static providers, maxWeight remains 1, so boost yields WeightMultiplierForStaticProviders (==10).
	p1 := NewConsumerSessionWithProvider("p1", nil, 1, 1, sdk.NewInt64Coin("ulava", 0))
	p2 := NewConsumerSessionWithProvider("p2", nil, 1, 1, sdk.NewInt64Coin("ulava", 0))
	p1.StaticProvider = true
	p2.StaticProvider = true

	providers := map[uint64]*ConsumerSessionsWithProvider{
		0: p1,
		1: p2,
	}

	weights := CalcWeightsByStake(providers)
	require.Equal(t, int64(WeightMultiplierForStaticProviders), weights["p1"])
	require.Equal(t, int64(WeightMultiplierForStaticProviders), weights["p2"])
}

func TestCalcWeightsByStake_StaticProviders_LegacyBoostAgainstMaxStake(t *testing.T) {
	// If there are other (non-static or explicitly-staked static) providers, omitted-stake static providers
	// get boosted relative to the max stake in the pairing list.
	chain := NewConsumerSessionWithProvider("chain", nil, 1, 1, sdk.NewInt64Coin("ulava", 100))
	staticOmitted := NewConsumerSessionWithProvider("staticOmitted", nil, 1, 1, sdk.NewInt64Coin("ulava", 0))
	staticOmitted.StaticProvider = true

	providers := map[uint64]*ConsumerSessionsWithProvider{
		0: chain,
		1: staticOmitted,
	}

	weights := CalcWeightsByStake(providers)
	require.Equal(t, int64(100), weights["chain"])
	require.Equal(t, int64(100*WeightMultiplierForStaticProviders), weights["staticOmitted"])
}

func TestCalcWeightsByStake_StaticProvider_ExplicitStake_NoBoost(t *testing.T) {
	// If stake is explicitly set (>0) for a static provider, it should be used as-is (no boost).
	staticExplicit := NewConsumerSessionWithProvider("staticExplicit", nil, 1, 1, sdk.NewInt64Coin("ulava", 25))
	staticExplicit.StaticProvider = true

	providers := map[uint64]*ConsumerSessionsWithProvider{
		0: staticExplicit,
	}

	weights := CalcWeightsByStake(providers)
	require.Equal(t, int64(25), weights["staticExplicit"])
}

