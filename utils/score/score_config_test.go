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
		config score.Config
		valid  bool
	}{
		{name: "valid", config: score.Config{Weight: 1, HalfLife: time.Second}, valid: true},
		{name: "invalid weight", config: score.Config{Weight: -1, HalfLife: time.Second}, valid: false},
		{name: "valid", config: score.Config{Weight: 1, HalfLife: -time.Second}, valid: false},
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
	config := score.Config{Weight: 1, HalfLife: time.Second}
	weight := float64(2)
	halfLife := 3 * time.Second

	opts := []score.Option{
		score.WithWeight(weight),
		score.WithDecayHalfLife(halfLife),
	}
	for _, opt := range opts {
		opt(&config)
	}

	require.Equal(t, weight, config.Weight)
	require.Equal(t, halfLife, config.HalfLife)
}
