package score_test

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/lavanet/lava/v3/utils/score"
	"github.com/stretchr/testify/require"
)

func TestScoreStoreCreation(t *testing.T) {
	num, denom, timestamp := float64(1), float64(2), time.Now()
	weight, halfLife, latencyCuFactor := float64(4), 5*time.Second, float64(1)
	opts := []score.Option_Refactor{score.WithWeight(weight), score.WithDecayHalfLife(halfLife)}
	negativeWeightOpts := []score.Option_Refactor{score.WithWeight(-weight), score.WithDecayHalfLife(halfLife), score.WithLatencyCuFactor(latencyCuFactor)}
	negativeHalflifeOpts := []score.Option_Refactor{score.WithWeight(weight), score.WithDecayHalfLife(-halfLife), score.WithLatencyCuFactor(latencyCuFactor)}
	negativeLatencyCuFactorOpts := []score.Option_Refactor{score.WithWeight(weight), score.WithDecayHalfLife(halfLife), score.WithLatencyCuFactor(-latencyCuFactor)}

	template := []struct {
		name      string
		scoreType string
		num       float64
		denom     float64
		timestamp time.Time
		opts      []score.Option_Refactor
		valid     bool
	}{
		{name: "valid", scoreType: score.LatencyScoreType_Refactor, num: num, denom: denom, timestamp: timestamp, opts: nil, valid: true},
		{name: "valid latency store with opts", scoreType: score.LatencyScoreType_Refactor, num: num, denom: denom, timestamp: timestamp, opts: opts, valid: true},
		{name: "valid sync store with opts", scoreType: score.SyncScoreType_Refactor, num: num, denom: denom, timestamp: timestamp, opts: opts, valid: true},
		{name: "valid availability store with opts", scoreType: score.AvailabilityScoreType_Refactor, num: num, denom: denom, timestamp: timestamp, opts: opts, valid: true},

		{name: "invalid negative num", scoreType: score.LatencyScoreType_Refactor, num: -num, denom: denom, timestamp: timestamp, opts: nil, valid: false},
		{name: "invalid negative denom", scoreType: score.LatencyScoreType_Refactor, num: num, denom: -denom, timestamp: timestamp, opts: nil, valid: false},
		{name: "invalid zero denom", scoreType: score.LatencyScoreType_Refactor, num: num, denom: 0, timestamp: timestamp, opts: nil, valid: false},
		{name: "invalid option - negative weight", scoreType: score.LatencyScoreType_Refactor, num: num, denom: denom, timestamp: timestamp, opts: negativeWeightOpts, valid: false},
		{name: "invalid option - negative half life", scoreType: score.LatencyScoreType_Refactor, num: num, denom: denom, timestamp: timestamp, opts: negativeHalflifeOpts, valid: false},
		{name: "invalid option - negative latency cu factor", scoreType: score.LatencyScoreType_Refactor, num: num, denom: denom, timestamp: timestamp, opts: negativeLatencyCuFactorOpts, valid: false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			store, err := score.NewCustomScoreStore_Refactor(tt.scoreType, tt.num, tt.denom, tt.timestamp, tt.opts...)
			if tt.valid {
				require.NoError(t, err)
				require.Equal(t, tt.scoreType, store.GetName())
				require.Equal(t, tt.num, store.GetNum())
				require.Equal(t, tt.denom, store.GetDenom())
				require.Equal(t, tt.timestamp, store.GetLastUpdateTime())
				if tt.opts != nil {
					require.Equal(t, weight, store.GetConfig().Weight)
					require.Equal(t, halfLife, store.GetConfig().HalfLife)
				}
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestDefaultScoreStoreCreation(t *testing.T) {
	template := []struct {
		name      string
		scoreType string
	}{
		{name: "latency store", scoreType: score.LatencyScoreType_Refactor},
		{name: "sync store", scoreType: score.SyncScoreType_Refactor},
		{name: "availability store", scoreType: score.AvailabilityScoreType_Refactor},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			store := score.NewScoreStore_Refactor(tt.scoreType)
			var expectedNum float64
			switch tt.scoreType {
			case score.LatencyScoreType_Refactor:
				expectedNum = score.DefaultLatencyNum_Refactor
			case score.SyncScoreType_Refactor:
				expectedNum = score.DefaultSyncNum_Refactor
			case score.AvailabilityScoreType_Refactor:
				expectedNum = score.DefaultAvailabilityNum_Refactor
			}

			require.Equal(t, tt.scoreType, store.GetName())
			require.Equal(t, expectedNum, store.GetNum())
			require.Equal(t, float64(1), store.GetDenom())
			require.InEpsilon(t, time.Now().Add(-score.InitialDataStaleness_Refactor).UTC().Unix(), store.GetLastUpdateTime().UTC().Unix(), 0.01)
			require.Equal(t, score.DefaultWeight_Refactor, store.GetConfig().Weight)
			require.Equal(t, score.DefaultHalfLifeTime_Refactor, store.GetConfig().HalfLife)
		})
	}
}

func TestScoreStoreValidation(t *testing.T) {
	validConfig := score.Config_Refactor{Weight: 1, HalfLife: time.Second, LatencyCuFactor: 1}
	invalidConfig1 := score.Config_Refactor{Weight: -1, HalfLife: time.Second, LatencyCuFactor: 1}
	invalidConfig2 := score.Config_Refactor{Weight: 1, HalfLife: time.Second, LatencyCuFactor: 1.01}

	template := []struct {
		name  string
		store score.ScoreStore_Refactor
		valid bool
	}{
		{name: "valid", store: score.ScoreStore_Refactor{Name: "dummy", Num: 1, Denom: 1, Time: time.Now(), Config: validConfig}, valid: true},
		{name: "invalid negative num", store: score.ScoreStore_Refactor{Name: "dummy", Num: -1, Denom: 1, Time: time.Now(), Config: validConfig}, valid: false},
		{name: "invalid negative denom", store: score.ScoreStore_Refactor{Name: "dummy", Num: 1, Denom: -1, Time: time.Now(), Config: validConfig}, valid: false},
		{name: "invalid zero denom", store: score.ScoreStore_Refactor{Name: "dummy", Num: 1, Denom: 0, Time: time.Now(), Config: validConfig}, valid: false},
		{name: "invalid config weight", store: score.ScoreStore_Refactor{Name: "dummy", Num: 1, Denom: 1, Time: time.Now(), Config: invalidConfig1}, valid: false},
		{name: "invalid config latency cu factor", store: score.ScoreStore_Refactor{Name: "dummy", Num: 1, Denom: 1, Time: time.Now(), Config: invalidConfig2}, valid: false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.store.Validate()
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestScoreStoreResolve(t *testing.T) {
	validConfig := score.Config_Refactor{Weight: 1, HalfLife: time.Second, LatencyCuFactor: 0.1}
	template := []struct {
		name   string
		store  score.ScoreStore_Refactor
		result float64
		valid  bool
	}{
		{name: "valid", store: score.ScoreStore_Refactor{Num: 5, Denom: 16, Config: validConfig}, result: 0.3125, valid: true},
		{name: "invalid num", store: score.ScoreStore_Refactor{Num: -5, Denom: 16, Config: validConfig}, result: 0.3125, valid: false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.store.Resolve()
			if tt.valid {
				require.NoError(t, err)
				require.Equal(t, tt.result, res)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestScoreStoreUpdateConfig(t *testing.T) {
	store := score.NewScoreStore_Refactor(score.LatencyScoreType_Refactor)
	weight, latencyCuFactor := float64(2), float64(1)
	halfLife := 3 * time.Second

	validOpts := []score.Option_Refactor{score.WithWeight(weight), score.WithDecayHalfLife(halfLife), score.WithLatencyCuFactor(latencyCuFactor)}
	invalidOpts := []score.Option_Refactor{score.WithWeight(-weight), score.WithDecayHalfLife(-halfLife), score.WithLatencyCuFactor(-latencyCuFactor)}

	err := store.UpdateConfig(validOpts...)
	require.NoError(t, err)
	require.Equal(t, weight, store.GetConfig().Weight)
	require.Equal(t, halfLife, store.GetConfig().HalfLife)
	require.Equal(t, latencyCuFactor, store.GetConfig().LatencyCuFactor)

	for _, opt := range invalidOpts {
		err = store.UpdateConfig(opt)
		require.Error(t, err)
		require.Equal(t, weight, store.GetConfig().Weight)
		require.Equal(t, halfLife, store.GetConfig().HalfLife)
		require.Equal(t, latencyCuFactor, store.GetConfig().LatencyCuFactor)
	}
}

func TestScoreStoreUpdate(t *testing.T) {
	num, denom, timestamp := float64(1), float64(2), time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)
	weight, halfLife, latencyCuFactor := float64(4), 5*time.Millisecond, 0.5
	sample, sampleTime := float64(1), timestamp.Add(10*time.Millisecond)

	// in this test, we add a sample after 10 milliseconds, so the exponent is:
	// time_since_last_update/half_life_time = 10ms / 5ms = 2
	expectedNum := num*math.Exp(-2*math.Ln2) + weight*sample
	expectedLatencyNum := math.Exp(-2*math.Ln2) + weight*sample*latencyCuFactor
	expectedDenom := denom*math.Exp(-2*math.Ln2) + weight

	template := []struct {
		name      string
		scoreType string
		sample    float64
		valid     bool
	}{
		{name: "valid latency", scoreType: score.LatencyScoreType_Refactor, sample: sample, valid: true},
		{name: "valid sync", scoreType: score.SyncScoreType_Refactor, sample: sample, valid: true},
		{name: "valid availability", scoreType: score.AvailabilityScoreType_Refactor, sample: sample, valid: true},

		{name: "invalid negative latency sample", scoreType: score.LatencyScoreType_Refactor, sample: -sample, valid: false},
		{name: "invalid negative sync sample", scoreType: score.SyncScoreType_Refactor, sample: -sample, valid: false},
		{name: "invalid negative availability sample", scoreType: score.AvailabilityScoreType_Refactor, sample: -sample, valid: false},
		{name: "invalid availability sample - not 0/1", scoreType: score.AvailabilityScoreType_Refactor, sample: 0.5, valid: false},
	}

	for _, tt := range template {
		t.Run(tt.name, func(t *testing.T) {
			store, err := score.NewCustomScoreStore_Refactor(tt.scoreType, num, denom, timestamp,
				score.WithWeight(weight), score.WithDecayHalfLife(halfLife), score.WithLatencyCuFactor(latencyCuFactor))
			require.NoError(t, err)

			err = store.Update(tt.sample, sampleTime)
			if tt.valid {
				if tt.scoreType == score.LatencyScoreType_Refactor {
					require.Equal(t, expectedLatencyNum, store.GetNum())
				} else {
					require.Equal(t, expectedNum, store.GetNum())
				}
				require.Equal(t, expectedDenom, store.GetDenom())
				require.Equal(t, sampleTime, store.GetLastUpdateTime())
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestScoreStoreUpdateIdenticalSamples verifies that updating the score with
// many identical samples should keep the score value. In other words, the
// ScoreStore's num and denom will change, but resolving the fracture
// should have the same results as always
func TestScoreStoreUpdateIdenticalSamples(t *testing.T) {
	num, denom, timestamp := float64(94), float64(17), time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)
	weight, halfLife := float64(4), 500*time.Millisecond

	store, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight), score.WithDecayHalfLife(halfLife))
	require.NoError(t, err)

	// update the ScoreStore with many identical samples
	iterations := 50
	sampleTime := timestamp
	sample := float64(20)
	for i := 0; i < iterations; i++ {
		sampleTime = sampleTime.Add(time.Duration(rand.Int63n(500)) * time.Millisecond)
		err = store.Update(sample, sampleTime)
		require.NoError(t, err)
	}

	// with many identical samples, the expected score should be the sample value
	expected := sample
	score, err := store.Resolve()
	require.NoError(t, err)
	require.InEpsilon(t, expected, score, 0.000001)
}

// TestScoreStoreUpdateIdenticalSamplesThenBetter verifies that updating the score with
// many identical samples and then better identical samples, the score value should be
// as the better sample value
func TestScoreStoreUpdateIdenticalSamplesThenBetter(t *testing.T) {
	num, denom, timestamp := float64(94), float64(17), time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)
	weight, halfLife := float64(4), 500*time.Millisecond

	store, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight), score.WithDecayHalfLife(halfLife))
	require.NoError(t, err)

	// update the ScoreStore with many identical samples
	iterations := 50
	sampleTime := timestamp
	sample := float64(20)
	for i := 0; i < iterations; i++ {
		sampleTime = sampleTime.Add(time.Duration(rand.Int63n(500)) * time.Millisecond)
		err = store.Update(sample, sampleTime)
		require.NoError(t, err)
	}

	// with many identical samples, the expected score should be the sample value
	expected := sample
	score, err := store.Resolve()
	require.NoError(t, err)
	require.InEpsilon(t, expected, score, 0.000001)

	// update the ScoreStore with many better identical samples
	betterSample := float64(3)
	for i := 0; i < iterations; i++ {
		sampleTime = sampleTime.Add(time.Duration(rand.Int63n(500)) * time.Millisecond)
		err = store.Update(betterSample, sampleTime)
		require.NoError(t, err)
	}

	// the expected score should be the better sample value
	expected = betterSample
	score, err = store.Resolve()
	require.NoError(t, err)
	require.InEpsilon(t, expected, score, 0.000001)
}

// TestScoreStoreUpdateDecayFactors checks that updating a ScoreStore after a
// short/long time has a different influence on the ScoreStore. Since updating
// involves multiplying the old score value with a decay factor, adding a new
// sample after a long time should change the score more drastically
func TestScoreStoreUpdateDecayFactors(t *testing.T) {
	num, denom, timestamp := float64(100), float64(20), time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)
	weight, halfLife := float64(4), 500*time.Millisecond
	originalScore := num / denom

	// setup two identical stores
	store1, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight), score.WithDecayHalfLife(halfLife))
	require.NoError(t, err)
	store2, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight), score.WithDecayHalfLife(halfLife))
	require.NoError(t, err)

	// update first store with a sample after a short time, and the other
	// with a sample after a long time
	err = store1.Update(1, timestamp.Add(10*time.Millisecond))
	require.NoError(t, err)
	err = store2.Update(1, timestamp.Add(500*time.Millisecond))
	require.NoError(t, err)

	// get the difference of each store's score from the original score
	// store 2 should have a larger difference
	score1, err := store1.Resolve()
	require.NoError(t, err)
	score2, err := store2.Resolve()
	require.NoError(t, err)
	require.Greater(t, math.Abs(score2-originalScore), math.Abs(score1-originalScore))
}

// TestScoreStoreStaysWithinRange tests that if all the samples
// are in range [x, y], then the resolved score is also between
// [x, y]. It should work for every decay factor and weights.
func TestScoreStoreStaysWithinRange(t *testing.T) {
	timestamp, halfLife := time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC), 500*time.Millisecond
	minRangeValue, maxRangeValue := float64(0), float64(100)

	store, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, 1, 1, timestamp,
		score.WithWeight(1), score.WithDecayHalfLife(halfLife))
	require.NoError(t, err)

	// update the ScoreStore with samples within the range with different weights and
	// decay factors
	iterations := 1000
	sampleTime := timestamp
	for i := 0; i < iterations; i++ {
		sampleTime = sampleTime.Add(time.Duration(rand.Int63n(500)) * time.Millisecond)
		store.UpdateConfig(score.WithWeight(float64(rand.Int63n(int64(maxRangeValue)))))
		err = store.Update(float64(rand.Int63n(int64(maxRangeValue))), sampleTime)
		require.NoError(t, err)
	}

	// the expected score should be within the defined range
	score, err := store.Resolve()
	require.NoError(t, err)
	require.LessOrEqual(t, score, maxRangeValue)
	require.GreaterOrEqual(t, score, minRangeValue)
}

