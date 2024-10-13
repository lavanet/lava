package score

import (
	"fmt"
	"time"
)

const (
	DefaultHalfLifeTime = time.Hour
	MaxHalfTime         = 3 * time.Hour

	DefaultWeight     float64 = 1
	ProbeUpdateWeight float64 = 0.25
	RelayUpdateWeight float64 = 1
)

// Config defines a collection of parameters that can be used by ScoreStore
type Config struct {
	Weight   float64
	HalfLife time.Duration
}

// Validate validates the Config's fields hold valid values
func (c Config) Validate() error {
	if c.Weight <= 0 {
		return fmt.Errorf("invalid config: weight must be strictly positive, weight: %f", c.Weight)
	}
	if c.HalfLife.Seconds() <= 0 {
		return fmt.Errorf("invalid config: half life time must be strictly positive, half life: %f", c.HalfLife.Seconds())
	}
	return nil
}

// String prints a Config's fields
func (c Config) String() string {
	return fmt.Sprintf("weight: %f, decay_half_life_time_sec: %f", c.Weight, c.HalfLife.Seconds())
}

// Option is used as a generic and elegant way to configure a new ScoreStore
type Option func(*Config)

func WithWeight(weight float64) Option {
	return func(c *Config) {
		c.Weight = weight
	}
}

func WithDecayHalfLife(halfLife time.Duration) Option {
	return func(c *Config) {
		c.HalfLife = halfLife
	}
}
