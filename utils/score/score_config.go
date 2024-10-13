package score

import (
	"fmt"
	"time"
)

const (
	DefaultHalfLifeTime_Refactor = time.Hour
	MaxHalfTime_Refactor         = 3 * time.Hour

	DefaultWeight_Refactor     float64 = 1
	ProbeUpdateWeight_Refactor float64 = 0.25
	RelayUpdateWeight_Refactor float64 = 1
)

// Config defines a collection of parameters that can be used by ScoreStore
type Config_Refactor struct {
	Weight   float64
	HalfLife time.Duration
}

// Validate validates the Config's fields hold valid values
func (c Config_Refactor) Validate() error {
	if c.Weight <= 0 {
		return fmt.Errorf("invalid config: weight must be strictly positive, weight: %f", c.Weight)
	}
	if c.HalfLife.Seconds() <= 0 {
		return fmt.Errorf("invalid config: half life time must be strictly positive, half life: %f", c.HalfLife.Seconds())
	}
	return nil
}

// String prints a Config's fields
func (c Config_Refactor) String() string {
	return fmt.Sprintf("weight: %f, decay_half_life_time_sec: %f", c.Weight, c.HalfLife.Seconds())
}

// Option is used as a generic and elegant way to configure a new ScoreStore
type Option_Refactor func(*Config_Refactor)

func WithWeight(weight float64) Option_Refactor {
	return func(c *Config_Refactor) {
		c.Weight = weight
	}
}

func WithDecayHalfLife(halfLife time.Duration) Option_Refactor {
	return func(c *Config_Refactor) {
		c.HalfLife = halfLife
	}
}