// TestScoreStoreHalfLife tests the update of ScoreStore for different
// half life factors. Assuming two identical stores, each with different
// half life factor, we update them in the same time. The store with the lower
// half life factor will be influenced more than the one with the higher half
// life factor
func TestScoreStoreHalfLife(t *testing.T) {
	num, denom, timestamp := float64(100), float64(20), time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)
	weight := float64(4)
	originalScore := num / denom
	shortHalfLife, longHalfLife := 10*time.Millisecond, 500*time.Millisecond

	// setup two identical stores (store1 = short, store2 = long)
	store1, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight), score.WithDecayHalfLife(shortHalfLife))
	require.NoError(t, err)
	store2, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight), score.WithDecayHalfLife(longHalfLife))
	require.NoError(t, err)

	// update the stores with the same sample and sample time
	err = store1.Update(1, timestamp.Add(100*time.Millisecond))
	require.NoError(t, err)
	err = store2.Update(1, timestamp.Add(100*time.Millisecond))
	require.NoError(t, err)

	// get the difference of each store's score from the original score
	// store 1 should have a larger difference (since it had the short
	// half life factor)
	score1, err := store1.Resolve()
	require.NoError(t, err)
	score2, err := store2.Resolve()
	require.NoError(t, err)
	require.Greater(t, math.Abs(score1-originalScore), math.Abs(score2-originalScore))
}

