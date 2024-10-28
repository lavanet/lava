package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/v3/x/pairing/types"
	"github.com/stretchr/testify/require"
)

func TestQosConfigValidation(t *testing.T) {
	template := []struct {
		name   string
		config types.Config_Refactor
		valid  bool
	}{
		{name: "valid", config: types.DefaultConfig, valid: true},
		{name: "valid - default block error probabililty (-1)", config: types.Config_Refactor{SyncFactor: sdk.OneDec(), FailureCost: 3, StrategyFactor: sdk.OneDec(), BlockErrorProbability: types.DefaultBlockErrorProbability}, valid: true},

		{name: "invalid negative sync factor", config: types.Config_Refactor{SyncFactor: sdk.NewDec(-1), FailureCost: 3, StrategyFactor: sdk.OneDec(), BlockErrorProbability: sdk.OneDec()}, valid: false},
		{name: "invalid greater than one sync factor", config: types.Config_Refactor{SyncFactor: sdk.NewDec(2), FailureCost: 3, StrategyFactor: sdk.OneDec(), BlockErrorProbability: sdk.OneDec()}, valid: false},
		{name: "invalid negative failure cost", config: types.Config_Refactor{SyncFactor: sdk.OneDec(), FailureCost: -3, StrategyFactor: sdk.OneDec(), BlockErrorProbability: sdk.OneDec()}, valid: false},
		{name: "invalid negative strategy factor", config: types.Config_Refactor{SyncFactor: sdk.OneDec(), FailureCost: 3, StrategyFactor: sdk.NewDec(-1), BlockErrorProbability: sdk.OneDec()}, valid: false},
		{name: "invalid negative block error probabililty (excluding default)", config: types.Config_Refactor{SyncFactor: sdk.OneDec(), FailureCost: 3, StrategyFactor: sdk.OneDec(), BlockErrorProbability: sdk.NewDec(-2)}, valid: false},
		{name: "invalid greater than 1 block error probabililty", config: types.Config_Refactor{SyncFactor: sdk.OneDec(), FailureCost: 3, StrategyFactor: sdk.OneDec(), BlockErrorProbability: sdk.NewDec(2)}, valid: false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestQosConfigModification(t *testing.T) {
	config := types.Config_Refactor{SyncFactor: sdk.OneDec(), FailureCost: 3, StrategyFactor: sdk.OneDec(), BlockErrorProbability: types.DefaultBlockErrorProbability}
	syncFactor := sdk.NewDec(2)
	failureCost := int64(3)
	strategyFactor := sdk.NewDec(2)
	blockErrorProbability := sdk.OneDec()

	opts := []types.Option{
		types.WithSyncFactor(syncFactor),
		types.WithFailureCost(failureCost),
		types.WithStrategyFactor(strategyFactor),
		types.WithBlockErrorProbability(blockErrorProbability),
	}
	for _, opt := range opts {
		opt(&config)
	}

	require.True(t, syncFactor.Equal(config.SyncFactor))
	require.Equal(t, failureCost, config.FailureCost)
	require.True(t, strategyFactor.Equal(config.StrategyFactor))
	require.True(t, blockErrorProbability.Equal(config.BlockErrorProbability))
}

func TestQosValidation(t *testing.T) {
	latency, sync, availability := sdk.OneDec(), sdk.OneDec(), sdk.OneDec()

	template := []struct {
		name  string
		qos   types.QualityOfServiceReport
		valid bool
	}{
		{name: "valid", qos: types.QualityOfServiceReport{Latency: latency, Sync: sync, Availability: availability}, valid: true},
		{name: "invalid negative latency", qos: types.QualityOfServiceReport{Latency: latency.Neg(), Sync: sync, Availability: availability}, valid: false},
		{name: "invalid negative sync", qos: types.QualityOfServiceReport{Latency: latency, Sync: sync.Neg(), Availability: availability}, valid: false},
		{name: "invalid negative availability", qos: types.QualityOfServiceReport{Latency: latency, Sync: sync, Availability: availability.Neg()}, valid: false},
		{name: "invalid zero availability", qos: types.QualityOfServiceReport{Latency: latency, Sync: sync, Availability: sdk.ZeroDec()}, valid: false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.qos.Validate()
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestQosCompute verifies the QoS object Compute() method works as expected for its two cases: normal
// and with configured block error probability
func TestQosCompute(t *testing.T) {
	blockErrorProbability := sdk.OneDec()
	qos := types.QualityOfServiceReport{Latency: sdk.OneDec(), Sync: sdk.OneDec(), Availability: sdk.OneDec()}

	// with the given QoS report and the default config, the expected score results:
	// normal: score = latency + sync*syncFactor + ((1/availability) - 1) * FailureCost = 1 + 1*0.1 + (1/1 - 1) * 3 = 1.1
	// with block error probability: score = latency + blockErrorProbability * FailureCost + ((1/availability) - 1) * FailureCost = 1 + 1*3 + (1/1 - 1) * 3 = 4
	expectedScoreDefault := sdk.NewDecWithPrec(11, 1)
	expectedScoreBlockErrorProbability := sdk.NewDec(4)

	template := []struct {
		name          string
		opts          []types.Option
		expectedScore sdk.Dec
	}{
		{name: "normal", opts: []types.Option{}, expectedScore: expectedScoreDefault},
		{name: "with block error probability", opts: []types.Option{types.WithBlockErrorProbability(blockErrorProbability)}, expectedScore: expectedScoreBlockErrorProbability},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			score, err := qos.ComputeQoSExcellence_Refactor(tt.opts...)
			require.NoError(t, err)
			require.True(t, tt.expectedScore.Equal(score))
		})
	}
}

// TestQosFailureCost checks that higher failure cost means worse score
func TestQosFailureCost(t *testing.T) {
	qos := types.QualityOfServiceReport{Latency: sdk.OneDec(), Sync: sdk.OneDec(), Availability: sdk.NewDecWithPrec(5, 1)}
	failureCost, highFailureCost := int64(1), int64(3)

	score, err := qos.ComputeQoSExcellence_Refactor(types.WithFailureCost(failureCost))
	require.NoError(t, err)
	scoreHighFailure, err := qos.ComputeQoSExcellence_Refactor(types.WithFailureCost(highFailureCost))
	require.NoError(t, err)
	require.True(t, scoreHighFailure.GT(score))

	scoreWithProb, err := qos.ComputeQoSExcellence_Refactor(types.WithFailureCost(failureCost), types.WithBlockErrorProbability(sdk.OneDec()))
	require.NoError(t, err)
	scoreHighFailureWithProb, err := qos.ComputeQoSExcellence_Refactor(types.WithFailureCost(highFailureCost), types.WithBlockErrorProbability(sdk.OneDec()))
	require.NoError(t, err)
	require.True(t, scoreHighFailureWithProb.GT(scoreWithProb))
}

// TestQosSyncFactor checks that higher syncFactor means worse score
func TestQosSyncFactor(t *testing.T) {
	qos := types.QualityOfServiceReport{Latency: sdk.OneDec(), Sync: sdk.OneDec(), Availability: sdk.NewDecWithPrec(5, 1)}
	syncFactor, highSyncFactor := sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(8, 1)

	score, err := qos.ComputeQoSExcellence_Refactor(types.WithSyncFactor(syncFactor))
	require.NoError(t, err)
	scoreHighSyncFactor, err := qos.ComputeQoSExcellence_Refactor(types.WithSyncFactor(highSyncFactor))
	require.NoError(t, err)
	require.True(t, scoreHighSyncFactor.GT(score))
}

// TestQosStrategyFactor checks that the strategy factor works as expected
// The strategy factor is an additional multiplier to the sync factor that is used
// to weaken/strengthen the influence of the sync score compared to the latency
func TestQosStrategyFactor(t *testing.T) {
	// we configure availability = 1 to zero the availability
	qos := types.QualityOfServiceReport{Latency: sdk.OneDec(), Sync: sdk.OneDec(), Availability: sdk.OneDec()}

	// we get the balancedScore with a balanced strategy and subtract the latency component of the balancedScore
	// this way, our balancedScore will only be syncFactor*sync (syncFactor = configuredSyncFactor * strategyFactor)
	balancedScore, err := qos.ComputeQoSExcellence_Refactor(types.WithStrategyFactor(types.BalancedStrategyFactor))
	require.NoError(t, err)
	balancedScore = balancedScore.Sub(sdk.OneDec())

	// calculate score with latency strategy - sync component should be smaller than the component in balancedScore
	latencyScore, err := qos.ComputeQoSExcellence_Refactor(types.WithStrategyFactor(types.LatencyStrategyFactor))
	require.NoError(t, err)
	latencyScore = latencyScore.Sub(sdk.OneDec())
	require.True(t, balancedScore.GT(latencyScore))

	// calculate score with sync freshness strategy - sync component should be bigger than the component in balancedScore
	syncScore, err := qos.ComputeQoSExcellence_Refactor(types.WithStrategyFactor(types.SyncFreshnessStrategyFactor))
	require.NoError(t, err)
	syncScore = syncScore.Sub(sdk.OneDec())
	require.True(t, balancedScore.LT(syncScore))
}

// TestQosBlockErrorProbability checks that larger block error probability means worse score
func TestQosBlockErrorProbability(t *testing.T) {
	qos := types.QualityOfServiceReport{Latency: sdk.OneDec(), Sync: sdk.OneDec(), Availability: sdk.OneDec()}
	probabililty, highProbabililty := sdk.NewDecWithPrec(5, 1), sdk.NewDecWithPrec(8, 1)

	score, err := qos.ComputeQoSExcellence_Refactor(types.WithBlockErrorProbability(probabililty))
	require.NoError(t, err)
	scoreHighProbabililty, err := qos.ComputeQoSExcellence_Refactor(types.WithBlockErrorProbability(highProbabililty))
	require.NoError(t, err)
	require.True(t, scoreHighProbabililty.GT(score))
}
