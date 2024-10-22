package score_test

import (
	"testing"
	"time"

	"github.com/lavanet/lava/v3/utils/score"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	template := []struct {
		name   string
		config score.Config_Refactor
		valid  bool
	}{
		{name: "valid", config: score.Config_Refactor{Weight: 1, HalfLife: time.Second, LatencyCuFactor: 1}, valid: true},
		{name: "invalid weight", config: score.Config_Refactor{Weight: -1, HalfLife: time.Second, LatencyCuFactor: 1}, valid: false},
		{name: "invalid half life", config: score.Config_Refactor{Weight: 1, HalfLife: -time.Second, LatencyCuFactor: 1}, valid: false},
		{name: "invalid zero latency cu factor", config: score.Config_Refactor{Weight: 1, HalfLife: time.Second, LatencyCuFactor: 0}, valid: false},
		{name: "invalid >1 latency cu factor", config: score.Config_Refactor{Weight: 1, HalfLife: time.Second, LatencyCuFactor: 1.01}, valid: false},
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

func TestConfigModification(t *testing.T) {
	config := score.Config_Refactor{Weight: 1, HalfLife: time.Second, LatencyCuFactor: 1}
	weight := float64(2)
	halfLife := 3 * time.Second
	latencyCuFactor := 0.5

	opts := []score.Option_Refactor{
		score.WithWeight(weight),
		score.WithDecayHalfLife(halfLife),
		score.WithLatencyCuFactor(latencyCuFactor),
	}
	for _, opt := range opts {
		opt(&config)
	}

	require.Equal(t, weight, config.Weight)
	require.Equal(t, halfLife, config.HalfLife)
	require.Equal(t, latencyCuFactor, config.LatencyCuFactor)
}