// TestScoreStoreWeight tests the update of ScoreStore for different
// weights. Assuming two identical stores, each with a different weight,
// we update them in the same time. The store with the higher weight
// will be influenced more than the other one
func TestScoreStoreWeight(t *testing.T) {
	num, denom, timestamp := float64(100), float64(20), time.Date(0, 0, 0, 0, 0, 1, 0, time.UTC)
	halfLife := 500 * time.Millisecond
	originalScore := num / denom
	weight1, weight2 := float64(4), float64(40)

	// setup two identical stores (store1 = low weight, store2 = high weight)
	store1, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight1), score.WithDecayHalfLife(halfLife))
	require.NoError(t, err)
	store2, err := score.NewCustomScoreStore_Refactor(score.LatencyScoreType_Refactor, num, denom, timestamp,
		score.WithWeight(weight2), score.WithDecayHalfLife(halfLife))
	require.NoError(t, err)

	// update the stores with the same sample and sample time
	err = store1.Update(1, timestamp.Add(100*time.Millisecond))
	require.NoError(t, err)
	err = store2.Update(1, timestamp.Add(100*time.Millisecond))
	require.NoError(t, err)

	// get the difference of each store's score from the original score
	// store 2 should have a larger difference (since it had the short
	// half life factor)
	score1, err := store1.Resolve()
	require.NoError(t, err)
	score2, err := store2.Resolve()
	require.NoError(t, err)
	require.Greater(t, math.Abs(score2-originalScore), math.Abs(score1-originalScore))
}

// TestScoreStoreAvailabilityResolveNonZero verifies that the Resolve()
// method of the AvailabilityScoreStore doesn't return zero when num/denom = 0
// Zero is undesirable since in QoS Compute() method we divide by the
// availability score
func TestScoreStoreAvailabilityResolveNonZero(t *testing.T) {
	store, err := score.NewCustomScoreStore_Refactor(score.AvailabilityScoreType_Refactor, 0, 1, time.Now())
	require.NoError(t, err)
	score, err := store.Resolve()
	require.NoError(t, err)
	require.NotZero(t, score)
}
