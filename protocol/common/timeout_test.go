package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// resetTimeoutGlobals restores the two configurable globals after each test
// so tests don't bleed into each other.
func resetTimeoutGlobals(t *testing.T, origMin, origDefault time.Duration) {
	t.Helper()
	t.Cleanup(func() {
		MinimumTimePerRelayDelay = origMin
		DefaultTimeout = origDefault
	})
}

// --- ValidateAndCapMinRelayTimeout ---

func TestValidateAndCapMinRelayTimeout(t *testing.T) {
	defaultReset := time.Duration(DefaultTimeoutSeconds) * time.Second

	tests := []struct {
		name        string
		inputMin    time.Duration
		inputDef    time.Duration
		wantMin     time.Duration
		wantDefault time.Duration
	}{
		{
			name:        "min < default: no change",
			inputMin:    5 * time.Second,
			inputDef:    30 * time.Second,
			wantMin:     5 * time.Second,
			wantDefault: 30 * time.Second,
		},
		{
			name:        "min == default: cap to 50%",
			inputMin:    30 * time.Second,
			inputDef:    30 * time.Second,
			wantMin:     15 * time.Second,
			wantDefault: 30 * time.Second,
		},
		{
			name:        "min > default: cap to 50%",
			inputMin:    60 * time.Second,
			inputDef:    30 * time.Second,
			wantMin:     15 * time.Second,
			wantDefault: 30 * time.Second,
		},
		{
			name:        "cap is exactly half of default",
			inputMin:    25 * time.Second,
			inputDef:    20 * time.Second,
			wantMin:     10 * time.Second,
			wantDefault: 20 * time.Second,
		},
		{
			name:        "zero default: resets to default constant, min unchanged",
			inputMin:    time.Second,
			inputDef:    0,
			wantMin:     time.Second,
			wantDefault: defaultReset,
		},
		{
			name:        "negative default: resets to default constant, min unchanged",
			inputMin:    time.Second,
			inputDef:    -5 * time.Second,
			wantMin:     time.Second,
			wantDefault: defaultReset,
		},
		{
			name:        "zero default then min > reset default: resets default then caps min",
			inputMin:    60 * time.Second,
			inputDef:    0,
			wantMin:     defaultReset / 2,
			wantDefault: defaultReset,
		},
		{
			name:        "very small default (1ns): cap rounds to zero, clamped to 1ms",
			inputMin:    time.Second,
			inputDef:    1, // 1 nanosecond
			wantMin:     time.Millisecond,
			wantDefault: 1,
		},
		{
			name:        "zero min: resets to 1s",
			inputMin:    0,
			inputDef:    30 * time.Second,
			wantMin:     time.Second,
			wantDefault: 30 * time.Second,
		},
		{
			name:        "negative min: resets to 1s",
			inputMin:    -time.Second,
			inputDef:    30 * time.Second,
			wantMin:     time.Second,
			wantDefault: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetTimeoutGlobals(t, MinimumTimePerRelayDelay, DefaultTimeout)
			MinimumTimePerRelayDelay = tt.inputMin
			DefaultTimeout = tt.inputDef

			ValidateAndCapMinRelayTimeout()

			require.Equal(t, tt.wantMin, MinimumTimePerRelayDelay)
			require.Equal(t, tt.wantDefault, DefaultTimeout)
		})
	}
}

// --- GetTimePerCu respects MinimumTimePerRelayDelay ---

func TestGetTimePerCu_DefaultFloor(t *testing.T) {
	resetTimeoutGlobals(t, MinimumTimePerRelayDelay, DefaultTimeout)

	MinimumTimePerRelayDelay = time.Second

	// cu=10 → 10×100ms = 1s == floor → returns floor
	require.Equal(t, time.Second, GetTimePerCu(10))
	// cu=5 → 500ms < floor → returns floor
	require.Equal(t, time.Second, GetTimePerCu(5))
	// cu=20 → 2s > floor → returns 2s
	require.Equal(t, 2*time.Second, GetTimePerCu(20))
}

func TestGetTimePerCu_RaisedFloor(t *testing.T) {
	resetTimeoutGlobals(t, MinimumTimePerRelayDelay, DefaultTimeout)

	MinimumTimePerRelayDelay = 5 * time.Second

	// cu=10 → 1s < new floor (5s) → returns 5s
	require.Equal(t, 5*time.Second, GetTimePerCu(10))
	// cu=20 → 2s < new floor (5s) → returns 5s
	require.Equal(t, 5*time.Second, GetTimePerCu(20))
	// cu=100 → 10s > new floor (5s) → returns 10s
	require.Equal(t, 10*time.Second, GetTimePerCu(100))
}

func TestGetTimePerCu_FloorAfterValidation(t *testing.T) {
	resetTimeoutGlobals(t, MinimumTimePerRelayDelay, DefaultTimeout)

	// Simulate: user sets --min-relay-timeout=40s --default-processing-timeout=30s
	DefaultTimeout = 30 * time.Second
	MinimumTimePerRelayDelay = 40 * time.Second
	ValidateAndCapMinRelayTimeout() // should cap to 15s

	// cu=10 → 1s < 15s → returns 15s
	require.Equal(t, 15*time.Second, GetTimePerCu(10))
	// cu=200 → 20s > 15s → returns 20s
	require.Equal(t, 20*time.Second, GetTimePerCu(200))
}
